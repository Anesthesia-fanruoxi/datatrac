// 数据源管理器模块
// 负责数据源的 CRUD 操作、连接测试和元数据查询

use crate::crypto::CryptoService;
use crate::storage::{DataSource, DataSourceType, Storage};
use anyhow::{Context, Result};
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;
use tauri::Emitter;

/// 连接测试结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConnectionResult {
    pub success: bool,
    pub message: String,
    pub details: Option<String>,
    pub steps: Option<Vec<ConnectionTestStep>>,
}

/// 连接测试步骤
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConnectionTestStep {
    pub step: usize,
    pub name: String,
    pub success: bool,
    pub message: String,
    pub duration: Option<u64>, // 耗时（毫秒）
}

/// 批量测试单个数据源的结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct BatchTestDataSourceResult {
    #[serde(rename = "dataSourceId")]
    pub data_source_id: String,
    #[serde(rename = "dataSourceName")]
    pub data_source_name: String,
    pub steps: Vec<ConnectionTestStep>,
    pub success: bool,
}

/// 批量测试结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct BatchTestResult {
    pub total: usize,
    pub success: usize,
    pub failed: usize,
    pub results: Vec<BatchTestDataSourceResult>,
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

    /// 测试数据源连接（带事件发送）
    /// 
    /// 根据数据源类型调用相应的连接测试方法
    pub async fn test_connection_with_events(
        &self,
        id: &str,
        window: Option<tauri::Window>,
    ) -> Result<ConnectionResult> {
        let ds = self.get_data_source(id).await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在: {}", id))?;
        
        match ds.source_type {
            DataSourceType::Mysql => self.test_mysql_connection_with_events(&ds, window).await,
            DataSourceType::Elasticsearch => self.test_elasticsearch_connection_with_events(&ds, window).await,
        }
    }
    
    /// 测试数据源连接（兼容旧接口）
    /// 
    /// 根据数据源类型调用相应的连接测试方法
    pub async fn test_connection(&self, id: &str) -> Result<ConnectionResult> {
        self.test_connection_with_events(id, None).await
    }

    /// 测试 MySQL 连接（带事件发送）
    pub async fn test_mysql_connection_with_events(
        &self,
        ds: &DataSource,
        window: Option<tauri::Window>,
    ) -> Result<ConnectionResult> {
        use sqlx::mysql::MySqlPoolOptions;
        use std::time::Instant;
        use std::net::TcpStream;
        use std::time::Duration;
        
        let mut steps = Vec::new();
        
        // 步骤 1: 测试端口连通性
        let step1_start = Instant::now();
        let port_test = TcpStream::connect_timeout(
            &format!("{}:{}", ds.host, ds.port).parse().unwrap(),
            Duration::from_secs(5)
        );
        let step1_duration = step1_start.elapsed().as_millis() as u64;
        
        let step1_success = port_test.is_ok();
        let step1 = ConnectionTestStep {
            step: 1,
            name: "测试端口连通性".to_string(),
            success: step1_success,
            message: if step1_success {
                format!("端口 {} 连通", ds.port)
            } else {
                format!("端口 {} 无法连接", ds.port)
            },
            duration: Some(step1_duration),
        };
        steps.push(step1.clone());
        
        // 发送步骤1事件
        if let Some(ref win) = window {
            let _ = win.emit("connection-test-step", &step1);
        }
        
        if !step1_success {
            return Ok(ConnectionResult {
                success: false,
                message: "端口连通性测试失败".to_string(),
                details: Some(format!("无法连接到 {}:{}", ds.host, ds.port)),
                steps: Some(steps),
            });
        }
        
        // 等待1秒
        tokio::time::sleep(Duration::from_secs(1)).await;
        
        // 步骤 2: 验证账号密码
        let step2_start = Instant::now();
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
        
        let connect_result = MySqlPoolOptions::new()
            .max_connections(1)
            .acquire_timeout(Duration::from_secs(10))
            .connect(&connection_string)
            .await;
        
        let step2_duration = step2_start.elapsed().as_millis() as u64;
        
        let (step2, result) = match connect_result {
            Ok(pool) => {
                // 执行简单查询验证连接
                match sqlx::query("SELECT 1").fetch_one(&pool).await {
                    Ok(_) => {
                        let step2 = ConnectionTestStep {
                            step: 2,
                            name: "验证账号密码".to_string(),
                            success: true,
                            message: "账号密码验证成功".to_string(),
                            duration: Some(step2_duration),
                        };
                        
                        let result = ConnectionResult {
                            success: true,
                            message: "连接测试成功".to_string(),
                            details: Some(format!("成功连接到 MySQL 服务器 {}:{}", ds.host, ds.port)),
                            steps: Some(vec![steps[0].clone(), step2.clone()]),
                        };
                        
                        (step2, result)
                    }
                    Err(e) => {
                        let step2 = ConnectionTestStep {
                            step: 2,
                            name: "验证账号密码".to_string(),
                            success: false,
                            message: format!("查询失败: {}", e),
                            duration: Some(step2_duration),
                        };
                        
                        let result = ConnectionResult {
                            success: false,
                            message: "查询失败".to_string(),
                            details: Some(format!("连接成功但查询失败: {}", e)),
                            steps: Some(vec![steps[0].clone(), step2.clone()]),
                        };
                        
                        (step2, result)
                    }
                }
            }
            Err(e) => {
                let error_msg = e.to_string();
                let step2_message = if error_msg.contains("Access denied") {
                    "账号或密码错误".to_string()
                } else if error_msg.contains("Unknown database") {
                    "数据库不存在".to_string()
                } else {
                    format!("连接失败: {}", error_msg)
                };
                
                let step2 = ConnectionTestStep {
                    step: 2,
                    name: "验证账号密码".to_string(),
                    success: false,
                    message: step2_message.clone(),
                    duration: Some(step2_duration),
                };
                
                let result = ConnectionResult {
                    success: false,
                    message: "账号密码验证失败".to_string(),
                    details: Some(step2_message),
                    steps: Some(vec![steps[0].clone(), step2.clone()]),
                };
                
                (step2, result)
            }
        };
        
        // 发送步骤2事件
        if let Some(ref win) = window {
            let _ = win.emit("connection-test-step", &step2);
        }
        
        Ok(result)
    }
    
    /// 测试 MySQL 连接（兼容旧接口）
    #[allow(dead_code)]
    async fn test_mysql_connection(&self, ds: &DataSource) -> Result<ConnectionResult> {
        self.test_mysql_connection_with_events(ds, None).await
    }

    /// 测试 Elasticsearch 连接（带事件发送）
    pub async fn test_elasticsearch_connection_with_events(
        &self,
        ds: &DataSource,
        window: Option<tauri::Window>,
    ) -> Result<ConnectionResult> {
        use elasticsearch::{Elasticsearch, http::transport::Transport};
        use std::time::Instant;
        use std::net::TcpStream;
        use std::time::Duration;
        
        let mut steps = Vec::new();
        
        // 步骤 1: 测试端口连通性
        let step1_start = Instant::now();
        let port_test = TcpStream::connect_timeout(
            &format!("{}:{}", ds.host, ds.port).parse().unwrap(),
            Duration::from_secs(5)
        );
        let step1_duration = step1_start.elapsed().as_millis() as u64;
        
        let step1_success = port_test.is_ok();
        let step1 = ConnectionTestStep {
            step: 1,
            name: "测试端口连通性".to_string(),
            success: step1_success,
            message: if step1_success {
                format!("端口 {} 连通", ds.port)
            } else {
                format!("端口 {} 无法连接", ds.port)
            },
            duration: Some(step1_duration),
        };
        steps.push(step1.clone());
        
        // 发送步骤1事件
        if let Some(ref win) = window {
            let _ = win.emit("connection-test-step", &step1);
        }
        
        if !step1_success {
            return Ok(ConnectionResult {
                success: false,
                message: "端口连通性测试失败".to_string(),
                details: Some(format!("无法连接到 {}:{}", ds.host, ds.port)),
                steps: Some(steps),
            });
        }
        
        // 等待1秒
        tokio::time::sleep(Duration::from_secs(1)).await;
        
        // 步骤 2: 验证 Elasticsearch 服务
        let step2_start = Instant::now();
        let url = format!("http://{}:{}", ds.host, ds.port);
        
        let transport = match Transport::single_node(&url) {
            Ok(t) => t,
            Err(e) => {
                let step2_duration = step2_start.elapsed().as_millis() as u64;
                let step2 = ConnectionTestStep {
                    step: 2,
                    name: "验证 Elasticsearch 服务".to_string(),
                    success: false,
                    message: format!("创建连接失败: {}", e),
                    duration: Some(step2_duration),
                };
                
                // 发送步骤2事件
                if let Some(ref win) = window {
                    let _ = win.emit("connection-test-step", &step2);
                }
                
                return Ok(ConnectionResult {
                    success: false,
                    message: "创建连接失败".to_string(),
                    details: Some(format!("无法创建 Elasticsearch 传输层: {}", e)),
                    steps: Some(vec![steps[0].clone(), step2]),
                });
            }
        };
        
        let client = Elasticsearch::new(transport);
        
        // 尝试 ping
        let (step2, result) = match client.ping().send().await {
            Ok(response) => {
                let step2_duration = step2_start.elapsed().as_millis() as u64;
                
                if response.status_code().is_success() {
                    let step2 = ConnectionTestStep {
                        step: 2,
                        name: "验证 Elasticsearch 服务".to_string(),
                        success: true,
                        message: "Elasticsearch 服务正常".to_string(),
                        duration: Some(step2_duration),
                    };
                    
                    let result = ConnectionResult {
                        success: true,
                        message: "连接测试成功".to_string(),
                        details: Some(format!("成功连接到 Elasticsearch 服务器 {}:{}", ds.host, ds.port)),
                        steps: Some(vec![steps[0].clone(), step2.clone()]),
                    };
                    
                    (step2, result)
                } else {
                    let step2 = ConnectionTestStep {
                        step: 2,
                        name: "验证 Elasticsearch 服务".to_string(),
                        success: false,
                        message: format!("返回错误状态码: {}", response.status_code()),
                        duration: Some(step2_duration),
                    };
                    
                    let result = ConnectionResult {
                        success: false,
                        message: "连接失败".to_string(),
                        details: Some(format!("Elasticsearch 返回错误状态码: {}", response.status_code())),
                        steps: Some(vec![steps[0].clone(), step2.clone()]),
                    };
                    
                    (step2, result)
                }
            }
            Err(e) => {
                let step2_duration = step2_start.elapsed().as_millis() as u64;
                let step2 = ConnectionTestStep {
                    step: 2,
                    name: "验证 Elasticsearch 服务".to_string(),
                    success: false,
                    message: format!("连接失败: {}", e),
                    duration: Some(step2_duration),
                };
                
                let result = ConnectionResult {
                    success: false,
                    message: "连接失败".to_string(),
                    details: Some(format!("无法连接到 Elasticsearch 服务器: {}", e)),
                    steps: Some(vec![steps[0].clone(), step2.clone()]),
                };
                
                (step2, result)
            }
        };
        
        // 发送步骤2事件
        if let Some(ref win) = window {
            let _ = win.emit("connection-test-step", &step2);
        }
        
        Ok(result)
    }
    
    /// 测试 Elasticsearch 连接（兼容旧接口）
    #[allow(dead_code)]
    async fn test_elasticsearch_connection(&self, ds: &DataSource) -> Result<ConnectionResult> {
        self.test_elasticsearch_connection_with_events(ds, None).await
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

    /// 批量测试所有数据源连接
    /// 
    /// 分两个步骤并发执行:
    /// 1. 测试端口连通性
    /// 2. 验证账号密码(只测试步骤1成功的)
    pub async fn batch_test_connections(
        &self,
        window: Option<tauri::Window>,
        skip_failed_step1: bool,
    ) -> Result<BatchTestResult> {
        // 获取所有数据源
        let data_sources = self.list_data_sources().await?;
        let total = data_sources.len();
        
        if total == 0 {
            return Ok(BatchTestResult {
                total: 0,
                success: 0,
                failed: 0,
                results: vec![],
            });
        }
        
        let mut results = Vec::new();
        
        // 步骤1: 并发测试端口连通性
        let mut step1_tasks = Vec::new();
        for ds in &data_sources {
            let ds_clone = ds.clone();
            step1_tasks.push(tokio::spawn(async move {
                Self::test_port_connectivity(&ds_clone).await
            }));
        }
        
        // 等待所有任务完成
        let mut step1_results = Vec::new();
        for task in step1_tasks {
            step1_results.push(task.await);
        }
        
        // 收集步骤1结果并发送事件
        let mut step1_success_ids = Vec::new();
        for (i, result) in step1_results.into_iter().enumerate() {
            let ds = &data_sources[i];
            match result {
                Ok(Ok(step)) => {
                    if step.success {
                        step1_success_ids.push(ds.id.clone());
                    }
                    
                    // 发送步骤1事件
                    if let Some(ref win) = window {
                        let _ = win.emit("batch-test-step", &BatchTestDataSourceResult {
                            data_source_id: ds.id.clone(),
                            data_source_name: ds.name.clone(),
                            steps: vec![step.clone()],
                            success: step.success,
                        });
                    }
                    
                    results.push(BatchTestDataSourceResult {
                        data_source_id: ds.id.clone(),
                        data_source_name: ds.name.clone(),
                        steps: vec![step],
                        success: false, // 暂时标记为false,等步骤2完成后更新
                    });
                }
                _ => {
                    // 测试失败
                    let step = ConnectionTestStep {
                        step: 1,
                        name: "测试端口连通性".to_string(),
                        success: false,
                        message: "测试失败".to_string(),
                        duration: None,
                    };
                    
                    if let Some(ref win) = window {
                        let _ = win.emit("batch-test-step", &BatchTestDataSourceResult {
                            data_source_id: ds.id.clone(),
                            data_source_name: ds.name.clone(),
                            steps: vec![step.clone()],
                            success: false,
                        });
                    }
                    
                    results.push(BatchTestDataSourceResult {
                        data_source_id: ds.id.clone(),
                        data_source_name: ds.name.clone(),
                        steps: vec![step],
                        success: false,
                    });
                }
            }
        }
        
        // 如果步骤1有失败且不跳过,返回结果
        if !skip_failed_step1 && step1_success_ids.len() < total {
            let success_count = results.iter().filter(|r| r.steps[0].success).count();
            return Ok(BatchTestResult {
                total,
                success: success_count,
                failed: total - success_count,
                results,
            });
        }
        
        // 等待1秒
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
        
        // 步骤2: 并发验证账号密码(只测试步骤1成功的)
        let mut step2_tasks = Vec::new();
        for ds in &data_sources {
            if step1_success_ids.contains(&ds.id) {
                let ds_clone = ds.clone();
                step2_tasks.push(Some(tokio::spawn(async move {
                    Self::test_credentials(&ds_clone).await
                })));
            } else {
                step2_tasks.push(None);
            }
        }
        
        // 等待所有任务完成
        let mut step2_results = Vec::new();
        for task in step2_tasks {
            match task {
                Some(t) => step2_results.push(Some(t.await)),
                None => step2_results.push(None),
            }
        }
        
        // 更新步骤2结果
        let mut success_count = 0;
        for (i, result) in step2_results.into_iter().enumerate() {
            if let Some(result) = result {
                match result {
                    Ok(Ok(step)) => {
                        results[i].steps.push(step.clone());
                        results[i].success = step.success;
                        
                        if step.success {
                            success_count += 1;
                        }
                        
                        // 发送步骤2事件
                        if let Some(ref win) = window {
                            let _ = win.emit("batch-test-step", &results[i]);
                        }
                    }
                    _ => {
                        let step = ConnectionTestStep {
                            step: 2,
                            name: "验证账号密码".to_string(),
                            success: false,
                            message: "测试失败".to_string(),
                            duration: None,
                        };
                        results[i].steps.push(step);
                        results[i].success = false;
                        
                        if let Some(ref win) = window {
                            let _ = win.emit("batch-test-step", &results[i]);
                        }
                    }
                }
            }
        }
        
        Ok(BatchTestResult {
            total,
            success: success_count,
            failed: total - success_count,
            results,
        })
    }
    
    /// 测试端口连通性(内部方法)
    async fn test_port_connectivity(ds: &DataSource) -> Result<ConnectionTestStep> {
        use std::net::TcpStream;
        use std::time::{Duration, Instant};
        
        let start = Instant::now();
        let result = TcpStream::connect_timeout(
            &format!("{}:{}", ds.host, ds.port).parse().unwrap(),
            Duration::from_secs(5)
        );
        let duration = start.elapsed().as_millis() as u64;
        
        Ok(ConnectionTestStep {
            step: 1,
            name: "测试端口连通性".to_string(),
            success: result.is_ok(),
            message: if result.is_ok() {
                format!("端口 {} 连通", ds.port)
            } else {
                format!("端口 {} 无法连接", ds.port)
            },
            duration: Some(duration),
        })
    }
    
    /// 验证账号密码(内部方法)
    async fn test_credentials(ds: &DataSource) -> Result<ConnectionTestStep> {
        use std::time::Instant;
        
        let start = Instant::now();
        
        let (success, message) = match ds.source_type {
            DataSourceType::Mysql => {
                use sqlx::mysql::MySqlPoolOptions;
                use std::time::Duration;
                
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
                
                match MySqlPoolOptions::new()
                    .max_connections(1)
                    .acquire_timeout(Duration::from_secs(10))
                    .connect(&connection_string)
                    .await
                {
                    Ok(pool) => {
                        match sqlx::query("SELECT 1").fetch_one(&pool).await {
                            Ok(_) => (true, "账号密码验证成功".to_string()),
                            Err(e) => (false, format!("查询失败: {}", e)),
                        }
                    }
                    Err(e) => {
                        let error_msg = e.to_string();
                        let message = if error_msg.contains("Access denied") {
                            "账号或密码错误".to_string()
                        } else if error_msg.contains("Unknown database") {
                            "数据库不存在".to_string()
                        } else {
                            format!("连接失败: {}", error_msg)
                        };
                        (false, message)
                    }
                }
            }
            DataSourceType::Elasticsearch => {
                use elasticsearch::{Elasticsearch, http::transport::Transport};
                
                let url = format!("http://{}:{}", ds.host, ds.port);
                match Transport::single_node(&url) {
                    Ok(transport) => {
                        let client = Elasticsearch::new(transport);
                        match client.ping().send().await {
                            Ok(response) => {
                                if response.status_code().is_success() {
                                    (true, "Elasticsearch 服务正常".to_string())
                                } else {
                                    (false, format!("返回错误状态码: {}", response.status_code()))
                                }
                            }
                            Err(e) => (false, format!("连接失败: {}", e)),
                        }
                    }
                    Err(e) => (false, format!("创建连接失败: {}", e)),
                }
            }
        };
        
        let duration = start.elapsed().as_millis() as u64;
        
        Ok(ConnectionTestStep {
            step: 2,
            name: "验证账号密码".to_string(),
            success,
            message,
            duration: Some(duration),
        })
    }
}


