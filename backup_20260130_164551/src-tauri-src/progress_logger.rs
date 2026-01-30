// 任务日志管理模块
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter};

/// 日志条目
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LogEntry {
    /// 时间戳
    pub timestamp: String,
    /// 日志级别
    pub level: LogLevel,
    /// 日志分类
    pub category: LogCategory,
    /// 日志消息
    pub message: String,
}

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
    Realtime,  // 实时日志（所有详细信息）
    Summary,   // 明细日志（批次摘要）
    Verify,    // 校验日志（数据校验结果）
    Error,     // 错误日志（错误信息）
}

/// 日志事件数据
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LogEvent {
    /// 任务 ID
    pub task_id: String,
    /// 日志条目
    pub log: LogEntry,
}

/// 任务日志管理器
pub struct TaskLogger {
    /// 任务日志映射 (task_id -> Vec<LogEntry>)，保留最近 1000 条
    task_logs: Arc<RwLock<HashMap<String, Vec<LogEntry>>>>,
    /// Tauri 应用句柄，用于发送事件
    app_handle: Arc<RwLock<Option<AppHandle>>>,
}

impl TaskLogger {
    /// 创建新的任务日志管理器
    pub fn new() -> Self {
        Self {
            task_logs: Arc::new(RwLock::new(HashMap::new())),
            app_handle: Arc::new(RwLock::new(None)),
        }
    }

    /// 设置应用句柄（用于发送事件）
    pub fn set_app_handle(&self, handle: AppHandle) {
        let mut app_handle = self.app_handle.write().unwrap();
        *app_handle = Some(handle);
    }

    /// 添加日志
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `level`: 日志级别
    /// - `category`: 日志分类
    /// - `message`: 日志消息
    pub fn add_log(&self, task_id: &str, level: LogLevel, category: LogCategory, message: String) {
        let now = chrono::Local::now().format("%Y-%m-%d %H:%M:%S").to_string();
        
        // 创建日志条目
        let log_entry = LogEntry {
            timestamp: now,
            level,
            category,
            message,
        };
        
        // 存储到内存（保留最近 1000 条）
        {
            let mut logs = self.task_logs.write().unwrap();
            let task_logs = logs.entry(task_id.to_string()).or_insert_with(Vec::new);
            task_logs.push(log_entry.clone());
            
            // 只保留最近 1000 条
            if task_logs.len() > 1000 {
                task_logs.remove(0);
            }
        }
        
        // 发送事件到前端
        self.emit_log_event(task_id, log_entry);
    }
    
    /// 发送日志事件到前端
    fn emit_log_event(&self, task_id: &str, log: LogEntry) {
        let app_handle = self.app_handle.read().unwrap();
        if let Some(handle) = app_handle.as_ref() {
            let event = LogEvent {
                task_id: task_id.to_string(),
                log: log.clone(),
            };
            
            if let Err(e) = handle.emit("task-log", &event) {
                log::error!("发送日志事件失败: {}", e);
            }
        }
    }

    /// 获取任务日志（从内存读取最近 1000 条）
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// 
    /// # 返回
    /// 日志列表
    pub fn get_logs(&self, task_id: &str) -> Vec<LogEntry> {
        let logs = self.task_logs.read().unwrap();
        logs.get(task_id).cloned().unwrap_or_default()
    }

    /// 清除任务日志（从内存中删除）
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn clear_logs(&self, task_id: &str) {
        let mut logs = self.task_logs.write().unwrap();
        logs.remove(task_id);
    }
}

impl Default for TaskLogger {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add_log() {
        let logger = TaskLogger::new();
        
        logger.add_log("task1", LogLevel::Info, LogCategory::Realtime, "测试日志".to_string());
        
        let logs = logger.get_logs("task1");
        assert_eq!(logs.len(), 1);
        assert_eq!(logs[0].message, "测试日志");
        assert_eq!(logs[0].level, LogLevel::Info);
        assert_eq!(logs[0].category, LogCategory::Realtime);
    }

    #[test]
    fn test_clear_logs() {
        let logger = TaskLogger::new();
        
        logger.add_log("task1", LogLevel::Info, LogCategory::Realtime, "测试日志".to_string());
        assert_eq!(logger.get_logs("task1").len(), 1);
        
        logger.clear_logs("task1");
        assert_eq!(logger.get_logs("task1").len(), 0);
    }
    
    #[test]
    fn test_multiple_tasks() {
        let logger = TaskLogger::new();
        
        // 任务1添加日志
        logger.add_log("task1", LogLevel::Info, LogCategory::Realtime, "任务1日志".to_string());
        // 任务2添加日志
        logger.add_log("task2", LogLevel::Info, LogCategory::Summary, "任务2日志".to_string());
        
        // 验证任务之间不互相影响
        assert_eq!(logger.get_logs("task1").len(), 1);
        assert_eq!(logger.get_logs("task2").len(), 1);
        assert_eq!(logger.get_logs("task1")[0].message, "任务1日志");
        assert_eq!(logger.get_logs("task2")[0].message, "任务2日志");
    }
    
    #[test]
    fn test_max_1000_logs() {
        let logger = TaskLogger::new();
        
        // 添加 1500 条日志
        for i in 0..1500 {
            logger.add_log("task1", LogLevel::Info, LogCategory::Realtime, format!("日志 {}", i));
        }
        
        // 验证只保留最近 1000 条
        let logs = logger.get_logs("task1");
        assert_eq!(logs.len(), 1000);
        assert_eq!(logs[0].message, "日志 500"); // 第一条应该是第 500 条
        assert_eq!(logs[999].message, "日志 1499"); // 最后一条应该是第 1499 条
    }
}
