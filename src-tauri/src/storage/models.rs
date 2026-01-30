// 数据模型定义
// 定义所有数据库表对应的 Rust 结构体

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 数据源类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum DataSourceType {
    Mysql,
    Elasticsearch,
}

impl DataSourceType {
    pub fn as_str(&self) -> &str {
        match self {
            DataSourceType::Mysql => "mysql",
            DataSourceType::Elasticsearch => "elasticsearch",
        }
    }

    pub fn from_str(s: &str) -> Result<Self, anyhow::Error> {
        match s.to_lowercase().as_str() {
            "mysql" => Ok(DataSourceType::Mysql),
            "elasticsearch" => Ok(DataSourceType::Elasticsearch),
            _ => Err(anyhow::anyhow!("不支持的数据源类型: {}", s)),
        }
    }
}

/// 数据源
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DataSource {
    pub id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub source_type: DataSourceType,
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub database: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// 任务状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskStatus {
    Idle,
    Running,
    Paused,
    Completed,
    Failed,
}

impl TaskStatus {
    pub fn as_str(&self) -> &str {
        match self {
            TaskStatus::Idle => "idle",
            TaskStatus::Running => "running",
            TaskStatus::Paused => "paused",
            TaskStatus::Completed => "completed",
            TaskStatus::Failed => "failed",
        }
    }
}

/// 同步任务
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncTask {
    pub id: String,
    pub name: String,
    pub source_id: String,
    pub target_id: String,
    pub source_type: DataSourceType,
    pub target_type: DataSourceType,
    pub config: String,
    pub status: TaskStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// 任务单元状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskUnitStatus {
    Pending,
    Running,
    Completed,
    Failed,
}

impl TaskUnitStatus {
    pub fn as_str(&self) -> &str {
        match self {
            TaskUnitStatus::Pending => "pending",
            TaskUnitStatus::Running => "running",
            TaskUnitStatus::Completed => "completed",
            TaskUnitStatus::Failed => "failed",
        }
    }

    pub fn from_str(s: &str) -> Result<Self, anyhow::Error> {
        match s.to_lowercase().as_str() {
            "pending" => Ok(TaskUnitStatus::Pending),
            "running" => Ok(TaskUnitStatus::Running),
            "completed" => Ok(TaskUnitStatus::Completed),
            "failed" => Ok(TaskUnitStatus::Failed),
            _ => Err(anyhow::anyhow!("不支持的任务单元状态: {}", s)),
        }
    }
}

/// 任务单元类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskUnitType {
    Table,
    Index,
}

impl TaskUnitType {
    pub fn as_str(&self) -> &str {
        match self {
            TaskUnitType::Table => "table",
            TaskUnitType::Index => "index",
        }
    }

    pub fn from_str(s: &str) -> Result<Self, anyhow::Error> {
        match s.to_lowercase().as_str() {
            "table" => Ok(TaskUnitType::Table),
            "index" => Ok(TaskUnitType::Index),
            _ => Err(anyhow::anyhow!("不支持的任务单元类型: {}", s)),
        }
    }
}

/// 任务单元
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnit {
    pub id: String,
    pub task_id: String,
    pub unit_name: String,
    pub unit_type: TaskUnitType,
    pub status: String,
    pub total_records: i64,
    pub processed_records: i64,
    pub error_message: Option<String>,
    pub keyword: Option<String>, // 关联的关键字（标签）
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// 关键字映射
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct KeywordMapping {
    pub keyword: String,      // 关键字
    pub indices: Vec<String>, // 匹配到的索引列表
}

/// 已同步索引
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncedIndex {
    pub index_name: String,
    pub first_synced_at: DateTime<Utc>,
    pub last_synced_at: DateTime<Utc>,
    pub sync_count: i32,
    pub last_task_id: Option<String>,
}
