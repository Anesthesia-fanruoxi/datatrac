// Elasticsearch 相关同步逻辑

use super::types::*;
use super::SyncEngine;
use crate::type_mapper::TypeMapper;
use anyhow::{Context, Result};
use elasticsearch::{
    Elasticsearch, 
    BulkParts, 
    indices::{IndicesCreateParts, IndicesDeleteParts, IndicesExistsParts},
    SearchParts,
    http::request::JsonBody,
};
use serde_json::{json, Value};

impl SyncEngine {
    /// 删除并重建 ES 索引
    pub(super) async fn drop_and_create_es_index(
        &self,
        client: &Elasticsearch,
        index: &str,
        schema: &TableSchema,
    ) -> Result<()> {
        // 检查索引是否存在
        let exists_response = client
            .indices()
            .exists(IndicesExistsParts::Index(&[index]))
            .send()
            .await
            .context("检查索引是否存在失败")?;

        // 如果存在则删除
        if exists_response.status_code().is_success() {
            client
                .indices()
                .delete(IndicesDeleteParts::Index(&[index]))
                .send()
                .await
                .context("删除索引失败")?;
        }

        // 构建映射
        let mut properties = serde_json::Map::new();
        for col in &schema.columns {
            let es_type = TypeMapper::mysql_to_es(&col.data_type);
            properties.insert(
                col.name.clone(),
                json!({ "type": es_type })
            );
        }

        let mappings = json!({
            "properties": properties
        });

        // 创建索引
        client
            .indices()
            .create(IndicesCreateParts::Index(index))
            .body(json!({ "mappings": mappings }))
            .send()
            .await
            .context("创建索引失败")?;

        Ok(())
    }

    /// 批量插入数据到 ES
    pub(super) async fn batch_insert_es(
        &self,
        client: &Elasticsearch,
        index: &str,
        batch: &BatchData,
        primary_key: Option<&str>,
    ) -> Result<()> {
        if batch.is_empty() {
            return Ok(());
        }

        let mut body: Vec<JsonBody<_>> = Vec::new();

        for record in batch {
            // 确定文档 ID
            let doc_id = if let Some(pk) = primary_key {
                record.get(pk)
                    .map(|v| TypeMapper::map_primary_key(v))
                    .unwrap_or_else(|| uuid::Uuid::new_v4().to_string())
            } else {
                uuid::Uuid::new_v4().to_string()
            };

            // 添加 index 操作
            body.push(json!({ "index": { "_id": doc_id } }).into());
            // 添加文档数据
            body.push(json!(record).into());
        }

        // 执行 bulk 操作
        let response = client
            .bulk(BulkParts::Index(index))
            .body(body)
            .send()
            .await
            .context("批量插入 ES 失败")?;

        // 检查响应
        let response_body = response.json::<Value>().await?;
        if let Some(errors) = response_body.get("errors") {
            if errors.as_bool().unwrap_or(false) {
                return Err(anyhow::anyhow!("批量插入存在错误"));
            }
        }

        Ok(())
    }

    /// 从 ES 读取批量数据（使用 scroll API）
    pub(super) async fn read_es_batch(
        &self,
        client: &Elasticsearch,
        index: &str,
        scroll_id: Option<&str>,
        size: usize,
    ) -> Result<(Vec<std::collections::HashMap<String, Value>>, Option<String>)> {
        if let Some(sid) = scroll_id {
            // 使用现有的 scroll_id 继续读取
            let response = client
                .scroll(elasticsearch::ScrollParts::None)
                .body(json!({
                    "scroll": "1m",
                    "scroll_id": sid
                }))
                .send()
                .await
                .context("Scroll 查询失败")?;

            let body = response.json::<Value>().await?;
            self.parse_es_search_response(&body)
        } else {
            // 初始查询
            let response = client
                .search(SearchParts::Index(&[index]))
                .scroll("1m")
                .size(size as i64)
                .body(json!({
                    "query": { "match_all": {} }
                }))
                .send()
                .await
                .context("初始查询失败")?;

            let body = response.json::<Value>().await?;
            self.parse_es_search_response(&body)
        }
    }

    /// 解析 ES 搜索响应
    fn parse_es_search_response(
        &self,
        body: &Value,
    ) -> Result<(Vec<std::collections::HashMap<String, Value>>, Option<String>)> {
        let mut batch = Vec::new();

        // 提取文档
        if let Some(hits) = body.get("hits").and_then(|h| h.get("hits")).and_then(|h| h.as_array()) {
            for hit in hits {
                if let Some(source) = hit.get("_source") {
                    let mut record = std::collections::HashMap::new();
                    
                    // 添加 _id 字段
                    if let Some(id) = hit.get("_id").and_then(|v| v.as_str()) {
                        record.insert("_id".to_string(), json!(id));
                    }
                    
                    // 添加其他字段
                    if let Some(obj) = source.as_object() {
                        for (key, value) in obj {
                            record.insert(key.clone(), value.clone());
                        }
                    }
                    
                    batch.push(record);
                }
            }
        }

        // 提取 scroll_id
        let scroll_id = body.get("_scroll_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        Ok((batch, scroll_id))
    }

    /// 获取 ES 索引的总文档数
    pub(super) async fn get_es_index_count(
        &self,
        client: &Elasticsearch,
        index: &str,
    ) -> Result<u64> {
        let response = client
            .count(elasticsearch::CountParts::Index(&[index]))
            .send()
            .await
            .context("查询文档数失败")?;

        let body = response.json::<Value>().await?;
        let count = body.get("count")
            .and_then(|v| v.as_u64())
            .unwrap_or(0);

        Ok(count)
    }

    /// 清除 scroll
    pub(super) async fn clear_es_scroll(
        &self,
        client: &Elasticsearch,
        scroll_id: &str,
    ) -> Result<()> {
        let _ = client
            .clear_scroll(elasticsearch::ClearScrollParts::None)
            .body(json!({ "scroll_id": scroll_id }))
            .send()
            .await;
        
        Ok(())
    }
}
