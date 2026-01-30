// 任务去重模块
// 负责任务内和跨任务的索引去重

use super::TaskManager;
use crate::storage::models::{TaskUnit, TaskUnitStatus, TaskUnitType};
use crate::storage::task_config::TaskConfig;
use anyhow::Result;
use chrono::Utc;
use std::collections::HashMap;

impl TaskManager {
    /// 创建任务并生成去重后的任务单元
    pub async fn create_task_with_deduplication(
        &self,
        task_id: &str,
        source_id: &str,
        config: &TaskConfig,
    ) -> Result<Vec<TaskUnit>> {
        log::info!("开始创建任务单元，任务ID: {}", task_id);

        // 1. 任务内去重
        let deduplicated = self.deduplicate_within_task(config)?;
        log::info!("任务内去重完成，剩余 {} 个索引", deduplicated.len());

        // 2. 跨任务去重（如果启用）
        let final_units = if config.skip_synced {
            self.deduplicate_across_tasks(source_id, deduplicated)
                .await?
        } else {
            deduplicated
        };
        log::info!("跨任务去重完成，最终 {} 个索引", final_units.len());

        // 3. 创建任务单元
        let mut units = Vec::new();
        for (index_name, keyword) in final_units {
            let unit = TaskUnit {
                id: uuid::Uuid::new_v4().to_string(),
                task_id: task_id.to_string(),
                unit_name: index_name.clone(),
                unit_type: TaskUnitType::Index,
                status: TaskUnitStatus::Pending.as_str().to_string(),
                total_records: 0,
                processed_records: 0,
                error_message: None,
                keyword: if keyword.is_empty() {
                    None
                } else {
                    Some(keyword)
                },
                created_at: Utc::now(),
                updated_at: Utc::now(),
            };

            self.storage.save_task_unit(&unit).await?;
            units.push(unit);
        }

        log::info!("任务单元创建完成，共 {} 个", units.len());
        Ok(units)
    }

    /// 任务内去重
    /// 
    /// 遍历所有关键字，记录每个索引第一次出现的关键字
    /// 如果索引重复出现，保留第一个关键字
    fn deduplicate_within_task(
        &self,
        config: &TaskConfig,
    ) -> Result<Vec<(String, String)>> {
        let mut index_map: HashMap<String, String> = HashMap::new();

        if let Some(keywords) = &config.keywords {
            // 有关键字映射，使用关键字去重
            for mapping in keywords {
                for index in &mapping.indices {
                    // 只记录第一次出现的关键字
                    index_map
                        .entry(index.clone())
                        .or_insert(mapping.keyword.clone());
                }
            }
            log::debug!(
                "使用关键字映射去重，原始索引数: {}, 去重后: {}",
                keywords.iter().map(|k| k.indices.len()).sum::<usize>(),
                index_map.len()
            );
        } else {
            // 没有关键字映射，直接使用 units
            for unit in &config.units {
                index_map.insert(unit.clone(), String::new());
            }
            log::debug!("未使用关键字映射，索引数: {}", index_map.len());
        }

        Ok(index_map.into_iter().collect())
    }

    /// 跨任务去重
    /// 
    /// 检查索引是否在之前的任务中已同步
    /// 过滤掉已同步的索引
    async fn deduplicate_across_tasks(
        &self,
        source_id: &str,
        units: Vec<(String, String)>,
    ) -> Result<Vec<(String, String)>> {
        let mut result = Vec::new();
        let mut skipped_count = 0;

        for (index_name, keyword) in units {
            // 检查索引是否已同步
            let is_synced = self
                .storage
                .is_index_synced(source_id, &index_name)
                .await?;

            if !is_synced {
                result.push((index_name, keyword));
            } else {
                log::info!("跳过已同步的索引: {}", index_name);
                skipped_count += 1;
            }
        }

        if skipped_count > 0 {
            log::info!("跨任务去重: 跳过 {} 个已同步的索引", skipped_count);
        }

        Ok(result)
    }

    /// 标记索引为已同步
    pub async fn mark_index_synced(
        &self,
        source_id: &str,
        index_name: &str,
        task_id: &str,
    ) -> Result<()> {
        self.storage
            .mark_index_synced(source_id, index_name, task_id)
            .await
    }
}
