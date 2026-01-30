// 数据源管理服务
// 负责数据源的管理、连接测试、元数据查询

mod mysql;
mod elasticsearch;

use crate::commands::datasource::{ConnectionResult, IndexMatchResult};
use crate::storage::models::{DataSource, DataSourceType};
use crate::storage::Storage;
use crate::utils::crypto::CryptoService;
use anyhow::Result;
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;

/// 数据源管理器
pub struct DataSourceManager {
    storage: Arc<Storage>,
    crypto: Arc<CryptoService>,
}

impl DataSourceManager {
    /// 创建新的数据源管理器
    pub fn new(storage: Arc<Storage>, crypto: Arc<CryptoService>) -> Self {
        Self { storage, crypto }
    }

    /// 获取所有数据源
    pub async fn list_datasources(&self) -> Result<Vec<DataSource>> {
        let mut datasources = self.storage.load_datasources().await?;

        // 解密密码
        for ds in &mut datasources {
            ds.password = self.crypto.decrypt(&ds.password)?;
        }

        Ok(datasources)
    }

    /// 获取单个数据源
    pub async fn get_datasource(&self, id: &str) -> Result<Option<DataSource>> {
        if let Some(mut ds) = self.storage.load_datasource(id).await? {
            ds.password = self.crypto.decrypt(&ds.password)?;
            Ok(Some(ds))
        } else {
            Ok(None)
        }
    }

    /// 创建数据源
    pub async fn create_datasource(
        &self,
        name: String,
        source_type: DataSourceType,
        host: String,
        port: u16,
        username: String,
        password: String,
        database: Option<String>,
    ) -> Result<String> {
        let id = Uuid::new_v4().to_string();
        let encrypted_password = self.crypto.encrypt(&password)?;

        let datasource = DataSource {
            id: id.clone(),
            name,
            source_type,
            host,
            port,
            username,
            password: encrypted_password,
            database,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };

        self.storage.save_datasource(&datasource).await?;

        Ok(id)
    }

    /// 更新数据源
    pub async fn update_datasource(&self, mut datasource: DataSource) -> Result<()> {
        // 如果密码不为空，重新加密
        if !datasource.password.is_empty() {
            datasource.password = self.crypto.encrypt(&datasource.password)?;
        }

        datasource.updated_at = Utc::now();
        self.storage.save_datasource(&datasource).await?;

        Ok(())
    }

    /// 删除数据源
    pub async fn delete_datasource(&self, id: &str) -> Result<()> {
        self.storage.delete_datasource(id).await
    }

    /// 测试连接
    pub async fn test_connection(&self, id: &str) -> Result<ConnectionResult> {
        let ds = self
            .get_datasource(id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在"))?;

        let start = std::time::Instant::now();

        let result = match ds.source_type {
            DataSourceType::Mysql => mysql::test_connection(&ds).await,
            DataSourceType::Elasticsearch => elasticsearch::test_connection(&ds).await,
        };

        let duration_ms = start.elapsed().as_millis() as u64;

        match result {
            Ok(_) => Ok(ConnectionResult {
                success: true,
                message: "连接成功".to_string(),
                duration_ms,
            }),
            Err(e) => Ok(ConnectionResult {
                success: false,
                message: e.to_string(),
                duration_ms,
            }),
        }
    }

    /// 获取 MySQL 数据库列表
    pub async fn get_databases(&self, id: &str) -> Result<Vec<String>> {
        let ds = self
            .get_datasource(id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在"))?;

        if ds.source_type != DataSourceType::Mysql {
            anyhow::bail!("数据源类型不是 MySQL");
        }

        mysql::get_databases(&ds).await
    }

    /// 获取 MySQL 表列表
    pub async fn get_tables(&self, id: &str, database: &str) -> Result<Vec<String>> {
        let ds = self
            .get_datasource(id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在"))?;

        if ds.source_type != DataSourceType::Mysql {
            anyhow::bail!("数据源类型不是 MySQL");
        }

        mysql::get_tables(&ds, database).await
    }

    /// 获取 ES 索引列表
    pub async fn get_indices(&self, id: &str) -> Result<Vec<String>> {
        let ds = self
            .get_datasource(id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在"))?;

        if ds.source_type != DataSourceType::Elasticsearch {
            anyhow::bail!("数据源类型不是 Elasticsearch");
        }

        elasticsearch::get_indices(&ds).await
    }

    /// 通配符匹配 ES 索引
    pub async fn match_indices(&self, id: &str, pattern: &str) -> Result<IndexMatchResult> {
        let ds = self
            .get_datasource(id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("数据源不存在"))?;

        if ds.source_type != DataSourceType::Elasticsearch {
            anyhow::bail!("数据源类型不是 Elasticsearch");
        }

        elasticsearch::match_indices(&ds, pattern).await
    }
}
