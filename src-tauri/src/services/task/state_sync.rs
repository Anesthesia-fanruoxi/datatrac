// 状态同步模块
// 负责任务单元状态的内存更新和数据库同步

use super::TaskManager;
use crate::core::monitor::{ProgressMonitor, TaskProgress};
use crate::storage::models::{TaskUnit, TaskUnitStatus};
use anyhow::Result;
use chrono::Utc;
use std::sync::Arc;

impl TaskManager {
    /// 更新任务单元状态（仅内存）
    pub fn update_unit_status(&self, task_id: &str, unit_name: &str, status: TaskUnitStatus) {
        let mut cache = self.task_units_cache.write().unwrap();
        
        if let Some(units) = cache.get_mut(task_id) {
            if let Some(unit) = units.iter_mut().find(|u| u.unit_name == unit_name) {
                unit.status = status.as_str().to_string();
                unit.updated_at = Utc::now();
            }
        }
    }

    /// 更新任务单元状态并同步到数据库和前端
    pub async fn update_unit_status_with_sync(
        &self,
        task_id: &str,
        unit_name: &str,
        status: TaskUnitStatus,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        // 1. 更新内存
        self.update_unit_status(task_id, unit_name, status.clone());

        // 2. 同步到数据库
        self.storage
            .update_unit_status(task_id, unit_name, status)
            .await?;

        // 3. 推送到前端
        self.push_progress_to_frontend(task_id, progress_monitor)?;

        Ok(())
    }

    /// 更新任务单元进度（仅内存）
    pub fn update_unit_progress(
        &self,
        task_id: &str,
        unit_name: &str,
        total: i64,
        processed: i64,
    ) {
        let mut cache = self.task_units_cache.write().unwrap();
        
        if let Some(units) = cache.get_mut(task_id) {
            if let Some(unit) = units.iter_mut().find(|u| u.unit_name == unit_name) {
                unit.total_records = total;
                unit.processed_records = processed;
                unit.updated_at = Utc::now();
            }
        }
    }

    /// 更新任务单元进度并同步到数据库和前端
    pub async fn update_unit_progress_with_sync(
        &self,
        task_id: &str,
        unit_name: &str,
        total: i64,
        processed: i64,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        // 1. 更新内存
        self.update_unit_progress(task_id, unit_name, total, processed);

        // 2. 同步到数据库
        self.storage
            .update_unit_progress(task_id, unit_name, total, processed)
            .await?;

        // 3. 推送到前端
        self.push_progress_to_frontend(task_id, progress_monitor)?;

        Ok(())
    }

    /// 标记任务单元失败（仅内存）
    pub fn fail_unit(&self, task_id: &str, unit_name: &str, error: String) {
        let mut cache = self.task_units_cache.write().unwrap();
        
        if let Some(units) = cache.get_mut(task_id) {
            if let Some(unit) = units.iter_mut().find(|u| u.unit_name == unit_name) {
                unit.status = TaskUnitStatus::Failed.as_str().to_string();
                unit.error_message = Some(error);
                unit.updated_at = Utc::now();
            }
        }
    }

    /// 标记任务单元失败并同步到数据库和前端
    pub async fn fail_unit_with_sync(
        &self,
        task_id: &str,
        unit_name: &str,
        error: String,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        // 1. 更新内存
        self.fail_unit(task_id, unit_name, error.clone());

        // 2. 同步到数据库
        self.storage.fail_unit(task_id, unit_name, &error).await?;

        // 3. 推送到前端
        self.push_progress_to_frontend(task_id, progress_monitor)?;

        Ok(())
    }

    /// 推送进度到前端
    fn push_progress_to_frontend(
        &self,
        task_id: &str,
        progress_monitor: &Arc<ProgressMonitor>,
    ) -> Result<()> {
        let units = self.get_task_units(task_id);

        // 聚合进度
        let total_units = units.len();
        let completed_units = units
            .iter()
            .filter(|u| u.status == TaskUnitStatus::Completed.as_str())
            .count();
        let failed_units = units
            .iter()
            .filter(|u| u.status == TaskUnitStatus::Failed.as_str())
            .count();

        let total_records: u64 = units.iter().map(|u| u.total_records as u64).sum();
        let processed_records: u64 = units.iter().map(|u| u.processed_records as u64).sum();

        let percentage = if total_records > 0 {
            (processed_records as f64 / total_records as f64) * 100.0
        } else {
            0.0
        };

        let progress = TaskProgress {
            task_id: task_id.to_string(),
            status: if completed_units == total_units {
                "completed".to_string()
            } else if failed_units > 0 {
                "failed".to_string()
            } else {
                "running".to_string()
            },
            total_units,
            completed_units,
            failed_units,
            total_records,
            processed_records,
            percentage,
        };

        progress_monitor.update_progress(progress);

        Ok(())
    }
}
