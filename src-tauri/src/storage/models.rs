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
    pub password: String, // 加密存储
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
