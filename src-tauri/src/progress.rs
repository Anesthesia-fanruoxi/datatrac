// 进度监控器模块
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter};

// 导入日志模块
use crate::progress_logger::TaskLogger;
// 导入任务管理器类型
use crate::task_manager::TaskUnit;

// 重新导出日志类型，保持向后兼容
pub use crate::progress_logger::{LogEntry, LogLevel, LogCategory};

/// 任务进度信息
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskProgress {
    /// 任务 ID
    pub task_id: String,
    /// 任务状态
    pub status: TaskStatus,
    /// 总记录数
    pub total_records: u64,
    /// 已处理记录数
    pub processed_records: u64,
    /// 进度百分比 (0-100)
    pub percentage: f64,
    /// 同步速度 (记录/秒)
    pub speed: f64,
    /// 预计剩余时间 (秒)
    pub estimated_time: u64,
    /// 开始时间 (ISO 8601 格式)
    pub start_time: String,
    /// 当前处理的表/索引
    pub current_table: Option<String>,
    /// 表级别进度列表
    pub table_progress: Vec<TableProgress>,
    /// 任务单元列表 (新增)
    pub task_units: Vec<TaskUnit>,
}

/// 表级别进度信息
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TableProgress {
    /// 表名
    pub table_name: String,
    /// 表状态
    pub status: TableStatus,
    /// 总记录数
    pub total_records: u64,
    /// 已处理记录数
    pub processed_records: u64,
    /// 进度百分比 (0-100)
    pub percentage: f64,
}

/// 表状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TableStatus {
    Waiting,
    Running,
    Completed,
    Failed,
}

/// 任务状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskStatus {
    Running,
    Paused,
    Completed,
    Failed,
}

/// 进度监控器
pub struct ProgressMonitor {
    /// 当前进度映射 (task_id -> TaskProgress)
    current_progress: Arc<RwLock<HashMap<String, TaskProgress>>>,
    /// 任务日志管理器
    logger: Arc<TaskLogger>,
    /// Tauri 应用句柄，用于发送事件
    app_handle: Option<AppHandle>,
}

impl ProgressMonitor {
    /// 创建新的进度监控器
    pub fn new() -> Self {
        Self {
            current_progress: Arc::new(RwLock::new(HashMap::new())),
            logger: Arc::new(TaskLogger::new()),
            app_handle: None,
        }
    }

    /// 设置应用句柄（用于发送事件）
    pub fn set_app_handle(&mut self, handle: AppHandle) {
        self.app_handle = Some(handle.clone());
        self.logger.set_app_handle(handle);
    }

    /// 启动任务跟踪
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `total`: 总记录数
    pub fn start_task(&self, task_id: &str, total: u64) {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        
        let start_time = chrono::DateTime::from_timestamp(now as i64, 0)
            .unwrap()
            .to_rfc3339();

        let progress = TaskProgress {
            task_id: task_id.to_string(),
            status: TaskStatus::Running,
            total_records: total,
            processed_records: 0,
            percentage: 0.0,
            speed: 0.0,
            estimated_time: 0,
            start_time,
            current_table: None,
            table_progress: Vec::new(),
            task_units: Vec::new(),
        };

        {
            let mut map = self.current_progress.write().unwrap();
            map.insert(task_id.to_string(), progress.clone());
        }

        // 清除旧日志
        self.logger.clear_logs(task_id);

        // 发送初始进度事件
        self.emit_progress_sync(&progress);
    }

    /// 更新任务进度
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `processed`: 已处理记录数
    pub fn update_progress(&self, task_id: &str, processed: u64) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.processed_records = processed;
            
            // 计算百分比
            if progress.total_records > 0 {
                progress.percentage = (processed as f64 / progress.total_records as f64) * 100.0;
            }
            
            // 计算速度和预估时间
            let start_timestamp = chrono::DateTime::parse_from_rfc3339(&progress.start_time)
                .unwrap()
                .timestamp() as u64;
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();
            let elapsed = now.saturating_sub(start_timestamp);
            
            if elapsed > 0 {
                progress.speed = processed as f64 / elapsed as f64;
                
                // 预估剩余时间
                if progress.speed > 0.0 {
                    let remaining = progress.total_records.saturating_sub(processed);
                    progress.estimated_time = (remaining as f64 / progress.speed) as u64;
                }
            }
            
            // 发送进度更新事件
            let progress_clone = progress.clone();
            drop(map); // 释放锁
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 更新当前处理的表/索引
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `table_name`: 表/索引名称
    pub fn update_current_table(&self, task_id: &str, table_name: Option<String>) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.current_table = table_name;
            
            // 发送进度更新事件
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 完成任务
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn complete_task(&self, task_id: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.status = TaskStatus::Completed;
            progress.percentage = 100.0;
            progress.processed_records = progress.total_records;
            progress.estimated_time = 0;
            
            // 发送完成事件
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 任务失败
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `error`: 错误信息
    pub fn fail_task(&self, task_id: &str, _error: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.status = TaskStatus::Failed;
            progress.estimated_time = 0;
            
            // 发送失败事件
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 暂停任务
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn pause_task(&self, task_id: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.status = TaskStatus::Paused;
            
            // 发送暂停事件
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 恢复任务
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn resume_task(&self, task_id: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.status = TaskStatus::Running;
            
            // 发送恢复事件
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 获取任务进度
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// 
    /// # 返回
    /// 任务进度信息，如果任务不存在则返回 None
    pub fn get_progress(&self, task_id: &str) -> Option<TaskProgress> {
        let map = self.current_progress.read().unwrap();
        map.get(task_id).cloned()
    }

    /// 清除任务进度
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn clear_progress(&self, task_id: &str) {
        let mut map = self.current_progress.write().unwrap();
        map.remove(task_id);
    }

    /// 发送进度事件到前端（同步版本）
    fn emit_progress_sync(&self, progress: &TaskProgress) {
        if let Some(handle) = &self.app_handle {
            // 使用 Tauri 的 emit 方法发送事件
            if let Err(e) = handle.emit("task-progress", progress) {
                log::error!("Failed to emit progress event: {}", e);
            }
        }
    }

    /// 发送进度事件到前端（异步版本）
    pub async fn emit_progress(&self, progress: &TaskProgress) {
        if let Some(handle) = &self.app_handle {
            // 使用 Tauri 的 emit 方法发送事件
            if let Err(e) = handle.emit("task-progress", progress) {
                log::error!("Failed to emit progress event: {}", e);
            }
        }
    }

    /// 初始化表进度列表
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `table_names`: 表名列表
    /// - `table_counts`: 每个表的记录数
    pub fn init_table_progress(&self, task_id: &str, table_names: Vec<String>, table_counts: Vec<u64>) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.table_progress = table_names
                .into_iter()
                .zip(table_counts.into_iter())
                .map(|(name, count)| TableProgress {
                    table_name: name,
                    status: TableStatus::Waiting,
                    total_records: count,
                    processed_records: 0,
                    percentage: 0.0,
                })
                .collect();
            
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 更新表进度
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `table_name`: 表名
    /// - `processed`: 已处理记录数
    pub fn update_table_progress(&self, task_id: &str, table_name: &str, processed: u64) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            if let Some(table) = progress.table_progress.iter_mut().find(|t| t.table_name == table_name) {
                table.processed_records = processed;
                if table.total_records > 0 {
                    table.percentage = (processed as f64 / table.total_records as f64) * 100.0;
                }
                if table.status == TableStatus::Waiting {
                    table.status = TableStatus::Running;
                }
            }
            
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 完成表同步
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `table_name`: 表名
    pub fn complete_table(&self, task_id: &str, table_name: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            if let Some(table) = progress.table_progress.iter_mut().find(|t| t.table_name == table_name) {
                table.status = TableStatus::Completed;
                table.percentage = 100.0;
                table.processed_records = table.total_records;
            }
            
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 表同步失败
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `table_name`: 表名
    pub fn fail_table(&self, task_id: &str, table_name: &str) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            if let Some(table) = progress.table_progress.iter_mut().find(|t| t.table_name == table_name) {
                table.status = TableStatus::Failed;
            }
            
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 添加日志
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `level`: 日志级别
    /// - `category`: 日志分类
    /// - `message`: 日志消息
    pub fn add_log(&self, task_id: &str, level: LogLevel, category: LogCategory, message: String) {
        self.logger.add_log(task_id, level, category, message);
    }

    /// 获取任务日志
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// 
    /// # 返回
    /// 日志列表
    pub fn get_logs(&self, task_id: &str) -> Vec<LogEntry> {
        self.logger.get_logs(task_id)
    }

    /// 清除任务日志
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn clear_logs(&self, task_id: &str) {
        self.logger.clear_logs(task_id);
    }

    /// 更新任务单元列表
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `task_units`: 任务单元列表
    pub fn update_task_units(&self, task_id: &str, task_units: Vec<TaskUnit>) {
        let mut map = self.current_progress.write().unwrap();
        
        if let Some(progress) = map.get_mut(task_id) {
            progress.task_units = task_units;
            
            let progress_clone = progress.clone();
            drop(map);
            self.emit_progress_sync(&progress_clone);
        }
    }

    /// 获取任务单元列表
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// 
    /// # 返回
    /// 任务单元列表
    pub fn get_task_units(&self, task_id: &str) -> Vec<TaskUnit> {
        let map = self.current_progress.read().unwrap();
        map.get(task_id)
            .map(|p| p.task_units.clone())
            .unwrap_or_default()
    }
}

impl Default for ProgressMonitor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_start_task() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.task_id, "task1");
        assert_eq!(progress.status, TaskStatus::Running);
        assert_eq!(progress.total_records, 1000);
        assert_eq!(progress.processed_records, 0);
        assert_eq!(progress.percentage, 0.0);
    }

    #[test]
    fn test_update_progress() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        
        // 等待一秒以便计算速度
        std::thread::sleep(std::time::Duration::from_secs(1));
        
        monitor.update_progress("task1", 500);
        
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.processed_records, 500);
        assert_eq!(progress.percentage, 50.0);
        assert!(progress.speed > 0.0);
    }

    #[test]
    fn test_complete_task() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        monitor.complete_task("task1");
        
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.status, TaskStatus::Completed);
        assert_eq!(progress.percentage, 100.0);
        assert_eq!(progress.processed_records, 1000);
    }

    #[test]
    fn test_fail_task() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        monitor.fail_task("task1", "Test error");
        
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.status, TaskStatus::Failed);
    }

    #[test]
    fn test_pause_resume_task() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        
        monitor.pause_task("task1");
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.status, TaskStatus::Paused);
        
        monitor.resume_task("task1");
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.status, TaskStatus::Running);
    }

    #[test]
    fn test_percentage_calculation() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        
        monitor.update_progress("task1", 250);
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.percentage, 25.0);
        
        monitor.update_progress("task1", 750);
        let progress = monitor.get_progress("task1").unwrap();
        assert_eq!(progress.percentage, 75.0);
    }

    #[test]
    fn test_clear_progress() {
        let monitor = ProgressMonitor::new();
        monitor.start_task("task1", 1000);
        
        assert!(monitor.get_progress("task1").is_some());
        
        monitor.clear_progress("task1");
        assert!(monitor.get_progress("task1").is_none());
    }
}
