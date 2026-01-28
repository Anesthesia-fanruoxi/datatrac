// 同步引擎模块
// 负责执行数据同步任务

mod types;
mod mysql_sync;
mod es_sync;

// 按同步类型分离的实现模块
mod implementations;

pub use types::*;

use crate::datasource::DataSourceManager;
use crate::error_logger::ErrorLogger;
use crate::progress::ProgressMonitor;
use crate::storage::DataSourceType;
use crate::type_mapper::TypeMapper;
use crate::task_manager::TaskManager;
use anyhow::Result;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use tokio::sync::Semaphore;

/// 同步引擎
/// 
/// 负责：
/// - 执行数据同步任务
/// - 管理任务状态
/// - 协调进度监控和错误日志
/// - 处理并发和批量操作
pub struct SyncEngine {
    /// 数据源管理器
    source_manager: Arc<DataSourceManager>,
    /// 进度监控器
    progress_monitor: Arc<ProgressMonitor>,
    /// 错误日志器
    error_logger: Arc<ErrorLogger>,
    /// 任务管理器
    task_manager: Arc<TaskManager>,
    /// 任务状态映射 (task_id -> SyncTaskState)
    task_states: Arc<RwLock<HashMap<String, SyncTaskState>>>,
}

impl SyncEngine {
    /// 创建新的同步引擎实例
    pub fn new(
        source_manager: Arc<DataSourceManager>,
        progress_monitor: Arc<ProgressMonitor>,
        error_logger: Arc<ErrorLogger>,
    ) -> Self {
        Self {
            source_manager,
            progress_monitor,
            error_logger,
            task_manager: Arc::new(TaskManager::new()),
            task_states: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 获取任务管理器的引用
    pub fn task_manager(&self) -> &Arc<TaskManager> {
        &self.task_manager
    }

    /// 获取任务状态
    fn get_task_state(&self, task_id: &str) -> Option<SyncTaskState> {
        let states = self.task_states.read().unwrap();
        states.get(task_id).cloned()
    }

    /// 设置任务状态
    fn set_task_state(&self, task_id: &str, state: SyncTaskState) {
        let mut states = self.task_states.write().unwrap();
        states.insert(task_id.to_string(), state);
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

    /// 启动同步任务
    pub async fn start_sync(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();

        // 只记录关键信息到日志
        log::info!("开始同步任务: {} ({:?})", task_id, config.sync_direction);

        if self.is_task_running(&task_id) {
            log::warn!("任务 {} 已在运行中，无法重复启动", task_id);
            anyhow::bail!("任务已在运行中");
        }

        let state = SyncTaskState::new(task_id.clone());
        self.set_task_state(&task_id, state.clone());

        {
            let mut states = self.task_states.write().unwrap();
            if let Some(s) = states.get_mut(&task_id) {
                s.is_running = true;
                s.is_paused = false;
            }
        }

        self.error_logger.clear_errors(&task_id);

        let result = match config.sync_direction {
            SyncDirection::MysqlToEs => {
                self.sync_mysql_to_es(config).await
            },
            SyncDirection::EsToMysql => {
                self.sync_es_to_mysql(config).await
            },
            SyncDirection::MysqlToMysql => {
                self.sync_mysql_to_mysql(config).await
            },
            SyncDirection::EsToEs => {
                self.sync_es_to_es(config).await
            },
        };

        {
            let mut states = self.task_states.write().unwrap();
            if let Some(s) = states.get_mut(&task_id) {
                s.is_running = false;
            }
        }

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
        log::info!("根据任务 ID 启动同步: {}", task_id);
        
        // 从数据库加载任务配置
        let storage = self.source_manager.storage();
        let task = storage.load_task(task_id).await?
            .ok_or_else(|| anyhow::anyhow!("任务不存在: {}", task_id))?;
        
        log::info!("加载任务配置: name={}, source_type={:?}, target_type={:?}", 
            task.name, task.source_type, task.target_type);
        
        // 解析配置 JSON
        let config_json: serde_json::Value = serde_json::from_str(&task.config)
            .map_err(|e| anyhow::anyhow!("解析任务配置失败: {}", e))?;
        
        log::info!("配置 JSON: {}", config_json);
        
        // 提取各部分配置
        let mysql_config: Option<MysqlSyncConfig> = config_json.get("mysqlConfig")
            .and_then(|v| serde_json::from_value(v.clone()).ok());
        
        let es_config: Option<EsSyncConfig> = config_json.get("esConfig")
            .and_then(|v| serde_json::from_value(v.clone()).ok());
        
        let sync_config: SyncConfig = config_json.get("syncConfig")
            .and_then(|v| serde_json::from_value(v.clone()).ok())
            .unwrap_or_else(|| SyncConfig {
                thread_count: 4,
                batch_size: 2500,
                error_strategy: ErrorStrategy::Skip,
                table_exists_strategy: TableExistsStrategy::Drop,
                db_name_transform: None,
            });
        
        log::info!("解析后的配置: mysql_config={:?}, es_config={:?}, sync_config={:?}", 
            mysql_config, es_config, sync_config);
        
        // 验证配置完整性
        if task.source_id.is_empty() || task.target_id.is_empty() {
            anyhow::bail!("任务未配置数据源，请先完成任务配置");
        }
        
        // 验证是否配置了同步的表/索引
        let has_mysql_config = mysql_config.as_ref()
            .map(|c| !c.databases.is_empty())
            .unwrap_or(false);
        
        let has_es_indices = es_config.as_ref()
            .and_then(|c| c.indices.as_ref())
            .map(|i| !i.is_empty())
            .unwrap_or(false);
        
        let has_es_selected_indices = es_config.as_ref()
            .and_then(|c| c.selected_indices.as_ref())
            .map(|i| !i.is_empty())
            .unwrap_or(false);
        
        if !has_mysql_config && !has_es_indices && !has_es_selected_indices {
            anyhow::bail!("任务未配置同步的表/索引，请先完成任务配置");
        }
        
        // 确定同步方向
        let sync_direction = match (&task.source_type, &task.target_type) {
            (DataSourceType::Mysql, DataSourceType::Elasticsearch) => SyncDirection::MysqlToEs,
            (DataSourceType::Elasticsearch, DataSourceType::Mysql) => SyncDirection::EsToMysql,
            (DataSourceType::Mysql, DataSourceType::Mysql) => SyncDirection::MysqlToMysql,
            (DataSourceType::Elasticsearch, DataSourceType::Elasticsearch) => SyncDirection::EsToEs,
        };
        
        // 构建同步任务配置
        let sync_task_config = SyncTaskConfig {
            task_id: task.id.clone(),
            task_name: task.name.clone(),
            source_id: task.source_id.clone(),
            target_id: task.target_id.clone(),
            sync_direction,
            sync_config,
            mysql_config,
            es_config,
        };
        
        log::info!("构建的同步配置: {:?}", sync_task_config);
        
        // 调用原有的 start_sync 方法
        self.start_sync(sync_task_config).await
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

        if let Some(state) = self.get_task_state(task_id) {
            state.pause_flag.store(true, std::sync::atomic::Ordering::SeqCst);
            
            {
                let mut states = self.task_states.write().unwrap();
                if let Some(s) = states.get_mut(task_id) {
                    s.is_paused = true;
                }
            }

            self.progress_monitor.pause_task(task_id);
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

        if let Some(state) = self.get_task_state(task_id) {
            state.pause_flag.store(false, std::sync::atomic::Ordering::SeqCst);
            
            {
                let mut states = self.task_states.write().unwrap();
                if let Some(s) = states.get_mut(task_id) {
                    s.is_paused = false;
                }
            }

            self.progress_monitor.resume_task(task_id);
            log::info!("任务 {} 已恢复", task_id);
            Ok(())
        } else {
            log::error!("任务 {} 状态不存在", task_id);
            anyhow::bail!("任务状态不存在");
        }
    }

    // 同步方法（调用实现）
    async fn sync_mysql_to_es(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Elasticsearch,
            config.sync_config.batch_size
        );
        implementations::mysql_to_es::sync_mysql_to_es(self, config).await
    }

    async fn sync_es_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        implementations::es_to_mysql::sync_es_to_mysql(self, config).await
    }

    async fn sync_mysql_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        implementations::mysql_to_mysql::sync_mysql_to_mysql(self, config).await
    }

    async fn sync_es_to_es(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Elasticsearch,
            config.sync_config.batch_size
        );
        implementations::es_to_es::sync_es_to_es(self, config).await
    }
    /// 根据源表结构创建目标表结构（使用类型映射）
    pub(super) fn convert_schema(
        &self,
        source_schema: &TableSchema,
        source_type: &DataSourceType,
        target_type: &DataSourceType,
    ) -> TableSchema {
        let mut target_schema = source_schema.clone();

        for col in &mut target_schema.columns {
            col.data_type = match (source_type, target_type) {
                (DataSourceType::Mysql, DataSourceType::Elasticsearch) => {
                    TypeMapper::mysql_to_es(&col.data_type)
                }
                (DataSourceType::Elasticsearch, DataSourceType::Mysql) => {
                    TypeMapper::es_to_mysql(&col.data_type)
                }
                _ => col.data_type.clone(),
            };
        }

        target_schema
    }

    /// 获取自适应批量大小
    fn get_adaptive_batch_size(&self, target_type: &DataSourceType, configured_size: usize) -> usize {
        if configured_size > 0 {
            configured_size
        } else {
            match target_type {
                DataSourceType::Mysql => 1000,
                DataSourceType::Elasticsearch => 500,
            }
        }
    }

    /// 转换数据库名称
    /// 
    /// 根据配置的转换规则，将源数据库名称转换为目标数据库名称
    pub(super) fn transform_database_name(&self, original_name: &str, transform: &Option<DbNameTransform>) -> String {
        if let Some(t) = transform {
            if !t.enabled {
                return original_name.to_string();
            }

            match t.mode {
                TransformMode::Prefix => {
                    // 前缀替换
                    if original_name.starts_with(&t.source_pattern) {
                        format!("{}{}", t.target_pattern, &original_name[t.source_pattern.len()..])
                    } else {
                        original_name.to_string()
                    }
                }
                TransformMode::Suffix => {
                    // 后缀替换
                    if original_name.ends_with(&t.source_pattern) {
                        format!("{}{}", &original_name[..original_name.len() - t.source_pattern.len()], t.target_pattern)
                    } else {
                        original_name.to_string()
                    }
                }
            }
        } else {
            original_name.to_string()
        }
    }

    /// 根据配置的转换规则，将源索引名称转换为目标索引名称
    pub(super) fn transform_index_name(&self, original_name: &str, transform: &IndexNameTransform) -> String {
        if !transform.enabled {
            return original_name.to_string();
        }

        match transform.mode {
            TransformMode::Prefix => {
                // 前缀替换
                if original_name.starts_with(&transform.source_pattern) {
                    format!("{}{}", transform.target_pattern, &original_name[transform.source_pattern.len()..])
                } else {
                    original_name.to_string()
                }
            }
            TransformMode::Suffix => {
                // 后缀替换
                if original_name.ends_with(&transform.source_pattern) {
                    format!("{}{}", &original_name[..original_name.len() - transform.source_pattern.len()], transform.target_pattern)
                } else {
                    original_name.to_string()
                }
            }
        }
    }

    /// 通用的并发批处理方法
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `thread_count`: 并发线程数
    /// - `batches`: 批次数据的迭代器
    /// - `process_fn`: 处理单个批次的异步函数
    /// 
    /// # 返回
    /// 处理成功的批次数量
    pub async fn concurrent_batch_process<T, F, Fut>(
        &self,
        task_id: &str,
        thread_count: usize,
        batches: Vec<T>,
        process_fn: F,
    ) -> Result<usize>
    where
        T: Send + 'static,
        F: Fn(T) -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = Result<()>> + Send,
    {
        let semaphore = Arc::new(Semaphore::new(thread_count));
        let process_fn = Arc::new(process_fn);
        let mut tasks = Vec::new();
        let total_batches = batches.len();
        let success_count = Arc::new(std::sync::atomic::AtomicUsize::new(0));

        log::info!("开始并发处理 {} 个批次，并发数: {}", total_batches, thread_count);

        for batch in batches {
            let permit = semaphore.clone().acquire_owned().await?;
            let process_fn = process_fn.clone();
            let success_count = success_count.clone();
            let task_id = task_id.to_string();

            let task = tokio::spawn(async move {
                let result = process_fn(batch).await;
                
                if result.is_ok() {
                    success_count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                }
                
                drop(permit);
                result
            });

            tasks.push(task);
        }

        // 等待所有任务完成
        let mut errors = Vec::new();
        for task in tasks {
            if let Err(e) = task.await {
                errors.push(format!("任务执行失败: {}", e));
            }
        }

        let success = success_count.load(std::sync::atomic::Ordering::SeqCst);
        
        if !errors.is_empty() {
            log::warn!("部分批次处理失败: {}/{}", errors.len(), total_batches);
        }

        log::info!("并发处理完成: 成功 {}/{}", success, total_batches);
        
        Ok(success)
    }
}
