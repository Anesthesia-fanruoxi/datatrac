// 已同步索引表操作
// 提供索引同步历史的查询和管理

use super::models::SyncedIndex;
use super::Storage;
use anyhow::Result;
use chrono::Utc;
use sqlx::Row;

impl Storage {
    /// 检查索引是否已同步
    pub async fn is_index_synced(&self, source_id: &str, index_name: &str) -> Result<bool> {
        let row = sqlx::query(
            "SELECT COUNT(*) as count FROM synced_indices WHERE source_id = ? AND index_name = ?",
        )
        .bind(source_id)
        .bind(index_name)
        .fetch_one(&self.pool)
        .await?;

        let count: i64 = row.get("count");
        Ok(count > 0)
    }

    /// 标记索引为已同步
    pub async fn mark_index_synced(
        &self,
        source_id: &str,
        index_name: &str,
        task_id: &str,
    ) -> Result<()> {
        let now = Utc::now().timestamp();
        let id = uuid::Uuid::new_v4().to_string();

        sqlx::query(
            r#"
            INSERT INTO synced_indices (id, source_id, index_name, first_synced_at, last_synced_at, sync_count, last_task_id)
            VALUES (?, ?, ?, ?, ?, 1, ?)
            ON CONFLICT(source_id, index_name) DO UPDATE SET
                last_synced_at = excluded.last_synced_at,
                sync_count = sync_count + 1,
                last_task_id = excluded.last_task_id
            "#
        )
        .bind(id)
        .bind(source_id)
        .bind(index_name)
        .bind(now)
        .bind(now)
        .bind(task_id)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 查询已同步的索引列表
    pub async fn list_synced_indices(&self, source_id: &str) -> Result<Vec<SyncedIndex>> {
        let rows = sqlx::query(
            r#"
            SELECT index_name, first_synced_at, last_synced_at, sync_count, last_task_id
            FROM synced_indices WHERE source_id = ?
            ORDER BY last_synced_at DESC
            "#,
        )
        .bind(source_id)
        .fetch_all(&self.pool)
        .await?;

        let mut indices = Vec::new();
        for row in rows {
            let first_synced_at =
                chrono::DateTime::from_timestamp(row.get("first_synced_at"), 0)
                    .unwrap_or_else(|| Utc::now());
            let last_synced_at = chrono::DateTime::from_timestamp(row.get("last_synced_at"), 0)
                .unwrap_or_else(|| Utc::now());

            indices.push(SyncedIndex {
                index_name: row.get("index_name"),
                first_synced_at,
                last_synced_at,
                sync_count: row.get("sync_count"),
                last_task_id: row.get("last_task_id"),
            });
        }

        Ok(indices)
    }

    /// 清除指定索引的同步历史
    pub async fn clear_synced_index(&self, source_id: &str, index_name: &str) -> Result<()> {
        sqlx::query("DELETE FROM synced_indices WHERE source_id = ? AND index_name = ?")
            .bind(source_id)
            .bind(index_name)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    /// 清除所有同步历史
    pub async fn clear_all_synced_indices(&self, source_id: &str) -> Result<()> {
        sqlx::query("DELETE FROM synced_indices WHERE source_id = ?")
            .bind(source_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }
}
