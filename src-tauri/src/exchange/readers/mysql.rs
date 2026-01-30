// MySQL Reader 实现
// 从 MySQL 表读取数据并转换为 DataRecord

use crate::exchange::core::data_record::{
    DataRecord, FieldInfo, FieldType, FieldValue, SchemaInfo,
};
use crate::storage::models::DataSource;
use anyhow::Result;
use sqlx::mysql::{MySqlConnectOptions, MySqlPool, MySqlPoolOptions};
use sqlx::{Column, Row, TypeInfo};
use std::str::FromStr;

/// MySQL Reader
pub struct MySqlReader {
    datasource: DataSource,
    table_name: String,
    pool: Option<MySqlPool>,
    offset: u64,
    total_count: u64,
    has_next: bool,
}

impl MySqlReader {
    /// 创建新的 MySQL Reader
    pub fn new(datasource: DataSource, table_name: String) -> Self {
        Self {
            datasource,
            table_name,
            pool: None,
            offset: 0,
            total_count: 0,
            has_next: true,
        }
    }

    /// 创建连接池
    async fn create_pool(&self) -> Result<MySqlPool> {
        let options = MySqlConnectOptions::from_str(&format!(
            "mysql://{}:{}@{}:{}/{}",
            self.datasource.username,
            self.datasource.password,
            self.datasource.host,
            self.datasource.port,
            self.datasource.database.as_deref().unwrap_or("mysql")
        ))?;

        let pool = MySqlPoolOptions::new()
            .max_connections(5)
            .connect_with(options)
            .await?;

        Ok(pool)
    }

    /// 将 MySQL 类型转换为 FieldType
    fn map_mysql_type(type_name: &str) -> FieldType {
        match type_name.to_uppercase().as_str() {
            "TINYINT" | "SMALLINT" | "MEDIUMINT" | "INT" | "BIGINT" => FieldType::Integer,
            "FLOAT" | "DOUBLE" | "DECIMAL" => FieldType::Float,
            "DATE" | "DATETIME" | "TIMESTAMP" => FieldType::DateTime,
            "JSON" => FieldType::Json,
            "BLOB" | "BINARY" | "VARBINARY" => FieldType::Binary,
            _ => FieldType::Text,
        }
    }

    /// 将 MySQL Row 转换为 DataRecord
    fn row_to_record(&self, row: &sqlx::mysql::MySqlRow) -> Result<DataRecord> {
        let mut record = DataRecord::new();

        for column in row.columns() {
            let col_name = column.name();
            let type_info = column.type_info();

            let value = match type_info.name() {
                "TINYINT" | "SMALLINT" | "MEDIUMINT" | "INT" | "BIGINT" => {
                    match row.try_get::<Option<i64>, _>(col_name) {
                        Ok(Some(v)) => FieldValue::Integer(v),
                        Ok(None) => FieldValue::Null,
                        Err(_) => FieldValue::Null,
                    }
                }
                "FLOAT" | "DOUBLE" | "DECIMAL" => {
                    match row.try_get::<Option<f64>, _>(col_name) {
                        Ok(Some(v)) => FieldValue::Float(v),
                        Ok(None) => FieldValue::Null,
                        Err(_) => FieldValue::Null,
                    }
                }
                "DATE" | "DATETIME" | "TIMESTAMP" => {
                    match row.try_get::<Option<chrono::NaiveDateTime>, _>(col_name) {
                        Ok(Some(v)) => {
                            let dt = chrono::DateTime::<chrono::Utc>::from_naive_utc_and_offset(
                                v,
                                chrono::Utc,
                            );
                            FieldValue::DateTime(dt)
                        }
                        Ok(None) => FieldValue::Null,
                        Err(_) => FieldValue::Null,
                    }
                }
                _ => match row.try_get::<Option<String>, _>(col_name) {
                    Ok(Some(v)) => FieldValue::Text(v),
                    Ok(None) => FieldValue::Null,
                    Err(_) => FieldValue::Null,
                },
            };

            record.add_field(col_name.to_string(), value);
        }

        Ok(record)
    }
}

impl MySqlReader {
    pub async fn open(&mut self) -> Result<()> {
        self.pool = Some(self.create_pool().await?);
        Ok(())
    }

    pub async fn get_schema(&self) -> Result<SchemaInfo> {
        let pool = self.pool.as_ref().unwrap();

        let query = format!(
            "SELECT * FROM `{}` LIMIT 1",
            self.table_name
        );

        let row = sqlx::query(&query).fetch_optional(pool).await?;

        let mut fields = Vec::new();

        if let Some(row) = row {
            for column in row.columns() {
                let field_info = FieldInfo {
                    name: column.name().to_string(),
                    field_type: Self::map_mysql_type(column.type_info().name()),
                    nullable: true,
                };
                fields.push(field_info);
            }
        }

        Ok(SchemaInfo {
            fields,
            primary_key: None,
        })
    }

    pub async fn get_total_count(&mut self) -> Result<u64> {
        let pool = self.pool.as_ref().unwrap();

        let query = format!("SELECT COUNT(*) as count FROM `{}`", self.table_name);
        let row: (i64,) = sqlx::query_as(&query).fetch_one(pool).await?;

        self.total_count = row.0 as u64;
        Ok(self.total_count)
    }

    pub async fn read_batch(&mut self, batch_size: usize) -> Result<Vec<DataRecord>> {
        let pool = self.pool.as_ref().unwrap();

        let query = format!(
            "SELECT * FROM `{}` LIMIT {} OFFSET {}",
            self.table_name, batch_size, self.offset
        );

        let rows = sqlx::query(&query).fetch_all(pool).await?;

        let mut records = Vec::new();
        for row in &rows {
            records.push(self.row_to_record(row)?);
        }

        self.offset += rows.len() as u64;
        self.has_next = rows.len() == batch_size;

        Ok(records)
    }

    pub fn has_next(&self) -> bool {
        self.has_next
    }

    pub async fn close(&mut self) -> Result<()> {
        if let Some(pool) = self.pool.take() {
            pool.close().await;
        }
        Ok(())
    }
}
