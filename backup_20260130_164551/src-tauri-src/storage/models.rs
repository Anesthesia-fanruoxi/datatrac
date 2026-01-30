// 数据模型定义

use anyhow::Result;
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

    pub fn from_str(s: &str) -> Result<Self> {
        match s {
            "mysql" => Ok(DataSourceType::Mysql),
            "elasticsearch" => Ok(DataSourceType::Elasticsearch),
            _ => anyhow::bail!("未知的数据源类型: {}", s),
        }
    }
}

/// 数据源配置
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
    pub password: String,         // 加密存储
    pub database: Option<String>, // MySQL 专用
    #[serde(rename = "createdAt", with = "chrono::serde::ts_milliseconds")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "updatedAt", with = "chrono::serde::ts_milliseconds")]
    pub updated_at: DateTime<Utc>,
}

/// 同步任务状态
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

    pub fn from_str(s: &str) -> Result<Self> {
        match s {
            "idle" => Ok(TaskStatus::Idle),
            "running" => Ok(TaskStatus::Running),
            "paused" => Ok(TaskStatus::Paused),
            "completed" => Ok(TaskStatus::Completed),
            "failed" => Ok(TaskStatus::Failed),
            _ => anyhow::bail!("未知的任务状态: {}", s),
        }
    }
}

/// 同步任务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncTask {
    pub id: String,
    pub name: String,
    #[serde(rename = "sourceId")]
    pub source_id: String,
    #[serde(rename = "targetId")]
    pub target_id: String,
    #[serde(rename = "sourceType")]
    pub source_type: DataSourceType,
    #[serde(rename = "targetType")]
    pub target_type: DataSourceType,
    pub config: String, // JSON 格式的配置
    pub status: TaskStatus,
    #[serde(rename = "createdAt", with = "chrono::serde::ts_milliseconds")]
    pub created_at: DateTime<Utc>,
    #[serde(rename = "updatedAt", with = "chrono::serde::ts_milliseconds")]
    pub updated_at: DateTime<Utc>,
}

/// 任务单元状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskUnitStatus {
    Pending,   // 等待执行
    Running,   // 执行中
    Completed, // 已完成
    Failed,    // 失败
    Paused,    // 暂停
}

impl TaskUnitStatus {
    pub fn as_str(&self) -> &str {
        match self {
            TaskUnitStatus::Pending => "pending",
            TaskUnitStatus::Running => "running",
            TaskUnitStatus::Completed => "completed",
            TaskUnitStatus::Failed => "failed",
            TaskUnitStatus::Paused => "paused",
        }
    }

    pub fn from_str(s: &str) -> Result<Self> {
        match s {
            "pending" => Ok(TaskUnitStatus::Pending),
            "running" => Ok(TaskUnitStatus::Running),
            "completed" => Ok(TaskUnitStatus::Completed),
            "failed" => Ok(TaskUnitStatus::Failed),
            "paused" => Ok(TaskUnitStatus::Paused),
            _ => anyhow::bail!("未知的任务单元状态: {}", s),
        }
    }
}

/// 任务单元类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum TaskUnitType {
    Table, // MySQL 表
    Index, // ES 索引
}

impl TaskUnitType {
    pub fn as_str(&self) -> &str {
        match self {
            TaskUnitType::Table => "table",
            TaskUnitType::Index => "index",
        }
    }

    pub fn from_str(s: &str) -> Result<Self> {
        match s {
            "table" => Ok(TaskUnitType::Table),
            "index" => Ok(TaskUnitType::Index),
            _ => anyhow::bail!("未知的任务单元类型: {}", s),
        }
    }
}

// ==================== 三表结构模型 ====================

/// 任务单元配置表
/// 记录任务的配置信息，只在修改任务配置时变动
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitConfig {
    pub id: String,
    #[serde(rename = "taskId")]
    pub task_id: String,
    #[serde(rename = "unitName")]
    pub unit_name: String,
    #[serde(rename = "unitType")]
    pub unit_type: TaskUnitType,
    #[serde(rename = "searchPattern")]
    pub search_pattern: Option<String>,
    #[serde(rename = "createdAt")]
    pub created_at: i64, // 毫秒时间戳
    #[serde(rename = "updatedAt")]
    pub updated_at: i64, // 毫秒时间戳
}

/// 任务单元运行记录表
/// 记录当前正在运行或待运行的单元状态
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitRuntime {
    pub id: String,
    #[serde(rename = "taskId")]
    pub task_id: String,
    #[serde(rename = "unitName")]
    pub unit_name: String,
    pub status: TaskUnitStatus, // pending/running/failed
    #[serde(rename = "totalRecords")]
    pub total_records: i64,
    #[serde(rename = "processedRecords")]
    pub processed_records: i64,
    #[serde(rename = "errorMessage")]
    pub error_message: Option<String>,
    #[serde(rename = "startedAt")]
    pub started_at: Option<i64>, // 毫秒时间戳
    #[serde(rename = "updatedAt")]
    pub updated_at: i64, // 毫秒时间戳
    /// 断点续传: 最后处理完成的批次号
    #[serde(rename = "lastProcessedBatch")]
    pub last_processed_batch: Option<i64>,
}

/// 任务单元历史记录表
/// 记录已完成的同步历史
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskUnitHistory {
    pub id: String,
    #[serde(rename = "taskId")]
    pub task_id: String,
    #[serde(rename = "unitName")]
    pub unit_name: String,
    #[serde(rename = "searchPattern")]
    pub search_pattern: Option<String>,
    #[serde(rename = "totalRecords")]
    pub total_records: i64,
    #[serde(rename = "completedAt")]
    pub completed_at: i64, // 毫秒时间戳
    #[serde(rename = "duration")]
    pub duration: i64, // 耗时（毫秒）
}
