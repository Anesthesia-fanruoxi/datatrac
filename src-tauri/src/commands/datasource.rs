// 数据源管理相关 Commands
// 提供数据源的 CRUD、连接测试、元数据查询等 API

use crate::commands::AppState;
use crate::storage::models::{DataSource, DataSourceType};
use serde::{Deserialize, Serialize};
use tauri::State;

/// 创建数据源请求
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateDataSourceRequest {
    pub name: String,
    #[serde(rename = "type")]
    pub source_type: String,
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub database: Option<String>,
}

/// 连接测试结果
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ConnectionResult {
    pub success: bool,
    pub message: String,
    pub duration_ms: u64,
}

/// 索引匹配结果
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct IndexMatchResult {
    pub pattern: String,
    pub matched_indices: Vec<String>,
    pub total_count: usize,
}

/// 获取所有数据源
#[tauri::command]
pub async fn list_datasources(state: State<'_, AppState>) -> Result<Vec<DataSource>, String> {
    state
        .datasource_manager
        .list_datasources()
        .await
        .map_err(|e| e.to_string())
}

/// 获取单个数据源
#[tauri::command]
pub async fn get_datasource(
    id: String,
    state: State<'_, AppState>,
) -> Result<Option<DataSource>, String> {
    state
        .datasource_manager
        .get_datasource(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 创建数据源
#[tauri::command]
pub async fn create_datasource(
    request: CreateDataSourceRequest,
    state: State<'_, AppState>,
) -> Result<String, String> {
    let source_type = match request.source_type.as_str() {
        "mysql" => DataSourceType::Mysql,
        "elasticsearch" => DataSourceType::Elasticsearch,
        _ => return Err("不支持的数据源类型".to_string()),
    };

    state
        .datasource_manager
        .create_datasource(
            request.name,
            source_type,
            request.host,
            request.port,
            request.username,
            request.password,
            request.database,
        )
        .await
        .map_err(|e| e.to_string())
}

/// 更新数据源
#[tauri::command]
pub async fn update_datasource(
    id: String,
    datasource: DataSource,
    state: State<'_, AppState>,
) -> Result<(), String> {
    if id != datasource.id {
        return Err("数据源 ID 不匹配".to_string());
    }

    state
        .datasource_manager
        .update_datasource(datasource)
        .await
        .map_err(|e| e.to_string())
}

/// 删除数据源
#[tauri::command]
pub async fn delete_datasource(id: String, state: State<'_, AppState>) -> Result<(), String> {
    state
        .datasource_manager
        .delete_datasource(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 测试数据源连接
#[tauri::command]
pub async fn test_connection(
    id: String,
    state: State<'_, AppState>,
) -> Result<ConnectionResult, String> {
    state
        .datasource_manager
        .test_connection(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 获取 MySQL 数据库列表
#[tauri::command]
pub async fn get_databases(
    id: String,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    state
        .datasource_manager
        .get_databases(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 获取 MySQL 表列表
#[tauri::command]
pub async fn get_tables(
    id: String,
    database: String,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    state
        .datasource_manager
        .get_tables(&id, &database)
        .await
        .map_err(|e| e.to_string())
}

/// 获取 ES 索引列表
#[tauri::command]
pub async fn get_indices(id: String, state: State<'_, AppState>) -> Result<Vec<String>, String> {
    state
        .datasource_manager
        .get_indices(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 通配符匹配 ES 索引
#[tauri::command]
pub async fn match_indices(
    id: String,
    pattern: String,
    state: State<'_, AppState>,
) -> Result<IndexMatchResult, String> {
    state
        .datasource_manager
        .match_indices(&id, &pattern)
        .await
        .map_err(|e| e.to_string())
}
