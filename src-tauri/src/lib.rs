// DataTrac 库入口
// 负责模块声明、应用初始化和状态管理

mod commands;
mod core;
mod exchange;
mod services;
mod storage;
mod utils;

use commands::AppState;
use core::{logger::TaskLogger, monitor::ProgressMonitor};
use services::{datasource::DataSourceManager, sync::SyncEngine, task::TaskManager};
use std::sync::Arc;
use storage::Storage;
use tauri::Manager;
use utils::crypto::CryptoService;

/// 运行 Tauri 应用
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            // 配置日志插件
            app.handle().plugin(
                tauri_plugin_log::Builder::default()
                    .level(log::LevelFilter::Info)
                    .timezone_strategy(tauri_plugin_log::TimezoneStrategy::UseLocal)
                    .targets([
                        tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::Stdout),
                        tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::LogDir {
                            file_name: None,
                        }),
                        tauri_plugin_log::Target::new(tauri_plugin_log::TargetKind::Webview),
                    ])
                    .build(),
            )?;

            // 设置窗口大小为屏幕的 80%
            if let Some(window) = app.get_webview_window("main") {
                if let Ok(monitor) = window.current_monitor() {
                    if let Some(monitor) = monitor {
                        let size = monitor.size();
                        let scale_factor = monitor.scale_factor();

                        let logical_width = (size.width as f64 / scale_factor) as u32;
                        let logical_height = (size.height as f64 / scale_factor) as u32;

                        let window_width = (logical_width as f64 * 0.8) as u32;
                        let window_height = (logical_height as f64 * 0.8) as u32;

                        let _ = window.set_size(tauri::Size::Logical(tauri::LogicalSize {
                            width: window_width as f64,
                            height: window_height as f64,
                        }));

                        let _ = window.center();

                        log::info!(
                            "窗口大小设置为: {}x{} (屏幕: {}x{})",
                            window_width,
                            window_height,
                            logical_width,
                            logical_height
                        );
                    }
                }
            }

            // 初始化后端服务
            let app_handle = app.handle().clone();

            tauri::async_runtime::block_on(async move {
                log::info!("开始初始化后端服务...");

                // 1. 初始化存储层
                let db_path = app_handle
                    .path()
                    .app_data_dir()
                    .expect("无法获取应用数据目录")
                    .join("datatrac.db");

                log::info!("数据库路径: {:?}", db_path);

                if let Some(parent) = db_path.parent() {
                    std::fs::create_dir_all(parent).expect("无法创建数据目录");
                }

                let db_path_str = db_path.to_str().unwrap().replace("\\", "/");
                let storage = Arc::new(Storage::new(&db_path_str).await.expect("初始化存储层失败"));
                log::info!("✓ 存储层初始化成功");

                // 2. 初始化加密服务
                let crypto = Arc::new(CryptoService::new().expect("初始化加密服务失败"));
                log::info!("✓ 加密服务初始化成功");

                // 3. 初始化数据源管理器
                let datasource_manager =
                    Arc::new(DataSourceManager::new(storage.clone(), crypto.clone()));
                log::info!("✓ 数据源管理器初始化成功");

                // 4. 初始化任务管理器
                let task_manager = Arc::new(TaskManager::new(storage.clone()));
                log::info!("✓ 任务管理器初始化成功");

                // 5. 初始化进度监控器
                let progress_monitor = Arc::new(ProgressMonitor::new(app_handle.clone()));
                log::info!("✓ 进度监控器初始化成功");

                // 6. 初始化日志管理器
                let task_logger = Arc::new(TaskLogger::new(app_handle.clone()));
                log::info!("✓ 日志管理器初始化成功");

                // 7. 初始化同步引擎
                let sync_engine = Arc::new(SyncEngine::new(
                    datasource_manager.clone(),
                    task_manager.clone(),
                    progress_monitor.clone(),
                    task_logger.clone(),
                ));
                log::info!("✓ 同步引擎初始化成功");

                // 8. 创建应用状态
                let app_state = AppState {
                    storage: storage.clone(),
                    datasource_manager,
                    task_manager,
                    sync_engine,
                    progress_monitor,
                    task_logger,
                };

                // 9. 注入应用状态
                app_handle.manage(app_state);
                log::info!("✓ 应用初始化完成！");
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            // 数据源管理
            commands::datasource::list_datasources,
            commands::datasource::get_datasource,
            commands::datasource::create_datasource,
            commands::datasource::update_datasource,
            commands::datasource::delete_datasource,
            commands::datasource::test_connection,
            commands::datasource::get_databases,
            commands::datasource::get_tables,
            commands::datasource::get_indices,
            commands::datasource::match_indices,
            // 任务管理
            commands::task::list_tasks,
            commands::task::get_task,
            commands::task::create_task,
            commands::task::update_task,
            commands::task::delete_task,
            commands::task::get_task_units,
            commands::task::reset_failed_units,
            commands::task::list_synced_indices,
            commands::task::clear_synced_index,
            commands::task::clear_all_synced_indices,
            // 同步控制
            commands::sync::start_sync,
            commands::sync::pause_sync,
            commands::sync::resume_sync,
            commands::sync::get_progress,
            commands::sync::get_logs,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
