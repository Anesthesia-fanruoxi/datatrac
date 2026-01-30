// 同步引擎模块
// 负责执行数据同步任务

mod types;
mod mysql_sync;
mod es_sync;
mod state_manager;
mod task_loader;
mod utils;

// 按同步类型分离的实现模块
mod implementations;

pub use types::*;
pub use utils::*;

use crate::datasource::DataSourceManager;
use crate::error_logger::ErrorLogger;
use crate::progress::ProgressMonitor;
use crate::storage::DataSourceType;
use crate::task_manager::TaskManager;
use anyhow::Result;
use state_manager::TaskStateManager;
use task_loader::TaskLoader;
use std::sync::Arc;

/// 同步引擎
/// 
/// 负责：
/// - 执行数据同步任务
/// - 管理任务状态
/// - 协调进度监控和错误日志
/// - 处理并发和批量操作
pub struct SyncEngine {
    /// 数据源管理器
    pub(crate) source_manager: Arc<DataSourceManager>,
    /// 进度监控器
    pub(crate) progress_monitor: Arc<ProgressMonitor>,
    /// 错误日志器
    pub(crate) error_logger: Arc<ErrorLogger>,
    /// 任务管理器
    task_manager: Arc<TaskManager>,
    /// 任务状态管理器
    pub(crate) state_manager: TaskStateManager,
    /// 任务加载器
    task_loader: TaskLoader,
}

impl SyncEngine {
    /// 创建新的同步引擎实例
    pub fn new(
        source_manager: Arc<DataSourceManager>,
        progress_monitor: Arc<ProgressMonitor>,
        error_logger: Arc<ErrorLogger>,
    ) -> Self {
        // 使用带存储的任务管理器
        let storage = Arc::clone(source_manager.storage());
        let task_manager = Arc::new(TaskManager::with_storage(storage));
        let task_loader = TaskLoader::new(source_manager.clone());
        
        Self {
            source_manager,
            progress_monitor,
            error_logger,
            task_manager,
            state_manager: TaskStateManager::new(),
            task_loader,
        }
    }

    /// 获取任务管理器的引用
    pub fn task_manager(&self) -> &Arc<TaskManager> {
        &self.task_manager
    }

    /// 检查任务是否正在运行
    pub fn is_task_running(&self, task_id: &str) -> bool {
        self.state_manager.is_task_running(task_id)
    }

    /// 检查任务是否暂停
    pub fn is_task_paused(&self, task_id: &str) -> bool {
        self.state_manager.is_task_paused(task_id)
    }

    /// 更新任务状态到数据库
    async fn update_task_status(&self, task_id: &str, status: crate::storage::TaskStatus) -> Result<()> {
        let storage = self.source_manager.storage();
        
        // 加载任务
        if let Some(mut task) = storage.load_task(task_id).await? {
            // 更新状态和时间
            task.status = status;
            task.updated_at = chrono::Utc::now();
            
            // 保存任务
            storage.save_task(&task).await?;
            log::info!("任务 {} 状态已更新为: {:?}", task_id, task.status);
        } else {
            log::warn!("任务 {} 不存在，无法更新状态", task_id);
        }
        
        Ok(())
    }
    /// 启动同步任务
    pub async fn start_sync(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();

        log::info!("开始同步任务: {} ({:?})", task_id, config.sync_direction);

        if self.is_task_running(&task_id) {
            log::warn!("任务 {} 已在运行中，无法重复启动", task_id);
            anyhow::bail!("任务已在运行中");
        }

        let state = SyncTaskState::new(task_id.clone());
        self.state_manager.set_task_state(&task_id, state);
        self.state_manager.set_running(&task_id, true);
        self.state_manager.set_paused(&task_id, false);

        self.error_logger.clear_errors(&task_id);

        let result = match config.sync_direction {
            SyncDirection::MysqlToEs => self.sync_mysql_to_es(config).await,
            SyncDirection::EsToMysql => self.sync_es_to_mysql(config).await,
            SyncDirection::MysqlToMysql => self.sync_mysql_to_mysql(config).await,
            SyncDirection::EsToEs => self.sync_es_to_es(config).await,
        };

        self.state_manager.set_running(&task_id, false);

        match &result {
            Ok(_) => {
                log::info!("任务 {} 同步完成", task_id);
                self.progress_monitor.complete_task(&task_id)
            },
            Err(e) => {
                log::error!("任务 {} 同步失败: {}", task_id, e);
                self.progress_monitor.fail_task(&task_id, &e.to_string())
            },
        }

        result
    }

    /// 根据任务 ID 启动同步任务（从数据库加载配置）
    pub async fn start_sync_by_id(&self, task_id: &str) -> Result<()> {
        // 更新任务状态为 running
        self.update_task_status(task_id, crate::storage::TaskStatus::Running).await?;
        
        // 加载并解析任务配置
        let config = self.task_loader.load_and_parse_task(task_id).await?;
        
        // 初始化任务单元（断点续传支持）
        self.task_loader.init_task_units(task_id).await?;
        
        // 启动同步任务
        let result = self.start_sync(config).await;
        
        // 根据结果更新任务状态
        match &result {
            Ok(_) => {
                self.update_task_status(task_id, crate::storage::TaskStatus::Completed).await?;
            }
            Err(e) => {
                log::error!("任务 {} 执行失败: {}", task_id, e);
                self.update_task_status(task_id, crate::storage::TaskStatus::Failed).await?;
            }
        }
        
        result
    }

    /// 暂停同步任务
    pub async fn pause_sync(&self, task_id: &str) -> Result<()> {
        log::info!("暂停任务: {}", task_id);
        
        if !self.is_task_running(task_id) {
            log::warn!("任务 {} 未在运行中，无法暂停", task_id);
            anyhow::bail!("任务未在运行中");
        }

        if self.is_task_paused(task_id) {
            log::warn!("任务 {} 已处于暂停状态", task_id);
            anyhow::bail!("任务已处于暂停状态");
        }

        if let Some(state) = self.state_manager.get_task_state(task_id) {
            state.pause_flag.store(true, std::sync::atomic::Ordering::SeqCst);
            self.state_manager.set_paused(task_id, true);
            self.progress_monitor.pause_task(task_id);
            
            // 更新任务状态为 paused
            self.update_task_status(task_id, crate::storage::TaskStatus::Paused).await?;
            
            log::info!("任务 {} 已暂停", task_id);
            Ok(())
        } else {
            log::error!("任务 {} 状态不存在", task_id);
            anyhow::bail!("任务状态不存在");
        }
    }

    /// 恢复同步任务
    pub async fn resume_sync(&self, task_id: &str) -> Result<()> {
        log::info!("恢复任务: {}", task_id);
        
        if !self.is_task_running(task_id) {
            log::warn!("任务 {} 未在运行中，无法恢复", task_id);
            anyhow::bail!("任务未在运行中");
        }

        if !self.is_task_paused(task_id) {
            log::warn!("任务 {} 未处于暂停状态", task_id);
            anyhow::bail!("任务未处于暂停状态");
        }

        if let Some(state) = self.state_manager.get_task_state(task_id) {
            state.pause_flag.store(false, std::sync::atomic::Ordering::SeqCst);
            self.state_manager.set_paused(task_id, false);
            self.progress_monitor.resume_task(task_id);
            
            // 更新任务状态为 running
            self.update_task_status(task_id, crate::storage::TaskStatus::Running).await?;
            
            log::info!("任务 {} 已恢复", task_id);
            Ok(())
        } else {
            log::error!("任务 {} 状态不存在", task_id);
            anyhow::bail!("任务状态不存在");
        }
    }

    // 同步方法（调用实现）
    async fn sync_mysql_to_es(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = get_adaptive_batch_size(
            &DataSourceType::Elasticsearch,
            config.sync_config.batch_size
        );
        implementations::mysql_to_es::sync_mysql_to_es(self, config).await
    }

    async fn sync_es_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        implementations::es_to_mysql::sync_es_to_mysql(self, config).await
    }

    async fn sync_mysql_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        implementations::mysql_to_mysql::sync_mysql_to_mysql(self, config).await
    }

    async fn sync_es_to_es(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = get_adaptive_batch_size(
            &DataSourceType::Elasticsearch,
            config.sync_config.batch_size
        );
        implementations::es_to_es::sync_es_to_es(self, config).await
    }
}
