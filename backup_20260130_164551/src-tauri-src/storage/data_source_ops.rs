// 数据源 CRUD 操作

use super::{DataSource, DataSourceType, Storage};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use sqlx::Row;

impl Storage {
    /// 保存数据源配置
    /// 
    /// 如果数据源 ID 已存在，则更新；否则插入新记录
    pub async fn save_data_source(&self, ds: &DataSource) -> Result<()> {
        sqlx::query(
            r#"
            INSERT INTO data_sources (
                id, name, type, host, port, username, password, database, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
        .bind(ds.source_type.as_str())
        .bind(&ds.host)
        .bind(ds.port as i64)
        .bind(&ds.username)
        .bind(&ds.password)
        .bind(&ds.database)
        .bind(ds.created_at.to_rfc3339())
        .bind(ds.updated_at.to_rfc3339())
        .execute(&self.pool)
        .await
        .context("保存数据源失败")?;

        Ok(())
    }

    /// 加载所有数据源配置
    pub async fn load_data_sources(&self) -> Result<Vec<DataSource>> {
        let rows = sqlx::query(
            r#"
            SELECT id, name, type, host, port, username, password, database, created_at, updated_at
            FROM data_sources
            ORDER BY created_at DESC
            "#,
        )
        .fetch_all(&self.pool)
        .await
        .context("加载数据源列表失败")?;

        let mut data_sources = Vec::new();
        for row in rows {
            let source_type_str: String = row.try_get("type")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            data_sources.push(DataSource {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                host: row.try_get("host")?,
                port: row.try_get::<i64, _>("port")? as u16,
                username: row.try_get("username")?,
                password: row.try_get("password")?,
                database: row.try_get("database")?,
                created_at: DateTime::parse_from_rfc3339(&created_at_str)
                    .context("解析 created_at 失败")?
                    .with_timezone(&Utc),
                updated_at: DateTime::parse_from_rfc3339(&updated_at_str)
                    .context("解析 updated_at 失败")?
                    .with_timezone(&Utc),
            });
        }

        Ok(data_sources)
    }

    /// 根据 ID 加载单个数据源
    pub async fn load_data_source(&self, id: &str) -> Result<Option<DataSource>> {
        let row = sqlx::query(
            r#"
            SELECT id, name, type, host, port, username, password, database, created_at, updated_at
            FROM data_sources
            WHERE id = ?
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .context("加载数据源失败")?;

        if let Some(row) = row {
            let source_type_str: String = row.try_get("type")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            Ok(Some(DataSource {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                host: row.try_get("host")?,
                port: row.try_get::<i64, _>("port")? as u16,
                username: row.try_get("username")?,
                password: row.try_get("password")?,
                database: row.try_get("database")?,
                created_at: DateTime::parse_from_rfc3339(&created_at_str)
                    .context("解析 created_at 失败")?
                    .with_timezone(&Utc),
                updated_at: DateTime::parse_from_rfc3339(&updated_at_str)
                    .context("解析 updated_at 失败")?
                    .with_timezone(&Utc),
            }))
        } else {
            Ok(None)
        }
    }

    /// 删除数据源配置
    pub async fn delete_data_source(&self, id: &str) -> Result<()> {
        let result = sqlx::query(
            r#"
            DELETE FROM data_sources WHERE id = ?
            "#,
        )
        .bind(id)
        .execute(&self.pool)
        .await
        .context("删除数据源失败")?;

        if result.rows_affected() == 0 {
            anyhow::bail!("数据源不存在: {}", id);
        }

        Ok(())
    }
}
