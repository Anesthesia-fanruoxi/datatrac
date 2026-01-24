// 存储层模块
// 使用 SQLite 持久化数据源和同步任务配置

mod models;
mod data_source_ops;
mod sync_task_ops;

pub use models::{DataSource, DataSourceType, SyncTask, TaskStatus};

use anyhow::{Context, Result};
use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};

/// 存储层，负责配置的持久化
pub struct Storage {
    pool: SqlitePool,
}

impl Storage {
    /// 创建新的存储实例
    /// 
    /// # 参数
    /// * `db_path` - SQLite 数据库文件路径
    pub async fn new(db_path: &str) -> Result<Self> {
        // 创建连接池
        let pool = SqlitePoolOptions::new()
            .max_connections(5)
            .connect(&format!("sqlite:{}", db_path))
            .await
            .context("连接 SQLite 数据库失败")?;

        let storage = Self { pool };
        
        // 初始化数据库 schema
        storage.init_schema().await?;

        Ok(storage)
    }

    /// 初始化数据库 schema
    /// 
    /// 创建数据源表、同步任务表和加密密钥表
    pub async fn init_schema(&self) -> Result<()> {
        // 创建数据源表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS data_sources (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                type TEXT NOT NULL CHECK(type IN ('mysql', 'elasticsearch')),
                host TEXT NOT NULL,
                port INTEGER NOT NULL,
                username TEXT NOT NULL,
                password TEXT NOT NULL,
                database TEXT,
                created_at TEXT NOT NULL,
                updated_at TEXT NOT NULL
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建数据源表失败")?;

        // 创建同步任务表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS sync_tasks (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                source_id TEXT NOT NULL,
                target_id TEXT NOT NULL,
                source_type TEXT NOT NULL,
                target_type TEXT NOT NULL,
                config TEXT NOT NULL,
                status TEXT NOT NULL,
                created_at TEXT NOT NULL,
                updated_at TEXT NOT NULL,
                FOREIGN KEY (source_id) REFERENCES data_sources(id),
                FOREIGN KEY (target_id) REFERENCES data_sources(id)
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建同步任务表失败")?;

        // 创建加密密钥表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS encryption_keys (
                id INTEGER PRIMARY KEY,
                key_data TEXT NOT NULL,
                created_at TEXT NOT NULL
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建加密密钥表失败")?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Utc;
    use uuid::Uuid;

    async fn create_test_storage() -> Result<Storage> {
        // 使用内存数据库进行测试
        Storage::new(":memory:").await
    }

    fn create_test_data_source() -> DataSource {
        DataSource {
            id: Uuid::new_v4().to_string(),
            name: "测试数据源".to_string(),
            source_type: DataSourceType::Mysql,
            host: "localhost".to_string(),
            port: 3306,
            username: "root".to_string(),
            password: "encrypted_password".to_string(),
            database: Some("test_db".to_string()),
            created_at: Utc::now(),
            updated_at: Utc::now(),
        }
    }

    fn create_test_sync_task(source_id: &str, target_id: &str) -> SyncTask {
        SyncTask {
            id: Uuid::new_v4().to_string(),
            name: "测试同步任务".to_string(),
            source_id: source_id.to_string(),
            target_id: target_id.to_string(),
            source_type: DataSourceType::Mysql,
            target_type: DataSourceType::Elasticsearch,
            config: r#"{"threadCount":4,"batchSize":1000}"#.to_string(),
            status: TaskStatus::Idle,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        }
    }

    #[tokio::test]
    async fn test_save_and_load_data_source() {
        let storage = create_test_storage().await.unwrap();
        let ds = create_test_data_source();

        // 保存数据源
        storage.save_data_source(&ds).await.unwrap();

        // 加载数据源
        let loaded = storage.load_data_source(&ds.id).await.unwrap();
        assert!(loaded.is_some());

        let loaded = loaded.unwrap();
        assert_eq!(loaded.id, ds.id);
        assert_eq!(loaded.name, ds.name);
        assert_eq!(loaded.source_type, ds.source_type);
        assert_eq!(loaded.host, ds.host);
        assert_eq!(loaded.port, ds.port);
    }

    #[tokio::test]
    async fn test_load_all_data_sources() {
        let storage = create_test_storage().await.unwrap();
        
        // 保存多个数据源
        let ds1 = create_test_data_source();
        let ds2 = create_test_data_source();
        
        storage.save_data_source(&ds1).await.unwrap();
        storage.save_data_source(&ds2).await.unwrap();

        // 加载所有数据源
        let all = storage.load_data_sources().await.unwrap();
        assert_eq!(all.len(), 2);
    }

    #[tokio::test]
    async fn test_update_data_source() {
        let storage = create_test_storage().await.unwrap();
        let mut ds = create_test_data_source();

        // 保存数据源
        storage.save_data_source(&ds).await.unwrap();

        // 更新数据源
        ds.name = "更新后的名称".to_string();
        ds.updated_at = Utc::now();
        storage.save_data_source(&ds).await.unwrap();

        // 验证更新
        let loaded = storage.load_data_source(&ds.id).await.unwrap().unwrap();
        assert_eq!(loaded.name, "更新后的名称");
        assert_eq!(loaded.id, ds.id); // ID 应该保持不变
    }

    #[tokio::test]
    async fn test_delete_data_source() {
        let storage = create_test_storage().await.unwrap();
        let ds = create_test_data_source();

        // 保存数据源
        storage.save_data_source(&ds).await.unwrap();

        // 删除数据源
        storage.delete_data_source(&ds.id).await.unwrap();

        // 验证删除
        let loaded = storage.load_data_source(&ds.id).await.unwrap();
        assert!(loaded.is_none());
    }

    #[tokio::test]
    async fn test_delete_nonexistent_data_source() {
        let storage = create_test_storage().await.unwrap();
        
        // 删除不存在的数据源应该返回错误
        let result = storage.delete_data_source("nonexistent_id").await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_save_and_load_sync_task() {
        let storage = create_test_storage().await.unwrap();
        
        // 先创建数据源
        let source = create_test_data_source();
        let target = create_test_data_source();
        storage.save_data_source(&source).await.unwrap();
        storage.save_data_source(&target).await.unwrap();

        // 创建同步任务
        let task = create_test_sync_task(&source.id, &target.id);
        storage.save_task(&task).await.unwrap();

        // 加载任务
        let loaded = storage.load_task(&task.id).await.unwrap();
        assert!(loaded.is_some());

        let loaded = loaded.unwrap();
        assert_eq!(loaded.id, task.id);
        assert_eq!(loaded.name, task.name);
        assert_eq!(loaded.source_id, task.source_id);
        assert_eq!(loaded.target_id, task.target_id);
    }

    #[tokio::test]
    async fn test_load_all_tasks() {
        let storage = create_test_storage().await.unwrap();
        
        // 创建数据源
        let source = create_test_data_source();
        let target = create_test_data_source();
        storage.save_data_source(&source).await.unwrap();
        storage.save_data_source(&target).await.unwrap();

        // 创建多个任务
        let task1 = create_test_sync_task(&source.id, &target.id);
        let task2 = create_test_sync_task(&source.id, &target.id);
        
        storage.save_task(&task1).await.unwrap();
        storage.save_task(&task2).await.unwrap();

        // 加载所有任务
        let all = storage.load_tasks().await.unwrap();
        assert_eq!(all.len(), 2);
    }

    #[tokio::test]
    async fn test_update_sync_task() {
        let storage = create_test_storage().await.unwrap();
        
        // 创建数据源
        let source = create_test_data_source();
        let target = create_test_data_source();
        storage.save_data_source(&source).await.unwrap();
        storage.save_data_source(&target).await.unwrap();

        // 创建任务
        let mut task = create_test_sync_task(&source.id, &target.id);
        storage.save_task(&task).await.unwrap();

        // 更新任务
        task.status = TaskStatus::Running;
        task.updated_at = Utc::now();
        storage.save_task(&task).await.unwrap();

        // 验证更新
        let loaded = storage.load_task(&task.id).await.unwrap().unwrap();
        assert_eq!(loaded.status, TaskStatus::Running);
        assert_eq!(loaded.id, task.id); // ID 应该保持不变
    }

    #[tokio::test]
    async fn test_delete_sync_task() {
        let storage = create_test_storage().await.unwrap();
        
        // 创建数据源
        let source = create_test_data_source();
        let target = create_test_data_source();
        storage.save_data_source(&source).await.unwrap();
        storage.save_data_source(&target).await.unwrap();

        // 创建任务
        let task = create_test_sync_task(&source.id, &target.id);
        storage.save_task(&task).await.unwrap();

        // 删除任务
        storage.delete_task(&task.id).await.unwrap();

        // 验证删除
        let loaded = storage.load_task(&task.id).await.unwrap();
        assert!(loaded.is_none());
    }

    #[tokio::test]
    async fn test_delete_nonexistent_task() {
        let storage = create_test_storage().await.unwrap();
        
        // 删除不存在的任务应该返回错误
        let result = storage.delete_task("nonexistent_id").await;
        assert!(result.is_err());
    }
}
