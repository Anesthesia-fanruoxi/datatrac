// Elasticsearch Reader 实现
// 从 ES 索引读取数据并转换为 DataRecord

use crate::exchange::core::data_record::{
    DataRecord, FieldInfo, FieldType, FieldValue, SchemaInfo,
};
use crate::storage::models::DataSource;
use anyhow::Result;
use elasticsearch::{
    auth::Credentials,
    http::transport::{SingleNodeConnectionPool, TransportBuilder},
    CountParts, Elasticsearch, SearchParts,
};
use serde_json::Value;
use url::Url;

/// Elasticsearch Reader
pub struct ElasticsearchReader {
    datasource: DataSource,
    index_name: String,
    client: Option<Elasticsearch>,
    scroll_id: Option<String>,
    total_count: u64,
    has_next: bool,
}

impl ElasticsearchReader {
    /// 创建新的 Elasticsearch Reader
    pub fn new(datasource: DataSource, index_name: String) -> Self {
        Self {
            datasource,
            index_name,
            client: None,
            scroll_id: None,
            total_count: 0,
            has_next: true,
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

    /// 将 JSON Value 转换为 FieldValue
    fn json_to_field_value(value: &Value) -> FieldValue {
        match value {
            Value::Null => FieldValue::Null,
            Value::Bool(b) => FieldValue::Boolean(*b),
            Value::Number(n) => {
                if let Some(i) = n.as_i64() {
                    FieldValue::Integer(i)
                } else if let Some(f) = n.as_f64() {
                    FieldValue::Float(f)
                } else {
                    FieldValue::Null
                }
            }
            Value::String(s) => {
                // 尝试解析为日期时间
                if let Ok(dt) = chrono::DateTime::parse_from_rfc3339(s) {
                    FieldValue::DateTime(dt.with_timezone(&chrono::Utc))
                } else {
                    FieldValue::Text(s.clone())
                }
            }
            Value::Array(_) | Value::Object(_) => FieldValue::Json(value.clone()),
        }
    }

    /// 将 ES 文档转换为 DataRecord
    fn doc_to_record(&self, doc: &Value) -> Result<DataRecord> {
        let mut record = DataRecord::new();

        if let Some(source) = doc.get("_source").and_then(|v| v.as_object()) {
            for (key, value) in source {
                record.add_field(key.clone(), Self::json_to_field_value(value));
            }
        }

        // 添加文档 ID 到元数据
        if let Some(id) = doc.get("_id").and_then(|v| v.as_str()) {
            record.add_metadata("_id".to_string(), id.to_string());
        }

        Ok(record)
    }
}

impl ElasticsearchReader {
    pub async fn open(&mut self) -> Result<()> {
        self.client = Some(self.create_client()?);
        Ok(())
    }

    pub async fn get_schema(&self) -> Result<SchemaInfo> {
        // ES 是无模式的，返回空 Schema
        // 实际字段会在读取数据时动态发现
        Ok(SchemaInfo {
            fields: Vec::new(),
            primary_key: Some("_id".to_string()),
        })
    }

    pub async fn get_total_count(&mut self) -> Result<u64> {
        let client = self.client.as_ref().unwrap();

        let response = client
            .count(CountParts::Index(&[&self.index_name]))
            .send()
            .await?;

        let body: Value = response.json().await?;
        let count = body["count"].as_u64().unwrap_or(0);

        self.total_count = count;
        Ok(count)
    }

    pub async fn read_batch(&mut self, batch_size: usize) -> Result<Vec<DataRecord>> {
        let client = self.client.as_ref().unwrap();

        let response = if let Some(scroll_id) = &self.scroll_id {
            // 使用 scroll_id 继续滚动
            client
                .scroll(elasticsearch::ScrollParts::None)
                .scroll_id(scroll_id)
                .scroll("1m")
                .send()
                .await?
        } else {
            // 首次查询，初始化 scroll
            client
                .search(SearchParts::Index(&[&self.index_name]))
                .scroll("1m")
                .size(batch_size as i64)
                .body(serde_json::json!({
                    "query": {
                        "match_all": {}
                    }
                }))
                .send()
                .await?
        };

        let body: Value = response.json().await?;

        // 更新 scroll_id
        if let Some(scroll_id) = body["_scroll_id"].as_str() {
            self.scroll_id = Some(scroll_id.to_string());
        }

        // 解析文档
        let empty_vec = vec![];
        let hits = body["hits"]["hits"].as_array().unwrap_or(&empty_vec);

        let mut records = Vec::new();
        for hit in hits {
            records.push(self.doc_to_record(hit)?);
        }

        self.has_next = !records.is_empty();

        Ok(records)
    }

    pub fn has_next(&self) -> bool {
        self.has_next
    }

    pub async fn close(&mut self) -> Result<()> {
        // 清理 scroll
        if let Some(scroll_id) = self.scroll_id.take() {
            if let Some(client) = &self.client {
                let body = serde_json::json!({
                    "scroll_id": [scroll_id]
                });
                let _ = client
                    .clear_scroll(elasticsearch::ClearScrollParts::None)
                    .body(body)
                    .send()
                    .await;
            }
        }
        Ok(())
    }
}
