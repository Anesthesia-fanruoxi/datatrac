// Elasticsearch 数据源操作
// 提供 ES 连接测试和元数据查询

use crate::commands::datasource::IndexMatchResult;
use crate::storage::models::DataSource;
use anyhow::Result;
use elasticsearch::{
    auth::Credentials,
    cat::CatIndicesParts,
    http::transport::{SingleNodeConnectionPool, TransportBuilder},
    Elasticsearch,
};
use url::Url;

/// 创建 ES 客户端
fn create_client(ds: &DataSource) -> Result<Elasticsearch> {
    let url = Url::parse(&format!("http://{}:{}", ds.host, ds.port))?;
    let conn_pool = SingleNodeConnectionPool::new(url);

    let credentials = Credentials::Basic(ds.username.clone(), ds.password.clone());

    let transport = TransportBuilder::new(conn_pool)
        .auth(credentials)
        .build()?;

    Ok(Elasticsearch::new(transport))
}

/// 测试 ES 连接
pub async fn test_connection(ds: &DataSource) -> Result<()> {
    let client = create_client(ds)?;
    client.ping().send().await?;
    Ok(())
}

/// 获取索引列表
pub async fn get_indices(ds: &DataSource) -> Result<Vec<String>> {
    let client = create_client(ds)?;

    let response = client
        .cat()
        .indices(CatIndicesParts::None)
        .format("json")
        .send()
        .await?;

    let body = response.json::<Vec<serde_json::Value>>().await?;

    let mut indices = Vec::new();
    for item in body {
        if let Some(index) = item.get("index").and_then(|v| v.as_str()) {
            // 过滤系统索引
            if !index.starts_with('.') {
                indices.push(index.to_string());
            }
        }
    }

    indices.sort();
    Ok(indices)
}

/// 通配符匹配索引
pub async fn match_indices(ds: &DataSource, pattern: &str) -> Result<IndexMatchResult> {
    let all_indices = get_indices(ds).await?;

    // 将通配符模式转换为正则表达式
    let regex_pattern = pattern
        .replace(".", "\\.")
        .replace("*", ".*")
        .replace("?", ".");

    let regex = regex::Regex::new(&format!("^{}$", regex_pattern))?;

    let matched_indices: Vec<String> = all_indices
        .into_iter()
        .filter(|index| regex.is_match(index))
        .collect();

    let total_count = matched_indices.len();

    Ok(IndexMatchResult {
        pattern: pattern.to_string(),
        matched_indices,
        total_count,
    })
}
