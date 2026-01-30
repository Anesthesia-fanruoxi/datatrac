// 进度监控器
// 负责聚合任务进度并推送到前端

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use tauri::{AppHandle, Emitter};

/// 任务进度
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskProgress {
    pub task_id: String,
    pub status: String,
    pub total_units: usize,
    pub completed_units: usize,
    pub failed_units: usize,
    pub total_records: u64,
    pub processed_records: u64,
    pub percentage: f64,
}

/// 进度监控器
pub struct ProgressMonitor {
    app_handle: AppHandle,
    progress_map: Arc<RwLock<HashMap<String, TaskProgress>>>,
}

impl ProgressMonitor {
    /// 创建新的进度监控器
    pub fn new(app_handle: AppHandle) -> Self {
        Self {
            app_handle,
            progress_map: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 更新任务进度
    pub fn update_progress(&self, progress: TaskProgress) {
        let task_id = progress.task_id.clone();

        // 更新内存中的进度
        {
            let mut map = self.progress_map.write().unwrap();
            map.insert(task_id.clone(), progress.clone());
        }

        // 推送到前端
        if let Err(e) = self.app_handle.emit("task-progress", &progress) {
            log::error!("推送进度事件失败: {}", e);
        }
    }

    /// 获取任务进度
    pub fn get_progress(&self, task_id: &str) -> Option<TaskProgress> {
        let map = self.progress_map.read().unwrap();
        map.get(task_id).cloned()
    }

    /// 清除任务进度
    pub fn clear_progress(&self, task_id: &str) {
        let mut map = self.progress_map.write().unwrap();
        map.remove(task_id);
    }
}
