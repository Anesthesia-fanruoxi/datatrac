// 其他同步方向的实现

use super::types::*;
use super::SyncEngine;
use crate::error_logger::ErrorLog;
use crate::storage::DataSourceType;
use anyhow::{Context, Result};
use serde_json::json;

impl SyncEngine {
    /// ES 到 MySQL 同步实现
    pub(super) async fn sync_es_to_mysql_impl(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();
        let batch_size = config.sync_config.batch_size;
        let error_strategy = config.sync_config.error_strategy.clone();

        let source = self.source_manager.get_data_source(&config.source_id).await?
            .ok_or_else(|| anyhow::anyhow!("源数据源不存在"))?;
        let target = self.source_manager.get_data_source(&config.target_id).await?
            .ok_or_else(|| anyhow::anyhow!("目标数据源不存在"))?;

        let es_url = format!("http://{}:{}", source.host, source.port);
        let es_transport = elasticsearch::http::transport::Transport::single_node(&es_url)
            .context("创建 ES 传输层失败")?;
        let es_client = elasticsearch::Elasticsearch::new(es_transport);

        let mysql_url = format!(
            "mysql://{}:{}@{}:{}",
            target.username, target.password, target.host, target.port
        );
        let mysql_pool = sqlx::mysql::MySqlPoolOptions::new()
            .max_connections(config.sync_config.thread_count as u32)
            .connect(&mysql_url)
            .await
            .context("连接 MySQL 失败")?;

        let es_config = config.es_config
            .ok_or_else(|| anyhow::anyhow!("缺少 ES 配置"))?;

        let mut all_indices = Vec::new();
        for index_sel in &es_config.indices {
            if let Some(matched) = &index_sel.matched_indices {
                all_indices.extend(matched.clone());
            } else {
                all_indices.push(index_sel.pattern.clone());
            }
        }

        let mut total_records = 0u64;
        for index in &all_indices {
            let count = self.get_es_index_count(&es_client, index).await?;
            total_records += count;
        }

        self.progress_monitor.start_task(&task_id, total_records);

        let mut processed_records = 0u64;
        let target_database = target.database.as_ref()
            .ok_or_else(|| anyhow::anyhow!("目标 MySQL 数据源缺少数据库名"))?;

        for index in &all_indices {
            self.progress_monitor.update_current_table(&task_id, Some(index.clone()));

            let (sample_batch, _) = self.read_es_batch(&es_client, index, None, 1).await?;
            
            if sample_batch.is_empty() {
                continue;
            }

            let source_schema = self.infer_schema_from_es_doc(&sample_batch[0], index);
            let target_schema = self.convert_schema(
                &source_schema,
                &DataSourceType::Elasticsearch,
                &DataSourceType::Mysql,
            );

            // 应用数据库名称转换（ES索引名作为数据库名）
            let transformed_db_name = self.transform_database_name(
                target_database,
                &config.sync_config.db_name_transform
            );

            self.handle_mysql_table_with_strategy(
                &mysql_pool,
                &transformed_db_name,
                index,
                &target_schema,
                &config.sync_config.table_exists_strategy
            ).await?;

            let mut scroll_id: Option<String> = None;
            
            loop {
                if let Some(state) = self.get_task_state(&task_id) {
                    while state.pause_flag.load(std::sync::atomic::Ordering::SeqCst) {
                        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                    }
                }

                let read_result = self.read_es_batch(
                    &es_client,
                    index,
                    scroll_id.as_deref(),
                    batch_size
                ).await;

                match read_result {
                    Ok((batch, new_scroll_id)) => {
                        if batch.is_empty() {
                            break;
                        }

                        scroll_id = new_scroll_id;

                        let insert_result = self.batch_insert_mysql(
                            &mysql_pool,
                            &transformed_db_name,
                            index,
                            &batch
                        ).await;

                        match insert_result {
                            Ok(_) => {
                                processed_records += batch.len() as u64;
                                self.progress_monitor.update_progress(&task_id, processed_records);
                            }
                            Err(e) => {
                                self.error_logger.log_error(
                                    &task_id,
                                    ErrorLog::new(
                                        "InsertError".to_string(),
                                        format!("插入数据到 MySQL 失败: {}", e),
                                        Some(json!({ "index": index }))
                                    )
                                );

                                match error_strategy {
                                    ErrorStrategy::Skip => continue,
                                    ErrorStrategy::Pause => {
                                        if let Some(sid) = scroll_id {
                                            let _ = self.clear_es_scroll(&es_client, &sid).await;
                                        }
                                        return Err(e);
                                    }
                                }
                            }
                        }
                    }
                    Err(e) => {
                        self.error_logger.log_error(
                            &task_id,
                            ErrorLog::new(
                                "ReadError".to_string(),
                                format!("读取 ES 数据失败: {}", e),
                                Some(json!({ "index": index }))
                            )
                        );

                        match error_strategy {
                            ErrorStrategy::Skip => break,
                            ErrorStrategy::Pause => {
                                if let Some(sid) = scroll_id {
                                    let _ = self.clear_es_scroll(&es_client, &sid).await;
                                }
                                return Err(e);
                            }
                        }
                    }
                }
            }

            if let Some(sid) = scroll_id {
                let _ = self.clear_es_scroll(&es_client, &sid).await;
            }
        }

        Ok(())
    }

    /// 从 ES 文档推断表结构
    fn infer_schema_from_es_doc(
        &self,
        doc: &std::collections::HashMap<String, serde_json::Value>,
        table_name: &str,
    ) -> TableSchema {
        let mut columns = Vec::new();
        let mut primary_key = None;

        for (key, value) in doc {
            if key == "_id" {
                primary_key = Some(key.clone());
                columns.push(ColumnDefinition {
                    name: key.clone(),
                    data_type: "text".to_string(),
                    nullable: false,
                    is_primary_key: true,
                });
            } else {
                let es_type = match value {
                    serde_json::Value::String(_) => "text",
                    serde_json::Value::Number(n) => {
                        if n.is_i64() || n.is_u64() {
                            "long"
                        } else {
                            "double"
                        }
                    }
                    serde_json::Value::Bool(_) => "boolean",
                    serde_json::Value::Array(_) => "text",
                    serde_json::Value::Object(_) => "object",
                    serde_json::Value::Null => "keyword",
                };

                columns.push(ColumnDefinition {
                    name: key.clone(),
                    data_type: es_type.to_string(),
                    nullable: true,
                    is_primary_key: false,
                });
            }
        }

        TableSchema {
            table_name: table_name.to_string(),
            columns,
            primary_key,
        }
    }

    /// MySQL 到 MySQL 同步实现
    pub(super) async fn sync_mysql_to_mysql_impl(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();
        let batch_size = config.sync_config.batch_size;
        let error_strategy = config.sync_config.error_strategy.clone();

        let source = self.source_manager.get_data_source(&config.source_id).await?
            .ok_or_else(|| anyhow::anyhow!("源数据源不存在"))?;
        let target = self.source_manager.get_data_source(&config.target_id).await?
            .ok_or_else(|| anyhow::anyhow!("目标数据源不存在"))?;

        let source_url = format!(
            "mysql://{}:{}@{}:{}",
            source.username, source.password, source.host, source.port
        );
        let source_pool = sqlx::mysql::MySqlPoolOptions::new()
            .max_connections(config.sync_config.thread_count as u32)
            .connect(&source_url)
            .await
            .context("连接源 MySQL 失败")?;

        let target_url = format!(
            "mysql://{}:{}@{}:{}",
            target.username, target.password, target.host, target.port
        );
        let target_pool = sqlx::mysql::MySqlPoolOptions::new()
            .max_connections(config.sync_config.thread_count as u32)
            .connect(&target_url)
            .await
            .context("连接目标 MySQL 失败")?;

        let mysql_config = config.mysql_config
            .ok_or_else(|| anyhow::anyhow!("缺少 MySQL 配置"))?;

        // 收集所有表名和记录数，用于初始化表进度
        let mut all_tables = Vec::new();
        let mut all_counts = Vec::new();
        
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                all_tables.push(format!("{}.{}", db_sel.database, table));
                let count = self.get_mysql_table_count(&source_pool, &db_sel.database, table).await?;
                all_counts.push(count);
            }
        }
        
        // 初始化表进度列表
        self.progress_monitor.init_table_progress(&task_id, all_tables.clone(), all_counts.clone());
        
        // 添加开始日志
        self.progress_monitor.add_log(
            &task_id,
            crate::progress::LogLevel::Info,
            format!("开始同步任务，共 {} 个表", all_tables.len())
        );

        // 计算总记录数
        let mut total_records = 0u64;
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                let count = self.get_mysql_table_count(&source_pool, &db_sel.database, table).await?;
                total_records += count;
            }
        }

        self.progress_monitor.start_task(&task_id, total_records);
        let mut processed_records = 0u64;

        // 遍历每个数据库
        for db_sel in &mysql_config.databases {
            let source_db = &db_sel.database;
            let table_count = db_sel.tables.len();
            
            // 应用数据库名称转换
            let target_db = self.transform_database_name(
                source_db,
                &config.sync_config.db_name_transform
            );
            
            // 遍历每个表
            for table in &db_sel.tables {
                let full_table_name = format!("{}.{}", source_db, table);
                
                self.progress_monitor.update_current_table(
                    &task_id,
                    Some(full_table_name.clone())
                );
                
                // 添加日志：开始同步表
                self.progress_monitor.add_log(
                    &task_id,
                    crate::progress::LogLevel::Info,
                    format!("开始同步表: {}", full_table_name)
                );

                // 获取表结构
                let source_schema = self.get_mysql_table_schema(
                    &source_pool,
                    source_db,
                    table
                ).await?;

                // 创建目标表
                self.handle_mysql_table_with_strategy(
                    &target_pool,
                    &target_db,
                    table,
                    &source_schema,
                    &config.sync_config.table_exists_strategy
                ).await?;

                // 获取表记录数
                let table_count = self.get_mysql_table_count(&source_pool, source_db, table).await?;
                
                if table_count == 0 {
                    self.progress_monitor.add_log(
                        &task_id,
                        crate::progress::LogLevel::Info,
                        format!("表 {} 无数据，跳过", full_table_name)
                    );
                    self.progress_monitor.complete_table(&task_id, &full_table_name);
                    continue;
                }
                
                // 计算批次信息
                let total_batches = ((table_count as f64) / (batch_size as f64)).ceil() as u64;
                
                self.progress_monitor.add_log(
                    &task_id,
                    crate::progress::LogLevel::Info,
                    format!("表 {} 总计: {} 条，分 {} 批次同步", full_table_name, table_count, total_batches)
                );
                
                let mut offset = 0u64;
                let mut current_batch = 1u64;
                let mut table_synced_count = 0u64;

                // 批量同步数据（使用原始行数据，避免类型转换）
                while offset < table_count {
                    // 检查暂停标志
                    if let Some(state) = self.get_task_state(&task_id) {
                        while state.pause_flag.load(std::sync::atomic::Ordering::SeqCst) {
                            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                        }
                    }

                    // 读取批量数据（原始行数据）
                    let batch_result = self.read_mysql_batch_raw(
                        &source_pool,
                        source_db,
                        table,
                        &source_schema,
                        offset,
                        batch_size
                    ).await;

                    match batch_result {
                        Ok(rows) => {
                            if rows.is_empty() {
                                break;
                            }

                            let batch_len = rows.len() as u64;

                            // 插入数据（使用原始行数据，传入表结构）
                            let insert_result = self.batch_insert_mysql_raw(
                                &target_pool,
                                &target_db,
                                table,
                                &rows,
                                &source_schema
                            ).await;

                            match insert_result {
                                Ok(_) => {
                                    processed_records += batch_len;
                                    table_synced_count += batch_len;
                                    offset += batch_len;
                                    
                                    // 更新表进度
                                    self.progress_monitor.update_table_progress(&task_id, &full_table_name, table_synced_count);
                                    
                                    let remaining_batches = total_batches - current_batch;
                                    let batch_log = format!("  批次 {}/{} 完成，已同步: {} 条，剩余: {} 批次", 
                                        current_batch, total_batches, table_synced_count, remaining_batches);
                                    
                                    // 推送批次进度日志到前端
                                    self.progress_monitor.add_log(
                                        &task_id,
                                        crate::progress::LogLevel::Info,
                                        batch_log
                                    );
                                    
                                    current_batch += 1;
                                    self.progress_monitor.update_progress(&task_id, processed_records);
                                }
                                Err(e) => {
                                    self.progress_monitor.add_log(
                                        &task_id,
                                        crate::progress::LogLevel::Error,
                                        format!("表 {} 批次 {}/{} 插入失败: {}", full_table_name, current_batch, total_batches, e)
                                    );
                                    
                                    self.error_logger.log_error(
                                        &task_id,
                                        ErrorLog::new(
                                            "InsertError".to_string(),
                                            format!("插入数据到目标 MySQL 失败: {}", e),
                                            Some(json!({
                                                "table": table,
                                                "offset": offset,
                                                "batch": current_batch
                                            }))
                                        )
                                    );

                                    match error_strategy {
                                        ErrorStrategy::Skip => {
                                            offset += batch_len;
                                            current_batch += 1;
                                            continue;
                                        }
                                        ErrorStrategy::Pause => {
                                            self.progress_monitor.fail_table(&task_id, &full_table_name);
                                            return Err(e);
                                        }
                                    }
                                }
                            }
                        }
                        Err(e) => {
                            self.error_logger.log_error(
                                &task_id,
                                ErrorLog::new(
                                    "ReadError".to_string(),
                                    format!("读取源 MySQL 数据失败: {}", e),
                                    Some(json!({
                                        "table": table,
                                        "offset": offset,
                                        "batch": current_batch
                                    }))
                                )
                            );

                            match error_strategy {
                                ErrorStrategy::Skip => {
                                    offset += batch_size as u64;
                                    current_batch += 1;
                                    continue;
                                }
                                ErrorStrategy::Pause => {
                                    return Err(e);
                                }
                            }
                        }
                    }
                }
                
                self.progress_monitor.add_log(
                    &task_id,
                    crate::progress::LogLevel::Info,
                    format!("表 {} 同步完成，共同步: {} 条", full_table_name, table_synced_count)
                );
                
                // 数据校验：对比源表和目标表的记录数
                let source_count = self.get_mysql_table_count(&source_pool, source_db, table).await?;
                let target_count = self.get_mysql_table_count(&target_pool, &target_db, table).await?;
                
                if source_count == target_count {
                    self.progress_monitor.add_log(
                        &task_id,
                        crate::progress::LogLevel::Info,
                        format!("✓ 表 {} 数据校验通过: {} 条", full_table_name, source_count)
                    );
                    self.progress_monitor.complete_table(&task_id, &full_table_name);
                } else {
                    let diff = if source_count > target_count {
                        source_count - target_count
                    } else {
                        target_count - source_count
                    };
                    let success_rate = if source_count > 0 {
                        target_count as f64 / source_count as f64 * 100.0
                    } else {
                        100.0
                    };
                    
                    self.progress_monitor.add_log(
                        &task_id,
                        crate::progress::LogLevel::Warn,
                        format!("✗ 表 {} 数据校验失败: 源 {} 条，目标 {} 条，差异 {} 条，成功率 {:.2}%", 
                            full_table_name, source_count, target_count, diff, success_rate)
                    );
                    self.progress_monitor.fail_table(&task_id, &full_table_name);
                }
            }
        }

        Ok(())
    }

    /// ES 到 ES 同步实现
    pub(super) async fn sync_es_to_es_impl(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();
        let batch_size = config.sync_config.batch_size;
        let error_strategy = config.sync_config.error_strategy.clone();

        let source = self.source_manager.get_data_source(&config.source_id).await?
            .ok_or_else(|| anyhow::anyhow!("源数据源不存在"))?;
        let target = self.source_manager.get_data_source(&config.target_id).await?
            .ok_or_else(|| anyhow::anyhow!("目标数据源不存在"))?;

        let source_url = format!("http://{}:{}", source.host, source.port);
        let source_transport = elasticsearch::http::transport::Transport::single_node(&source_url)
            .context("创建源 ES 传输层失败")?;
        let source_client = elasticsearch::Elasticsearch::new(source_transport);

        let target_url = format!("http://{}:{}", target.host, target.port);
        let target_transport = elasticsearch::http::transport::Transport::single_node(&target_url)
            .context("创建目标 ES 传输层失败")?;
        let target_client = elasticsearch::Elasticsearch::new(target_transport);

        let es_config = config.es_config
            .ok_or_else(|| anyhow::anyhow!("缺少 ES 配置"))?;

        let mut all_indices = Vec::new();
        for index_sel in &es_config.indices {
            if let Some(matched) = &index_sel.matched_indices {
                all_indices.extend(matched.clone());
            } else {
                all_indices.push(index_sel.pattern.clone());
            }
        }

        let mut total_records = 0u64;
        for index in &all_indices {
            let count = self.get_es_index_count(&source_client, index).await?;
            total_records += count;
        }

        self.progress_monitor.start_task(&task_id, total_records);

        let mut processed_records = 0u64;

        for index in &all_indices {
            self.progress_monitor.update_current_table(&task_id, Some(index.clone()));

            let mapping_response = source_client
                .indices()
                .get_mapping(elasticsearch::indices::IndicesGetMappingParts::Index(&[index.as_str()]))
                .send()
                .await
                .context("获取索引映射失败")?;

            let mapping_body = mapping_response.json::<serde_json::Value>().await?;
            let source_schema = self.infer_schema_from_es_mapping(&mapping_body, index)?;

            self.handle_es_index_with_strategy(
                &target_client, 
                index, 
                &source_schema,
                &config.sync_config.table_exists_strategy
            ).await?;

            let mut scroll_id: Option<String> = None;

            loop {
                if let Some(state) = self.get_task_state(&task_id) {
                    while state.pause_flag.load(std::sync::atomic::Ordering::SeqCst) {
                        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                    }
                }

                let read_result = self.read_es_batch(
                    &source_client,
                    index,
                    scroll_id.as_deref(),
                    batch_size
                ).await;

                match read_result {
                    Ok((batch, new_scroll_id)) => {
                        if batch.is_empty() {
                            break;
                        }

                        scroll_id = new_scroll_id;

                        let insert_result = self.batch_insert_es(
                            &target_client,
                            index,
                            &batch,
                            Some("_id")
                        ).await;

                        match insert_result {
                            Ok(_) => {
                                processed_records += batch.len() as u64;
                                self.progress_monitor.update_progress(&task_id, processed_records);
                            }
                            Err(e) => {
                                self.error_logger.log_error(
                                    &task_id,
                                    ErrorLog::new(
                                        "InsertError".to_string(),
                                        format!("插入数据到目标 ES 失败: {}", e),
                                        Some(json!({ "index": index }))
                                    )
                                );

                                match error_strategy {
                                    ErrorStrategy::Skip => continue,
                                    ErrorStrategy::Pause => {
                                        if let Some(sid) = scroll_id {
                                            let _ = self.clear_es_scroll(&source_client, &sid).await;
                                        }
                                        return Err(e);
                                    }
                                }
                            }
                        }
                    }
                    Err(e) => {
                        self.error_logger.log_error(
                            &task_id,
                            ErrorLog::new(
                                "ReadError".to_string(),
                                format!("读取源 ES 数据失败: {}", e),
                                Some(json!({ "index": index }))
                            )
                        );

                        match error_strategy {
                            ErrorStrategy::Skip => break,
                            ErrorStrategy::Pause => {
                                if let Some(sid) = scroll_id {
                                    let _ = self.clear_es_scroll(&source_client, &sid).await;
                                }
                                return Err(e);
                            }
                        }
                    }
                }
            }

            if let Some(sid) = scroll_id {
                let _ = self.clear_es_scroll(&source_client, &sid).await;
            }
        }

        Ok(())
    }

    /// 从 ES 映射推断表结构
    fn infer_schema_from_es_mapping(
        &self,
        mapping: &serde_json::Value,
        index_name: &str,
    ) -> Result<TableSchema> {
        let mut columns = Vec::new();

        if let Some(index_mapping) = mapping.get(index_name) {
            if let Some(mappings) = index_mapping.get("mappings") {
                if let Some(properties) = mappings.get("properties") {
                    if let Some(props_obj) = properties.as_object() {
                        for (field_name, field_def) in props_obj {
                            let field_type = field_def.get("type")
                                .and_then(|v| v.as_str())
                                .unwrap_or("keyword");

                            columns.push(ColumnDefinition {
                                name: field_name.clone(),
                                data_type: field_type.to_string(),
                                nullable: true,
                                is_primary_key: false,
                            });
                        }
                    }
                }
            }
        }

        if columns.is_empty() {
            columns.push(ColumnDefinition {
                name: "_id".to_string(),
                data_type: "keyword".to_string(),
                nullable: false,
                is_primary_key: true,
            });
        }

        Ok(TableSchema {
            table_name: index_name.to_string(),
            columns,
            primary_key: Some("_id".to_string()),
        })
    }
}
