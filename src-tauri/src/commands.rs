// Tauri Commands 模块
// 提供前后端通信接口

use crate::datasource::{BatchTestResult, ConnectionResult, DataSourceManager, IndexMatchResult};
use crate::error_logger::{ErrorLog, ErrorLogger};
use crate::progress::{ProgressMonitor, TaskProgress};
use crate::storage::{DataSource, SyncTask};
use crate::sync_engine::SyncEngine;
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

/// 任务单元进度信息（用于前端显示）
#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitProgress {
    pub id: String,
    pub task_id: String,
    pub unit_name: String,
    pub unit_type: String,
    pub status: String,
    pub total_records: i64,
    pub processed_records: i64,
    pub error_message: Option<String>,
    pub started_at: Option<i64>,
    pub updated_at: i64,
    pub search_pattern: Option<String>,
}

/// 任务单元列表响应（包含状态统计）
#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitsResponse {
    /// 新增同步单元（配置表 + 运行记录表）
    pub new_units: Vec<TaskUnitProgress>,
    /// 已完成同步单元（历史记录表）
    pub completed_units: Vec<CompletedUnitProgress>,
    /// 状态统计
    pub statistics: TaskUnitStatistics,
}

/// 已完成单元进度信息
#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CompletedUnitProgress {
    pub id: String,
    pub task_id: String,
    pub unit_name: String,
    pub search_pattern: Option<String>,
    pub total_records: i64,
    pub completed_at: i64,
    pub duration: i64,
}

/// 任务单元状态统计
#[derive(Debug, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitStatistics {
    /// 总数
    pub total: usize,
    /// 等待中
    pub pending: usize,
    /// 进行中
    pub running: usize,
    /// 已完成
    pub completed: usize,
    /// 失败
    pub failed: usize,
    /// 暂停
    pub paused: usize,
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

// ============================================================================
// 任务单元管理相关 Commands
// ============================================================================

/// 获取任务的所有单元（包含状态统计）
/// 
/// 新逻辑:
/// 1. 先从任务配置 JSON 更新配置表(确保配置是最新的)
/// 2. 从配置表读取所有配置的单元
/// 3. 从运行记录表读取运行状态
/// 4. 从历史记录表读取已完成的单元
/// 5. 合并数据返回: new_units(配置+运行) + completed_units(历史)
#[tauri::command]
pub async fn get_task_units(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<TaskUnitsResponse, String> {
    log::info!("[get_task_units] 接收请求, task_id: {}", task_id);
    
    let storage = state.data_source_manager.storage();
    
    // 0. 先加载任务配置,更新配置表
    if let Some(task) = storage.load_task(&task_id).await.map_err(|e| e.to_string())? {
        // 从 JSON 更新配置表
        storage.update_task_config_from_json(&task.id, &task.config, &task.source_type, &task.target_type)
            .await
            .map_err(|e| {
                log::error!("[get_task_units] 更新配置表失败: {}", e);
                e.to_string()
            })?;
    }
    
    // 1. 加载配置表
    let configs = storage
        .load_unit_configs(&task_id)
        .await
        .map_err(|e| {
            log::error!("[get_task_units] 加载配置失败: {}", e);
            e.to_string()
        })?;
    
    // 2. 加载运行记录表
    let runtimes = storage
        .load_unit_runtimes(&task_id)
        .await
        .map_err(|e| {
            log::error!("[get_task_units] 加载运行记录失败: {}", e);
            e.to_string()
        })?;
    
    // 3. 加载历史记录表
    let histories = storage
        .load_unit_histories(&task_id)
        .await
        .map_err(|e| {
            log::error!("[get_task_units] 加载历史记录失败: {}", e);
            e.to_string()
        })?;
    
    // 4. 构建运行记录的 Map
    let runtime_map: std::collections::HashMap<String, _> = runtimes
        .into_iter()
        .map(|r| (r.unit_name.clone(), r))
        .collect();
    
    // 5. 构建历史记录的 Set（用于判断是否已完成）
    let history_names: std::collections::HashSet<String> = histories
        .iter()
        .map(|h| h.unit_name.clone())
        .collect();
    
    // 6. 合并配置表和运行记录表 -> new_units（排除已在历史记录中的）
    let mut new_units = Vec::new();
    let mut statistics = TaskUnitStatistics {
        total: configs.len(),
        pending: 0,
        running: 0,
        completed: histories.len(),
        failed: 0,
        paused: 0,
    };
    
    for config in configs {
        // 如果已在历史记录中，跳过（不显示在新增同步中）
        if history_names.contains(&config.unit_name) {
            continue;
        }
        
        let (status, total_records, processed_records, error_message, started_at) = 
            if let Some(runtime) = runtime_map.get(&config.unit_name) {
                // 有运行记录
                let status_str = match runtime.status {
                    crate::storage::TaskUnitStatus::Pending => {
                        statistics.pending += 1;
                        "pending"
                    },
                    crate::storage::TaskUnitStatus::Running => {
                        statistics.running += 1;
                        "running"
                    },
                    crate::storage::TaskUnitStatus::Failed => {
                        statistics.failed += 1;
                        "failed"
                    },
                    crate::storage::TaskUnitStatus::Completed => {
                        // 不应该出现在这里，因为已完成的应该在历史记录中
                        statistics.completed += 1;
                        "completed"
                    },
                    crate::storage::TaskUnitStatus::Paused => {
                        statistics.paused += 1;
                        "paused"
                    },
                };
                (
                    status_str.to_string(),
                    runtime.total_records,
                    runtime.processed_records,
                    runtime.error_message.clone(),
                    runtime.started_at,
                )
            } else {
                // 没有运行记录,说明是新增的
                statistics.pending += 1;
                (
                    "pending".to_string(),
                    0,
                    0,
                    None,
                    None,
                )
            };
        
        let unit_type_str = match config.unit_type {
            crate::storage::TaskUnitType::Table => "table",
            crate::storage::TaskUnitType::Index => "index",
        };
        
        new_units.push(TaskUnitProgress {
            id: config.id.clone(),
            task_id: config.task_id.clone(),
            unit_name: config.unit_name.clone(),
            unit_type: unit_type_str.to_string(),
            status,
            total_records,
            processed_records,
            error_message,
            started_at,
            updated_at: config.updated_at,
            search_pattern: config.search_pattern.clone(),
        });
    }
    
    // 7. 转换历史记录 -> completed_units
    let completed_units: Vec<CompletedUnitProgress> = histories
        .into_iter()
        .map(|h| CompletedUnitProgress {
            id: h.id,
            task_id: h.task_id,
            unit_name: h.unit_name,
            search_pattern: h.search_pattern,
            total_records: h.total_records,
            completed_at: h.completed_at,
            duration: h.duration,
        })
        .collect();
    
    log::info!("[get_task_units] 成功加载: 新增 {} 个, 已完成 {} 个", new_units.len(), completed_units.len());
    log::info!("[get_task_units] 状态统计: 总数={}, 等待={}, 进行中={}, 已完成={}, 失败={}, 暂停={}", 
        statistics.total, statistics.pending, statistics.running, 
        statistics.completed, statistics.failed, statistics.paused);
    
    Ok(TaskUnitsResponse {
        new_units,
        completed_units,
        statistics,
    })
}

/// 获取任务的历史记录
#[tauri::command]
pub async fn get_task_histories(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<crate::storage::TaskUnitHistory>, String> {
    log::info!("[get_task_histories] 接收请求, task_id: {}", task_id);
    
    let histories = state
        .data_source_manager
        .storage()
        .load_unit_histories(&task_id)
        .await
        .map_err(|e| {
            log::error!("[get_task_histories] 加载失败: {}", e);
            e.to_string()
        })?;
    
    log::info!("[get_task_histories] 成功加载 {} 条历史记录", histories.len());
    
    Ok(histories)
}

/// 重置失败的任务单元
/// 
/// 将所有失败的单元重置为 pending 状态，以便重新执行
/// 新逻辑: 重置运行记录表中的失败状态
#[tauri::command]
pub async fn reset_failed_units(
    task_id: String,
    _window: tauri::Window,
    state: State<'_, AppState>,
) -> Result<usize, String> {
    log::info!("[reset_failed_units] 重置任务 {} 的失败单元", task_id);
    
    let count = state
        .data_source_manager
        .storage()
        .reset_failed_runtimes(&task_id)
        .await
        .map_err(|e| {
            log::error!("[reset_failed_units] 重置失败: {}", e);
            e.to_string()
        })?;
    
    log::info!("[reset_failed_units] 成功重置 {} 个失败单元", count);
    
    Ok(count)
}

/// 重置指定的任务单元
/// 
/// 将指定单元重置为 pending 状态，以便重新执行
#[tauri::command]
pub async fn reset_unit(
    task_id: String,
    unit_name: String,
    _window: tauri::Window,
    state: State<'_, AppState>,
) -> Result<(), String> {
    log::info!("[reset_unit] 重置单元 {} / {}", task_id, unit_name);
    
    let storage = state.data_source_manager.storage();
    
    // 重置运行记录
    storage
        .reset_runtime(&task_id, &unit_name)
        .await
        .map_err(|e| {
            log::error!("[reset_unit] 重置失败: {}", e);
            e.to_string()
        })?;
    
    log::info!("[reset_unit] 重置成功");
    
    Ok(())
}

/// 按搜索关键字清除任务单元记录
/// 
/// 删除指定任务中匹配特定搜索关键字的历史记录
#[tauri::command]
pub async fn clear_task_units_by_pattern(
    task_id: String,
    pattern: String,
    _window: tauri::Window,
    state: State<'_, AppState>,
) -> Result<usize, String> {
    log::info!("[clear_task_units_by_pattern] 清除任务 {} 的关键字 {} 的历史记录", task_id, pattern);
    
    let count = state
        .data_source_manager
        .storage()
        .clear_histories_by_pattern(&task_id, &pattern)
        .await
        .map_err(|e| {
            log::error!("[clear_task_units_by_pattern] 清除失败: {}", e);
            e.to_string()
        })?;
    
    log::info!("[clear_task_units_by_pattern] 成功清除 {} 条历史记录", count);
    
    Ok(count)
}

