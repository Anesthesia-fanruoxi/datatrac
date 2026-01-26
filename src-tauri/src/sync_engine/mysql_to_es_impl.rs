// MySQL 到 ES 同步实现

use super::types::*;
use super::SyncEngine;
use crate::error_logger::ErrorLog;
use crate::storage::DataSourceType;
use anyhow::{Context, Result};
use serde_json::json;

impl SyncEngine {
    /// MySQL 到 ES 同步实现
    pub(super) async fn sync_mysql_to_es_impl(&self, config: SyncTaskConfig) -> Result<()> {
        let task_id = config.task_id.clone();
        let batch_size = config.sync_config.batch_size;
        let error_strategy = config.sync_config.error_strategy.clone();

        log::info!("开始 MySQL -> ES 同步实现");
        log::info!("批量大小: {}, 错误策略: {:?}", batch_size, error_strategy);

        // 获取源和目标数据源
        log::info!("获取数据源配置...");
        let source = self.source_manager.get_data_source(&config.source_id).await?
            .ok_or_else(|| anyhow::anyhow!("源数据源不存在"))?;
        let target = self.source_manager.get_data_source(&config.target_id).await?
            .ok_or_else(|| anyhow::anyhow!("目标数据源不存在"))?;
        
        log::info!("源数据源: {}:{}", source.host, source.port);
        log::info!("目标数据源: {}:{}", target.host, target.port);

        // 创建 MySQL 连接池
        log::info!("创建 MySQL 连接池...");
        let mysql_url = format!(
            "mysql://{}:{}@{}:{}",
            source.username, source.password, source.host, source.port
        );
        let mysql_pool = sqlx::mysql::MySqlPoolOptions::new()
            .max_connections(config.sync_config.thread_count as u32)
            .connect(&mysql_url)
            .await
            .context("连接 MySQL 失败")?;
        log::info!("MySQL 连接池创建成功");

        // 创建 ES 客户端
        log::info!("创建 Elasticsearch 客户端...");
        let es_url = format!("http://{}:{}", target.host, target.port);
        let es_transport = elasticsearch::http::transport::Transport::single_node(&es_url)
            .context("创建 ES 传输层失败")?;
        let es_client = elasticsearch::Elasticsearch::new(es_transport);
        log::info!("Elasticsearch 客户端创建成功");

        // 获取 MySQL 配置
        let mysql_config = config.mysql_config
            .ok_or_else(|| anyhow::anyhow!("缺少 MySQL 配置"))?;

        log::info!("开始计算总记录数...");
        // 计算总记录数
        let mut total_records = 0u64;
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                let count = self.get_mysql_table_count(&mysql_pool, &db_sel.database, table).await?;
                log::info!("  {}.{}: {} 条记录", db_sel.database, table, count);
                total_records += count;
            }
        }
        log::info!("总记录数: {}", total_records);

        // 启动进度监控
        self.progress_monitor.start_task(&task_id, total_records);

        let mut processed_records = 0u64;

        // 遍历所有数据库和表
        log::info!("开始同步数据...");
        for db_sel in &mysql_config.databases {
            log::info!("处理数据库: {}", db_sel.database);
            
            for table in &db_sel.tables {
                log::info!("  处理表: {}", table);
                
                // 更新当前处理的表
                self.progress_monitor.update_current_table(
                    &task_id,
                    Some(format!("{}.{}", db_sel.database, table))
                );

                // 获取源表结构
                log::info!("    获取表结构...");
                let source_schema = self.get_mysql_table_schema(
                    &mysql_pool,
                    &db_sel.database,
                    table
                ).await?;
                log::info!("    表结构获取成功，字段数: {}", source_schema.columns.len());

                // 转换为 ES 索引结构
                log::info!("    转换为 ES 索引结构...");
                let target_schema = self.convert_schema(
                    &source_schema,
                    &DataSourceType::Mysql,
                    &DataSourceType::Elasticsearch,
                );

                // 应用数据库名称转换
                let transformed_db_name = self.transform_database_name(
                    &db_sel.database,
                    &config.sync_config.db_name_transform
                );
                if transformed_db_name != db_sel.database {
                    log::info!("    数据库名称转换: {} -> {}", db_sel.database, transformed_db_name);
                }

                // 删除并重建 ES 索引（使用转换后的数据库名称）
                let index_name = format!("{}_{}", transformed_db_name, table).to_lowercase();
                log::info!("    目标索引: {}", index_name);
                log::info!("    处理索引策略: {:?}", config.sync_config.table_exists_strategy);
                
                self.handle_es_index_with_strategy(
                    &es_client, 
                    &index_name, 
                    &target_schema,
                    &config.sync_config.table_exists_strategy
                ).await?;
                log::info!("    索引准备完成");

                // 分批读取并插入数据
                let table_count = self.get_mysql_table_count(&mysql_pool, &db_sel.database, table).await?;
                let mut offset = 0u64;
                log::info!("    开始同步数据，总记录数: {}", table_count);

                while offset < table_count {
                    // 检查暂停标志
                    if let Some(state) = self.get_task_state(&task_id) {
                        while state.pause_flag.load(std::sync::atomic::Ordering::SeqCst) {
                            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                        }
                    }

                    // 读取批量数据
                    let batch = self.read_mysql_batch(
                        &mysql_pool,
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

                            log::info!("      批次 {}-{}: 读取 {} 条记录", 
                                offset, offset + batch.len() as u64, batch.len());

                            // 插入到 ES
                            let insert_result = self.batch_insert_es(
                                &es_client,
                                &index_name,
                                &batch,
                                source_schema.primary_key.as_deref()
                            ).await;

                            match insert_result {
                                Ok(_) => {
                                    processed_records += batch.len() as u64;
                                    offset += batch.len() as u64;
                                    self.progress_monitor.update_progress(&task_id, processed_records);
                                    log::info!("      批次插入成功，已处理: {}/{}", processed_records, total_records);
                                }
                                Err(e) => {
                                    log::error!("      批次插入失败: {}", e);
                                    self.error_logger.log_error(
                                        &task_id,
                                        ErrorLog::new(
                                            "InsertError".to_string(),
                                            format!("插入数据到 ES 失败: {}", e),
                                            Some(json!({
                                                "table": table,
                                                "offset": offset
                                            }))
                                        )
                                    );

                                    match error_strategy {
                                        ErrorStrategy::Skip => {
                                            log::warn!("      跳过错误，继续处理下一批");
                                            offset += batch.len() as u64;
                                            continue;
                                        }
                                        ErrorStrategy::Pause => {
                                            log::error!("      遇错暂停，停止同步");
                                            return Err(e);
                                        }
                                    }
                                }
                            }
                        }
                        Err(e) => {
                            log::error!("      读取数据失败: {}", e);
                            self.error_logger.log_error(
                                &task_id,
                                ErrorLog::new(
                                    "ReadError".to_string(),
                                    format!("读取 MySQL 数据失败: {}", e),
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
}
