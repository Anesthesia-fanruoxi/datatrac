// 任务调度模块
// 负责并发执行任务单元

use super::TaskManager;
use crate::core::monitor::ProgressMonitor;
use crate::storage::models::TaskUnitStatus;
use anyhow::Result;
use std::future::Future;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use tokio::sync::Semaphore;

impl TaskManager {
    /// 自动模式执行任务
    /// 
    /// 并发执行所有未完成的任务单元
    /// 
    /// # 参数
    /// - `task_id`: 任务 ID
    /// - `concurrency`: 并发数（线程数）
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
        Fut: Future<Output = Result<()>> + Send,
    {
        log::info!(
            "开始自动模式执行任务: {}, 并发数: {}",
            task_id,
            concurrency
        );

        // 1. 从数据库加载任务单元
        let units = self.load_task_units_from_db(task_id).await?;

        // 2. 过滤出未完成的单元
        let pending_units: Vec<_> = units
            .into_iter()
            .filter(|u| u.status != TaskUnitStatus::Completed.as_str())
            .collect();

        if pending_units.is_empty() {
            log::info!("没有待执行的任务单元");
            return Ok(0);
        }

        log::info!("待执行的任务单元数量: {}", pending_units.len());

        // 3. 创建并发控制信号量
        let semaphore = Arc::new(Semaphore::new(concurrency));

        // 4. 创建成功计数器
        let success_count = Arc::new(AtomicUsize::new(0));

        // 5. 创建任务列表
        let mut handles = Vec::new();

        let task_id = task_id.to_string();
        let process_fn = Arc::new(process_fn);

        for unit in pending_units {
            let semaphore = semaphore.clone();
            let task_manager = Arc::new(self.clone_for_async());
            let progress_monitor = progress_monitor.clone();
            let process_fn = process_fn.clone();
            let success_count = success_count.clone();
            let task_id = task_id.clone();
            let unit_name = unit.unit_name.clone();
            let unit_id = unit.id.clone();

            // 创建异步任务
            let handle = tokio::spawn(async move {
                // 获取信号量许可
                let _permit = semaphore.acquire().await.unwrap();

                log::info!("开始执行任务单元: {}", unit_name);

                // 标记为 Running
                if let Err(e) = task_manager
                    .update_unit_status_with_sync(
                        &task_id,
                        &unit_name,
                        TaskUnitStatus::Running,
                        &progress_monitor,
                    )
                    .await
                {
                    log::error!("更新单元状态失败: {}", e);
                    return;
                }

                // 执行同步逻辑
                let result = process_fn(unit_id.clone(), unit_name.clone()).await;

                match result {
                    Ok(_) => {
                        // 标记为 Completed
                        if let Err(e) = task_manager
                            .update_unit_status_with_sync(
                                &task_id,
                                &unit_name,
                                TaskUnitStatus::Completed,
                                &progress_monitor,
                            )
                            .await
                        {
                            log::error!("更新单元状态失败: {}", e);
                        } else {
                            success_count.fetch_add(1, Ordering::SeqCst);
                            log::info!("任务单元执行成功: {}", unit_name);
                        }
                    }
                    Err(e) => {
                        // 标记为 Failed
                        log::error!("任务单元执行失败: {}, 错误: {}", unit_name, e);
                        if let Err(e) = task_manager
                            .fail_unit_with_sync(
                                &task_id,
                                &unit_name,
                                e.to_string(),
                                &progress_monitor,
                            )
                            .await
                        {
                            log::error!("标记单元失败状态失败: {}", e);
                        }
                    }
                }

                // 许可会在这里自动释放
            });

            handles.push(handle);
        }

        // 6. 等待所有任务完成
        for handle in handles {
            if let Err(e) = handle.await {
                log::error!("任务执行出错: {}", e);
            }
        }

        let final_count = success_count.load(Ordering::SeqCst);
        log::info!("自动模式执行完成，成功: {}", final_count);

        Ok(final_count)
    }

    /// 克隆用于异步任务
    /// 
    /// 由于 TaskManager 包含 RwLock，不能直接 Clone
    /// 这里创建一个新的实例，共享相同的 storage 和 cache
    fn clone_for_async(&self) -> Self {
        Self {
            storage: self.storage.clone(),
            task_units_cache: self.task_units_cache.clone(),
        }
    }
}
