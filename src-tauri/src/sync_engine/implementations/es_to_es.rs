// Elasticsearch → Elasticsearch 同步实现

use crate::sync_engine::{SyncEngine, SyncTaskConfig};
use crate::progress::ProgressMonitor;
use crate::error_logger::ErrorLogger;
use anyhow::{Context, Result};
use elasticsearch::Elasticsearch;
use serde_json::{json, Value};
use std::sync::Arc;

/// Elasticsearch → Elasticsearch 同步实现
/// 
/// 功能：
/// - 从源 ES 读取数据
/// - 写入目标 ES
/// - 支持索引名称转换
/// - 支持多个搜索条件组
/// - 支持批量重索引
/// - 支持多索引并发同步（使用任务管理器）
/// - 支持暂停/恢复
pub async fn sync_es_to_es(engine: &SyncEngine, config: SyncTaskConfig) -> Result<()> {
    log::info!("开始 Elasticsearch → Elasticsearch 同步");
    
    // 获取任务状态（用于暂停检查）
    let task_state = engine.get_task_state(&config.task_id);
    let pause_flag = task_state.as_ref().map(|s| s.pause_flag.clone());
    
    // 获取 ES 配置
    let es_config = config.es_config
        .as_ref()
        .context("缺少 ES 配置")?;
    
    // 获取要同步的索引列表
    let indices = if let Some(selected_indices) = &es_config.selected_indices {
        selected_indices.clone()
    } else if let Some(search_groups) = &es_config.search_groups {
        // 从搜索条件组中提取所有匹配的索引
        search_groups.iter()
            .flat_map(|group| group.matched_indices.clone())
            .collect()
    } else {
        anyhow::bail!("未配置要同步的索引");
    };
    
    if indices.is_empty() {
        anyhow::bail!("没有要同步的索引");
    }
    
    let thread_count = config.sync_config.thread_count;
    log::info!("准备同步 {} 个索引，并发线程数: {}", indices.len(), thread_count);
    
    // 获取源和目标数据源信息
    let source_ds = engine.source_manager.get_data_source(&config.source_id).await?
        .context("源数据源不存在")?;
    let target_ds = engine.source_manager.get_data_source(&config.target_id).await?
        .context("目标数据源不存在")?;
    
    // 创建 ES 客户端（支持认证和超时配置）
    use elasticsearch::auth::Credentials;
    use elasticsearch::http::transport::TransportBuilder;
    use std::time::Duration;
    use std::sync::Arc;
    
    let source_url = format!("http://{}:{}", source_ds.host, source_ds.port);
    let target_url = format!("http://{}:{}", target_ds.host, target_ds.port);
    
    log::info!("源 ES 地址: {}", source_url);
    log::info!("目标 ES 地址: {}", target_url);
    
    // 构建源客户端
    let mut source_builder = TransportBuilder::new(elasticsearch::http::transport::SingleNodeConnectionPool::new(
        source_url.parse()?
    ))
    .timeout(Duration::from_secs(60))
    .disable_proxy();
    
    if !source_ds.username.is_empty() {
        log::info!("源 ES 使用认证: {}", source_ds.username);
        source_builder = source_builder.auth(Credentials::Basic(
            source_ds.username.clone(),
            source_ds.password.clone(),
        ));
    }
    let source_transport = source_builder.build()?;
    let source_client = Arc::new(Elasticsearch::new(source_transport));
    
    // 构建目标客户端
    let mut target_builder = TransportBuilder::new(elasticsearch::http::transport::SingleNodeConnectionPool::new(
        target_url.parse()?
    ))
    .timeout(Duration::from_secs(60))
    .disable_proxy();
    
    if !target_ds.username.is_empty() {
        log::info!("目标 ES 使用认证: {}", target_ds.username);
        target_builder = target_builder.auth(Credentials::Basic(
            target_ds.username.clone(),
            target_ds.password.clone(),
        ));
    }
    let target_transport = target_builder.build()?;
    let target_client = Arc::new(Elasticsearch::new(target_transport));
    
    // 初始化进度
    engine.progress_monitor.start_task(&config.task_id, 0);
    
    // 准备索引任务列表
    let index_tasks: Vec<_> = indices.into_iter().map(|source_index| {
        // 转换索引名称
        let target_index = if let Some(transform) = &es_config.index_name_transform {
            engine.transform_index_name(&source_index, transform)
        } else {
            source_index.clone()
        };
        
        IndexSyncTask {
            source_index,
            target_index,
        }
    }).collect();
    
    let total_indices = index_tasks.len();
    let index_names: Vec<String> = index_tasks.iter().map(|t| t.source_index.clone()).collect();
    
    // 使用任务管理器执行并发同步
    let task_manager = engine.task_manager().clone();
    let task_manager_for_closure = task_manager.clone();
    let progress_monitor = engine.progress_monitor.clone();
    let error_logger = engine.error_logger.clone();
    let task_id = config.task_id.clone();
    let config_clone = config.clone();
    let index_tasks_clone = index_tasks.clone();
    let pause_flag_clone = pause_flag.clone();
    
    let success = task_manager.execute_units(
        &task_id,
        index_names,
        thread_count,
        progress_monitor.clone(),
        move |source_index| {
            let source_client = source_client.clone();
            let target_client = target_client.clone();
            let config = config_clone.clone();
            let progress_monitor = progress_monitor.clone();
            let error_logger = error_logger.clone();
            let task_manager = task_manager_for_closure.clone();
            let index_tasks = index_tasks_clone.clone();
            let pause_flag = pause_flag_clone.clone();
            
            // 找到对应的目标索引
            let target_index = index_tasks.iter()
                .find(|t| t.source_index == source_index)
                .map(|t| t.target_index.clone())
                .unwrap_or_else(|| source_index.clone());
            
            async move {
                log::info!("同步索引: {} -> {}", source_index, target_index);
                
                let result = sync_single_index_impl(
                    &progress_monitor,
                    &error_logger,
                    &task_manager,
                    &source_client,
                    &target_client,
                    &source_index,
                    &target_index,
                    &config,
                    pause_flag.as_ref(),
                ).await;
                
                match &result {
                    Ok(count) => {
                        log::info!("索引 {} 同步完成，共 {} 条记录", source_index, count);
                        progress_monitor.update_table_progress(
                            &config.task_id,
                            &source_index,
                            *count as u64,
                        );
                    }
                    Err(e) => {
                        log::error!("索引 {} 同步失败: {}", source_index, e);
                        
                        // 构建错误日志
                        let error_log = crate::error_logger::ErrorLog::new(
                            "IndexSyncError".to_string(),
                            format!("索引 {} 同步失败: {}", source_index, e),
                            Some(serde_json::json!({
                                "index": source_index,
                                "error": e.to_string()
                            }))
                        );
                        error_logger.log_error(&config.task_id, error_log);
                    }
                }
                
                result.map(|_| ())
            }
        }
    ).await?;
    
    log::info!("所有索引同步完成: 成功 {}/{}", success, total_indices);
    
    // 推送总体校验日志到前端
    if success == total_indices {
        engine.progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Verify,
            format!("✓ 所有索引同步完成：成功 {}/{} 个索引", success, total_indices)
        );
    } else {
        engine.progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Warn,
            crate::progress::LogCategory::Verify,
            format!("⚠ 部分索引同步失败：成功 {}/{} 个索引", success, total_indices)
        );
    }
    
    Ok(())
}

/// 索引同步任务
#[derive(Clone)]
struct IndexSyncTask {
    source_index: String,
    target_index: String,
}

/// 同步单个索引的实现
async fn sync_single_index_impl(
    progress_monitor: &Arc<ProgressMonitor>,
    _error_logger: &Arc<ErrorLogger>,
    task_manager: &Arc<crate::task_manager::TaskManager>,
    source_client: &Elasticsearch,
    target_client: &Elasticsearch,
    source_index: &str,
    target_index: &str,
    config: &SyncTaskConfig,
    pause_flag: Option<&Arc<std::sync::atomic::AtomicBool>>,
) -> Result<usize> {
    let batch_size = config.sync_config.batch_size;
    let scroll_timeout = "5m";
    let mut total_count = 0;
    
    // 初始化 scroll 查询
    let response = source_client
        .search(elasticsearch::SearchParts::Index(&[source_index]))
        .scroll(scroll_timeout)
        .size(batch_size as i64)
        .body(json!({
            "query": {
                "match_all": {}
            }
        }))
        .send()
        .await;
    
    let response = match response {
        Ok(r) => r,
        Err(e) => {
            log::error!("scroll 请求发送失败: {:?}", e);
            log::error!("错误详情: {}", e);
            return Err(anyhow::anyhow!("初始化 scroll 查询失败: {}", e));
        }
    };
    
    // 检查响应状态
    let status = response.status_code();
    if !response.status_code().is_success() {
        let error_text = response.text().await.unwrap_or_else(|_| "无法读取错误信息".to_string());
        log::error!("ES 返回错误状态 {}: {}", status, error_text);
        return Err(anyhow::anyhow!("初始化 scroll 查询失败: HTTP {}, {}", status, error_text));
    }
    
    let response_body: Value = response.json().await
        .context("解析响应失败")?;
    
    // 获取总记录数
    let total_hits = response_body["hits"]["total"]["value"]
        .as_u64()
        .or_else(|| response_body["hits"]["total"].as_u64())
        .unwrap_or(0);
    
    // 推送总记录数到前端
    progress_monitor.add_log(
        &config.task_id,
        crate::progress::LogLevel::Info,
        crate::progress::LogCategory::Realtime,
        format!("索引 {} 总记录数: {}", source_index, total_hits)
    );
    
    // 更新任务单元的总记录数
    task_manager.update_unit_progress_with_sync(
        &config.task_id,
        source_index,
        total_hits,
        0,
        progress_monitor,
    );
    
    let mut scroll_id = response_body["_scroll_id"]
        .as_str()
        .context("获取 scroll_id 失败")?
        .to_string();
    
    let mut current_response = response_body;
    
    let mut batch_count = 0;  // 批次计数
    
    // 循环读取数据
    loop {
        // 检查暂停标志
        if let Some(flag) = pause_flag {
            if flag.load(std::sync::atomic::Ordering::SeqCst) {
                progress_monitor.add_log(
                    &config.task_id,
                    crate::progress::LogLevel::Info,
                    crate::progress::LogCategory::Realtime,
                    format!("索引 {} 同步已暂停", source_index)
                );
                
                // 清理 scroll
                let _ = source_client
                    .clear_scroll(elasticsearch::ClearScrollParts::None)
                    .body(json!({
                        "scroll_id": [scroll_id]
                    }))
                    .send()
                    .await;
                
                return Err(anyhow::anyhow!("任务已暂停"));
            }
        }
        
        let hits = current_response["hits"]["hits"]
            .as_array()
            .context("解析 hits 失败")?;
        
        if hits.is_empty() {
            progress_monitor.add_log(
                &config.task_id,
                crate::progress::LogLevel::Info,
                crate::progress::LogCategory::Realtime,
                "数据读取完成".to_string()
            );
            break;
        }
        
        batch_count += 1;
        
        // 推送实时日志到前端
        progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Realtime,
            format!("第 {} 批：读取到 {} 条记录", batch_count, hits.len())
        );
        
        // 推送明细日志到前端（批次摘要）
        progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Summary,
            format!("批次 {}/{} 开始处理，本批 {} 条记录", batch_count, "?", hits.len())
        );
        
        // 构建 bulk 请求体(Vec<String>格式)
        let mut bulk_body = Vec::new();
        for hit in hits {
            let doc_id = hit["_id"].as_str().unwrap_or("");
            let source = &hit["_source"];
            
            // index 操作行
            let index_line = json!({
                "index": {
                    "_index": target_index,
                    "_id": doc_id
                }
            });
            bulk_body.push(index_line.to_string());
            
            // 文档数据行
            bulk_body.push(source.to_string());
        }
        
        // 推送实时日志到前端
        progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Realtime,
            format!("第 {} 批：开始写入 {} 条记录到目标索引", batch_count, hits.len())
        );
        
        // 执行 bulk 写入
        let bulk_response = target_client
            .bulk(elasticsearch::BulkParts::None)
            .body(bulk_body)
            .send()
            .await
            .context("bulk 写入失败")?;
        
        let bulk_result: Value = bulk_response.json().await
            .context("解析 bulk 响应失败")?;
        
        // 检查是否有错误
        if bulk_result["errors"].as_bool().unwrap_or(false) {
            progress_monitor.add_log(
                &config.task_id,
                crate::progress::LogLevel::Warn,
                crate::progress::LogCategory::Error,
                format!("第 {} 批：bulk 写入部分失败", batch_count)
            );
        }
        
        total_count += hits.len();
        
        // 更新任务单元进度
        task_manager.update_unit_progress_with_sync(
            &config.task_id,
            source_index,
            total_hits,
            total_count as u64,
            progress_monitor,
        );
        
        let progress_percent = if total_hits > 0 {
            (total_count as f64 / total_hits as f64 * 100.0) as u32
        } else {
            0
        };
        
        // 推送实时进度日志到前端
        progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Realtime,
            format!("已同步 {} / {} 条记录 ({}%)", total_count, total_hits, progress_percent)
        );
        
        // 推送明细日志到前端（批次完成摘要）
        let remaining_batches = if total_hits > 0 {
            ((total_hits - total_count as u64) as f64 / batch_size as f64).ceil() as usize
        } else {
            0
        };
        progress_monitor.add_log(
            &config.task_id,
            crate::progress::LogLevel::Info,
            crate::progress::LogCategory::Summary,
            format!("批次 {} 完成，已同步: {} 条，剩余: {} 批次", batch_count, total_count, remaining_batches)
        );
        
        // 继续 scroll
        let scroll_response = source_client
            .scroll(elasticsearch::ScrollParts::None)
            .body(json!({
                "scroll": scroll_timeout,
                "scroll_id": scroll_id
            }))
            .send()
            .await
            .context("scroll 查询失败")?;
        
        current_response = scroll_response.json().await
            .context("解析 scroll 响应失败")?;
        
        scroll_id = current_response["_scroll_id"]
            .as_str()
            .context("获取 scroll_id 失败")?
            .to_string();
    }
    
    // 清理 scroll
    let _ = source_client
        .clear_scroll(elasticsearch::ClearScrollParts::None)
        .body(json!({
            "scroll_id": [scroll_id]
        }))
        .send()
        .await;
    
    // 推送校验日志到前端
    progress_monitor.add_log(
        &config.task_id,
        crate::progress::LogLevel::Info,
        crate::progress::LogCategory::Verify,
        format!("索引 {} 同步完成，共同步 {} 条记录", source_index, total_count)
    );
    
    // 可选：验证目标索引的记录数
    let verify_response = target_client
        .count(elasticsearch::CountParts::Index(&[target_index]))
        .send()
        .await;
    
    if let Ok(verify_resp) = verify_response {
        if let Ok(verify_body) = verify_resp.json::<Value>().await {
            if let Some(target_count) = verify_body["count"].as_u64() {
                if target_count == total_count as u64 {
                    progress_monitor.add_log(
                        &config.task_id,
                        crate::progress::LogLevel::Info,
                        crate::progress::LogCategory::Verify,
                        format!("✓ 索引 {} 数据校验通过：源 {} 条 = 目标 {} 条", source_index, total_count, target_count)
                    );
                } else {
                    progress_monitor.add_log(
                        &config.task_id,
                        crate::progress::LogLevel::Warn,
                        crate::progress::LogCategory::Verify,
                        format!("⚠ 索引 {} 数据校验失败：源 {} 条 ≠ 目标 {} 条", source_index, total_count, target_count)
                    );
                }
            }
        }
    }
    
    Ok(total_count)
}
