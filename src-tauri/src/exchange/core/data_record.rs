// 数据记录定义
// 定义统一的中间数据格式

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// 字段值类型
/// 支持所有常见的数据类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum FieldValue {
    Null,
    Boolean(bool),
    Integer(i64),
    Float(f64),
    Text(String),
    DateTime(DateTime<Utc>),
    Json(serde_json::Value),
    Binary(Vec<u8>),
}

impl FieldValue {
    /// 判断是否为 Null
    pub fn is_null(&self) -> bool {
        matches!(self, FieldValue::Null)
    }

    /// 转换为字符串（用于日志）
    pub fn to_string(&self) -> String {
        match self {
            FieldValue::Null => "NULL".to_string(),
            FieldValue::Boolean(b) => b.to_string(),
            FieldValue::Integer(i) => i.to_string(),
            FieldValue::Float(f) => f.to_string(),
            FieldValue::Text(s) => s.clone(),
            FieldValue::DateTime(dt) => dt.to_rfc3339(),
            FieldValue::Json(v) => v.to_string(),
            FieldValue::Binary(b) => format!("<binary {} bytes>", b.len()),
        }
    }
}

/// 数据记录
/// 所有数据源在传输过程中都转换为这个统一格式
#[derive(Debug, Clone)]
pub struct DataRecord {
    /// 字段数据
    pub fields: HashMap<String, FieldValue>,
    /// 元数据（如主键 ID、版本号等）
    pub metadata: HashMap<String, String>,
}

impl DataRecord {
    /// 创建新的数据记录
    pub fn new() -> Self {
        Self {
            fields: HashMap::new(),
            metadata: HashMap::new(),
        }
    }

    /// 添加字段
    pub fn add_field(&mut self, name: String, value: FieldValue) {
        self.fields.insert(name, value);
    }

    /// 获取字段
    pub fn get_field(&self, name: &str) -> Option<&FieldValue> {
        self.fields.get(name)
    }

    /// 添加元数据
    pub fn add_metadata(&mut self, key: String, value: String) {
        self.metadata.insert(key, value);
    }

    /// 获取元数据
    pub fn get_metadata(&self, key: &str) -> Option<&String> {
        self.metadata.get(key)
    }

    /// 获取所有字段名
    pub fn field_names(&self) -> Vec<String> {
        self.fields.keys().cloned().collect()
    }
}

impl Default for DataRecord {
    fn default() -> Self {
        Self::new()
    }
}

/// Schema 信息
/// 用于在 Reader 和 Writer 之间传递字段定义
#[derive(Debug, Clone)]
pub struct SchemaInfo {
    /// 字段定义
    pub fields: Vec<FieldInfo>,
    /// 主键字段名
    pub primary_key: Option<String>,
}

/// 字段信息
#[derive(Debug, Clone)]
pub struct FieldInfo {
    /// 字段名
    pub name: String,
    /// 字段类型
    pub field_type: FieldType,
    /// 是否可为空
    pub nullable: bool,
}

/// 字段类型
#[derive(Debug, Clone)]
pub enum FieldType {
    Boolean,
    Integer,
    Float,
    Text,
    DateTime,
    Json,
    Binary,
}
