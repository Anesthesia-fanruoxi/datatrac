// 任务管理服务
// 负责任务的 CRUD、任务单元管理、并发调度

mod deduplication;
mod scheduler;
mod state_sync;

use crate::core::monitor::ProgressMonitor;
use crate::storage::models::{SyncTask, TaskStatus, TaskUnit, TaskUnitStatus, TaskUnitType};
use crate::storage::Storage;
use anyhow::Result;
use chrono::Utc;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use uuid::Uuid;

pub use scheduler::*;
pub use state_sync::*;

/// 任务管理器
pub struct TaskManager {
    storage: Arc<Storage>,
    /// 内存缓存：task_id -> Vec<TaskUnit>
    task_units_cache: Arc<RwLock<HashMap<String, Vec<TaskUnit>>>>,
}

impl TaskManager {
    /// 创建新的任务管理器
    pub fn new(storage: Arc<Storage>) -> Self {
        Self {
            storage,
            task_units_cache: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 获取所有任务
    pub async fn list_tasks(&self) -> Result<Vec<SyncTask>> {
        self.storage.load_tasks().await
    }

    /// 获取单个任务
    pub async fn get_task(&self, id: &str) -> Result<Option<SyncTask>> {
        self.storage.load_task(id).await
    }

    /// 创建任务
    pub async fn create_task(&self, mut task: SyncTask) -> Result<String> {
        if task.id.is_empty() {
            task.id = Uuid::new_v4().to_string();
        }

        task.status = TaskStatus::Idle;
        task.created_at = Utc::now();
        task.updated_at = Utc::now();

        self.storage.save_task(&task).await?;

        // 解析配置并创建任务单元
        self.init_task_units(&task).await?;

        Ok(task.id)
    }

    /// 更新任务
    pub async fn update_task(&self, mut task: SyncTask) -> Result<()> {
        task.updated_at = Utc::now();
        self.storage.save_task(&task).await?;

        // 重新初始化任务单元
        self.init_task_units(&task).await?;

        Ok(())
    }

    /// 删除任务
    pub async fn delete_task(&self, id: &str) -> Result<()> {
        // 清除内存缓存
        self.clear_task_units(id);
        
        // 删除数据库记录
        self.storage.delete_task(id).await
    }

    /// 获取任务单元列表（从内存缓存）
    pub fn get_task_units(&self, task_id: &str) -> Vec<TaskUnit> {
        let cache = self.task_units_cache.read().unwrap();
        cache.get(task_id).cloned().unwrap_or_default()
    }

    /// 从数据库加载任务单元到内存
    pub async fn load_task_units_from_db(&self, task_id: &str) -> Result<Vec<TaskUnit>> {
        let units = self.storage.load_task_units(task_id).await?;
        
        // 更新内存缓存
        {
            let mut cache = self.task_units_cache.write().unwrap();
            cache.insert(task_id.to_string(), units.clone());
        }
        
        Ok(units)
    }

    /// 重置失败的任务单元
    pub async fn reset_failed_units(&self, task_id: &str) -> Result<usize> {
        let count = self.storage.reset_failed_units(task_id).await?;
        
        // 重新加载到内存
        self.load_task_units_from_db(task_id).await?;
        
        Ok(count)
    }

    /// 清除任务单元缓存
    pub fn clear_task_units(&self, task_id: &str) {
        let mut cache = self.task_units_cache.write().unwrap();
        cache.remove(task_id);
    }

    /// 初始化任务单元（从任务配置）
    async fn init_task_units(&self, task: &SyncTask) -> Result<()> {
        // 解析配置 JSON
        let config: serde_json::Value = serde_json::from_str(&task.config)?;

        // 获取单元列表
        let units = if let Some(units) = config.get("units").and_then(|v| v.as_array()) {
            units
                .iter()
                .filter_map(|u| u.as_str().map(String::from))
                .collect::<Vec<_>>()
        } else {
            Vec::new()
        };

        // 确定单元类型
        let unit_type = match (task.source_type.as_str(), task.target_type.as_str()) {
            ("mysql", _) | (_, "mysql") => TaskUnitType::Table,
            ("elasticsearch", _) | (_, "elasticsearch") => TaskUnitType::Index,
            _ => TaskUnitType::Table,
        };

        // 获取现有的任务单元
        let existing_units = self.storage.load_task_units(&task.id).await?;
        let existing_names: std::collections::HashSet<String> = existing_units
            .iter()
            .map(|u| u.unit_name.clone())
            .collect();

        // 创建新的任务单元
        for unit_name in units {
            // 如果单元已存在，跳过
            if existing_names.contains(&unit_name) {
                continue;
            }

            let unit = TaskUnit {
                id: Uuid::new_v4().to_string(),
                task_id: task.id.clone(),
                unit_name,
                unit_type: unit_type.clone(),
                status: TaskUnitStatus::Pending.as_str().to_string(),
                total_records: 0,
                processed_records: 0,
                error_message: None,
                keyword: None,
                created_at: Utc::now(),
                updated_at: Utc::now(),
            };

            self.storage.save_task_unit(&unit).await?;
        }

        // 加载到内存缓存
        self.load_task_units_from_db(&task.id).await?;

        Ok(())
    }
}
