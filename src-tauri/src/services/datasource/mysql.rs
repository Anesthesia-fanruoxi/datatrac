// MySQL 数据源操作
// 提供 MySQL 连接测试和元数据查询

use crate::storage::models::DataSource;
use anyhow::Result;
use sqlx::mysql::{MySqlConnectOptions, MySqlPool, MySqlPoolOptions};
use sqlx::Row;
use std::str::FromStr;

/// 创建 MySQL 连接池
async fn create_pool(ds: &DataSource) -> Result<MySqlPool> {
    let options = MySqlConnectOptions::from_str(&format!(
        "mysql://{}:{}@{}:{}/{}",
        ds.username,
        ds.password,
        ds.host,
        ds.port,
        ds.database.as_deref().unwrap_or("mysql")
    ))?;

    let pool = MySqlPoolOptions::new()
        .max_connections(5)
        .connect_with(options)
        .await?;

    Ok(pool)
}

/// 测试 MySQL 连接
pub async fn test_connection(ds: &DataSource) -> Result<()> {
    let pool = create_pool(ds).await?;
    sqlx::query("SELECT 1").execute(&pool).await?;
    pool.close().await;
    Ok(())
}

/// 获取数据库列表
pub async fn get_databases(ds: &DataSource) -> Result<Vec<String>> {
    let pool = create_pool(ds).await?;

    let rows = sqlx::query("SHOW DATABASES")
        .fetch_all(&pool)
        .await?;

    let mut databases = Vec::new();
    for row in rows {
        let db: String = row.get(0);
        // 过滤系统数据库
        if !["information_schema", "mysql", "performance_schema", "sys"].contains(&db.as_str()) {
            databases.push(db);
        }
    }

    pool.close().await;
    Ok(databases)
}

/// 获取表列表
pub async fn get_tables(ds: &DataSource, database: &str) -> Result<Vec<String>> {
    let pool = create_pool(ds).await?;

    let query = format!("SHOW TABLES FROM `{}`", database);
    let rows = sqlx::query(&query).fetch_all(&pool).await?;

    let mut tables = Vec::new();
    for row in rows {
        let table: String = row.get(0);
        tables.push(table);
    }

    pool.close().await;
    Ok(tables)
}
