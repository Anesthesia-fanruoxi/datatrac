// 新的三表结构数据库操作

use super::Storage;
use super::models::{TaskUnitConfig, TaskUnitRuntime, TaskUnitHistory, TaskUnitType, TaskUnitStatus};
use anyhow::{Context, Result};
use chrono::Utc;
use uuid::Uuid;
use sqlx::Row;

// ==================== 配置表操作 ====================

impl Storage {
    /// 保存任务单元配置
    pub async fn save_unit_config(&self, config: &TaskUnitConfig) -> Result<()> {
        sqlx::query(
            r#"
            INSERT INTO task_unit_config (
                id, task_id, unit_name, unit_type, search_pattern, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(task_id, unit_name) DO UPDATE SET
                unit_type = excluded.unit_type,
                search_pattern = excluded.search_pattern,
                updated_at = excluded.updated_at
            "#,
        )
        .bind(&config.id)
        .bind(&config.task_id)
        .bind(&config.unit_name)
        .bind(config.unit_type.as_str())
        .bind(&config.search_pattern)
        .bind(config.created_at)
        .bind(config.updated_at)
        .execute(&self.pool)
        .await
        .context("保存任务单元配置失败")?;

        Ok(())
    }

    /// 批量保存任务单元配置
    pub async fn save_unit_configs(&self, configs: &[TaskUnitConfig]) -> Result<()> {
        let mut tx = self.pool.begin().await?;

        for config in configs {
            sqlx::query(
                r#"
                INSERT INTO task_unit_config (
                    id, task_id, unit_name, unit_type, search_pattern, created_at, updated_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT(task_id, unit_name) DO UPDATE SET
                    unit_type = excluded.unit_type,
                    search_pattern = excluded.search_pattern,
                    updated_at = excluded.updated_at
                "#,
            )
            .bind(&config.id)
            .bind(&config.task_id)
            .bind(&config.unit_name)
            .bind(config.unit_type.as_str())
            .bind(&config.search_pattern)
            .bind(config.created_at)
            .bind(config.updated_at)
            .execute(&mut *tx)
            .await
            .context("批量保存任务单元配置失败")?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 加载任务的所有配置
    pub async fn load_unit_configs(&self, task_id: &str) -> Result<Vec<TaskUnitConfig>> {
        let rows = sqlx::query(
            r#"
            SELECT id, task_id, unit_name, unit_type, search_pattern, created_at, updated_at
            FROM task_unit_config
            WHERE task_id = ?
            ORDER BY unit_name
            "#,
        )
        .bind(task_id)
        .fetch_all(&self.pool)
        .await
        .context("加载任务单元配置失败")?;

        let mut configs = Vec::new();
        for row in rows {
            let config = TaskUnitConfig {
                id: row.get("id"),
                task_id: row.get("task_id"),
                unit_name: row.get("unit_name"),
                unit_type: TaskUnitType::from_str(row.get("unit_type"))?,
                search_pattern: row.get("search_pattern"),
                created_at: row.get("created_at"),
                updated_at: row.get("updated_at"),
            };
            configs.push(config);
        }

        Ok(configs)
    }

    /// 删除任务的所有配置
    pub async fn delete_unit_configs(&self, task_id: &str) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM task_unit_config
            WHERE task_id = ?
            "#,
        )
        .bind(task_id)
        .execute(&self.pool)
        .await
        .context("删除任务单元配置失败")?;

        Ok(())
    }
}

// ==================== 运行记录表操作 ====================

impl Storage {
    /// 保存任务单元运行记录
    pub async fn save_unit_runtime(&self, runtime: &TaskUnitRuntime) -> Result<()> {
        sqlx::query(
            r#"
            INSERT INTO task_unit_runtime (
                id, task_id, unit_name, status, total_records, processed_records,
                error_message, started_at, updated_at, last_processed_batch
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(task_id, unit_name) DO UPDATE SET
                status = excluded.status,
                total_records = excluded.total_records,
                processed_records = excluded.processed_records,
                error_message = excluded.error_message,
                started_at = excluded.started_at,
                updated_at = excluded.updated_at,
                last_processed_batch = excluded.last_processed_batch
            "#,
        )
        .bind(&runtime.id)
        .bind(&runtime.task_id)
        .bind(&runtime.unit_name)
        .bind(runtime.status.as_str())
        .bind(runtime.total_records)
        .bind(runtime.processed_records)
        .bind(&runtime.error_message)
        .bind(runtime.started_at)
        .bind(runtime.updated_at)
        .bind(runtime.last_processed_batch)
        .execute(&self.pool)
        .await
        .context("保存任务单元运行记录失败")?;

        Ok(())
    }

    /// 加载任务的所有运行记录
    pub async fn load_unit_runtimes(&self, task_id: &str) -> Result<Vec<TaskUnitRuntime>> {
        let rows = sqlx::query(
            r#"
            SELECT id, task_id, unit_name, status, total_records, processed_records,
                   error_message, started_at, updated_at, last_processed_batch
            FROM task_unit_runtime
            WHERE task_id = ?
            ORDER BY unit_name
            "#,
        )
        .bind(task_id)
        .fetch_all(&self.pool)
        .await
        .context("加载任务单元运行记录失败")?;

        let mut runtimes = Vec::new();
        for row in rows {
            let runtime = TaskUnitRuntime {
                id: row.get("id"),
                task_id: row.get("task_id"),
                unit_name: row.get("unit_name"),
                status: TaskUnitStatus::from_str(row.get("status"))?,
                total_records: row.get("total_records"),
                processed_records: row.get("processed_records"),
                error_message: row.get("error_message"),
                started_at: row.get("started_at"),
                updated_at: row.get("updated_at"),
                last_processed_batch: row.get("last_processed_batch"),
            };
            runtimes.push(runtime);
        }

        Ok(runtimes)
    }

    /// 更新运行记录状态
    pub async fn update_runtime_status(&self, task_id: &str, unit_name: &str, status: TaskUnitStatus) -> Result<()> {
        let now = Utc::now().timestamp_millis();
        
        sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET status = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(status.as_str())
        .bind(now)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("更新运行记录状态失败")?;

        Ok(())
    }

    /// 更新运行记录进度
    pub async fn update_runtime_progress(
        &self,
        task_id: &str,
        unit_name: &str,
        total_records: i64,
        processed_records: i64,
    ) -> Result<()> {
        let now = Utc::now().timestamp_millis();
        
        sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET total_records = ?, processed_records = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(total_records)
        .bind(processed_records)
        .bind(now)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("更新运行记录进度失败")?;

        Ok(())
    }
    
    /// 更新运行记录的批次号 (用于断点续传)
    pub async fn update_runtime_batch(
        &self,
        task_id: &str,
        unit_name: &str,
        batch_number: i64,
    ) -> Result<()> {
        let now = Utc::now().timestamp_millis();
        
        sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET last_processed_batch = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(batch_number)
        .bind(now)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("更新运行记录批次号失败")?;

        Ok(())
    }

    /// 标记运行记录失败
    pub async fn fail_runtime(&self, task_id: &str, unit_name: &str, error_message: &str) -> Result<()> {
        let now = Utc::now().timestamp_millis();
        
        sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET status = 'failed', error_message = ?, updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(error_message)
        .bind(now)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("标记运行记录失败")?;

        Ok(())
    }

    /// 删除运行记录
    pub async fn delete_runtime(&self, task_id: &str, unit_name: &str) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM task_unit_runtime
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("删除运行记录失败")?;

        Ok(())
    }

    /// 清理不在配置表中的运行记录
    pub async fn cleanup_orphan_runtimes(&self, task_id: &str) -> Result<usize> {
        let result = sqlx::query(
            r#"
            DELETE FROM task_unit_runtime
            WHERE task_id = ? 
            AND unit_name NOT IN (
                SELECT unit_name FROM task_unit_config WHERE task_id = ?
            )
            "#,
        )
        .bind(task_id)
        .bind(task_id)
        .execute(&self.pool)
        .await
        .context("清理孤立运行记录失败")?;

        Ok(result.rows_affected() as usize)
    }

    /// 重置失败的运行记录为 pending 状态
    pub async fn reset_failed_runtimes(&self, task_id: &str) -> Result<usize> {
        let now = Utc::now().timestamp_millis();
        
        let result = sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET status = 'pending',
                processed_records = 0,
                error_message = NULL,
                started_at = NULL,
                last_processed_batch = NULL,
                updated_at = ?
            WHERE task_id = ? AND status = 'failed'
            "#,
        )
        .bind(now)
        .bind(task_id)
        .execute(&self.pool)
        .await
        .context("重置失败运行记录失败")?;

        Ok(result.rows_affected() as usize)
    }

    /// 重置指定的运行记录为 pending 状态
    pub async fn reset_runtime(&self, task_id: &str, unit_name: &str) -> Result<()> {
        let now = Utc::now().timestamp_millis();
        
        sqlx::query(
            r#"
            UPDATE task_unit_runtime
            SET status = 'pending',
                processed_records = 0,
                error_message = NULL,
                started_at = NULL,
                last_processed_batch = NULL,
                updated_at = ?
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(now)
        .bind(task_id)
        .bind(unit_name)
        .execute(&self.pool)
        .await
        .context("重置运行记录失败")?;

        Ok(())
    }
}

// ==================== 历史记录表操作 ====================

impl Storage {
    /// 保存任务单元历史记录
    pub async fn save_unit_history(&self, history: &TaskUnitHistory) -> Result<()> {
        sqlx::query(
            r#"
            INSERT INTO task_unit_history (
                id, task_id, unit_name, search_pattern, total_records, completed_at, duration
            ) VALUES (?, ?, ?, ?, ?, ?, ?)
            "#,
        )
        .bind(&history.id)
        .bind(&history.task_id)
        .bind(&history.unit_name)
        .bind(&history.search_pattern)
        .bind(history.total_records)
        .bind(history.completed_at)
        .bind(history.duration)
        .execute(&self.pool)
        .await
        .context("保存任务单元历史记录失败")?;

        Ok(())
    }

    /// 加载任务的所有历史记录
    pub async fn load_unit_histories(&self, task_id: &str) -> Result<Vec<TaskUnitHistory>> {
        let rows = sqlx::query(
            r#"
            SELECT id, task_id, unit_name, search_pattern, total_records, completed_at, duration
            FROM task_unit_history
            WHERE task_id = ?
            ORDER BY completed_at DESC
            "#,
        )
        .bind(task_id)
        .fetch_all(&self.pool)
        .await
        .context("加载任务单元历史记录失败")?;

        let mut histories = Vec::new();
        for row in rows {
            let history = TaskUnitHistory {
                id: row.get("id"),
                task_id: row.get("task_id"),
                unit_name: row.get("unit_name"),
                search_pattern: row.get("search_pattern"),
                total_records: row.get("total_records"),
                completed_at: row.get("completed_at"),
                duration: row.get("duration"),
            };
            histories.push(history);
        }

        Ok(histories)
    }

    /// 按搜索关键字删除历史记录
    pub async fn clear_histories_by_pattern(&self, task_id: &str, pattern: &str) -> Result<usize> {
        log::info!("[clear_histories_by_pattern] 删除任务 {} 的关键字 {} 的历史记录", task_id, pattern);
        
        let result = sqlx::query(
            r#"
            DELETE FROM task_unit_history
            WHERE task_id = ? AND search_pattern = ?
            "#,
        )
        .bind(task_id)
        .bind(pattern)
        .execute(&self.pool)
        .await
        .context("按关键字删除历史记录失败")?;
        
        let count = result.rows_affected() as usize;
        log::info!("[clear_histories_by_pattern] 成功删除 {} 条历史记录", count);
        
        Ok(count)
    }

    /// 删除任务的所有历史记录
    pub async fn delete_unit_histories(&self, task_id: &str) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM task_unit_history
            WHERE task_id = ?
            "#,
        )
        .bind(task_id)
        .execute(&self.pool)
        .await
        .context("删除任务单元历史记录失败")?;

        Ok(())
    }

    /// 将运行记录移动到历史记录
    /// 
    /// 用于同步完成后的数据迁移
    pub async fn move_runtime_to_history(
        &self,
        task_id: &str,
        unit_name: &str,
        search_pattern: Option<String>,
        duration: i64,
    ) -> Result<()> {
        // 1. 查询运行记录
        let runtime_row = sqlx::query(
            r#"
            SELECT total_records FROM task_unit_runtime
            WHERE task_id = ? AND unit_name = ?
            "#,
        )
        .bind(task_id)
        .bind(unit_name)
        .fetch_optional(&self.pool)
        .await
        .context("查询运行记录失败")?;

        if let Some(row) = runtime_row {
            let total_records: i64 = row.get("total_records");
            let now = Utc::now().timestamp_millis();

            // 2. 插入历史记录
            let history = TaskUnitHistory {
                id: Uuid::new_v4().to_string(),
                task_id: task_id.to_string(),
                unit_name: unit_name.to_string(),
                search_pattern,
                total_records,
                completed_at: now,
                duration,
            };
            self.save_unit_history(&history).await?;

            // 3. 删除运行记录
            self.delete_runtime(task_id, unit_name).await?;

            log::info!("[move_runtime_to_history] 成功将运行记录移动到历史: {}", unit_name);
        }

        Ok(())
    }
}
