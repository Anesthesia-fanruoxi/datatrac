// 同步任务 CRUD 操作

use super::{DataSourceType, Storage, SyncTask, TaskStatus, TaskUnitConfig, TaskUnitType};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use sqlx::Row;
use uuid::Uuid;

impl Storage {
    /// 保存同步任务配置
    /// 
    /// 如果任务 ID 已存在，则更新；否则插入新记录
    /// 同时更新配置表
    pub async fn save_task(&self, task: &SyncTask) -> Result<()> {
        log::info!("save_task - 准备保存任务: id={}, name={}", task.id, task.name);
        log::info!("save_task - source_id={}, target_id={}", task.source_id, task.target_id);
        log::info!("save_task - source_type={}, target_type={}", task.source_type.as_str(), task.target_type.as_str());
        log::info!("save_task - config={}", task.config);
        log::info!("save_task - status={}", task.status.as_str());
        
        sqlx::query(
            r#"
            INSERT INTO sync_tasks (
                id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                name = excluded.name,
                source_id = excluded.source_id,
                target_id = excluded.target_id,
                source_type = excluded.source_type,
                target_type = excluded.target_type,
                config = excluded.config,
                status = excluded.status,
                updated_at = excluded.updated_at
            "#,
        )
        .bind(&task.id)
        .bind(&task.name)
        .bind(&task.source_id)
        .bind(&task.target_id)
        .bind(task.source_type.as_str())
        .bind(task.target_type.as_str())
        .bind(&task.config)
        .bind(task.status.as_str())
        .bind(task.created_at.to_rfc3339())
        .bind(task.updated_at.to_rfc3339())
        .execute(&self.pool)
        .await
        .context("保存同步任务失败")?;

        log::info!("save_task - 保存成功");
        
        // 更新配置表
        self.update_task_config_from_json(&task.id, &task.config, &task.source_type, &task.target_type).await?;
        
        Ok(())
    }
    
    /// 从任务配置 JSON 更新配置表
    /// 
    /// 解析 config JSON,提取索引/表列表,更新配置表
    pub async fn update_task_config_from_json(
        &self,
        task_id: &str,
        config_json: &str,
        source_type: &DataSourceType,
        target_type: &DataSourceType,
    ) -> Result<()> {
        log::info!("[update_task_config_from_json] 开始更新配置表, task_id: {}", task_id);
        
        // 解析 JSON
        let config: serde_json::Value = serde_json::from_str(config_json)
            .context("解析任务配置 JSON 失败")?;
        
        let mut configs = Vec::new();
        let now = Utc::now().timestamp_millis();
        
        // 根据数据源类型提取单元列表
        if source_type == &DataSourceType::Mysql && target_type == &DataSourceType::Mysql {
            // MySQL → MySQL
            if let Some(mysql_config) = config.get("mysqlConfig") {
                if let Some(databases) = mysql_config.get("databases").and_then(|v| v.as_array()) {
                    for db in databases {
                        let db_name = db.get("database").and_then(|v| v.as_str()).unwrap_or("");
                        if let Some(tables) = db.get("tables").and_then(|v| v.as_array()) {
                            for table in tables {
                                if let Some(table_name) = table.as_str() {
                                    let unit_name = format!("{}.{}", db_name, table_name);
                                    configs.push(TaskUnitConfig {
                                        id: Uuid::new_v4().to_string(),
                                        task_id: task_id.to_string(),
                                        unit_name,
                                        unit_type: TaskUnitType::Table,
                                        search_pattern: None,
                                        created_at: now,
                                        updated_at: now,
                                    });
                                }
                            }
                        }
                    }
                }
            }
        } else if source_type == &DataSourceType::Elasticsearch {
            // ES → ES 或 ES → MySQL
            if let Some(es_config) = config.get("esConfig") {
                // 优先使用 selectedIndices (用户实际选择的索引)
                if let Some(selected_indices) = es_config.get("selectedIndices").and_then(|v| v.as_array()) {
                    // 从 searchGroups 中获取每个索引对应的搜索关键字
                    // 注意：searchGroups 中的 matchedIndices 包含所有过滤出来的索引
                    // selectedIndices 字段标记了用户选中的索引
                    let mut pattern_map: std::collections::HashMap<String, String> = std::collections::HashMap::new();
                    if let Some(search_groups) = es_config.get("searchGroups").and_then(|v| v.as_array()) {
                        for group in search_groups {
                            if let Some(pattern) = group.get("pattern").and_then(|v| v.as_str()) {
                                // 从 selectedIndices 字段获取该组中被选中的索引
                                if let Some(selected_in_group) = group.get("selectedIndices").and_then(|v| v.as_array()) {
                                    log::info!("[update_task_config_from_json] 组 '{}' 的 selectedIndices: {:?}", pattern, selected_in_group);
                                    for index in selected_in_group {
                                        if let Some(index_name) = index.as_str() {
                                            pattern_map.insert(index_name.to_string(), pattern.to_string());
                                        }
                                    }
                                } else {
                                    // 注意：这里不应该回退到 matchedIndices
                                    // 如果 selectedIndices 字段不存在，说明是旧数据，才使用 matchedIndices
                                    // 如果 selectedIndices 字段存在但为空数组，说明用户没有选择该组的任何索引
                                    log::warn!("[update_task_config_from_json] 组 '{}' 没有 selectedIndices 字段，跳过", pattern);
                                }
                            }
                        }
                    }
                    
                    log::info!("[update_task_config_from_json] pattern_map: {:?}", pattern_map);
                    log::info!("[update_task_config_from_json] selectedIndices: {:?}", selected_indices);
                    
                    // 只保存用户选择的索引
                    for index in selected_indices {
                        if let Some(index_name) = index.as_str() {
                            let search_pattern = pattern_map.get(index_name).cloned();
                            configs.push(TaskUnitConfig {
                                id: Uuid::new_v4().to_string(),
                                task_id: task_id.to_string(),
                                unit_name: index_name.to_string(),
                                unit_type: TaskUnitType::Index,
                                search_pattern,
                                created_at: now,
                                updated_at: now,
                            });
                        }
                    }
                } else if let Some(search_groups) = es_config.get("searchGroups").and_then(|v| v.as_array()) {
                    // 兼容旧逻辑: 如果没有 selectedIndices,使用所有 matchedIndices
                    for group in search_groups {
                        let pattern = group.get("pattern").and_then(|v| v.as_str()).map(|s| s.to_string());
                        if let Some(indices) = group.get("matchedIndices").and_then(|v| v.as_array()) {
                            for index in indices {
                                if let Some(index_name) = index.as_str() {
                                    configs.push(TaskUnitConfig {
                                        id: Uuid::new_v4().to_string(),
                                        task_id: task_id.to_string(),
                                        unit_name: index_name.to_string(),
                                        unit_type: TaskUnitType::Index,
                                        search_pattern: pattern.clone(),
                                        created_at: now,
                                        updated_at: now,
                                    });
                                }
                            }
                        }
                    }
                } else if let Some(indices) = es_config.get("indices").and_then(|v| v.as_array()) {
                    // 兼容更旧的方案
                    for index_sel in indices {
                        let pattern = index_sel.get("pattern").and_then(|v| v.as_str()).map(|s| s.to_string());
                        if let Some(matched_indices) = index_sel.get("matchedIndices").and_then(|v| v.as_array()) {
                            for index in matched_indices {
                                if let Some(index_name) = index.as_str() {
                                    configs.push(TaskUnitConfig {
                                        id: Uuid::new_v4().to_string(),
                                        task_id: task_id.to_string(),
                                        unit_name: index_name.to_string(),
                                        unit_type: TaskUnitType::Index,
                                        search_pattern: pattern.clone(),
                                        created_at: now,
                                        updated_at: now,
                                    });
                                }
                            }
                        }
                    }
                }
            }
        } else if source_type == &DataSourceType::Mysql && target_type == &DataSourceType::Elasticsearch {
            // MySQL → ES
            if let Some(mysql_config) = config.get("mysqlConfig") {
                if let Some(databases) = mysql_config.get("databases").and_then(|v| v.as_array()) {
                    for db in databases {
                        let db_name = db.get("database").and_then(|v| v.as_str()).unwrap_or("");
                        if let Some(tables) = db.get("tables").and_then(|v| v.as_array()) {
                            for table in tables {
                                if let Some(table_name) = table.as_str() {
                                    let unit_name = format!("{}.{}", db_name, table_name);
                                    configs.push(TaskUnitConfig {
                                        id: Uuid::new_v4().to_string(),
                                        task_id: task_id.to_string(),
                                        unit_name,
                                        unit_type: TaskUnitType::Table,
                                        search_pattern: None,
                                        created_at: now,
                                        updated_at: now,
                                    });
                                }
                            }
                        }
                    }
                }
            }
        }
        
        log::info!("[update_task_config_from_json] 提取到 {} 个配置单元", configs.len());
        
        if !configs.is_empty() {
            // 去重处理：排除已在历史记录表中的索引
            log::info!("[update_task_config_from_json] 开始去重处理，查询历史记录表");
            let histories = self.load_unit_histories(task_id).await?;
            let completed_names: std::collections::HashSet<String> = histories
                .into_iter()
                .map(|h| h.unit_name)
                .collect();
            
            log::info!("[update_task_config_from_json] 历史记录表中有 {} 个已完成的单元", completed_names.len());
            
            // 过滤掉已完成的单元
            let original_count = configs.len();
            configs.retain(|c| !completed_names.contains(&c.unit_name));
            let filtered_count = configs.len();
            
            if original_count > filtered_count {
                log::info!("[update_task_config_from_json] 过滤掉 {} 个已完成的单元，剩余 {} 个", 
                    original_count - filtered_count, filtered_count);
            }
            
            // 先删除旧配置
            log::info!("[update_task_config_from_json] 删除旧配置");
            self.delete_unit_configs(task_id).await?;
            
            if !configs.is_empty() {
                // 保存新配置
                log::info!("[update_task_config_from_json] 保存 {} 个新配置", configs.len());
                self.save_unit_configs(&configs).await?;
                log::info!("[update_task_config_from_json] 配置表更新成功");
            } else {
                log::info!("[update_task_config_from_json] 所有单元都已完成，无需保存新配置");
            }
        } else {
            log::warn!("[update_task_config_from_json] 没有提取到配置单元,可能配置格式不正确");
            log::warn!("[update_task_config_from_json] config_json: {}", config_json);
        }
        
        Ok(())
    }

    /// 加载所有同步任务
    pub async fn load_tasks(&self) -> Result<Vec<SyncTask>> {
        let rows = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks
            ORDER BY created_at DESC
            "#,
        )
        .fetch_all(&self.pool)
        .await
        .context("加载同步任务列表失败")?;

        let mut tasks = Vec::new();
        for row in rows {
            let source_type_str: String = row.try_get("source_type")?;
            let target_type_str: String = row.try_get("target_type")?;
            let status_str: String = row.try_get("status")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            tasks.push(SyncTask {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_id: row.try_get("source_id")?,
                target_id: row.try_get("target_id")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                target_type: DataSourceType::from_str(&target_type_str)?,
                config: row.try_get("config")?,
                status: TaskStatus::from_str(&status_str)?,
                created_at: DateTime::parse_from_rfc3339(&created_at_str)
                    .context("解析 created_at 失败")?
                    .with_timezone(&Utc),
                updated_at: DateTime::parse_from_rfc3339(&updated_at_str)
                    .context("解析 updated_at 失败")?
                    .with_timezone(&Utc),
            });
        }

        Ok(tasks)
    }

    /// 根据 ID 加载单个同步任务
    pub async fn load_task(&self, id: &str) -> Result<Option<SyncTask>> {
        let row = sqlx::query(
            r#"
            SELECT id, name, source_id, target_id, source_type, target_type, config, status, created_at, updated_at
            FROM sync_tasks
            WHERE id = ?
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .context("加载同步任务失败")?;

        if let Some(row) = row {
            let source_type_str: String = row.try_get("source_type")?;
            let target_type_str: String = row.try_get("target_type")?;
            let status_str: String = row.try_get("status")?;
            let created_at_str: String = row.try_get("created_at")?;
            let updated_at_str: String = row.try_get("updated_at")?;

            Ok(Some(SyncTask {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                source_id: row.try_get("source_id")?,
                target_id: row.try_get("target_id")?,
                source_type: DataSourceType::from_str(&source_type_str)?,
                target_type: DataSourceType::from_str(&target_type_str)?,
                config: row.try_get("config")?,
                status: TaskStatus::from_str(&status_str)?,
                created_at: DateTime::parse_from_rfc3339(&created_at_str)
                    .context("解析 created_at 失败")?
                    .with_timezone(&Utc),
                updated_at: DateTime::parse_from_rfc3339(&updated_at_str)
                    .context("解析 updated_at 失败")?
                    .with_timezone(&Utc),
            }))
        } else {
            Ok(None)
        }
    }

    /// 删除同步任务
    pub async fn delete_task(&self, id: &str) -> Result<()> {
        let result = sqlx::query(
            r#"
            DELETE FROM sync_tasks WHERE id = ?
            "#,
        )
        .bind(id)
        .execute(&self.pool)
        .await
        .context("删除同步任务失败")?;

        if result.rows_affected() == 0 {
            anyhow::bail!("同步任务不存在: {}", id);
        }

        Ok(())
    }
}
