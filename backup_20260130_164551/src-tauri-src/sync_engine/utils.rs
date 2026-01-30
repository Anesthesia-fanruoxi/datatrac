// 同步引擎工具方法模块

use super::types::*;
use crate::storage::DataSourceType;
use crate::type_mapper::TypeMapper;
use anyhow::Result;
use std::sync::Arc;
use tokio::sync::Semaphore;

/// 根据源表结构创建目标表结构（使用类型映射）
pub fn convert_schema(
    source_schema: &TableSchema,
    source_type: &DataSourceType,
    target_type: &DataSourceType,
) -> TableSchema {
    let mut target_schema = source_schema.clone();

    for col in &mut target_schema.columns {
        col.data_type = match (source_type, target_type) {
            (DataSourceType::Mysql, DataSourceType::Elasticsearch) => {
                TypeMapper::mysql_to_es(&col.data_type)
            }
            (DataSourceType::Elasticsearch, DataSourceType::Mysql) => {
                TypeMapper::es_to_mysql(&col.data_type)
            }
            _ => col.data_type.clone(),
        };
    }

    target_schema
}

/// 获取自适应批量大小
pub fn get_adaptive_batch_size(target_type: &DataSourceType, configured_size: usize) -> usize {
    if configured_size > 0 {
        configured_size
    } else {
        match target_type {
            DataSourceType::Mysql => 1000,
            DataSourceType::Elasticsearch => 500,
        }
    }
}

/// 转换数据库名称
/// 
/// 根据配置的转换规则，将源数据库名称转换为目标数据库名称
pub fn transform_database_name(original_name: &str, transform: &Option<DbNameTransform>) -> String {
    if let Some(t) = transform {
        if !t.enabled {
            return original_name.to_string();
        }

        match t.mode {
            TransformMode::Prefix => {
                // 前缀替换
                if original_name.starts_with(&t.source_pattern) {
                    format!("{}{}", t.target_pattern, &original_name[t.source_pattern.len()..])
                } else {
                    original_name.to_string()
                }
            }
            TransformMode::Suffix => {
                // 后缀替换
                if original_name.ends_with(&t.source_pattern) {
                    format!("{}{}", &original_name[..original_name.len() - t.source_pattern.len()], t.target_pattern)
                } else {
                    original_name.to_string()
                }
            }
        }
    } else {
        original_name.to_string()
    }
}

/// 根据配置的转换规则，将源索引名称转换为目标索引名称
pub fn transform_index_name(original_name: &str, transform: &IndexNameTransform) -> String {
    if !transform.enabled {
        return original_name.to_string();
    }

    match transform.mode {
        TransformMode::Prefix => {
            // 前缀替换
            if original_name.starts_with(&transform.source_pattern) {
                format!("{}{}", transform.target_pattern, &original_name[transform.source_pattern.len()..])
            } else {
                original_name.to_string()
            }
        }
        TransformMode::Suffix => {
            // 后缀替换
            if original_name.ends_with(&transform.source_pattern) {
                format!("{}{}", &original_name[..original_name.len() - transform.source_pattern.len()], transform.target_pattern)
            } else {
                original_name.to_string()
            }
        }
    }
}

/// 通用的并发批处理方法
/// 
/// # 参数
/// - `thread_count`: 并发线程数
/// - `batches`: 批次数据的迭代器
/// - `process_fn`: 处理单个批次的异步函数
/// 
/// # 返回
/// 处理成功的批次数量
pub async fn concurrent_batch_process<T, F, Fut>(
    thread_count: usize,
    batches: Vec<T>,
    process_fn: F,
) -> Result<usize>
where
    T: Send + 'static,
    F: Fn(T) -> Fut + Send + Sync + 'static,
    Fut: std::future::Future<Output = Result<()>> + Send,
{
    let semaphore = Arc::new(Semaphore::new(thread_count));
    let process_fn = Arc::new(process_fn);
    let mut tasks = Vec::new();
    let total_batches = batches.len();
    let success_count = Arc::new(std::sync::atomic::AtomicUsize::new(0));

    log::info!("开始并发处理 {} 个批次，并发数: {}", total_batches, thread_count);

    for batch in batches {
        let permit = semaphore.clone().acquire_owned().await?;
        let process_fn = process_fn.clone();
        let success_count = success_count.clone();

        let task = tokio::spawn(async move {
            let result = process_fn(batch).await;
            
            if result.is_ok() {
                success_count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
            }
            
            drop(permit);
            result
        });

        tasks.push(task);
    }

    // 等待所有任务完成
    let mut errors = Vec::new();
    for task in tasks {
        if let Err(e) = task.await {
            errors.push(format!("任务执行失败: {}", e));
        }
    }

    let success = success_count.load(std::sync::atomic::Ordering::SeqCst);
    
    if !errors.is_empty() {
        log::warn!("部分批次处理失败: {}/{}", errors.len(), total_batches);
    }

    log::info!("并发处理完成: 成功 {}/{}", success, total_batches);
    
    Ok(success)
}
