// Commands 模块
// 提供前后端通信的 API 接口层

pub mod datasource;
pub mod sync;
pub mod task;

use crate::core::{logger::TaskLogger, monitor::ProgressMonitor};
use crate::services::{datasource::DataSourceManager, sync::SyncEngine, task::TaskManager};
use crate::storage::Storage;
use std::sync::Arc;

/// 应用状态
/// 包含所有后端服务的引用
pub struct AppState {
    pub storage: Arc<Storage>,
    pub datasource_manager: Arc<DataSourceManager>,
    pub task_manager: Arc<TaskManager>,
    pub sync_engine: Arc<SyncEngine>,
    pub progress_monitor: Arc<ProgressMonitor>,
    pub task_logger: Arc<TaskLogger>,
}
