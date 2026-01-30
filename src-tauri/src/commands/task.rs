// 任务管理相关 Commands
// 提供任务的 CRUD、任务单元管理等 API

use crate::commands::AppState;
use crate::storage::models::{SyncTask, TaskUnit};
use serde::Serialize;
use tauri::State;

/// 任务单元响应
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitsResponse {
    pub units: Vec<TaskUnit>,
    pub statistics: TaskUnitStatistics,
}

/// 任务单元统计
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitStatistics {
    pub total: usize,
    pub pending: usize,
    pub running: usize,
    pub completed: usize,
    pub failed: usize,
}

/// 获取所有任务
#[tauri::command]
pub async fn list_tasks(state: State<'_, AppState>) -> Result<Vec<SyncTask>, String> {
    state
        .task_manager
        .list_tasks()
        .await
        .map_err(|e| e.to_string())
}

/// 获取单个任务
#[tauri::command]
pub async fn get_task(
    id: String,
    state: State<'_, AppState>,
) -> Result<Option<SyncTask>, String> {
    state
        .task_manager
        .get_task(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 创建任务
#[tauri::command]
pub async fn create_task(
    task: SyncTask,
    state: State<'_, AppState>,
) -> Result<String, String> {
    state
        .task_manager
        .create_task(task)
        .await
        .map_err(|e| e.to_string())
}

/// 更新任务
#[tauri::command]
pub async fn update_task(
    id: String,
    task: SyncTask,
    state: State<'_, AppState>,
) -> Result<(), String> {
    if id != task.id {
        return Err("任务 ID 不匹配".to_string());
    }

    state
        .task_manager
        .update_task(task)
        .await
        .map_err(|e| e.to_string())
}

/// 删除任务
#[tauri::command]
pub async fn delete_task(id: String, state: State<'_, AppState>) -> Result<(), String> {
    state
        .task_manager
        .delete_task(&id)
        .await
        .map_err(|e| e.to_string())
}

/// 获取任务单元列表
#[tauri::command]
pub async fn get_task_units(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<TaskUnitsResponse, String> {
    let units = state.task_manager.get_task_units(&task_id);

    // 统计各状态数量
    let mut statistics = TaskUnitStatistics {
        total: units.len(),
        pending: 0,
        running: 0,
        completed: 0,
        failed: 0,
    };

    for unit in &units {
        match unit.status.as_str() {
            "pending" => statistics.pending += 1,
            "running" => statistics.running += 1,
            "completed" => statistics.completed += 1,
            "failed" => statistics.failed += 1,
            _ => {}
        }
    }

    Ok(TaskUnitsResponse { units, statistics })
}

/// 重置失败的任务单元
#[tauri::command]
pub async fn reset_failed_units(
    task_id: String,
    state: State<'_, AppState>,
) -> Result<usize, String> {
    state
        .task_manager
        .reset_failed_units(&task_id)
        .await
        .map_err(|e| e.to_string())
}

/// 查询已同步的索引列表
#[tauri::command]
pub async fn list_synced_indices(
    source_id: String,
    state: State<'_, AppState>,
) -> Result<Vec<crate::storage::models::SyncedIndex>, String> {
    state
        .storage
        .list_synced_indices(&source_id)
        .await
        .map_err(|e| e.to_string())
}

/// 清除指定索引的同步历史
#[tauri::command]
pub async fn clear_synced_index(
    source_id: String,
    index_name: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .storage
        .clear_synced_index(&source_id, &index_name)
        .await
        .map_err(|e| e.to_string())
}

/// 清除所有同步历史
#[tauri::command]
pub async fn clear_all_synced_indices(
    source_id: String,
    state: State<'_, AppState>,
) -> Result<(), String> {
    state
        .storage
        .clear_all_synced_indices(&source_id)
        .await
        .map_err(|e| e.to_string())
}
