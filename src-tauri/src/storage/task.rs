// 任务表操作
// 提供同步任务的 CRUD 操作

use super::models::{DataSourceType, SyncTask, TaskStatus};
use super::Storage;
use anyhow::Result;
use chrono::Utc;
use sqlx::Row;

impl Storage {
    /// 保存任务
    pub async fn save_task(&self, task: &SyncTask) -> Result<()> {
        let created_at = task.created_at.timestamp();
        let updated_at = task.updated_at.timestamp();
        let source_type = task.source_type.as_str();
        let target_type = task.target_type.as_str();
        let status = task.status.as_str();

        sqlx::query(
            r#"
            INSERT INTO sync_tasks (id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
        .bind(source_type)
        .bind(target_type)
        .bind(&task.config)
        .bind(status)
        .bind(created_at)
        .bind(updated_at)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 加载单个任务
    pub async fn load_task(&self, id: &str) -> Result<Option<SyncTask>> {
        let row = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks WHERE id = ?
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await?;

        if let Some(row) = row {
            let source_type = DataSourceType::from_str(row.get("source_type"))?;
            let target_type = DataSourceType::from_str(row.get("target_type"))?;
            let status_str: String = row.get("status");
            let status = match status_str.as_str() {
                "idle" => TaskStatus::Idle,
                "running" => TaskStatus::Running,
                "paused" => TaskStatus::Paused,
                "completed" => TaskStatus::Completed,
                "failed" => TaskStatus::Failed,
                _ => TaskStatus::Idle,
            };
            let created_at = chrono::DateTime::from_timestamp(row.get("created_at"), 0)
                .unwrap_or_else(|| Utc::now());
            let updated_at = chrono::DateTime::from_timestamp(row.get("updated_at"), 0)
                .unwrap_or_else(|| Utc::now());

            Ok(Some(SyncTask {
                id: row.get("id"),
                name: row.get("name"),
                source_id: row.get("source_id"),
                target_id: row.get("target_id"),
                source_type,
                target_type,
                config: row.get("config"),
                status,
                created_at,
                updated_at,
            }))
        } else {
            Ok(None)
        }
    }

    /// 加载所有任务
    pub async fn load_tasks(&self) -> Result<Vec<SyncTask>> {
        let rows = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks ORDER BY created_at DESC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        let mut tasks = Vec::new();
        for row in rows {
            let source_type = DataSourceType::from_str(row.get("source_type"))?;
            let target_type = DataSourceType::from_str(row.get("target_type"))?;
            let status_str: String = row.get("status");
            let status = match status_str.as_str() {
                "idle" => TaskStatus::Idle,
                "running" => TaskStatus::Running,
                "paused" => TaskStatus::Paused,
                "completed" => TaskStatus::Completed,
                "failed" => TaskStatus::Failed,
                _ => TaskStatus::Idle,
            };
            let created_at = chrono::DateTime::from_timestamp(row.get("created_at"), 0)
                .unwrap_or_else(|| Utc::now());
            let updated_at = chrono::DateTime::from_timestamp(row.get("updated_at"), 0)
                .unwrap_or_else(|| Utc::now());

            tasks.push(SyncTask {
                id: row.get("id"),
                name: row.get("name"),
                source_id: row.get("source_id"),
                target_id: row.get("target_id"),
                source_type,
                target_type,
                config: row.get("config"),
                status,
                created_at,
                updated_at,
            });
        }

        Ok(tasks)
    }

    /// 删除任务
    pub async fn delete_task(&self, id: &str) -> Result<()> {
        let result = sqlx::query("DELETE FROM sync_tasks WHERE id = ?")
            .bind(id)
            .execute(&self.pool)
            .await?;

        if result.rows_affected() == 0 {
            anyhow::bail!("任务不存在: {}", id);
        }

        Ok(())
    }
}
