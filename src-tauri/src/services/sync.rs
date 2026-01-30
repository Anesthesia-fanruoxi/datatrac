// 同步引擎服务
// 基于 Exchange Framework 的同步引擎

use crate::core::{logger::TaskLogger, monitor::ProgressMonitor};
use crate::exchange::core::factory::{ReaderFactory, WriterFactory};
use crate::exchange::core::pipeline::SyncPipeline;
use crate::services::{datasource::DataSourceManager, task::TaskManager};
use crate::storage::task_config::TaskConfig;
use anyhow::{anyhow, Result};
use std::sync::Arc;

/// 同步引擎
/// 负责协调数据同步任务的执行
pub struct SyncEngine {
    datasource_manager: Arc<DataSourceManager>,
    task_manager: Arc<TaskManager>,
    progress_monitor: Arc<ProgressMonitor>,
    task_logger: Arc<TaskLogger>,
}

impl SyncEngine {
    /// 创建新的同步引擎
    pub fn new(
        datasource_manager: Arc<DataSourceManager>,
        task_manager: Arc<TaskManager>,
        progress_monitor: Arc<ProgressMonitor>,
        task_logger: Arc<TaskLogger>,
    ) -> Self {
        Self {
            datasource_manager,
            task_manager,
            progress_monitor,
            task_logger,
        }
    }

    /// 启动同步任务
    pub async fn start_sync(&self, task_id: &str) -> Result<()> {
        log::info!("启动同步任务: {}", task_id);

        // 1. 加载任务信息
        let task = self
            .task_manager
            .get_task(task_id)
            .await?
            .ok_or_else(|| anyhow!("任务不存在: {}", task_id))?;

        // 2. 解析任务配置
        let config: TaskConfig = serde_json::from_str(&task.config)
            .map_err(|e| anyhow!("解析任务配置失败: {}", e))?;

        log::info!(
            "任务配置 - 线程数: {}, 批次大小: {}",
            config.sync_config.thread_count,
            config.sync_config.batch_size
        );

        // 3. 加载数据源信息
        let source_ds = self
            .datasource_manager
            .get_datasource(&task.source_id)
            .await?
            .ok_or_else(|| anyhow!("源数据源不存在: {}", task.source_id))?;

        let target_ds = self
            .datasource_manager
            .get_datasource(&task.target_id)
            .await?
            .ok_or_else(|| anyhow!("目标数据源不存在: {}", task.target_id))?;

        // 4. 从数据库加载任务单元
        self.task_manager
            .load_task_units_from_db(task_id)
            .await?;

        // 5. 使用 TaskManager 的自动模式执行
        let batch_size = config.sync_config.batch_size;
        let source_ds_clone = source_ds.clone();
        let target_ds_clone = target_ds.clone();
        let source_id = task.source_id.clone();
        let task_id_str = task_id.to_string();
        let task_manager_clone = self.task_manager.clone();
        let progress_monitor_clone = self.progress_monitor.clone();

        let success_count = self
            .task_manager
            .execute_auto_mode(
                task_id,
                config.sync_config.thread_count,
                self.progress_monitor.clone(),
                move |_unit_id, unit_name| {
                    let source_ds = source_ds_clone.clone();
                    let target_ds = target_ds_clone.clone();
                    let source_id = source_id.clone();
                    let task_id = task_id_str.clone();
                    let task_manager = task_manager_clone.clone();
                    let progress_monitor = progress_monitor_clone.clone();

                    async move {
                        log::info!("开始同步单元: {}", unit_name);

                        // 创建 Reader 和 Writer
                        let reader = ReaderFactory::create(&source_ds, &unit_name)?;
                        let writer = WriterFactory::create(&target_ds, &unit_name)?;

                        // 创建 Pipeline
                        let pipeline = SyncPipeline::new(reader, writer, batch_size);

                        // 初始化 Pipeline
                        pipeline.open().await?;
                        pipeline.prepare().await?;

                        // 执行同步，带进度回调
                        let unit_name_for_progress = unit_name.clone();
                        let task_id_for_progress = task_id.clone();
                        let task_manager_for_progress = task_manager.clone();
                        
                        let total_processed = pipeline
                            .execute(move |processed, total| {
                                // 每批次处理完成后，更新进度并推送到前端
                                let unit_name = unit_name_for_progress.clone();
                                let task_id = task_id_for_progress.clone();
                                let task_manager = task_manager_for_progress.clone();
                                let progress_monitor = progress_monitor.clone();

                                // 在后台异步更新进度，不阻塞主流程
                                tokio::spawn(async move {
                                    if let Err(e) = task_manager
                                        .update_unit_progress_with_sync(
                                            &task_id,
                                            &unit_name,
                                            total as i64,
                                            processed as i64,
                                            &progress_monitor,
                                        )
                                        .await
                                    {
                                        log::error!("更新进度失败: {}", e);
                                    }
                                });
                            })
                            .await?;

                        // 关闭 Pipeline
                        pipeline.close().await?;

                        log::info!(
                            "单元 {} 同步完成，共处理 {} 条记录",
                            unit_name,
                            total_processed
                        );

                        // 同步成功后，记录到 synced_indices
                        if let Err(e) = task_manager
                            .mark_index_synced(&source_id, &unit_name, &task_id)
                            .await
                        {
                            log::error!("记录已同步索引失败: {}", e);
                        }

                        Ok(())
                    }
                },
            )
            .await?;

        log::info!(
            "任务 {} 执行完成，成功: {} 个单元",
            task_id,
            success_count
        );

        Ok(())
    }

    /// 暂停同步任务
    pub async fn pause_sync(&self, task_id: &str) -> Result<()> {
        // TODO: 实现暂停逻辑
        // 需要添加任务状态管理和中断机制
        log::info!("暂停同步任务: {}", task_id);
        Ok(())
    }

    /// 恢复同步任务
    pub async fn resume_sync(&self, task_id: &str) -> Result<()> {
        // TODO: 实现恢复逻辑
        // 从数据库加载未完成的任务单元，继续执行
        log::info!("恢复同步任务: {}", task_id);
        self.start_sync(task_id).await
    }
}
