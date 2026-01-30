// 数据源表操作
// 提供数据源的 CRUD 操作

use super::models::{DataSource, DataSourceType};
use super::Storage;
use anyhow::Result;
use chrono::Utc;
use sqlx::Row;

impl Storage {
    /// 保存数据源
    pub async fn save_datasource(&self, ds: &DataSource) -> Result<()> {
        let created_at = ds.created_at.timestamp();
        let updated_at = ds.updated_at.timestamp();
        let source_type = ds.source_type.as_str();

        sqlx::query(
            r#"
            INSERT INTO datasources (id, name, type, host, port, username, password, database, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                name = excluded.name,
                type = excluded.type,
                host = excluded.host,
                port = excluded.port,
                username = excluded.username,
                password = excluded.password,
                database = excluded.database,
                updated_at = excluded.updated_at
            "#,
        )
        .bind(&ds.id)
        .bind(&ds.name)
        .bind(source_type)
        .bind(&ds.host)
        .bind(ds.port as i64)
        .bind(&ds.username)
        .bind(&ds.password)
        .bind(&ds.database)
        .bind(created_at)
        .bind(updated_at)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 加载单个数据源
    pub async fn load_datasource(&self, id: &str) -> Result<Option<DataSource>> {
        let row = sqlx::query(
            r#"
            SELECT id, name, type, host, port, username, password, database, created_at, updated_at
            FROM datasources WHERE id = ?
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await?;

        if let Some(row) = row {
            let source_type = DataSourceType::from_str(row.get("type"))?;
            let created_at = chrono::DateTime::from_timestamp(row.get("created_at"), 0)
                .unwrap_or_else(|| Utc::now());
            let updated_at = chrono::DateTime::from_timestamp(row.get("updated_at"), 0)
                .unwrap_or_else(|| Utc::now());

            Ok(Some(DataSource {
                id: row.get("id"),
                name: row.get("name"),
                source_type,
                host: row.get("host"),
                port: row.get::<i64, _>("port") as u16,
                username: row.get("username"),
                password: row.get("password"),
                database: row.get("database"),
                created_at,
                updated_at,
            }))
        } else {
            Ok(None)
        }
    }

    /// 加载所有数据源
    pub async fn load_datasources(&self) -> Result<Vec<DataSource>> {
        let rows = sqlx::query(
            r#"
            SELECT id, name, type, host, port, username, password, database, created_at, updated_at
            FROM datasources ORDER BY created_at DESC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        let mut datasources = Vec::new();
        for row in rows {
            let source_type = DataSourceType::from_str(row.get("type"))?;
            let created_at = chrono::DateTime::from_timestamp(row.get("created_at"), 0)
                .unwrap_or_else(|| Utc::now());
            let updated_at = chrono::DateTime::from_timestamp(row.get("updated_at"), 0)
                .unwrap_or_else(|| Utc::now());

            datasources.push(DataSource {
                id: row.get("id"),
                name: row.get("name"),
                source_type,
                host: row.get("host"),
                port: row.get::<i64, _>("port") as u16,
                username: row.get("username"),
                password: row.get("password"),
                database: row.get("database"),
                created_at,
                updated_at,
            });
        }

        Ok(datasources)
    }

    /// 删除数据源
    pub async fn delete_datasource(&self, id: &str) -> Result<()> {
        let result = sqlx::query("DELETE FROM datasources WHERE id = ?")
            .bind(id)
            .execute(&self.pool)
            .await?;

        if result.rows_affected() == 0 {
            anyhow::bail!("数据源不存在: {}", id);
        }

        Ok(())
    }
}
