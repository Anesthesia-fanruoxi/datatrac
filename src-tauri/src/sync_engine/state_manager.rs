// 任务状态管理模块

use super::types::SyncTaskState;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};

/// 任务状态管理器
pub struct TaskStateManager {
    /// 任务状态映射 (task_id -> SyncTaskState)
    task_states: Arc<RwLock<HashMap<String, SyncTaskState>>>,
}

impl TaskStateManager {
    /// 创建新的任务状态管理器
    pub fn new() -> Self {
        Self {
            task_states: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 获取任务状态
    pub fn get_task_state(&self, task_id: &str) -> Option<SyncTaskState> {
        let states = self.task_states.read().unwrap();
        states.get(task_id).cloned()
    }

    /// 设置任务状态
    pub fn set_task_state(&self, task_id: &str, state: SyncTaskState) {
        let mut states = self.task_states.write().unwrap();
        states.insert(task_id.to_string(), state);
    }

    /// 更新任务运行状态
    pub fn set_running(&self, task_id: &str, is_running: bool) {
        let mut states = self.task_states.write().unwrap();
        if let Some(s) = states.get_mut(task_id) {
            s.is_running = is_running;
        }
    }

    /// 更新任务暂停状态
    pub fn set_paused(&self, task_id: &str, is_paused: bool) {
        let mut states = self.task_states.write().unwrap();
        if let Some(s) = states.get_mut(task_id) {
            s.is_paused = is_paused;
        }
    }

    /// 检查任务是否正在运行
    pub fn is_task_running(&self, task_id: &str) -> bool {
        self.get_task_state(task_id)
            .map(|state| state.is_running)
            .unwrap_or(false)
    }

    /// 检查任务是否暂停
    pub fn is_task_paused(&self, task_id: &str) -> bool {
        self.get_task_state(task_id)
            .map(|state| state.is_paused)
            .unwrap_or(false)
    }
}
