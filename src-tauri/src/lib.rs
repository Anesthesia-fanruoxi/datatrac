// 模块声明
mod crypto;
mod storage;
mod datasource;
mod type_mapper;
mod progress;
pub mod error_logger;
pub mod sync_engine;
mod commands;

use commands::AppState;
use crypto::CryptoService;
use datasource::DataSourceManager;
use error_logger::ErrorLogger;
use progress::ProgressMonitor;
use storage::Storage;
use sync_engine::SyncEngine;
use std::sync::Arc;
use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .setup(|app| {
      if cfg!(debug_assertions) {
        app.handle().plugin(
          tauri_plugin_log::Builder::default()
            .level(log::LevelFilter::Info)
            .build(),
        )?;
      }

      // 初始化后端服务
      let app_handle = app.handle().clone();
      
      tauri::async_runtime::block_on(async move {
        // 1. 初始化存储层
        let db_path = app_handle
          .path()
          .app_data_dir()
          .expect("无法获取应用数据目录")
          .join("datasync.db");
        
        log::info!("数据库路径: {:?}", db_path);
        
        // 确保目录存在
        if let Some(parent) = db_path.parent() {
          log::info!("创建数据目录: {:?}", parent);
          std::fs::create_dir_all(parent).expect("无法创建数据目录");
        }
        
        // 将路径转换为字符串，并替换反斜杠为正斜杠（Windows 兼容性）
        let db_path_str = db_path.to_str().unwrap().replace("\\", "/");
        log::info!("数据库连接字符串: {}", db_path_str);
        
        log::info!("初始化存储层...");
        let storage = Arc::new(
          Storage::new(&db_path_str)
            .await
            .expect("初始化存储层失败")
        );
        log::info!("存储层初始化成功");

        // 2. 初始化加密服务
        log::info!("初始化加密服务...");
        let crypto = Arc::new(CryptoService::new().expect("初始化加密服务失败"));
        log::info!("加密服务初始化成功");

        // 3. 初始化数据源管理器
        log::info!("初始化数据源管理器...");
        let data_source_manager = Arc::new(DataSourceManager::new(
          storage.clone(),
          crypto.clone(),
        ));
        log::info!("数据源管理器初始化成功");

        // 4. 初始化进度监控器
        log::info!("初始化进度监控器...");
        let mut progress_monitor = ProgressMonitor::new();
        progress_monitor.set_app_handle(app_handle.clone());
        let progress_monitor = Arc::new(progress_monitor);
        log::info!("进度监控器初始化成功");

        // 5. 初始化错误日志器
        log::info!("初始化错误日志器...");
        let error_logger = Arc::new(ErrorLogger::new().with_app_handle(app_handle.clone()));
        log::info!("错误日志器初始化成功");

        // 6. 初始化同步引擎
        log::info!("初始化同步引擎...");
        let sync_engine = Arc::new(SyncEngine::new(
          data_source_manager.clone(),
          progress_monitor.clone(),
          error_logger.clone(),
        ));
        log::info!("同步引擎初始化成功");

        // 7. 创建应用状态
        log::info!("创建应用状态...");
        let app_state = AppState {
          data_source_manager,
          progress_monitor,
          error_logger,
          sync_engine,
        };

        // 8. 将状态注入到 Tauri
        log::info!("注入应用状态到 Tauri...");
        app_handle.manage(app_state);
        log::info!("应用初始化完成！");
      });

      Ok(())
    })
    .invoke_handler(tauri::generate_handler![
      // 数据源管理
      commands::list_data_sources,
      commands::get_data_source,
      commands::create_data_source,
      commands::update_data_source,
      commands::delete_data_source,
      commands::test_connection,
      // 元数据查询
      commands::get_databases,
      commands::get_tables,
      commands::get_indices,
      commands::match_indices,
      // 同步任务管理
      commands::list_tasks,
      commands::get_task,
      commands::create_task,
      commands::update_task,
      commands::delete_task,
      // 任务执行控制
      commands::start_sync,
      commands::pause_sync,
      commands::resume_sync,
      commands::get_progress,
      commands::get_errors,
    ])
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}
