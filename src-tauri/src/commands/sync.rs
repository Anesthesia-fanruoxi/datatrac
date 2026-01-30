// 同步控制相关 Commands
// 提供任务启动、暂停、恢复、进度查询等 API

use crate::commands::AppState;
use crate::core::logger::LogEntry;
use crate::core::monitor::TaskProgress;
use tauri::State;

/// 启动同步任务
#[tauri::command]
pub async fn start_sync(task_id: String, state: State<'_, AppState>) -> Result<(), String> {
    // 在后台启动同步任务
    let engine = state.sync_engine.clone();

    tokio::spawn(async move {
        if let Err(e) = engine.start_sync(&task_id).await {
            log::error!("同步任务执行失败: {}", e);
        }
    });

    Ok(())
}

/// 暂停同步任务
#[tauri::command]
pub async fn pause_sync(task_id: String, state: State<'_, AppState>) -> Result<(), String> {
    state
        .sync_engine
        .pause_sync(&task_id)
        .await
        .map_err(|e| e.to_string())
}

/// 恢复同步任务
#[tauri::command]
pub async fn resume_sync(task_id: String, state: State<'_, AppState>) -> Result<(), String> {
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

/// 获取任务日志
#[tauri::command]
pub async fn get_logs(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<LogEntry>, String> {
    Ok(state.task_logger.get_logs(&task_id))
}
