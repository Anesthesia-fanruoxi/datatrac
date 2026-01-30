// Elasticsearch Writer 实现
// 将 DataRecord 写入 ES 索引

use crate::exchange::core::data_record::{DataRecord, FieldValue, SchemaInfo};
use crate::storage::models::DataSource;
use anyhow::Result;
use elasticsearch::{
    auth::Credentials,
    http::transport::{SingleNodeConnectionPool, TransportBuilder},
    BulkParts, Elasticsearch,
};
use serde_json::{json, Value};
use url::Url;

/// Elasticsearch Writer
pub struct ElasticsearchWriter {
    datasource: DataSource,
    index_name: String,
    client: Option<Elasticsearch>,
}

impl ElasticsearchWriter {
    /// 创建新的 Elasticsearch Writer
    pub fn new(datasource: DataSource, index_name: String) -> Self {
        Self {
            datasource,
            index_name,
            client: None,
        }
    }

    /// 创建 ES 客户端
    fn create_client(&self) -> Result<Elasticsearch> {
        let url = Url::parse(&format!(
            "http://{}:{}",
            self.datasource.host, self.datasource.port
        ))?;
        let conn_pool = SingleNodeConnectionPool::new(url);

        let credentials = Credentials::Basic(
            self.datasource.username.clone(),
            self.datasource.password.clone(),
        );

        let transport = TransportBuilder::new(conn_pool)
            .auth(credentials)
            .build()?;

        Ok(Elasticsearch::new(transport))
    }

    /// 将 FieldValue 转换为 JSON Value
    fn field_value_to_json(value: &FieldValue) -> Value {
        match value {
            FieldValue::Null => Value::Null,
            FieldValue::Boolean(b) => json!(b),
            FieldValue::Integer(i) => json!(i),
            FieldValue::Float(f) => json!(f),
            FieldValue::Text(s) => json!(s),
            FieldValue::DateTime(dt) => json!(dt.to_rfc3339()),
            FieldValue::Json(v) => v.clone(),
            FieldValue::Binary(_) => Value::Null, // 暂不支持二进制
        }
    }

    /// 将 DataRecord 转换为 JSON
    fn record_to_json(&self, record: &DataRecord) -> Value {
        let mut doc = serde_json::Map::new();

        for (key, value) in &record.fields {
            doc.insert(key.clone(), Self::field_value_to_json(value));
        }

        Value::Object(doc)
    }
}

impl ElasticsearchWriter {
    pub async fn open(&mut self) -> Result<()> {
        self.client = Some(self.create_client()?);
        Ok(())
    }

    pub async fn prepare_target(&mut self, _schema_info: &SchemaInfo) -> Result<()> {
        // ES 是无模式的，不需要预先创建索引结构
        // 索引会在第一次写入时自动创建
        Ok(())
    }

    pub async fn write_batch(&mut self, batch: Vec<DataRecord>) -> Result<()> {
        if batch.is_empty() {
            return Ok(());
        }

        let client = self.client.as_ref().unwrap();

        // 构建 Bulk 请求体 - 使用字符串格式
        let mut body_lines = Vec::new();

        for record in &batch {
            // 获取文档 ID（如果有）
            let doc_id = record.get_metadata("_id");

            // 添加 action 行
            let action = if let Some(id) = doc_id {
                format!(
                    r#"{{"index":{{"_index":"{}","_id":"{}"}}}}"#,
                    self.index_name, id
                )
            } else {
                format!(r#"{{"index":{{"_index":"{}"}}}}"#, self.index_name)
            };
            body_lines.push(action);

            // 添加文档行
            let doc = self.record_to_json(record);
            body_lines.push(doc.to_string());
        }

        // 拼接成 NDJSON 格式（每行一个 JSON，以换行符分隔）
        let body_str = body_lines.join("\n") + "\n";

        // 发送 Bulk 请求
        let response = client
            .bulk(BulkParts::None)
            .body(vec![body_str])
            .send()
            .await?;

        let response_body = response.json::<Value>().await?;

        // 检查是否有错误
        if let Some(errors) = response_body["errors"].as_bool() {
            if errors {
                return Err(anyhow::anyhow!("Bulk 写入出现错误"));
            }
        }

        Ok(())
    }

    pub async fn commit(&mut self) -> Result<()> {
        // ES 会自动刷新，无需额外操作
        Ok(())
    }

    pub async fn close(&mut self) -> Result<()> {
        // 无需特殊清理
        Ok(())
    }
}
