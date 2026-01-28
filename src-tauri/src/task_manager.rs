// 任务管理器模块
// 负责统一调度和管理同步任务

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use tokio::sync::Semaphore;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use crate::progress::ProgressMonitor;

/// 任务单元状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskUnitStatus {
    Pending,    // 等待执行
    Running,    // 执行中
    Completed,  // 已完成
    Failed,     // 失败
}

/// 任务单元 (表或索引)
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnit {
    /// 单元 ID (表名或索引名)
    pub id: String,
    /// 单元名称
    pub name: String,
    /// 状态
    pub status: TaskUnitStatus,
    /// 总记录数
    pub total_records: u64,
    /// 已处理记录数
    pub processed_records: u64,
    /// 进度百分比
    pub percentage: f64,
    /// 错误信息 (如果失败)
    pub error_message: Option<String>,
}

impl TaskUnit {
    /// 创建新的任务单元
    pub fn new(id: String, name: String) -> Self {
        Self {
            id,
            name,
            status: TaskUnitStatus::Pending,
            total_records: 0,
            processed_records: 0,
            percentage: 0.0,
            error_message: None,
        }
    }
}

/// 任务管理器
/// 
/// 职责:
/// - 统一管理所有任务单元 (表/索引)
/// - 控制并发执行
/// - 跟踪任务状态
/// - 分配任务给工作线程
pub struct TaskManager {
    /// 任务单元映射 (task_id -> Vec<TaskUnit>)
    task_units: Arc<RwLock<HashMap<String, Vec<TaskUnit>>>>,
}

impl TaskManager {
    /// 创建新的任务管理器
    pub fn new() -> Self {
        Self {
            task_units: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 初始化任务单元列表
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_names`: 单元名称列表 (表名或索引名)
    pub fn init_task_units(&self, task_id: &str, unit_names: Vec<String>) {
        let mut units = self.task_units.write().unwrap();
        
        let task_units: Vec<TaskUnit> = unit_names
            .into_iter()
            .map(|name| TaskUnit::new(name.clone(), name))
            .collect();
        
        units.insert(task_id.to_string(), task_units);
    }

    /// 获取任务单元列表
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// 
    /// # 返回
    /// 任务单元列表
    pub fn get_task_units(&self, task_id: &str) -> Vec<TaskUnit> {
        let units = self.task_units.read().unwrap();
        units.get(task_id).cloned().unwrap_or_default()
    }

    /// 更新任务单元状态
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `status`: 新状态
    pub fn update_unit_status(&self, task_id: &str, unit_id: &str, status: TaskUnitStatus) {
        let mut units = self.task_units.write().unwrap();
        
        if let Some(task_units) = units.get_mut(task_id) {
            if let Some(unit) = task_units.iter_mut().find(|u| u.id == unit_id) {
                unit.status = status;
            }
        }
    }

    /// 更新任务单元状态并同步到 ProgressMonitor
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `status`: 新状态
    /// - `progress_monitor`: 进度监控器
    pub fn update_unit_status_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        status: TaskUnitStatus,
        progress_monitor: &Arc<ProgressMonitor>,
    ) {
        self.update_unit_status(task_id, unit_id, status);
        
        // 同步到 ProgressMonitor
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
    }

    /// 更新任务单元进度
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `total`: 总记录数
    /// - `processed`: 已处理记录数
    pub fn update_unit_progress(&self, task_id: &str, unit_id: &str, total: u64, processed: u64) {
        let mut units = self.task_units.write().unwrap();
        
        if let Some(task_units) = units.get_mut(task_id) {
            if let Some(unit) = task_units.iter_mut().find(|u| u.id == unit_id) {
                unit.total_records = total;
                unit.processed_records = processed;
                
                if total > 0 {
                    unit.percentage = (processed as f64 / total as f64) * 100.0;
                }
            }
        }
    }

    /// 更新任务单元进度并同步到 ProgressMonitor
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `total`: 总记录数
    /// - `processed`: 已处理记录数
    /// - `progress_monitor`: 进度监控器
    pub fn update_unit_progress_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        total: u64,
        processed: u64,
        progress_monitor: &Arc<ProgressMonitor>,
    ) {
        self.update_unit_progress(task_id, unit_id, total, processed);
        
        // 同步到 ProgressMonitor
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
    }

    /// 标记任务单元失败
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `error`: 错误信息
    pub fn fail_unit(&self, task_id: &str, unit_id: &str, error: String) {
        let mut units = self.task_units.write().unwrap();
        
        if let Some(task_units) = units.get_mut(task_id) {
            if let Some(unit) = task_units.iter_mut().find(|u| u.id == unit_id) {
                unit.status = TaskUnitStatus::Failed;
                unit.error_message = Some(error);
            }
        }
    }

    /// 标记任务单元失败并同步到 ProgressMonitor
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_id`: 单元 ID
    /// - `error`: 错误信息
    /// - `progress_monitor`: 进度监控器
    pub fn fail_unit_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        error: String,
        progress_monitor: &Arc<ProgressMonitor>,
    ) {
        self.fail_unit(task_id, unit_id, error);
        
        // 同步到 ProgressMonitor
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
    }

    /// 清除任务单元
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub fn clear_task_units(&self, task_id: &str) {
        let mut units = self.task_units.write().unwrap();
        units.remove(task_id);
    }

    /// 并发执行任务单元
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `unit_names`: 单元名称列表
    /// - `concurrency`: 并发数
    /// - `progress_monitor`: 进度监控器
    /// - `process_fn`: 处理单个单元的异步函数
    /// 
    /// # 返回
    /// 成功执行的单元数量
    pub async fn execute_units<F, Fut>(
        &self,
        task_id: &str,
        unit_names: Vec<String>,
        concurrency: usize,
        progress_monitor: Arc<ProgressMonitor>,
        process_fn: F,
    ) -> Result<usize>
    where
        F: Fn(String) -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = Result<()>> + Send,
    {
        // 初始化任务单元
        self.init_task_units(task_id, unit_names.clone());
        
        // 同步初始状态到 ProgressMonitor
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
        
        let semaphore = Arc::new(Semaphore::new(concurrency));
        let process_fn = Arc::new(process_fn);
        let mut tasks = Vec::new();
        let success_count = Arc::new(std::sync::atomic::AtomicUsize::new(0));
        
        log::info!("任务管理器: 开始执行 {} 个单元，并发数: {}", unit_names.len(), concurrency);
        
        for unit_name in unit_names {
            let permit = semaphore.clone().acquire_owned().await?;
            let process_fn = process_fn.clone();
            let success_count = success_count.clone();
            let task_id = task_id.to_string();
            let unit_id = unit_name.clone();
            let progress_monitor = progress_monitor.clone();
            
            // 标记为运行中
            self.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Running, &progress_monitor);
            
            let task_manager = self.clone();
            
            let task = tokio::spawn(async move {
                log::info!("任务管理器: 开始执行单元 {}", unit_id);
                let result = process_fn(unit_name).await;
                
                match result {
                    Ok(_) => {
                        log::info!("任务管理器: 单元 {} 执行成功", unit_id);
                        task_manager.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Completed, &progress_monitor);
                        success_count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                    }
                    Err(e) => {
                        log::error!("任务管理器: 单元 {} 执行失败: {}", unit_id, e);
                        task_manager.fail_unit_with_sync(&task_id, &unit_id, e.to_string(), &progress_monitor);
                    }
                }
                
                drop(permit);
            });
            
            tasks.push(task);
        }
        
        // 等待所有任务完成
        for task in tasks {
            let _ = task.await;
        }
        
        let success = success_count.load(std::sync::atomic::Ordering::SeqCst);
        log::info!("任务管理器: 所有单元执行完成，成功: {}/{}", success, self.get_task_units(task_id).len());
        
        Ok(success)
    }
}

impl Clone for TaskManager {
    fn clone(&self) -> Self {
        Self {
            task_units: Arc::clone(&self.task_units),
        }
    }
}

impl Default for TaskManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_init_task_units() {
        let manager = TaskManager::new();
        let unit_names = vec!["table1".to_string(), "table2".to_string()];
        
        manager.init_task_units("task1", unit_names);
        
        let units = manager.get_task_units("task1");
        assert_eq!(units.len(), 2);
        assert_eq!(units[0].name, "table1");
        assert_eq!(units[1].name, "table2");
        assert_eq!(units[0].status, TaskUnitStatus::Pending);
    }

    #[test]
    fn test_update_unit_status() {
        let manager = TaskManager::new();
        manager.init_task_units("task1", vec!["table1".to_string()]);
        
        manager.update_unit_status("task1", "table1", TaskUnitStatus::Running);
        
        let units = manager.get_task_units("task1");
        assert_eq!(units[0].status, TaskUnitStatus::Running);
    }

    #[test]
    fn test_update_unit_progress() {
        let manager = TaskManager::new();
        manager.init_task_units("task1", vec!["table1".to_string()]);
        
        manager.update_unit_progress("task1", "table1", 1000, 500);
        
        let units = manager.get_task_units("task1");
        assert_eq!(units[0].total_records, 1000);
        assert_eq!(units[0].processed_records, 500);
        assert_eq!(units[0].percentage, 50.0);
    }

    #[test]
    fn test_fail_unit() {
        let manager = TaskManager::new();
        manager.init_task_units("task1", vec!["table1".to_string()]);
        
        manager.fail_unit("task1", "table1", "Test error".to_string());
        
        let units = manager.get_task_units("task1");
        assert_eq!(units[0].status, TaskUnitStatus::Failed);
        assert_eq!(units[0].error_message, Some("Test error".to_string()));
    }
}
