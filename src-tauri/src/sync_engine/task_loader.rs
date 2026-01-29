// 任务加载和配置解析模块

use super::types::*;
use crate::datasource::DataSourceManager;
use crate::storage::DataSourceType;
use anyhow::Result;
use std::sync::Arc;

/// 任务加载器
pub struct TaskLoader {
    source_manager: Arc<DataSourceManager>,
}

impl TaskLoader {
    /// 创建新的任务加载器
    pub fn new(source_manager: Arc<DataSourceManager>) -> Self {
        Self { source_manager }
    }

    /// 从数据库加载任务配置并解析
    pub async fn load_and_parse_task(&self, task_id: &str) -> Result<SyncTaskConfig> {
        log::info!("根据任务 ID 加载配置: {}", task_id);
        
        // 从数据库加载任务配置
        let storage = self.source_manager.storage();
        let task = storage.load_task(task_id).await?
            .ok_or_else(|| anyhow::anyhow!("任务不存在: {}", task_id))?;
        
        log::info!("加载任务配置: name={}, source_type={:?}, target_type={:?}", 
            task.name, task.source_type, task.target_type);
        
        // 解析配置 JSON
        let config_json: serde_json::Value = serde_json::from_str(&task.config)
            .map_err(|e| anyhow::anyhow!("解析任务配置失败: {}", e))?;
        
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
        
        // 验证配置完整性
        self.validate_config(&task.source_id, &task.target_id, &mysql_config, &es_config)?;
        
        // 确定同步方向
        let sync_direction = self.determine_sync_direction(&task.source_type, &task.target_type);
        
        // 构建同步任务配置
        Ok(SyncTaskConfig {
            task_id: task.id.clone(),
            task_name: task.name.clone(),
            source_id: task.source_id.clone(),
            target_id: task.target_id.clone(),
            sync_direction,
            sync_config,
            mysql_config,
            es_config,
        })
    }

    /// 验证配置完整性
    fn validate_config(
        &self,
        source_id: &str,
        target_id: &str,
        mysql_config: &Option<MysqlSyncConfig>,
        es_config: &Option<EsSyncConfig>,
    ) -> Result<()> {
        // 验证数据源配置
        if source_id.is_empty() || target_id.is_empty() {
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
        
        Ok(())
    }

    /// 确定同步方向
    fn determine_sync_direction(
        &self,
        source_type: &DataSourceType,
        target_type: &DataSourceType,
    ) -> SyncDirection {
        match (source_type, target_type) {
            (DataSourceType::Mysql, DataSourceType::Elasticsearch) => SyncDirection::MysqlToEs,
            (DataSourceType::Elasticsearch, DataSourceType::Mysql) => SyncDirection::EsToMysql,
            (DataSourceType::Mysql, DataSourceType::Mysql) => SyncDirection::MysqlToMysql,
            (DataSourceType::Elasticsearch, DataSourceType::Elasticsearch) => SyncDirection::EsToEs,
        }
    }

    /// 初始化任务单元
    /// 
    /// 新逻辑:
    /// 1. 从配置表读取所有单元
    /// 2. 为每个配置单元创建或更新运行记录表
    /// 3. 清理配置表中不存在的运行记录
    /// 
    /// 支持断点续传：
    /// - 已有运行记录的保持状态
    /// - 没有运行记录的创建为 pending
    pub async fn init_task_units(&self, task_id: &str) -> Result<()> {
        log::info!("[init_task_units] 初始化任务单元: task_id={}", task_id);
        
        let storage = self.source_manager.storage();
        
        // 1. 从配置表读取所有单元
        let configs = storage.load_unit_configs(task_id).await?;
        
        if configs.is_empty() {
            anyhow::bail!("配置表中没有找到任务单元，请先保存任务配置");
        }
        
        log::info!("[init_task_units] 从配置表加载到 {} 个单元", configs.len());
        
        // 2. 加载现有的运行记录
        let existing_runtimes = storage.load_unit_runtimes(task_id).await?;
        let runtime_map: std::collections::HashMap<String, _> = existing_runtimes
            .into_iter()
            .map(|r| (r.unit_name.clone(), r))
            .collect();
        
        log::info!("[init_task_units] 现有运行记录: {} 个", runtime_map.len());
        
        // 3. 为每个配置单元创建或保持运行记录
        let now = chrono::Utc::now().timestamp_millis();
        for config in &configs {
            if let Some(existing) = runtime_map.get(&config.unit_name) {
                // 已有运行记录
                match existing.status {
                    crate::storage::TaskUnitStatus::Running => {
                        // 运行中状态重置为 pending（可能是异常中断）
                        log::info!("[init_task_units] 单元 {} 运行中被中断，重置为 pending", config.unit_name);
                        storage.reset_runtime(task_id, &config.unit_name).await?;
                    }
                    crate::storage::TaskUnitStatus::Pending | crate::storage::TaskUnitStatus::Failed => {
                        // pending 和 failed 状态保持不变
                        log::info!("[init_task_units] 单元 {} 状态为 {:?}，保持不变", config.unit_name, existing.status);
                    }
                    _ => {}
                }
            } else {
                // 没有运行记录，创建新的
                log::info!("[init_task_units] 创建新的运行记录: {}", config.unit_name);
                let runtime = crate::storage::TaskUnitRuntime {
                    id: uuid::Uuid::new_v4().to_string(),
                    task_id: task_id.to_string(),
                    unit_name: config.unit_name.clone(),
                    status: crate::storage::TaskUnitStatus::Pending,
                    total_records: 0,
                    processed_records: 0,
                    error_message: None,
                    started_at: None,
                    updated_at: now,
                    last_processed_batch: None,
                };
                storage.save_unit_runtime(&runtime).await?;
            }
        }
        
        // 4. 清理配置表中不存在的运行记录
        let cleaned = storage.cleanup_orphan_runtimes(task_id).await?;
        if cleaned > 0 {
            log::info!("[init_task_units] 清理了 {} 个孤立的运行记录", cleaned);
        }
        
        // 5. 统计状态
        let runtimes = storage.load_unit_runtimes(task_id).await?;
        let pending_count = runtimes.iter().filter(|r| r.status == crate::storage::TaskUnitStatus::Pending).count();
        let failed_count = runtimes.iter().filter(|r| r.status == crate::storage::TaskUnitStatus::Failed).count();
        
        log::info!("[init_task_units] 任务单元初始化完成，共 {} 个单元", runtimes.len());
        log::info!("[init_task_units] 状态统计: 待执行={}, 失败={}", pending_count, failed_count);
        
        Ok(())
    }
}
