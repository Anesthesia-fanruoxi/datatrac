// 数据源管理器模块
// 负责数据源的 CRUD 操作、连接测试和元数据查询

use crate::crypto::CryptoService;
use crate::storage::{DataSource, DataSourceType, Storage};
use anyhow::{Context, Result};
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;

/// 连接测试结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConnectionResult {
    pub success: bool,
    pub message: String,
    pub details: Option<String>,
}

/// 索引匹配结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct IndexMatchResult {
    pub pattern: String,
    pub count: usize,
    pub preview: Vec<String>, // 前 10 个匹配的索引
}

/// 数据源管理器
/// 
/// 负责：
/// - 数据源配置的增删改查
/// - 密码的自动加密/解密
/// - 数据库连接测试
/// - 元数据查询（数据库、表、索引）
pub struct DataSourceManager {
    storage: Arc<Storage>,
    crypto: Arc<CryptoService>,
}

impl DataSourceManager {
    /// 创建新的数据源管理器实例
    pub fn new(storage: Arc<Storage>, crypto: Arc<CryptoService>) -> Self {
        Self { storage, crypto }
    }

    /// 获取 Storage 引用（用于任务管理）
    pub fn storage(&self) -> &Arc<Storage> {
        &self.storage
    }

    /// 获取所有数据源列表
    /// 
    /// 返回的数据源中，密码字段已解密
    pub async fn list_data_sources(&self) -> Result<Vec<DataSource>> {
        let mut data_sources = self.storage.load_data_sources().await?;
        
        // 解密所有密码
        for ds in &mut data_sources {
            ds.password = self.crypto.decrypt(&ds.password)
                .context(format!("解密数据源 {} 的密码失败", ds.name))?;
        }
        
        Ok(data_sources)
    }

    /// 根据 ID 获取单个数据源
    /// 
    /// 返回的数据源中，密码字段已解密
    pub async fn get_data_source(&self, id: &str) -> Result<Option<DataSource>> {
        if let Some(mut ds) = self.storage.load_data_source(id).await? {
            ds.password = self.crypto.decrypt(&ds.password)
                .context(format!("解密数据源 {} 的密码失败", ds.name))?;
            Ok(Some(ds))
        } else {
            Ok(None)
        }
    }

    /// 创建新数据源
    /// 
    /// 自动生成 ID，加密密码后保存
    /// 返回新创建的数据源 ID
    pub async fn create_data_source(&self, mut ds: DataSource) -> Result<String> {
        // 生成新 ID
        ds.id = Uuid::new_v4().to_string();
        
        // 设置时间戳
        let now = Utc::now();
        ds.created_at = now;
        ds.updated_at = now;
        
        // 加密密码
        let encrypted_password = self.crypto.encrypt(&ds.password)
            .context("加密密码失败")?;
        ds.password = encrypted_password;
        
        // 保存到存储
        self.storage.save_data_source(&ds).await?;
        
        Ok(ds.id)
    }

    /// 更新数据源
    /// 
    /// 保持原有 ID 和创建时间，更新其他字段
    /// 如果密码字段不为空，则重新加密
    pub async fn update_data_source(&self, id: &str, mut ds: DataSource) -> Result<()> {
        // 加载原有数据源以获取创建时间
        let original = self.storage.load_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        // 保持原有 ID 和创建时间
        ds.id = id.to_string();
        ds.created_at = original.created_at;
        ds.updated_at = Utc::now();
        
        // 加密密码
        let encrypted_password = self.crypto.encrypt(&ds.password)
            .context("加密密码失败")?;
        ds.password = encrypted_password;
        
        // 保存到存储
        self.storage.save_data_source(&ds).await?;
        
        Ok(())
    }

    /// 删除数据源
    pub async fn delete_data_source(&self, id: &str) -> Result<()> {
        self.storage.delete_data_source(id).await
    }

    /// 测试数据源连接
    /// 
    /// 根据数据源类型调用相应的连接测试方法
    pub async fn test_connection(&self, id: &str) -> Result<ConnectionResult> {
        let ds = self.get_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        match ds.source_type {
            DataSourceType::Mysql => self.test_mysql_connection(&ds).await,
            DataSourceType::Elasticsearch => self.test_elasticsearch_connection(&ds).await,
        }
    }

    /// 测试 MySQL 连接
    async fn test_mysql_connection(&self, ds: &DataSource) -> Result<ConnectionResult> {
        use sqlx::mysql::MySqlPoolOptions;
        
        // 构建连接字符串
        let connection_string = if let Some(database) = &ds.database {
            format!(
                "mysql://{}:{}@{}:{}/{}",
                ds.username, ds.password, ds.host, ds.port, database
            )
        } else {
            format!(
                "mysql://{}:{}@{}:{}",
                ds.username, ds.password, ds.host, ds.port
            )
        };
        
        // 尝试连接
        match MySqlPoolOptions::new()
            .max_connections(1)
            .connect(&connection_string)
            .await
        {
            Ok(pool) => {
                // 执行简单查询验证连接
                match sqlx::query("SELECT 1").fetch_one(&pool).await {
                    Ok(_) => Ok(ConnectionResult {
                        success: true,
                        message: "连接成功".to_string(),
                        details: Some(format!("成功连接到 MySQL 服务器 {}:{}", ds.host, ds.port)),
                    }),
                    Err(e) => Ok(ConnectionResult {
                        success: false,
                        message: "查询失败".to_string(),
                        details: Some(format!("连接成功但查询失败: {}", e)),
                    }),
                }
            }
            Err(e) => Ok(ConnectionResult {
                success: false,
                message: "连接失败".to_string(),
                details: Some(format!("无法连接到 MySQL 服务器: {}", e)),
            }),
        }
    }

    /// 测试 Elasticsearch 连接
    async fn test_elasticsearch_connection(&self, ds: &DataSource) -> Result<ConnectionResult> {
        use elasticsearch::{Elasticsearch, http::transport::Transport};
        
        // 构建 URL
        let url = format!("http://{}:{}", ds.host, ds.port);
        
        // 创建传输层
        let transport = match Transport::single_node(&url) {
            Ok(t) => t,
            Err(e) => {
                return Ok(ConnectionResult {
                    success: false,
                    message: "创建连接失败".to_string(),
                    details: Some(format!("无法创建 Elasticsearch 传输层: {}", e)),
                });
            }
        };
        
        let client = Elasticsearch::new(transport);
        
        // 尝试 ping
        match client.ping().send().await {
            Ok(response) => {
                if response.status_code().is_success() {
                    Ok(ConnectionResult {
                        success: true,
                        message: "连接成功".to_string(),
                        details: Some(format!("成功连接到 Elasticsearch 服务器 {}:{}", ds.host, ds.port)),
                    })
                } else {
                    Ok(ConnectionResult {
                        success: false,
                        message: "连接失败".to_string(),
                        details: Some(format!("Elasticsearch 返回错误状态码: {}", response.status_code())),
                    })
                }
            }
            Err(e) => Ok(ConnectionResult {
                success: false,
                message: "连接失败".to_string(),
                details: Some(format!("无法连接到 Elasticsearch 服务器: {}", e)),
            }),
        }
    }

    /// 获取 MySQL 数据库列表
    /// 
    /// # 参数
    /// * `id` - 数据源 ID
    pub async fn get_databases(&self, id: &str) -> Result<Vec<String>> {
        let ds = self.get_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        if ds.source_type != DataSourceType::Mysql {
            anyhow::bail!("只有 MySQL 数据源支持获取数据库列表");
        }
        
        use sqlx::mysql::MySqlPoolOptions;
        use sqlx::Row;
        
        // 构建连接字符串（不指定数据库）
        let connection_string = format!(
            "mysql://{}:{}@{}:{}",
            ds.username, ds.password, ds.host, ds.port
        );
        
        let pool = MySqlPoolOptions::new()
            .max_connections(1)
            .connect(&connection_string)
            .await
            .context("连接 MySQL 失败")?;
        
        // 查询数据库列表
        let rows = sqlx::query("SHOW DATABASES")
            .fetch_all(&pool)
            .await
            .context("查询数据库列表失败")?;
        
        let databases: Vec<String> = rows
            .iter()
            .filter_map(|row| row.try_get::<String, _>(0).ok())
            .collect();
        
        Ok(databases)
    }

    /// 获取 MySQL 指定数据库的表列表
    /// 
    /// # 参数
    /// * `id` - 数据源 ID
    /// * `database` - 数据库名称
    pub async fn get_tables(&self, id: &str, database: &str) -> Result<Vec<String>> {
        let ds = self.get_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        if ds.source_type != DataSourceType::Mysql {
            anyhow::bail!("只有 MySQL 数据源支持获取表列表");
        }
        
        use sqlx::mysql::MySqlPoolOptions;
        use sqlx::Row;
        
        // 构建连接字符串（指定数据库）
        let connection_string = format!(
            "mysql://{}:{}@{}:{}/{}",
            ds.username, ds.password, ds.host, ds.port, database
        );
        
        let pool = MySqlPoolOptions::new()
            .max_connections(1)
            .connect(&connection_string)
            .await
            .context("连接 MySQL 失败")?;
        
        // 查询表列表
        let rows = sqlx::query("SHOW TABLES")
            .fetch_all(&pool)
            .await
            .context("查询表列表失败")?;
        
        let tables: Vec<String> = rows
            .iter()
            .filter_map(|row| row.try_get::<String, _>(0).ok())
            .collect();
        
        Ok(tables)
    }

    /// 获取 Elasticsearch 索引列表
    /// 
    /// # 参数
    /// * `id` - 数据源 ID
    pub async fn get_indices(&self, id: &str) -> Result<Vec<String>> {
        let ds = self.get_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        if ds.source_type != DataSourceType::Elasticsearch {
            anyhow::bail!("只有 Elasticsearch 数据源支持获取索引列表");
        }
        
        use elasticsearch::{Elasticsearch, http::transport::Transport, cat::CatIndicesParts};
        
        let url = format!("http://{}:{}", ds.host, ds.port);
        let transport = Transport::single_node(&url)
            .context("创建 Elasticsearch 传输层失败")?;
        let client = Elasticsearch::new(transport);
        
        // 获取索引列表
        let response = client
            .cat()
            .indices(CatIndicesParts::None)
            .format("json")
            .send()
            .await
            .context("查询索引列表失败")?;
        
        let body = response.json::<Vec<serde_json::Value>>().await
            .context("解析索引列表响应失败")?;
        
        let indices: Vec<String> = body
            .iter()
            .filter_map(|item| {
                item.get("index")
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string())
            })
            .collect();
        
        Ok(indices)
    }

    /// 通配符匹配 Elasticsearch 索引
    /// 
    /// # 参数
    /// * `id` - 数据源 ID
    /// * `pattern` - 通配符模式（例如：logs-*）
    pub async fn match_indices(&self, id: &str, pattern: &str) -> Result<IndexMatchResult> {
        let all_indices = self.get_indices(id).await?;
        
        // 将通配符模式转换为正则表达式
        let regex_pattern = pattern
            .replace(".", "\\.")
            .replace("*", ".*")
            .replace("?", ".");
        
        let regex = regex::Regex::new(&format!("^{}$", regex_pattern))
            .context("无效的通配符模式")?;
        
        // 匹配索引
        let matched: Vec<String> = all_indices
            .into_iter()
            .filter(|index| regex.is_match(index))
            .collect();
        
        let count = matched.len();
        let preview = matched.iter().take(10).cloned().collect();
        
        Ok(IndexMatchResult {
            pattern: pattern.to_string(),
            count,
            preview,
        })
    }
}

