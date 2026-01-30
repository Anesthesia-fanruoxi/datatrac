// 任务配置相关的数据结构
// 用于解析和处理任务配置 JSON

use super::models::KeywordMapping;
use serde::{Deserialize, Serialize};

/// 同步配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncConfig {
    pub thread_count: usize,
    pub batch_size: usize,
    pub error_strategy: String,
}

/// 任务配置
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskConfig {
    pub units: Vec<String>,
    pub sync_config: SyncConfig,
    
    // 关键字映射（可选）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub keywords: Option<Vec<KeywordMapping>>,
    
    // 是否跳过已同步的索引（默认 true）
    #[serde(default = "default_skip_synced")]
    pub skip_synced: bool,
}

fn default_skip_synced() -> bool {
    true
}
