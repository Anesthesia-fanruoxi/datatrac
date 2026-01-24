// 错误日志器模块
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use chrono::{DateTime, Utc};
use tauri::{AppHandle, Emitter};

/// 错误日志条目
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorLog {
    /// 时间戳
    pub timestamp: DateTime<Utc>,
    /// 错误类型
    pub error_type: String,
    /// 错误消息
    pub message: String,
    /// 相关数据（可选）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
}

impl ErrorLog {
    /// 创建新的错误日志
    pub fn new(error_type: String, message: String, data: Option<serde_json::Value>) -> Self {
        Self {
            timestamp: Utc::now(),
            error_type,
            message,
            data,
        }
    }
}

/// 错误日志器
/// 
/// 负责记录和管理同步任务的错误日志
#[derive(Clone)]
pub struct ErrorLogger {
    /// 存储每个任务的错误日志列表
    /// Key: task_id, Value: Vec<ErrorLog>
    errors: Arc<RwLock<HashMap<String, Vec<ErrorLog>>>>,
    /// Tauri 应用句柄，用于发送事件
    app_handle: Option<AppHandle>,
}

impl ErrorLogger {
    /// 创建新的错误日志器实例
    pub fn new() -> Self {
        Self {
            errors: Arc::new(RwLock::new(HashMap::new())),
            app_handle: None,
        }
    }

    /// 设置 Tauri 应用句柄
    pub fn with_app_handle(mut self, app_handle: AppHandle) -> Self {
        self.app_handle = Some(app_handle);
        self
    }

    /// 记录错误
    /// 
    /// # 参数
    /// * `task_id` - 任务 ID
    /// * `error` - 错误日志
    pub fn log_error(&self, task_id: &str, error: ErrorLog) {
        // 记录到内存
        {
            let mut errors = self.errors.write().unwrap();
            errors.entry(task_id.to_string())
                .or_insert_with(Vec::new)
                .push(error.clone());
        }

        // 发送事件到前端
        if let Some(app_handle) = &self.app_handle {
            let _ = self.emit_error(app_handle, task_id, &error);
        }
    }

    /// 获取指定任务的所有错误日志
    /// 
    /// # 参数
    /// * `task_id` - 任务 ID
    /// 
    /// # 返回
    /// 错误日志列表
    pub fn get_errors(&self, task_id: &str) -> Vec<ErrorLog> {
        let errors = self.errors.read().unwrap();
        errors.get(task_id)
            .cloned()
            .unwrap_or_default()
    }

    /// 清空指定任务的错误日志
    /// 
    /// # 参数
    /// * `task_id` - 任务 ID
    pub fn clear_errors(&self, task_id: &str) {
        let mut errors = self.errors.write().unwrap();
        errors.remove(task_id);
    }

    /// 获取指定任务的错误数量
    /// 
    /// # 参数
    /// * `task_id` - 任务 ID
    /// 
    /// # 返回
    /// 错误数量
    pub fn get_error_count(&self, task_id: &str) -> usize {
        let errors = self.errors.read().unwrap();
        errors.get(task_id)
            .map(|logs| logs.len())
            .unwrap_or(0)
    }

    /// 发送错误事件到前端
    /// 
    /// # 参数
    /// * `app_handle` - Tauri 应用句柄
    /// * `task_id` - 任务 ID
    /// * `error` - 错误日志
    fn emit_error(&self, app_handle: &AppHandle, task_id: &str, error: &ErrorLog) -> Result<(), tauri::Error> {
        #[derive(Serialize, Clone)]
        struct ErrorEvent {
            task_id: String,
            error: ErrorLog,
        }

        let event = ErrorEvent {
            task_id: task_id.to_string(),
            error: error.clone(),
        };

        app_handle.emit("sync-error", event)
    }
}

impl Default for ErrorLogger {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_create_error_log() {
        let error = ErrorLog::new(
            "TestError".to_string(),
            "测试错误消息".to_string(),
            Some(json!({"key": "value"}))
        );

        assert_eq!(error.error_type, "TestError");
        assert_eq!(error.message, "测试错误消息");
        assert!(error.data.is_some());
    }

    #[test]
    fn test_error_logger_new() {
        let logger = ErrorLogger::new();
        assert_eq!(logger.get_error_count("task-1"), 0);
    }

    #[test]
    fn test_log_and_get_errors() {
        let logger = ErrorLogger::new();
        let task_id = "task-001";

        // 记录第一个错误
        let error1 = ErrorLog::new(
            "ConnectionError".to_string(),
            "无法连接到数据库".to_string(),
            None
        );
        logger.log_error(task_id, error1);

        // 记录第二个错误
        let error2 = ErrorLog::new(
            "DataError".to_string(),
            "数据格式错误".to_string(),
            Some(json!({"table": "users"}))
        );
        logger.log_error(task_id, error2);

        // 验证错误数量
        assert_eq!(logger.get_error_count(task_id), 2);

        // 获取错误列表
        let errors = logger.get_errors(task_id);
        assert_eq!(errors.len(), 2);
        assert_eq!(errors[0].error_type, "ConnectionError");
        assert_eq!(errors[1].error_type, "DataError");
    }

    #[test]
    fn test_clear_errors() {
        let logger = ErrorLogger::new();
        let task_id = "task-002";

        // 记录错误
        let error = ErrorLog::new(
            "TestError".to_string(),
            "测试".to_string(),
            None
        );
        logger.log_error(task_id, error);
        assert_eq!(logger.get_error_count(task_id), 1);

        // 清空错误
        logger.clear_errors(task_id);
        assert_eq!(logger.get_error_count(task_id), 0);
        assert_eq!(logger.get_errors(task_id).len(), 0);
    }

    #[test]
    fn test_multiple_tasks() {
        let logger = ErrorLogger::new();

        // 任务 1 的错误
        let error1 = ErrorLog::new(
            "Error1".to_string(),
            "任务1错误".to_string(),
            None
        );
        logger.log_error("task-1", error1);

        // 任务 2 的错误
        let error2 = ErrorLog::new(
            "Error2".to_string(),
            "任务2错误".to_string(),
            None
        );
        logger.log_error("task-2", error2.clone());
        logger.log_error("task-2", error2);

        // 验证各任务的错误数量
        assert_eq!(logger.get_error_count("task-1"), 1);
        assert_eq!(logger.get_error_count("task-2"), 2);
        assert_eq!(logger.get_error_count("task-3"), 0);
    }

    #[test]
    fn test_error_log_with_data() {
        let logger = ErrorLogger::new();
        let task_id = "task-003";

        let error = ErrorLog::new(
            "DataConversionError".to_string(),
            "类型转换失败".to_string(),
            Some(json!({
                "table": "users",
                "column": "age",
                "value": "invalid",
                "expected_type": "integer"
            }))
        );
        logger.log_error(task_id, error);

        let errors = logger.get_errors(task_id);
        assert_eq!(errors.len(), 1);
        
        let data = errors[0].data.as_ref().unwrap();
        assert_eq!(data["table"], "users");
        assert_eq!(data["column"], "age");
    }

    #[test]
    fn test_get_errors_empty_task() {
        let logger = ErrorLogger::new();
        let errors = logger.get_errors("non-existent-task");
        assert_eq!(errors.len(), 0);
    }
}
