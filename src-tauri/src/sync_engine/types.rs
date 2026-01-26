// 同步引擎类型定义

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// 同步任务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncTaskConfig {
    /// 任务 ID
    pub task_id: String,
    /// 任务名称
    pub task_name: String,
    /// 源数据源 ID
    pub source_id: String,
    /// 目标数据源 ID
    pub target_id: String,
    /// 同步方向
    pub sync_direction: SyncDirection,
    /// 同步配置
    pub sync_config: SyncConfig,
    /// MySQL 配置（如果源或目标是 MySQL）
    pub mysql_config: Option<MysqlSyncConfig>,
    /// ES 配置（如果源或目标是 ES）
    pub es_config: Option<EsSyncConfig>,
}

/// 同步方向
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SyncDirection {
    MysqlToEs,
    EsToMysql,
    MysqlToMysql,
    EsToEs,
}

/// 同步配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncConfig {
    /// 线程数（1-32）
    pub thread_count: usize,
    /// 批量大小
    pub batch_size: usize,
    /// 错误处理策略
    pub error_strategy: ErrorStrategy,
    /// 数据库名称转换配置（可选）
    pub db_name_transform: Option<DbNameTransform>,
    /// 目标表存在时的处理策略
    pub table_exists_strategy: TableExistsStrategy,
}

/// 目标表存在时的处理策略
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TableExistsStrategy {
    /// 删除并重建（默认）
    Drop,
    /// 清空数据但保留表结构
    Truncate,
    /// 备份原表后重建
    Backup,
}

/// 数据库名称转换配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DbNameTransform {
    /// 是否启用
    pub enabled: bool,
    /// 转换模式
    pub mode: TransformMode,
    /// 源模式（前缀或后缀）
    pub source_pattern: String,
    /// 目标模式（前缀或后缀）
    pub target_pattern: String,
}

/// 转换模式
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TransformMode {
    /// 前缀替换
    Prefix,
    /// 后缀替换
    Suffix,
}

/// 错误处理策略
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum ErrorStrategy {
    /// 跳过错误并继续
    Skip,
    /// 遇到错误时暂停
    Pause,
}

/// MySQL 同步配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MysqlSyncConfig {
    /// 数据库和表的选择
    pub databases: Vec<DatabaseSelection>,
}

/// 数据库选择
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DatabaseSelection {
    /// 数据库名称
    pub database: String,
    /// 选中的表列表
    pub tables: Vec<String>,
}

/// ES 同步配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct EsSyncConfig {
    /// 索引选择
    pub indices: Vec<IndexSelection>,
}

/// 索引选择
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct IndexSelection {
    /// 索引模式（可以是具体索引名或通配符）
    pub pattern: String,
    /// 匹配到的索引列表（通配符展开后）
    pub matched_indices: Option<Vec<String>>,
}

/// 同步任务运行时状态
#[derive(Debug, Clone)]
pub struct SyncTaskState {
    /// 任务 ID
    pub task_id: String,
    /// 是否正在运行
    pub is_running: bool,
    /// 是否暂停
    pub is_paused: bool,
    /// 暂停标志（用于通知工作线程暂停）
    pub pause_flag: std::sync::Arc<std::sync::atomic::AtomicBool>,
    /// 停止标志（用于通知工作线程停止）
    pub stop_flag: std::sync::Arc<std::sync::atomic::AtomicBool>,
}

impl SyncTaskState {
    pub fn new(task_id: String) -> Self {
        Self {
            task_id,
            is_running: false,
            is_paused: false,
            pause_flag: std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false)),
            stop_flag: std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false)),
        }
    }
}

/// 表结构信息
#[derive(Debug, Clone)]
pub struct TableSchema {
    /// 表名
    pub table_name: String,
    /// 列定义
    pub columns: Vec<ColumnDefinition>,
    /// 主键列名
    pub primary_key: Option<String>,
}

/// 列定义
#[derive(Debug, Clone)]
pub struct ColumnDefinition {
    /// 列名
    pub name: String,
    /// 数据类型
    pub data_type: String,
    /// 是否可为空
    pub nullable: bool,
    /// 是否是主键
    pub is_primary_key: bool,
}

/// 批量数据
pub type BatchData = Vec<HashMap<String, serde_json::Value>>;
