// MySQL 相关同步逻辑

use super::types::*;
use super::SyncEngine;
use anyhow::{Context, Result};
use sqlx::mysql::MySqlPool;
use sqlx::{Column, Row};

impl SyncEngine {
    /// 获取 MySQL 表结构（使用完整的列类型）
    pub(super) async fn get_mysql_table_schema(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
    ) -> Result<TableSchema> {
        // 查询表结构 - 使用 COLUMN_TYPE 获取完整类型信息（包括长度、精度等）
        let query = format!(
            "SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA
             FROM INFORMATION_SCHEMA.COLUMNS 
             WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
             ORDER BY ORDINAL_POSITION"
        );

        let rows = sqlx::query(&query)
            .bind(database)
            .bind(table)
            .fetch_all(pool)
            .await
            .context(format!("查询表结构失败: {}.{}", database, table))?;

        if rows.is_empty() {
            anyhow::bail!("表 {}.{} 不存在或没有列", database, table);
        }

        let mut columns = Vec::new();
        let mut primary_key = None;

        for row in rows {
            let column_name: String = row.try_get("COLUMN_NAME")?;
            let column_type: String = row.try_get("COLUMN_TYPE")?; // 完整类型，如 varchar(255)
            let is_nullable: String = row.try_get("IS_NULLABLE")?;
            let column_key: String = row.try_get("COLUMN_KEY")?;

            let is_primary_key = column_key == "PRI";
            if is_primary_key {
                primary_key = Some(column_name.clone());
            }

            columns.push(ColumnDefinition {
                name: column_name,
                data_type: column_type, // 使用完整的列类型
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
        
        log::info!("删除表: {}.{}", database, table);
        
        sqlx::query(&drop_query)
            .execute(pool)
            .await
            .context(format!("删除表失败: {}.{}", database, table))?;

        // 构建创建表的 SQL
        let mut create_sql = format!("CREATE TABLE `{}`.`{}` (", database, table);
        
        for (i, col) in schema.columns.iter().enumerate() {
            if i > 0 {
                create_sql.push_str(", ");
            }
            
            // 使用完整的列类型（已包含长度等信息）
            create_sql.push_str(&format!(
                "`{}` {}{}",
                col.name,
                col.data_type, // 现在是完整类型，如 varchar(255)
                if col.nullable { "" } else { " NOT NULL" }
            ));
        }

        // 添加主键
        if let Some(pk) = &schema.primary_key {
            create_sql.push_str(&format!(", PRIMARY KEY (`{}`)", pk));
        }

        create_sql.push(')');
        create_sql.push_str(" ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci");

        log::info!("创建表: {}.{} ({} 列)", database, table, schema.columns.len());
        
        // 创建表
        sqlx::query(&create_sql)
            .execute(pool)
            .await
            .context(format!("创建表失败: {}.{}", database, table))?;

        Ok(())
    }

    /// 根据策略处理 MySQL 表（支持 drop/truncate/backup）
    pub(super) async fn handle_mysql_table_with_strategy(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        schema: &TableSchema,
        strategy: &TableExistsStrategy,
    ) -> Result<()> {
        use chrono::Utc;
        
        // 首先确保数据库存在
        self.ensure_mysql_database_exists(pool, database).await?;
        
        // 检查表是否存在
        let check_query = format!(
            "SELECT COUNT(*) as count FROM INFORMATION_SCHEMA.TABLES 
             WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
        );
        let row: (i64,) = sqlx::query_as(&check_query)
            .bind(database)
            .bind(table)
            .fetch_one(pool)
            .await
            .context("检查表是否存在失败")?;
        
        let table_exists = row.0 > 0;

        match strategy {
            TableExistsStrategy::Drop => {
                if table_exists {
                    log::info!("目标表已存在，执行删除策略: {}.{}", database, table);
                }
                // 删除并重建（默认行为）
                self.drop_and_create_mysql_table(pool, database, table, schema).await?;
            }
            TableExistsStrategy::Truncate => {
                if table_exists {
                    log::info!("目标表已存在，执行清空策略: {}.{}", database, table);
                    // 清空表数据
                    let truncate_query = format!("TRUNCATE TABLE `{}`.`{}`", database, table);
                    sqlx::query(&truncate_query)
                        .execute(pool)
                        .await
                        .context("清空表失败")?;
                } else {
                    log::info!("目标表不存在，创建新表: {}.{}", database, table);
                    // 表不存在，创建新表
                    self.drop_and_create_mysql_table(pool, database, table, schema).await?;
                }
            }
            TableExistsStrategy::Backup => {
                if table_exists {
                    // 备份原表（重命名）
                    let timestamp = Utc::now().format("%Y%m%d_%H%M%S");
                    let backup_table = format!("{}_backup_{}", table, timestamp);
                    
                    log::info!("目标表已存在，执行备份策略: {}.{} -> {}.{}", 
                        database, table, database, backup_table);
                    
                    let rename_query = format!(
                        "RENAME TABLE `{}`.`{}` TO `{}`.`{}`",
                        database, table, database, backup_table
                    );
                    sqlx::query(&rename_query)
                        .execute(pool)
                        .await
                        .context("备份表失败")?;
                } else {
                    log::info!("目标表不存在，创建新表: {}.{}", database, table);
                }
                // 创建新表
                self.drop_and_create_mysql_table(pool, database, table, schema).await?;
            }
        }

        Ok(())
    }

    /// 确保 MySQL 数据库存在（如果不存在则创建）
    pub(super) async fn ensure_mysql_database_exists(
        &self,
        pool: &MySqlPool,
        database: &str,
    ) -> Result<()> {
        // 检查数据库是否存在
        let check_query = "SELECT COUNT(*) as count FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?";
        let row: (i64,) = sqlx::query_as(check_query)
            .bind(database)
            .fetch_one(pool)
            .await
            .context("检查数据库是否存在失败")?;
        
        let db_exists = row.0 > 0;
        
        if !db_exists {
            // 创建数据库
            let create_query = format!("CREATE DATABASE `{}` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database);
            sqlx::query(&create_query)
                .execute(pool)
                .await
                .context(format!("创建数据库 {} 失败", database))?;
            
            log::info!("已创建目标数据库: {}", database);
        }
        
        Ok(())
    }

    /// 从 MySQL 读取批量数据（返回原始行数据）
    /// 注意：YEAR 和 DECIMAL 类型会被转换以避免 sqlx 的类型限制
    pub(super) async fn read_mysql_batch_raw(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        schema: &TableSchema,
        offset: u64,
        limit: usize,
    ) -> Result<Vec<sqlx::mysql::MySqlRow>> {
        // 构建列选择列表，将特殊类型转换
        let column_list = schema.columns.iter()
            .map(|col| {
                let data_type = col.data_type.to_lowercase();
                if data_type.starts_with("year") {
                    // YEAR 类型转换为 INT，以避免 sqlx 的类型限制
                    format!("CAST(`{}` AS SIGNED) as `{}`", col.name, col.name)
                } else if data_type.starts_with("decimal") || data_type.starts_with("numeric") {
                    // DECIMAL/NUMERIC 类型转换为 DOUBLE，以避免 sqlx 的类型限制
                    format!("CAST(`{}` AS DOUBLE) as `{}`", col.name, col.name)
                } else {
                    format!("`{}`", col.name)
                }
            })
            .collect::<Vec<_>>()
            .join(", ");

        let query = format!(
            "SELECT {} FROM `{}`.`{}` LIMIT {} OFFSET {}",
            column_list, database, table, limit, offset
        );

        let rows = sqlx::query(&query)
            .fetch_all(pool)
            .await
            .context(format!("读取数据失败: {}.{}", database, table))?;

        Ok(rows)
    }

    /// 从 MySQL 读取批量数据（JSON 格式，用于 ES 同步）
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
            .context(format!("读取数据失败: {}.{}", database, table))?;

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

    /// 批量插入数据到 MySQL（根据表结构直接绑定对应类型）
    pub(super) async fn batch_insert_mysql_raw(
        &self,
        pool: &MySqlPool,
        database: &str,
        table: &str,
        rows: &[sqlx::mysql::MySqlRow],
        schema: &TableSchema,
    ) -> Result<()> {
        if rows.is_empty() {
            return Ok(());
        }

        // 获取列名
        let columns: Vec<String> = schema.columns.iter().map(|c| c.name.clone()).collect();
        
        // 构建批量插入 SQL
        let column_list = columns
            .iter()
            .map(|c| format!("`{}`", c))
            .collect::<Vec<_>>()
            .join(", ");
        
        let value_placeholder = format!(
            "({})",
            columns.iter().map(|_| "?").collect::<Vec<_>>().join(", ")
        );
        
        let placeholders = vec![value_placeholder; rows.len()].join(", ");
        
        let insert_sql = format!(
            "INSERT INTO `{}`.`{}` ({}) VALUES {}",
            database, table, column_list, placeholders
        );

        // 构建查询并绑定所有值
        let mut query = sqlx::query(&insert_sql);
        
        use sqlx::Row;
        use sqlx::ValueRef;
        
        // 遍历每一行，根据字段类型绑定值
        for row in rows {
            for (col_idx, col_def) in schema.columns.iter().enumerate() {
                let value_ref = row.try_get_raw(col_idx)?;
                
                // 如果是 NULL，直接绑定 NULL
                if value_ref.is_null() {
                    query = query.bind(None::<String>);
                    continue;
                }
                
                // 根据字段类型绑定对应的 Rust 类型
                let data_type = col_def.data_type.to_lowercase();
                let is_unsigned = data_type.contains("unsigned");
                
                // 添加调试日志
                log::debug!("处理列 {} (索引 {}): 类型 = {}", col_def.name, col_idx, data_type);
                
                // 先检查 ENUM 和 SET 类型（必须在其他类型之前，因为它们可能包含其他关键字）
                if data_type.starts_with("enum(") || data_type.starts_with("set(") {
                    // ENUM 和 SET 类型作为字符串处理
                    query = query.bind(row.try_get::<String, _>(col_idx)?);
                }
                // JSON 类型（必须在其他类型之前检查）
                else if data_type.starts_with("json") {
                    // JSON 类型作为字符串处理
                    query = query.bind(row.try_get::<String, _>(col_idx)?);
                }
                // 时间类型（使用 starts_with 避免误匹配）
                else if data_type.starts_with("timestamp") {
                    // TIMESTAMP 使用 DateTime<Utc>
                    query = query.bind(row.try_get::<chrono::DateTime<chrono::Utc>, _>(col_idx)?);
                } else if data_type.starts_with("datetime") {
                    // DATETIME 使用 NaiveDateTime
                    query = query.bind(row.try_get::<chrono::NaiveDateTime, _>(col_idx)?);
                } else if data_type.starts_with("date") {
                    // DATE 使用 NaiveDate
                    query = query.bind(row.try_get::<chrono::NaiveDate, _>(col_idx)?);
                } else if data_type.starts_with("time") {
                    // TIME 使用 NaiveTime
                    query = query.bind(row.try_get::<chrono::NaiveTime, _>(col_idx)?);
                } else if data_type.starts_with("year") {
                    // YEAR 类型：读取时已转换为 INT，这里用 i32 读取，绑定时转回 i16
                    // MySQL YEAR 范围是 1901-2155，用 i16 足够
                    log::debug!("检测到 YEAR 类型，列 {}: {}", col_def.name, data_type);
                    let year_value: i32 = row.try_get(col_idx)?;
                    log::debug!("YEAR 值: {}", year_value);
                    query = query.bind(year_value as i16);
                }
                // 布尔类型（必须在 tinyint 之前检查）
                else if data_type.starts_with("tinyint(1)") || data_type.starts_with("bool") {
                    query = query.bind(row.try_get::<bool, _>(col_idx)?);
                }
                // 整数类型（按从大到小的顺序检查，避免误匹配）
                else if data_type.starts_with("bigint") {
                    if is_unsigned {
                        query = query.bind(row.try_get::<u64, _>(col_idx)?);
                    } else {
                        query = query.bind(row.try_get::<i64, _>(col_idx)?);
                    }
                } else if data_type.starts_with("mediumint") {
                    // MEDIUMINT 用 i32/u32
                    if is_unsigned {
                        query = query.bind(row.try_get::<u32, _>(col_idx)?);
                    } else {
                        query = query.bind(row.try_get::<i32, _>(col_idx)?);
                    }
                } else if data_type.starts_with("int") {
                    // INT 用 i32/u32
                    if is_unsigned {
                        query = query.bind(row.try_get::<u32, _>(col_idx)?);
                    } else {
                        query = query.bind(row.try_get::<i32, _>(col_idx)?);
                    }
                } else if data_type.starts_with("smallint") {
                    if is_unsigned {
                        query = query.bind(row.try_get::<u16, _>(col_idx)?);
                    } else {
                        query = query.bind(row.try_get::<i16, _>(col_idx)?);
                    }
                } else if data_type.starts_with("tinyint") {
                    if is_unsigned {
                        query = query.bind(row.try_get::<u8, _>(col_idx)?);
                    } else {
                        query = query.bind(row.try_get::<i8, _>(col_idx)?);
                    }
                }
                // BIT 类型
                else if data_type.starts_with("bit") {
                    // BIT 类型用 u64
                    query = query.bind(row.try_get::<u64, _>(col_idx)?);
                }
                // 浮点类型
                else if data_type.starts_with("double") || data_type.starts_with("decimal") || data_type.starts_with("numeric") {
                    query = query.bind(row.try_get::<f64, _>(col_idx)?);
                } else if data_type.starts_with("float") {
                    query = query.bind(row.try_get::<f32, _>(col_idx)?);
                }
                // 二进制类型
                else if data_type.starts_with("blob") 
                    || data_type.starts_with("binary") 
                    || data_type.starts_with("varbinary") {
                    query = query.bind(row.try_get::<Vec<u8>, _>(col_idx)?);
                }
                // 空间类型（所有空间类型都作为二进制处理）
                else if data_type.starts_with("geometry")
                    || data_type.starts_with("point")
                    || data_type.starts_with("linestring")
                    || data_type.starts_with("polygon")
                    || data_type.starts_with("multipoint")
                    || data_type.starts_with("multilinestring")
                    || data_type.starts_with("multipolygon")
                    || data_type.starts_with("geometrycollection") {
                    query = query.bind(row.try_get::<Vec<u8>, _>(col_idx)?);
                }
                // 字符串类型（VARCHAR, TEXT, CHAR 等，作为默认类型）
                else {
                    query = query.bind(row.try_get::<String, _>(col_idx)?);
                }
            }
        }

        // 执行插入
        match query.execute(pool).await {
            Ok(_) => Ok(()),
            Err(e) => {
                log::error!("批量插入失败，表: {}.{}, 列: {}", 
                    database, table, columns.join(", "));
                log::error!("数据库错误: {}", e);
                Err(anyhow::anyhow!("批量插入失败: {}", e))
            }
        }
    }

    /// 批量插入数据到 MySQL（JSON 格式，用于 ES 同步）
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
                
                // 根据值类型正确绑定参数
                match value {
                    serde_json::Value::Null => {
                        query = query.bind(None::<String>);
                    }
                    serde_json::Value::Bool(b) => {
                        query = query.bind(b);
                    }
                    serde_json::Value::Number(n) => {
                        if let Some(i) = n.as_i64() {
                            query = query.bind(i);
                        } else if let Some(u) = n.as_u64() {
                            query = query.bind(u as i64);
                        } else if let Some(f) = n.as_f64() {
                            query = query.bind(f);
                        } else {
                            query = query.bind(n.to_string());
                        }
                    }
                    serde_json::Value::String(s) => {
                        query = query.bind(s);
                    }
                    serde_json::Value::Array(_) | serde_json::Value::Object(_) => {
                        query = query.bind(value.to_string());
                    }
                }
            }
        }

        // 执行插入
        match query.execute(pool).await {
            Ok(_) => Ok(()),
            Err(e) => {
                log::error!("批量插入失败，SQL 结构: INSERT INTO `{}`.`{}` ({}) VALUES (...)", 
                    database, table, columns.join(", "));
                log::error!("数据库错误: {}", e);
                Err(anyhow::anyhow!("批量插入失败: {}", e))
            }
        }
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
            .context(format!("查询记录数失败: {}.{}", database, table))?;
        
        let count: i64 = row.try_get("count")?;
        Ok(count as u64)
    }
}
