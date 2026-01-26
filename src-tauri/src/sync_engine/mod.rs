// 同步引擎模块
// 负责执行数据同步任务

mod types;
mod mysql_sync;
mod es_sync;
mod mysql_to_es_impl;
mod other_sync_impl;

pub use types::*;

use crate::datasource::DataSourceManager;
use crate::error_logger::ErrorLogger;
use crate::progress::ProgressMonitor;
use crate::storage::DataSourceType;
use crate::type_mapper::TypeMapper;
use anyhow::Result;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};

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
            task_states: Arc::new(RwLock::new(HashMap::new())),
        }
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

        log::info!("========================================");
        log::info!("开始启动同步任务: {}", task_id);
        log::info!("同步方向: {:?}", config.sync_direction);
        log::info!("线程数: {}, 批量大小: {}", config.sync_config.thread_count, config.sync_config.batch_size);
        log::info!("错误策略: {:?}", config.sync_config.error_strategy);
        log::info!("========================================");

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
        log::info!("任务 {} 状态已初始化，开始执行同步", task_id);

        let result = match config.sync_direction {
            SyncDirection::MysqlToEs => {
                log::info!("执行 MySQL -> Elasticsearch 同步");
                self.sync_mysql_to_es(config).await
            },
            SyncDirection::EsToMysql => {
                log::info!("执行 Elasticsearch -> MySQL 同步");
                self.sync_es_to_mysql(config).await
            },
            SyncDirection::MysqlToMysql => {
                log::info!("执行 MySQL -> MySQL 同步");
                self.sync_mysql_to_mysql(config).await
            },
            SyncDirection::EsToEs => {
                log::info!("执行 Elasticsearch -> Elasticsearch 同步");
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
                log::info!("========================================");
                log::info!("任务 {} 同步完成", task_id);
                log::info!("========================================");
                self.progress_monitor.complete_task(&task_id)
            },
            Err(e) => {
                log::error!("========================================");
                log::error!("任务 {} 同步失败: {}", task_id, e);
                log::error!("========================================");
                self.progress_monitor.fail_task(&task_id, &e.to_string())
            },
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
        self.sync_mysql_to_es_impl(config).await
    }

    async fn sync_es_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        self.sync_es_to_mysql_impl(config).await
    }

    async fn sync_mysql_to_mysql(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Mysql,
            config.sync_config.batch_size
        );
        self.sync_mysql_to_mysql_impl(config).await
    }

    async fn sync_es_to_es(&self, mut config: SyncTaskConfig) -> Result<()> {
        config.sync_config.batch_size = self.get_adaptive_batch_size(
            &DataSourceType::Elasticsearch,
            config.sync_config.batch_size
        );
        self.sync_es_to_es_impl(config).await
    }

    /// 根据源表结构创建目标表结构（使用类型映射）
    fn convert_schema(
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
    fn transform_database_name(&self, original_name: &str, transform: &Option<DbNameTransform>) -> String {
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
}
