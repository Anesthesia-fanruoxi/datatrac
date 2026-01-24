// MySQL 相关同步逻辑

use super::types::*;
use super::SyncEngine;
use anyhow::{Context, Result};
use sqlx::mysql::MySqlPool;
use sqlx::{Column, Row};

impl SyncEngine {
    /// 获取 MySQL 表结构
    pub(super) async fn get_mysql_table_schema(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
    ) -> Result<TableSchema> {
        // 查询表结构
        let query = format!(
            "SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY 
             FROM INFORMATION_SCHEMA.COLUMNS 
             WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
             ORDER BY ORDINAL_POSITION"
        );

        let rows = sqlx::query(&query)
            .bind(database)
            .bind(table)
            .fetch_all(pool)
            .await
            .context("查询表结构失败")?;

        let mut columns = Vec::new();
        let mut primary_key = None;

        for row in rows {
            let column_name: String = row.try_get("COLUMN_NAME")?;
            let data_type: String = row.try_get("DATA_TYPE")?;
            let is_nullable: String = row.try_get("IS_NULLABLE")?;
            let column_key: String = row.try_get("COLUMN_KEY")?;

            let is_primary_key = column_key == "PRI";
            if is_primary_key {
                primary_key = Some(column_name.clone());
            }

            columns.push(ColumnDefinition {
                name: column_name,
                data_type,
                nullable: is_nullable == "YES",
                is_primary_key,
            });
        }

        Ok(TableSchema {
            table_name: table.to_string(),
            columns,
            primary_key,
        })
    }

    /// 删除并重建 MySQL 表
    pub(super) async fn drop_and_create_mysql_table(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        schema: &TableSchema,
    ) -> Result<()> {
        // 删除表（如果存在）
        let drop_query = format!("DROP TABLE IF EXISTS `{}`.`{}`", database, table);
        sqlx::query(&drop_query)
            .execute(pool)
            .await
            .context("删除表失败")?;

        // 构建创建表的 SQL
        let mut create_sql = format!("CREATE TABLE `{}`.`{}` (", database, table);
        
        for (i, col) in schema.columns.iter().enumerate() {
            if i > 0 {
                create_sql.push_str(", ");
            }
            create_sql.push_str(&format!(
                "`{}` {}{}",
                col.name,
                col.data_type,
                if col.nullable { "" } else { " NOT NULL" }
            ));
        }

        // 添加主键
        if let Some(pk) = &schema.primary_key {
            create_sql.push_str(&format!(", PRIMARY KEY (`{}`)", pk));
        }

        create_sql.push(')');

        // 创建表
        sqlx::query(&create_sql)
            .execute(pool)
            .await
            .context("创建表失败")?;

        Ok(())
    }

    /// 从 MySQL 读取批量数据
    pub(super) async fn read_mysql_batch(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        offset: u64,
        limit: usize,
    ) -> Result<BatchData> {
        let query = format!(
            "SELECT * FROM `{}`.`{}` LIMIT {} OFFSET {}",
            database, table, limit, offset
        );

        let rows = sqlx::query(&query)
            .fetch_all(pool)
            .await
            .context("读取数据失败")?;

        let mut batch = Vec::new();
        for row in rows {
            let mut record = std::collections::HashMap::new();
            
            // 获取所有列
            for col in row.columns() {
                let col_name = col.name();
                let value: serde_json::Value = row.try_get(col_name)
                    .unwrap_or(serde_json::Value::Null);
                record.insert(col_name.to_string(), value);
            }
            
            batch.push(record);
        }

        Ok(batch)
    }

    /// 批量插入数据到 MySQL
    pub(super) async fn batch_insert_mysql(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        batch: &BatchData,
    ) -> Result<()> {
        if batch.is_empty() {
            return Ok(());
        }

        // 获取列名
        let columns: Vec<String> = batch[0].keys().cloned().collect();
        
        // 构建批量插入 SQL
        let mut insert_sql = format!(
            "INSERT INTO `{}`.`{}` ({}) VALUES ",
            database,
            table,
            columns.iter().map(|c| format!("`{}`", c)).collect::<Vec<_>>().join(", ")
        );

        // 添加值占位符
        let placeholders: Vec<String> = batch
            .iter()
            .map(|_| {
                format!(
                    "({})",
                    columns.iter().map(|_| "?").collect::<Vec<_>>().join(", ")
                )
            })
            .collect();
        
        insert_sql.push_str(&placeholders.join(", "));

        // 构建查询
        let mut query = sqlx::query(&insert_sql);
        
        // 绑定所有值
        for record in batch {
            for col in &columns {
                let value = record.get(col).unwrap_or(&serde_json::Value::Null);
                query = query.bind(value);
            }
        }

        // 执行插入
        query.execute(pool)
            .await
            .context("批量插入失败")?;

        Ok(())
    }

    /// 获取 MySQL 表的总记录数
    pub(super) async fn get_mysql_table_count(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
    ) -> Result<u64> {
        let query = format!("SELECT COUNT(*) as count FROM `{}`.`{}`", database, table);
        
        let row = sqlx::query(&query)
            .fetch_one(pool)
            .await
            .context("查询记录数失败")?;
        
        let count: i64 = row.try_get("count")?;
        Ok(count as u64)
    }
}
