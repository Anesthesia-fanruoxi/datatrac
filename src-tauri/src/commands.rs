// Tauri Commands 模块
// 提供前后端通信接口

use crate::datasource::{BatchTestResult, ConnectionResult, DataSourceManager, IndexMatchResult};
use crate::error_logger::{ErrorLog, ErrorLogger};
use crate::progress::{ProgressMonitor, TaskProgress};
use crate::storage::{DataSource, SyncTask};
use crate::sync_engine::{SyncEngine, SyncTaskConfig};
use std::sync::Arc;
use tauri::State;

// ============================================================================
// 应用状态
// ============================================================================

/// 应用状态，包含所有后端服务
pub struct AppState {
    pub data_source_manager: Arc<DataSourceManager>,
    pub progress_monitor: Arc<ProgressMonitor>,
    pub error_logger: Arc<ErrorLogger>,
    pub sync_engine: Arc<SyncEngine>,
}

// ============================================================================
// 数据传输对象 (DTO)
// ============================================================================

/// 创建数据源的请求数据
#[derive(Debug, serde::Deserialize)]
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

// ============================================================================
// 子任务 9.1: 数据源管理相关 Commands
// ============================================================================

/// 获取所有数据源
#[tauri::command]
pub async fn list_data_sources(
    state: State<'_, AppState>,
) -> Result<Vec<DataSource>, String> {
    state
        .data_source_manager
        .list_data_sources()
        .await
        .map_err(|e| e.to_string())
}

/// 获取单个数据源
#[tauri::command]
pub async fn get_data_source(
    id: String,
    state: State<'_, AppState>,
) -> Result<Option<DataSource>, String> {
    state
        .data_source_manager
        .get_data_source(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 创建数据源
#[tauri::command]
pub async fn create_data_source(
    data_source: CreateDataSourceRequest,
    state: State<'_, AppState>,
) -> Result<String, String> {
    use crate::storage::{DataSource, DataSourceType};
    use chrono::Utc;
    
    // 将请求数据转换为 DataSource
    let source_type = DataSourceType::from_str(&data_source.source_type)
        .map_err(|e| e.to_string())?;
    
    let ds = DataSource {
        id: String::new(), // 会被 create_data_source 方法重新生成
        name: data_source.name,
        source_type,
        host: data_source.host,
        port: data_source.port,
        username: data_source.username,
        password: data_source.password,
        database: data_source.database,
        created_at: Utc::now(),
        updated_at: Utc::now(),
    };
    
    state
        .data_source_manager
        .create_data_source(ds)
        .await
        .map_err(|e| e.to_string())
}

/// 更新数据源
#[tauri::command]
pub async fn update_data_source(
    id: String,
    data_source: DataSource,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .data_source_manager
        .update_data_source(&id, data_source)
        .await
        .map_err(|e| e.to_string())
}

/// 删除数据源
#[tauri::command]
pub async fn delete_data_source(
    id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .data_source_manager
        .delete_data_source(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 测试数据源连接
#[tauri::command]
pub async fn test_connection(
    id: String,
    window: tauri::Window,
    state: State<'_, AppState>,
) -> Result<ConnectionResult, String> {
    state
        .data_source_manager
        .test_connection_with_events(&id, Some(window))
        .await
        .map_err(|e| e.to_string())
}

/// 批量测试所有数据源连接
#[tauri::command]
pub async fn batch_test_connections(
    window: tauri::Window,
    skip_failed_step1: bool,
    state: State<'_, AppState>,
) -> Result<BatchTestResult, String> {
    state
        .data_source_manager
        .batch_test_connections(Some(window), skip_failed_step1)
        .await
        .map_err(|e| e.to_string())
}

// ============================================================================
// 子任务 9.2: 元数据查询相关 Commands
// ============================================================================

/// 获取 MySQL 数据库列表
#[tauri::command]
pub async fn get_databases(
    source_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    state
        .data_source_manager
        .get_databases(&source_id)
        .await
        .map_err(|e| e.to_string())
}

/// 获取 MySQL 表列表
#[tauri::command]
pub async fn get_tables(
    source_id: String,
    database: String,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    state
        .data_source_manager
        .get_tables(&source_id, &database)
        .await
        .map_err(|e| e.to_string())
}

/// 获取 ES 索引列表
#[tauri::command]
pub async fn get_indices(
    source_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    state
        .data_source_manager
        .get_indices(&source_id)
        .await
        .map_err(|e| e.to_string())
}

/// 通配符匹配 ES 索引
#[tauri::command]
pub async fn match_indices(
    source_id: String,
    pattern: String,
    state: State<'_, AppState>,
) -> Result<IndexMatchResult, String> {
    state
        .data_source_manager
        .match_indices(&source_id, &pattern)
        .await
        .map_err(|e| e.to_string())
}

// ============================================================================
// 子任务 9.3: 同步任务管理相关 Commands
// ============================================================================

/// 获取所有同步任务
#[tauri::command]
pub async fn list_tasks(
    state: State<'_, AppState>,
) -> Result<Vec<SyncTask>, String> {
    state
        .data_source_manager
        .storage()
        .load_tasks()
        .await
        .map_err(|e| e.to_string())
}

/// 获取单个同步任务
#[tauri::command]
pub async fn get_task(
    id: String,
    state: State<'_, AppState>,
) -> Result<Option<SyncTask>, String> {
    state
        .data_source_manager
        .storage()
        .load_task(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 创建同步任务
#[tauri::command]
pub async fn create_task(
    task: SyncTask,
    state: State<'_, AppState>,
) -> Result<String, String> {
    state
        .data_source_manager
        .storage()
        .save_task(&task)
        .await
        .map_err(|e| e.to_string())?;
    
    Ok(task.id)
}

/// 更新同步任务
#[tauri::command]
pub async fn update_task(
    id: String,
    task: SyncTask,
    state: State<'_, AppState>,
) -> Result<(), String> {
    // 确保 ID 一致
    if id != task.id {
        return Err("任务 ID 不匹配".to_string());
    }
    
    state
        .data_source_manager
        .storage()
        .save_task(&task)
        .await
        .map_err(|e| e.to_string())
}

/// 删除同步任务
#[tauri::command]
pub async fn delete_task(
    id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .data_source_manager
        .storage()
        .delete_task(&id)
        .await
        .map_err(|e| e.to_string())
}

// ============================================================================
// 子任务 9.4: 任务执行控制相关 Commands
// ============================================================================

/// 启动同步任务
#[tauri::command]
pub async fn start_sync(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    // 在后台启动同步任务
    let engine = state.sync_engine.clone();
    
    tokio::spawn(async move {
        if let Err(e) = engine.start_sync_by_id(&task_id).await {
            log::error!("同步任务执行失败: {}", e);
        }
    });
    
    Ok(())
}

/// 暂停同步任务
#[tauri::command]
pub async fn pause_sync(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .sync_engine
        .pause_sync(&task_id)
        .await
        .map_err(|e| e.to_string())
}

/// 恢复同步任务
#[tauri::command]
pub async fn resume_sync(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .sync_engine
        .resume_sync(&task_id)
        .await
        .map_err(|e| e.to_string())
}

/// 获取任务进度
#[tauri::command]
pub async fn get_progress(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<Option<TaskProgress>, String> {
    Ok(state.progress_monitor.get_progress(&task_id))
}

/// 获取错误日志
#[tauri::command]
pub async fn get_errors(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<ErrorLog>, String> {
    Ok(state.error_logger.get_errors(&task_id))
}

/// 获取任务日志
#[tauri::command]
pub async fn get_task_logs(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<crate::progress::LogEntry>, String> {
    Ok(state.progress_monitor.get_logs(&task_id))
}
