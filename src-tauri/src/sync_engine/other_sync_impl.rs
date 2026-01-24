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

            self.drop_and_create_mysql_table(
                &mysql_pool,
                target_database,
                index,
                &target_schema
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
                            target_database,
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

        let mut total_records = 0u64;
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                let count = self.get_mysql_table_count(&source_pool, &db_sel.database, table).await?;
                total_records += count;
            }
        }

        self.progress_monitor.start_task(&task_id, total_records);

        let mut processed_records = 0u64;
        let target_database = target.database.as_ref()
            .ok_or_else(|| anyhow::anyhow!("目标 MySQL 数据源缺少数据库名"))?;

        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                self.progress_monitor.update_current_table(
                    &task_id,
                    Some(format!("{}.{}", db_sel.database, table))
                );

                let source_schema = self.get_mysql_table_schema(
                    &source_pool,
                    &db_sel.database,
                    table
                ).await?;

                self.drop_and_create_mysql_table(
                    &target_pool,
                    target_database,
                    table,
                    &source_schema
                ).await?;

                let table_count = self.get_mysql_table_count(&source_pool, &db_sel.database, table).await?;
                let mut offset = 0u64;

                while offset < table_count {
                    if let Some(state) = self.get_task_state(&task_id) {
                        while state.pause_flag.load(std::sync::atomic::Ordering::SeqCst) {
                            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                        }
                    }

                    let batch = self.read_mysql_batch(
                        &source_pool,
                        &db_sel.database,
                        table,
                        offset,
                        batch_size
                    ).await;

                    match batch {
                        Ok(batch) => {
                            if batch.is_empty() {
                                break;
                            }

                            let insert_result = self.batch_insert_mysql(
                                &target_pool,
                                target_database,
                                table,
                                &batch
                            ).await;

                            match insert_result {
                                Ok(_) => {
                                    processed_records += batch.len() as u64;
                                    offset += batch.len() as u64;
                                    self.progress_monitor.update_progress(&task_id, processed_records);
                                }
                                Err(e) => {
                                    self.error_logger.log_error(
                                        &task_id,
                                        ErrorLog::new(
                                            "InsertError".to_string(),
                                            format!("插入数据到目标 MySQL 失败: {}", e),
                                            Some(json!({
                                                "table": table,
                                                "offset": offset
                                            }))
                                        )
                                    );

                                    match error_strategy {
                                        ErrorStrategy::Skip => {
                                            offset += batch.len() as u64;
                                            continue;
                                        }
                                        ErrorStrategy::Pause => {
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
                                        "offset": offset
                                    }))
                                )
                            );

                            match error_strategy {
                                ErrorStrategy::Skip => {
                                    offset += batch_size as u64;
                                    continue;
                                }
                                ErrorStrategy::Pause => {
                                    return Err(e);
                                }
                            }
                        }
                    }
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

            self.drop_and_create_es_index(&target_client, index, &source_schema).await?;

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
