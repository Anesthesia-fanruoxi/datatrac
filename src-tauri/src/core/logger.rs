// 日志管理器
// 负责管理任务日志并推送到前端

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use tauri::{AppHandle, Emitter};

/// 日志级别
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum LogLevel {
    Info,
    Warn,
    Error,
}

/// 日志分类
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum LogCategory {
    Realtime,
    Summary,
    Verify,
    Error,
}

/// 日志条目
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LogEntry {
    pub timestamp: i64,
    pub level: LogLevel,
    pub category: LogCategory,
    pub message: String,
}

/// 日志事件
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct LogEvent {
    task_id: String,
    log: LogEntry,
}

/// 任务日志管理器
pub struct TaskLogger {
    app_handle: AppHandle,
    logs_map: Arc<RwLock<HashMap<String, Vec<LogEntry>>>>,
}

impl TaskLogger {
    /// 创建新的日志管理器
    pub fn new(app_handle: AppHandle) -> Self {
        Self {
            app_handle,
            logs_map: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 添加日志
    pub fn add_log(
        &self,
        task_id: &str,
        level: LogLevel,
        category: LogCategory,
        message: String,
    ) {
        let log_entry = LogEntry {
            timestamp: chrono::Utc::now().timestamp_millis(),
            level,
            category,
            message,
        };

        // 存储到内存（保留最近 1000 条）
        {
            let mut logs = self.logs_map.write().unwrap();
            let task_logs = logs.entry(task_id.to_string()).or_insert_with(Vec::new);
            task_logs.push(log_entry.clone());

            if task_logs.len() > 1000 {
                task_logs.remove(0);
            }
        }

        // 推送到前端
        let event = LogEvent {
            task_id: task_id.to_string(),
            log: log_entry,
        };

        if let Err(e) = self.app_handle.emit("task-log", &event) {
            log::error!("推送日志事件失败: {}", e);
        }
    }

    /// 获取任务日志
    pub fn get_logs(&self, task_id: &str) -> Vec<LogEntry> {
        let logs = self.logs_map.read().unwrap();
        logs.get(task_id).cloned().unwrap_or_default()
    }

    /// 清除任务日志
    pub fn clear_logs(&self, task_id: &str) {
        let mut logs = self.logs_map.write().unwrap();
        logs.remove(task_id);
    }
}
