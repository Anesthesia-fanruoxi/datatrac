// 任务管理器模块
// 负责统一调度和管理同步任务

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use tokio::sync::Semaphore;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use crate::progress::ProgressMonitor;
use crate::storage::Storage;

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
/// - 与数据库同步任务单元状态
pub struct TaskManager {
    /// 任务单元映射 (task_id -> Vec<TaskUnit>) - 内存缓存
    task_units: Arc<RwLock<HashMap<String, Vec<TaskUnit>>>>,
    /// 数据库存储 (可选,用于持久化)
    storage: Option<Arc<Storage>>,
}

impl TaskManager {
    /// 创建新的任务管理器
    pub fn new() -> Self {
        Self {
            task_units: Arc::new(RwLock::new(HashMap::new())),
            storage: None,
        }
    }

    /// 创建带数据库持久化的任务管理器
    pub fn with_storage(storage: Arc<Storage>) -> Self {
        Self {
            task_units: Arc::new(RwLock::new(HashMap::new())),
            storage: Some(storage),
        }
    }
    
    /// 获取 Storage 实例
    pub fn storage(&self) -> Option<&Arc<Storage>> {
        self.storage.as_ref()
    }

    /// 从数据库加载任务单元
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    pub async fn load_task_units_from_db(&self, task_id: &str) -> Result<Vec<TaskUnit>> {
        if let Some(storage) = &self.storage {
            // 使用新的三表结构：从运行记录表加载
            let db_units = storage.load_unit_runtimes(task_id).await?;
            
            log::info!("[load_task_units_from_db] 从运行记录表加载到 {} 个单元", db_units.len());
            
            // 转换为内存格式
            let units: Vec<TaskUnit> = db_units.into_iter().map(|u| {
                let percentage = if u.total_records > 0 {
                    (u.processed_records as f64 / u.total_records as f64) * 100.0
                } else {
                    0.0
                };
                
                TaskUnit {
                    id: u.id,
                    name: u.unit_name,
                    status: match u.status {
                        crate::storage::TaskUnitStatus::Pending => TaskUnitStatus::Pending,
                        crate::storage::TaskUnitStatus::Running => TaskUnitStatus::Running,
                        crate::storage::TaskUnitStatus::Completed => TaskUnitStatus::Completed,
                        crate::storage::TaskUnitStatus::Failed => TaskUnitStatus::Failed,
                        crate::storage::TaskUnitStatus::Paused => TaskUnitStatus::Pending,
                    },
                    total_records: u.total_records as u64,
                    processed_records: u.processed_records as u64,
                    percentage,
                    error_message: u.error_message,
                }
            }).collect();
            
            // 更新内存缓存
            let mut cache = self.task_units.write().unwrap();
            cache.insert(task_id.to_string(), units.clone());
            
            Ok(units)
        } else {
            Ok(Vec::new())
        }
    }

    /// 初始化任务单元列表
    pub fn init_task_units(&self, task_id: &str, unit_names: Vec<String>) {
        let mut units = self.task_units.write().unwrap();
        
        let task_units: Vec<TaskUnit> = unit_names
            .into_iter()
            .map(|name| TaskUnit::new(name.clone(), name))
            .collect();
        
        units.insert(task_id.to_string(), task_units);
    }

    /// 获取任务单元列表
    pub fn get_task_units(&self, task_id: &str) -> Vec<TaskUnit> {
        let units = self.task_units.read().unwrap();
        units.get(task_id).cloned().unwrap_or_default()
    }

    /// 更新任务单元状态
    pub fn update_unit_status(&self, task_id: &str, unit_id: &str, status: TaskUnitStatus) {
        let mut units = self.task_units.write().unwrap();
        
        if let Some(task_units) = units.get_mut(task_id) {
            if let Some(unit) = task_units.iter_mut().find(|u| u.id == unit_id) {
                unit.status = status;
            }
        }
    }

    /// 更新任务单元状态并同步到数据库和 ProgressMonitor
    pub async fn update_unit_status_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        status: TaskUnitStatus,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        self.update_unit_status(task_id, unit_id, status.clone());
        
        if let Some(storage) = &self.storage {
            let db_status = match status {
                TaskUnitStatus::Pending => crate::storage::TaskUnitStatus::Pending,
                TaskUnitStatus::Running => crate::storage::TaskUnitStatus::Running,
                TaskUnitStatus::Completed => crate::storage::TaskUnitStatus::Completed,
                TaskUnitStatus::Failed => crate::storage::TaskUnitStatus::Failed,
            };
            // 使用新的三表结构方法：通过 unit_name 更新运行记录表
            // 注意：unit_id 在新结构中实际上是 unit_name
            storage.update_runtime_status(task_id, unit_id, db_status).await?;
        }
        
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
        
        Ok(())
    }

    /// 更新任务单元进度
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

    /// 更新任务单元进度并同步到数据库和 ProgressMonitor
    /// 
    /// 新逻辑: 使用 unit_name 而不是 unit_id
    pub async fn update_unit_progress_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        total: u64,
        processed: u64,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        self.update_unit_progress(task_id, unit_id, total, processed);
        
        if let Some(storage) = &self.storage {
            // unit_id 实际上是 unit_name (索引名或表名)
            storage.update_runtime_progress(task_id, unit_id, total as i64, processed as i64).await?;
        }
        
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
        
        Ok(())
    }

    /// 标记任务单元失败
    pub fn fail_unit(&self, task_id: &str, unit_id: &str, error: String) {
        let mut units = self.task_units.write().unwrap();
        
        if let Some(task_units) = units.get_mut(task_id) {
            if let Some(unit) = task_units.iter_mut().find(|u| u.id == unit_id) {
                unit.status = TaskUnitStatus::Failed;
                unit.error_message = Some(error);
            }
        }
    }

    /// 标记任务单元失败并同步到数据库和 ProgressMonitor
    /// 
    /// 新逻辑: 使用 unit_name 而不是 unit_id
    pub async fn fail_unit_with_sync(
        &self,
        task_id: &str,
        unit_id: &str,
        error: String,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        self.fail_unit(task_id, unit_id, error.clone());
        
        if let Some(storage) = &self.storage {
            // unit_id 实际上是 unit_name (索引名或表名)
            storage.fail_runtime(task_id, unit_id, &error).await?;
        }
        
        let units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, units);
        
        Ok(())
    }

    /// 清除任务单元
    pub fn clear_task_units(&self, task_id: &str) {
        let mut units = self.task_units.write().unwrap();
        units.remove(task_id);
    }

    /// 并发执行任务单元
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
        self.init_task_units(task_id, unit_names.clone());
        
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
            
            if let Err(e) = self.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Running, &progress_monitor).await {
                log::error!("更新单元状态失败: {}", e);
            }
            
            let task_manager = self.clone();
            
            let task = tokio::spawn(async move {
                log::info!("任务管理器: 开始执行单元 {}", unit_id);
                let result = process_fn(unit_name).await;
                
                match result {
                    Ok(_) => {
                        log::info!("任务管理器: 单元 {} 执行成功", unit_id);
                        if let Err(e) = task_manager.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Completed, &progress_monitor).await {
                            log::error!("更新单元状态失败: {}", e);
                        }
                        success_count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                    }
                    Err(e) => {
                        log::error!("任务管理器: 单元 {} 执行失败: {}", unit_id, e);
                        if let Err(e) = task_manager.fail_unit_with_sync(&task_id, &unit_id, e.to_string(), &progress_monitor).await {
                            log::error!("标记单元失败: {}", e);
                        }
                    }
                }
                
                drop(permit);
            });
            
            tasks.push(task);
        }
        
        for task in tasks {
            let _ = task.await;
        }
        
        let success = success_count.load(std::sync::atomic::Ordering::SeqCst);
        log::info!("任务管理器: 所有单元执行完成，成功: {}/{}", success, self.get_task_units(task_id).len());
        
        Ok(success)
    }

    /// 自动模式: 执行所有未完成的任务单元
    /// 
    /// 从数据库加载任务单元,过滤出未完成的单元,然后并发执行
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `concurrency`: 并发数
    /// - `progress_monitor`: 进度监控器
    /// - `process_fn`: 处理单个单元的异步函数
    /// 
    /// # 返回
    /// 成功执行的单元数量
    pub async fn execute_auto_mode<F, Fut>(
        &self,
        task_id: &str,
        concurrency: usize,
        progress_monitor: Arc<ProgressMonitor>,
        process_fn: F,
    ) -> Result<usize>
    where
        F: Fn(String, String) -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = Result<()>> + Send,
    {
        // 从数据库加载任务单元
        let units = self.load_task_units_from_db(task_id).await?;
        
        log::info!("自动模式: 加载了 {} 个任务单元", units.len());
        
        // 过滤出未完成的单元 (排除 completed 状态)
        let pending_units: Vec<_> = units
            .into_iter()
            .filter(|u| u.status != TaskUnitStatus::Completed)
            .collect();
        
        log::info!("自动模式: 需要执行 {} 个未完成的单元", pending_units.len());
        
        if pending_units.is_empty() {
            log::info!("自动模式: 所有单元已完成,无需执行");
            return Ok(0);
        }
        
        // 同步初始状态到 ProgressMonitor
        let all_units = self.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, all_units);
        
        let semaphore = Arc::new(Semaphore::new(concurrency));
        let process_fn = Arc::new(process_fn);
        let mut tasks = Vec::new();
        let success_count = Arc::new(std::sync::atomic::AtomicUsize::new(0));
        
        log::info!("自动模式: 开始执行，并发数: {}", concurrency);
        
        for unit in pending_units {
            let permit = semaphore.clone().acquire_owned().await?;
            let process_fn = process_fn.clone();
            let success_count = success_count.clone();
            let task_id = task_id.to_string();
            let unit_id = unit.id.clone();
            let unit_name = unit.name.clone();
            let progress_monitor = progress_monitor.clone();
            
            // 标记为运行中
            if let Err(e) = self.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Running, &progress_monitor).await {
                log::error!("更新单元状态失败: {}", e);
            }
            
            let task_manager = self.clone();
            
            let task = tokio::spawn(async move {
                log::info!("自动模式: 开始执行单元 {} ({})", unit_name, unit_id);
                
                // 调用处理函数,传入 unit_id 和 unit_name
                let result = process_fn(unit_id.clone(), unit_name.clone()).await;
                
                match result {
                    Ok(_) => {
                        log::info!("自动模式: 单元 {} 执行成功", unit_name);
                        if let Err(e) = task_manager.update_unit_status_with_sync(&task_id, &unit_id, TaskUnitStatus::Completed, &progress_monitor).await {
                            log::error!("更新单元状态失败: {}", e);
                        }
                        success_count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                    }
                    Err(e) => {
                        log::error!("自动模式: 单元 {} 执行失败: {}", unit_name, e);
                        if let Err(e) = task_manager.fail_unit_with_sync(&task_id, &unit_id, e.to_string(), &progress_monitor).await {
                            log::error!("标记单元失败: {}", e);
                        }
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
        let total = self.get_task_units(task_id).len();
        log::info!("自动模式: 所有单元执行完成，成功: {}/{}", success, total);
        
        Ok(success)
    }
}

impl Clone for TaskManager {
    fn clone(&self) -> Self {
        Self {
            task_units: Arc::clone(&self.task_units),
            storage: self.storage.clone(),
        }
    }
}

impl Default for TaskManager {
    fn default() -> Self {
        Self::new()
    }
}
