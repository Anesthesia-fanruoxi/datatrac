// MySQL to MySQL sync implementation

use crate::sync_engine::{SyncEngine, SyncTaskConfig};
use anyhow::{Context, Result};
use sqlx::mysql::MySqlPool;
use sqlx::{Row, Column};
use std::sync::Arc;

pub async fn sync_mysql_to_mysql(engine: &SyncEngine, config: SyncTaskConfig) -> Result<()> {
    log::info!("MySQL to MySQL sync started, task ID: {}", config.task_id);
    engine.progress_monitor.start_task(&config.task_id, 0);
    
    let source_ds = engine.source_manager.get_data_source(&config.source_id).await?.context("Source not found")?;
    let target_ds = engine.source_manager.get_data_source(&config.target_id).await?.context("Target not found")?;
    
    let source_url = format!("mysql://{}:{}@{}:{}/", source_ds.username, source_ds.password, source_ds.host, source_ds.port);
    let target_url = format!("mysql://{}:{}@{}:{}/", target_ds.username, target_ds.password, target_ds.host, target_ds.port);
    
    let source_pool = Arc::new(MySqlPool::connect(&source_url).await?);
    let target_pool = Arc::new(MySqlPool::connect(&target_url).await?);
    
    let mysql_config = config.mysql_config.as_ref().ok_or_else(|| anyhow::anyhow!("MySQL config missing"))?;
    let concurrency = config.sync_config.thread_count as usize;
    
    log::info!("Concurrency: {}", concurrency);
    
    let mut all_tables = Vec::new();
    for db_selection in &mysql_config.databases {
        let source_db = &db_selection.database;
        let target_db = source_db.clone();
        
        sqlx::query(&format!("CREATE DATABASE IF NOT EXISTS `{}`", target_db))
            .execute(target_pool.as_ref()).await?;
        
        for table in &db_selection.tables {
            all_tables.push((source_db.clone(), target_db.clone(), table.clone()));
        }
    }
    
    log::info!("Total tables: {}", all_tables.len());
    
    use tokio::sync::Semaphore;
    let semaphore = Arc::new(Semaphore::new(concurrency));
    let mut tasks = Vec::new();
    let config_arc = Arc::new(config);
    
    for (source_db, target_db, table) in all_tables {
        let permit = semaphore.clone().acquire_owned().await?;
        let config_clone = Arc::clone(&config_arc);
        let source_pool_clone = Arc::clone(&source_pool);
        let target_pool_clone = Arc::clone(&target_pool);
        let task_manager = engine.task_manager().clone();
        let progress_monitor = Arc::clone(&engine.progress_monitor);
        
        let task = tokio::spawn(async move {
            let unit_name = format!("{}.{}", source_db, table);
            let result = sync_table(&task_manager, &progress_monitor, &config_clone, &source_pool_clone, &target_pool_clone, &source_db, &target_db, &table).await;
            drop(permit);
            (unit_name, result)
        });
        
        tasks.push(task);
    }
    
    for task in tasks {
        if let Ok((unit_name, result)) = task.await {
            if let Err(e) = result {
                log::error!("Table {} sync failed: {:?}", unit_name, e);
                if matches!(config_arc.sync_config.error_strategy, crate::sync_engine::ErrorStrategy::Pause) {
                    return Err(e);
                }
            }
        }
    }
    
    log::info!("MySQL to MySQL sync completed");
    Ok(())
}

async fn sync_table(
    task_manager: &Arc<crate::task_manager::TaskManager>,
    progress_monitor: &Arc<crate::progress::ProgressMonitor>,
    config: &SyncTaskConfig,
    source_pool: &MySqlPool,
    target_pool: &MySqlPool,
    source_db: &str,
    target_db: &str,
    table: &str,
) -> Result<()> {
    let unit_name = format!("{}.{}", source_db, table);
    let task_id = &config.task_id;
    
    log::info!("Syncing table: {}", unit_name);
    
    task_manager.update_unit_status_with_sync(task_id, &unit_name, crate::task_manager::TaskUnitStatus::Running, progress_monitor).await?;
    
    let ddl_query = format!("SHOW CREATE TABLE `{}`.`{}`", source_db, table);
    let row = sqlx::query(&ddl_query).fetch_one(source_pool).await?;
    let create_sql: String = row.try_get(1)?;
    
    let target_create_sql = create_sql.replace(&format!("CREATE TABLE `{}`", table), &format!("CREATE TABLE `{}`.`{}`", target_db, table));
    
    let _ = sqlx::query(&format!("DROP TABLE IF EXISTS `{}`.`{}`", target_db, table)).execute(target_pool).await;
    sqlx::query(&target_create_sql).execute(target_pool).await?;
    
    let count_query = format!("SELECT COUNT(*) as count FROM `{}`.`{}`", source_db, table);
    let row: (i64,) = sqlx::query_as(&count_query).fetch_one(source_pool).await?;
    let total_records = row.0;
    
    if total_records == 0 {
        task_manager.update_unit_status_with_sync(task_id, &unit_name, crate::task_manager::TaskUnitStatus::Completed, progress_monitor).await?;
        return Ok(());
    }
    
    let batch_size = config.sync_config.batch_size as i64;
    let mut offset = 0i64;
    let mut processed = 0i64;
    
    while offset < total_records {
        let select_query = format!("SELECT * FROM `{}`.`{}` LIMIT {} OFFSET {}", source_db, table, batch_size, offset);
        let rows = sqlx::query(&select_query).fetch_all(source_pool).await?;
        if rows.is_empty() { break; }
        
        let columns = rows[0].columns();
        let column_count = columns.len();
        let column_names: Vec<String> = columns.iter().map(|c| format!("`{}`", c.name())).collect();
        let placeholders = (0..column_count).map(|_| "?").collect::<Vec<_>>().join(", ");
        let insert_sql = format!("INSERT INTO `{}`.`{}` ({}) VALUES ({})", target_db, table, column_names.join(", "), placeholders);
        
        for row in &rows {
            let mut query = sqlx::query(&insert_sql);
            for i in 0..column_count {
                query = if let Ok(v) = row.try_get::<Option<chrono::NaiveDateTime>, _>(i) {
                    query.bind(v)
                } else if let Ok(v) = row.try_get::<Option<i64>, _>(i) {
                    query.bind(v)
                } else if let Ok(v) = row.try_get::<Option<f64>, _>(i) {
                    query.bind(v)
                } else if let Ok(v) = row.try_get::<Option<String>, _>(i) {
                    query.bind(v)
                } else if let Ok(v) = row.try_get::<Option<Vec<u8>>, _>(i) {
                    query.bind(v)
                } else {
                    query.bind(None::<String>)
                };
            }
            query.execute(target_pool).await?;
        }
        
        processed += rows.len() as i64;
        offset += batch_size;
        
        task_manager.update_unit_progress(task_id, &unit_name, total_records as u64, processed as u64);
        let updated_units = task_manager.get_task_units(task_id);
        progress_monitor.update_task_units(task_id, updated_units);
    }
    
    task_manager.update_unit_status_with_sync(task_id, &unit_name, crate::task_manager::TaskUnitStatus::Completed, progress_monitor).await?;
    log::info!("Table {} sync completed", unit_name);
    Ok(())
}
