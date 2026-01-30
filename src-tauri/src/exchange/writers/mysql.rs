// MySQL Writer 实现
// 将 DataRecord 写入 MySQL 表

use crate::exchange::core::data_record::{DataRecord, FieldType, FieldValue, SchemaInfo};
use crate::storage::models::DataSource;
use anyhow::Result;
use sqlx::mysql::{MySqlConnectOptions, MySqlPool, MySqlPoolOptions};
use std::str::FromStr;

/// MySQL Writer
pub struct MySqlWriter {
    datasource: DataSource,
    table_name: String,
    pool: Option<MySqlPool>,
    schema: Option<SchemaInfo>,
}

impl MySqlWriter {
    /// 创建新的 MySQL Writer
    pub fn new(datasource: DataSource, table_name: String) -> Self {
        Self {
            datasource,
            table_name,
            pool: None,
            schema: None,
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

    /// 将 FieldType 转换为 MySQL 类型
    fn field_type_to_mysql(field_type: &FieldType) -> &str {
        match field_type {
            FieldType::Boolean => "TINYINT(1)",
            FieldType::Integer => "BIGINT",
            FieldType::Float => "DOUBLE",
            FieldType::Text => "TEXT",
            FieldType::DateTime => "DATETIME",
            FieldType::Json => "JSON",
            FieldType::Binary => "BLOB",
        }
    }

    /// 将 FieldValue 转换为 SQL 字符串
    fn field_value_to_sql(&self, value: &FieldValue) -> String {
        match value {
            FieldValue::Null => "NULL".to_string(),
            FieldValue::Boolean(b) => if *b { "1" } else { "0" }.to_string(),
            FieldValue::Integer(i) => i.to_string(),
            FieldValue::Float(f) => f.to_string(),
            FieldValue::Text(s) => format!("'{}'", s.replace("'", "''")),
            FieldValue::DateTime(dt) => format!("'{}'", dt.format("%Y-%m-%d %H:%M:%S")),
            FieldValue::Json(v) => format!("'{}'", v.to_string().replace("'", "''")),
            FieldValue::Binary(_) => "NULL".to_string(), // 暂不支持二进制
        }
    }
}

impl MySqlWriter {
    pub async fn open(&mut self) -> Result<()> {
        self.pool = Some(self.create_pool().await?);
        Ok(())
    }

    pub async fn prepare_target(&mut self, schema_info: &SchemaInfo) -> Result<()> {
        self.schema = Some(schema_info.clone());

        let pool = self.pool.as_ref().unwrap();

        // 检查表是否存在
        let check_query = format!(
            "SELECT COUNT(*) as count FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '{}'",
            self.table_name
        );

        let row: (i64,) = sqlx::query_as(&check_query).fetch_one(pool).await?;

        if row.0 == 0 {
            // 表不存在，创建表
            let mut columns = Vec::new();

            for field in &schema_info.fields {
                let nullable = if field.nullable { "NULL" } else { "NOT NULL" };
                columns.push(format!(
                    "`{}` {} {}",
                    field.name,
                    Self::field_type_to_mysql(&field.field_type),
                    nullable
                ));
            }

            let create_query = format!(
                "CREATE TABLE `{}` ({})",
                self.table_name,
                columns.join(", ")
            );

            sqlx::query(&create_query).execute(pool).await?;
        }

        Ok(())
    }

    pub async fn write_batch(&mut self, batch: Vec<DataRecord>) -> Result<()> {
        if batch.is_empty() {
            return Ok(());
        }

        let pool = self.pool.as_ref().unwrap();

        // 获取字段名
        let field_names: Vec<String> = batch[0].field_names();

        // 构建 INSERT 语句
        let mut values_list = Vec::new();

        for record in &batch {
            let mut values = Vec::new();
            for field_name in &field_names {
                if let Some(value) = record.get_field(field_name) {
                    values.push(self.field_value_to_sql(value));
                } else {
                    values.push("NULL".to_string());
                }
            }
            values_list.push(format!("({})", values.join(", ")));
        }

        let insert_query = format!(
            "INSERT INTO `{}` ({}) VALUES {}",
            self.table_name,
            field_names
                .iter()
                .map(|n| format!("`{}`", n))
                .collect::<Vec<_>>()
                .join(", "),
            values_list.join(", ")
        );

        sqlx::query(&insert_query).execute(pool).await?;

        Ok(())
    }

    pub async fn commit(&mut self) -> Result<()> {
        // MySQL 自动提交，无需额外操作
        Ok(())
    }

    pub async fn close(&mut self) -> Result<()> {
        if let Some(pool) = self.pool.take() {
            pool.close().await;
        }
        Ok(())
    }
}
