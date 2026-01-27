// 同步任务 CRUD 操作

use super::{DataSourceType, Storage, SyncTask, TaskStatus};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use sqlx::Row;

impl Storage {
    /// 保存同步任务配置
    /// 
    /// 如果任务 ID 已存在，则更新；否则插入新记录
    pub async fn save_task(&self, task: &SyncTask) -> Result<()> {
        log::info!("save_task - 准备保存任务: id={}, name={}", task.id, task.name);
        log::info!("save_task - source_id={}, target_id={}", task.source_id, task.target_id);
        log::info!("save_task - source_type={}, target_type={}", task.source_type.as_str(), task.target_type.as_str());
        log::info!("save_task - config={}", task.config);
        log::info!("save_task - status={}", task.status.as_str());
        
        sqlx::query(
            r#"
            INSERT INTO sync_tasks (
                id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                name = excluded.name,
                source_id = excluded.source_id,
                target_id = excluded.target_id,
                source_type = excluded.source_type,
                target_type = excluded.target_type,
                config = excluded.config,
                status = excluded.status,
                updated_at = excluded.updated_at
            "#,
        )
        .bind(&task.id)
        .bind(&task.name)
        .bind(&task.source_id)
        .bind(&task.target_id)
        .bind(task.source_type.as_str())
        .bind(task.target_type.as_str())
        .bind(&task.config)
        .bind(task.status.as_str())
        .bind(task.created_at.to_rfc3339())
        .bind(task.updated_at.to_rfc3339())
        .execute(&self.pool)
        .await
        .context("保存同步任务失败")?;

        log::info!("save_task - 保存成功");
        
        Ok(())
    }

    /// 加载所有同步任务
    pub async fn load_tasks(&self) -> Result<Vec<SyncTask>> {
        let rows = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks
            ORDER BY created_at DESC
            "#,
        )
        .fetch_all(&self.pool)
        .await
        .context("加载同步任务列表失败")?;

        let mut tasks = Vec::new();
        for row in rows {
            let source_type_str: String = row.try_get("source_type")?;
            let target_type_str: String = row.try_get("target_type")?;
            let status_str: String = row.try_get("status")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            tasks.push(SyncTask {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_id: row.try_get("source_id")?,
                target_id: row.try_get("target_id")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                target_type: DataSourceType::from_str(&target_type_str)?,
                config: row.try_get("config")?,
                status: TaskStatus::from_str(&status_str)?,
                created_at: DateTime::parse_from_rfc3339(&created_at_str)
                    .context("解析 created_at 失败")?
                    .with_timezone(&Utc),
                updated_at: DateTime::parse_from_rfc3339(&updated_at_str)
                    .context("解析 updated_at 失败")?
                    .with_timezone(&Utc),
            });
        }

        Ok(tasks)
    }

    /// 根据 ID 加载单个同步任务
    pub async fn load_task(&self, id: &str) -> Result<Option<SyncTask>> {
        let row = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks
            WHERE id = ?
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .context("加载同步任务失败")?;

        if let Some(row) = row {
            let source_type_str: String = row.try_get("source_type")?;
            let target_type_str: String = row.try_get("target_type")?;
            let status_str: String = row.try_get("status")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            Ok(Some(SyncTask {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_id: row.try_get("source_id")?,
                target_id: row.try_get("target_id")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                target_type: DataSourceType::from_str(&target_type_str)?,
                config: row.try_get("config")?,
                status: TaskStatus::from_str(&status_str)?,
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

    /// 删除同步任务
    pub async fn delete_task(&self, id: &str) -> Result<()> {
        let result = sqlx::query(
            r#"
            DELETE FROM sync_tasks WHERE id = ?
            "#,
        )
        .bind(id)
        .execute(&self.pool)
        .await
        .context("删除同步任务失败")?;

        if result.rows_affected() == 0 {
            anyhow::bail!("同步任务不存在: {}", id);
        }

        Ok(())
    }
}
