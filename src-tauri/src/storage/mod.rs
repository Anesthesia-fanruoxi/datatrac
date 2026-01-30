// 存储层模块
// 负责 SQLite 数据库操作和数据持久化

pub mod datasource;
pub mod models;
pub mod synced_index;
pub mod task;
pub mod task_config;
pub mod task_unit;

use anyhow::{Context, Result};
use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};

/// 存储层
/// 提供数据库连接和基础操作
pub struct Storage {
    pool: SqlitePool,
}

impl Storage {
    /// 创建新的存储实例
    pub async fn new(db_path: &str) -> Result<Self> {
        let connection_string = if db_path.contains(':') {
            format!("sqlite:///{}?mode=rwc", db_path)
        } else {
            format!("sqlite://{}?mode=rwc", db_path)
        };

        log::info!("SQLite 连接字符串: {}", connection_string);

        let pool = SqlitePoolOptions::new()
            .max_connections(5)
            .connect(&connection_string)
            .await
            .context("连接 SQLite 数据库失败")?;

        let storage = Self { pool };
        storage.init_schema().await?;

        Ok(storage)
    }

    /// 初始化数据库 schema
    async fn init_schema(&self) -> Result<()> {
        // 创建数据源表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS datasources (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                type TEXT NOT NULL CHECK(type IN ('mysql', 'elasticsearch')),
                host TEXT NOT NULL,
                port INTEGER NOT NULL,
                username TEXT NOT NULL,
                password TEXT NOT NULL,
                database TEXT,
                created_at INTEGER NOT NULL,
                updated_at INTEGER NOT NULL
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
                created_at INTEGER NOT NULL,
                updated_at INTEGER NOT NULL
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建同步任务表失败")?;

        // 创建任务单元表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS task_units (
                id TEXT PRIMARY KEY,
                task_id TEXT NOT NULL,
                unit_name TEXT NOT NULL,
                unit_type TEXT NOT NULL CHECK(unit_type IN ('table', 'index')),
                status TEXT NOT NULL CHECK(status IN ('pending', 'running', 'completed', 'failed')),
                total_records INTEGER DEFAULT 0,
                processed_records INTEGER DEFAULT 0,
                error_message TEXT,
                keyword TEXT,
                created_at INTEGER NOT NULL,
                updated_at INTEGER NOT NULL,
                UNIQUE(task_id, unit_name),
                FOREIGN KEY (task_id) REFERENCES sync_tasks(id) ON DELETE CASCADE
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建任务单元表失败")?;

        // 创建已同步索引表
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS synced_indices (
                id TEXT PRIMARY KEY,
                source_id TEXT NOT NULL,
                index_name TEXT NOT NULL,
                first_synced_at INTEGER NOT NULL,
                last_synced_at INTEGER NOT NULL,
                sync_count INTEGER DEFAULT 1,
                last_task_id TEXT,
                UNIQUE(source_id, index_name)
            )
            "#,
        )
        .execute(&self.pool)
        .await
        .context("创建已同步索引表失败")?;

        // 创建索引
        sqlx::query("CREATE INDEX IF NOT EXISTS idx_task_units_task_id ON task_units(task_id)")
            .execute(&self.pool)
            .await
            .context("创建任务单元索引失败")?;

        sqlx::query("CREATE INDEX IF NOT EXISTS idx_task_units_status ON task_units(status)")
            .execute(&self.pool)
            .await
            .context("创建任务单元状态索引失败")?;

        sqlx::query("CREATE INDEX IF NOT EXISTS idx_synced_indices_source ON synced_indices(source_id)")
            .execute(&self.pool)
            .await
            .context("创建已同步索引索引失败")?;

        sqlx::query("CREATE INDEX IF NOT EXISTS idx_synced_indices_name ON synced_indices(source_id, index_name)")
            .execute(&self.pool)
            .await
            .context("创建已同步索引名称索引失败")?;

        log::info!("数据库 schema 初始化完成");
        Ok(())
    }

    /// 获取数据库连接池
    pub fn pool(&self) -> &SqlitePool {
        &self.pool
    }
}
