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

        // 获取源和目标数据源
        let source = self.source_manager.get_data_source(&config.source_id).await?
            .ok_or_else(|| anyhow::anyhow!("源数据源不存在"))?;
        let target = self.source_manager.get_data_source(&config.target_id).await?
            .ok_or_else(|| anyhow::anyhow!("目标数据源不存在"))?;

        // 创建 MySQL 连接池
        let mysql_url = format!(
            "mysql://{}:{}@{}:{}",
            source.username, source.password, source.host, source.port
        );
        let mysql_pool = sqlx::mysql::MySqlPoolOptions::new()
            .max_connections(config.sync_config.thread_count as u32)
            .connect(&mysql_url)
            .await
            .context("连接 MySQL 失败")?;

        // 创建 ES 客户端
        let es_url = format!("http://{}:{}", target.host, target.port);
        let es_transport = elasticsearch::http::transport::Transport::single_node(&es_url)
            .context("创建 ES 传输层失败")?;
        let es_client = elasticsearch::Elasticsearch::new(es_transport);

        // 获取 MySQL 配置
        let mysql_config = config.mysql_config
            .ok_or_else(|| anyhow::anyhow!("缺少 MySQL 配置"))?;

        // 计算总记录数
        let mut total_records = 0u64;
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                let count = self.get_mysql_table_count(&mysql_pool, &db_sel.database, table).await?;
                total_records += count;
            }
        }

        // 启动进度监控
        self.progress_monitor.start_task(&task_id, total_records);

        let mut processed_records = 0u64;

        // 遍历所有数据库和表
        for db_sel in &mysql_config.databases {
            for table in &db_sel.tables {
                // 更新当前处理的表
                self.progress_monitor.update_current_table(
                    &task_id,
                    Some(format!("{}.{}", db_sel.database, table))
                );

                // 获取源表结构
                let source_schema = self.get_mysql_table_schema(
                    &mysql_pool,
                    &db_sel.database,
                    table
                ).await?;

                // 转换为 ES 索引结构
                let target_schema = self.convert_schema(
                    &source_schema,
                    &DataSourceType::Mysql,
                    &DataSourceType::Elasticsearch,
                );

                // 删除并重建 ES 索引
                let index_name = format!("{}_{}", db_sel.database, table).to_lowercase();
                self.drop_and_create_es_index(&es_client, &index_name, &target_schema).await?;

                // 分批读取并插入数据
                let table_count = self.get_mysql_table_count(&mysql_pool, &db_sel.database, table).await?;
                let mut offset = 0u64;

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
                                }
                                Err(e) => {
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
