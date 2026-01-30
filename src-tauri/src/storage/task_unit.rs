// 任务单元表操作
// 提供任务单元的 CRUD 操作

use super::models::{TaskUnit, TaskUnitStatus, TaskUnitType};
use super::Storage;
use anyhow::Result;
use chrono::Utc;
use sqlx::Row;

impl Storage {
    /// 保存任务单元
    pub async fn save_task_unit(&self, unit: &TaskUnit) -> Result<()> {
        let created_at = unit.created_at.timestamp();
        let updated_at = unit.updated_at.timestamp();
        let unit_type = unit.unit_type.as_str();

        sqlx::query(
            r#"
            INSERT INTO task_units (id, task_id, unit_name, unit_type, status, total_records, processed_records, error_message, keyword, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(task_id, unit_name) DO UPDATE SET
                status = excluded.status,
                total_records = excluded.total_records,
                processed_records = excluded.processed_records,
                error_message = excluded.error_message,
                keyword = excluded.keyword,
                updated_at = excluded.updated_at
            "#,
        )
        .bind(&unit.id)
        .bind(&unit.task_id)
        .bind(&unit.unit_name)
        .bind(unit_type)
        .bind(&unit.status)
        .bind(unit.total_records)
        .bind(unit.processed_records)
        .bind(&unit.error_message)
        .bind(&unit.keyword)
        .bind(created_at)
        .bind(updated_at)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 加载任务的所有单元
    pub async fn load_task_units(&self, task_id: &str) -> Result<Vec<TaskUnit>> {
        let rows = sqlx::query(
            r#"
            SELECT id, task_id, unit_name, unit_type, status, total_records, processed_records, error_message, keyword, created_at, updated_at
            FROM task_units WHERE task_id = ? ORDER BY created_at ASC
            "#,
        )
        .bind(task_id)
        .fetch_all(&self.pool)
        .await?;

        let mut units = Vec::new();
        for row in rows {
            let unit_type = TaskUnitType::from_str(row.get("unit_type"))?;
            let created_at = chrono::DateTime::from_timestamp(row.get("created_at"), 0)
                .unwrap_or_else(|| Utc::now());
            let updated_at = chrono::DateTime::from_timestamp(row.get("updated_at"), 0)
                .unwrap_or_else(|| Utc::now());

            units.push(TaskUnit {
                id: row.get("id"),
                task_id: row.get("task_id"),
                unit_name: row.get("unit_name"),
                unit_type,
                status: row.get("status"),
                total_records: row.get("total_records"),
                processed_records: row.get("processed_records"),
                error_message: row.get("error_message"),
                keyword: row.get("keyword"),
                created_at,
                updated_at,
            });
        }

        Ok(units)
    }

    /// 更新任务单元状态
    pub async fn update_unit_status(
        &self,
        task_id: &str,
        unit_name: &str,
        status: TaskUnitStatus,
    ) -> Result<()> {
        let status_str = status.as_str();
        let updated_at = Utc::now().timestamp();

        sqlx::query(
            r#"
            UPDATE task_units SET status = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(status_str)
        .bind(updated_at)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新任务单元进度
    pub async fn update_unit_progress(
        &self,
        task_id: &str,
        unit_name: &str,
        total: i64,
        processed: i64,
    ) -> Result<()> {
        let updated_at = Utc::now().timestamp();

        sqlx::query(
            r#"
            UPDATE task_units SET total_records = ?, processed_records = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(total)
        .bind(processed)
        .bind(updated_at)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 标记任务单元失败
    pub async fn fail_unit(&self, task_id: &str, unit_name: &str, error: &str) -> Result<()> {
        let updated_at = Utc::now().timestamp();

        sqlx::query(
            r#"
            UPDATE task_units SET status = 'failed', error_message = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(error)
        .bind(updated_at)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 重置失败的任务单元
    pub async fn reset_failed_units(&self, task_id: &str) -> Result<usize> {
        let updated_at = Utc::now().timestamp();

        let result = sqlx::query(
            r#"
            UPDATE task_units SET status = 'pending', error_message = NULL, updated_at = ?
            WHERE task_id = ? AND status = 'failed'
            "#,
        )
        .bind(updated_at)
        .bind(task_id)
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() as usize)
    }
}
