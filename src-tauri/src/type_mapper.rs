use serde_json::Value;

/// 类型映射器
/// 负责在 MySQL 和 Elasticsearch 之间进行数据类型转换
pub struct TypeMapper;

impl TypeMapper {
    /// 将 MySQL 数据类型映射到 Elasticsearch 类型
    /// 
    /// # 参数
    /// * `mysql_type` - MySQL 数据类型字符串（如 "INT", "VARCHAR", "TEXT" 等）
    /// 
    /// # 返回
    /// 对应的 Elasticsearch 类型字符串
    /// 
    /// # 映射规则
    /// - INT, TINYINT, SMALLINT, MEDIUMINT, BIGINT -> long
    /// - VARCHAR, CHAR -> keyword
    /// - TEXT, MEDIUMTEXT, LONGTEXT -> text
    /// - DATETIME, TIMESTAMP -> date
    /// - DATE -> date
    /// - TIME -> keyword
    /// - BOOLEAN, BOOL, TINYINT(1) -> boolean
    /// - FLOAT, DOUBLE, DECIMAL -> double
    /// - JSON -> object
    /// - BLOB, BINARY -> binary
    /// - 其他 -> keyword (默认)
    pub fn mysql_to_es(mysql_type: &str) -> String {
        let normalized = mysql_type.to_uppercase();
        
        // 处理带括号的类型，如 VARCHAR(255), TINYINT(1)
        let base_type = if let Some(pos) = normalized.find('(') {
            &normalized[..pos]
        } else {
            &normalized
        };
        
        match base_type {
            // 整数类型
            "INT" | "INTEGER" | "TINYINT" | "SMALLINT" | "MEDIUMINT" | "BIGINT" => {
                // 特殊处理 TINYINT(1) 作为布尔值
                if normalized.starts_with("TINYINT(1)") {
                    "boolean".to_string()
                } else {
                    "long".to_string()
                }
            },
            
            // 字符串类型
            "VARCHAR" | "CHAR" => "keyword".to_string(),
            "TEXT" | "MEDIUMTEXT" | "LONGTEXT" | "TINYTEXT" => "text".to_string(),
            
            // 日期时间类型
            "DATETIME" | "TIMESTAMP" => "date".to_string(),
            "DATE" => "date".to_string(),
            "TIME" => "keyword".to_string(),
            
            // 布尔类型
            "BOOLEAN" | "BOOL" => "boolean".to_string(),
            
            // 浮点数类型
            "FLOAT" | "DOUBLE" | "DECIMAL" | "NUMERIC" | "REAL" => "double".to_string(),
            
            // JSON 类型
            "JSON" => "object".to_string(),
            
            // 二进制类型
            "BLOB" | "MEDIUMBLOB" | "LONGBLOB" | "TINYBLOB" | "BINARY" | "VARBINARY" => {
                "binary".to_string()
            },
            
            // 默认类型
            _ => "keyword".to_string(),
        }
    }
    
    /// 将 Elasticsearch 数据类型映射到 MySQL 类型
    /// 
    /// # 参数
    /// * `es_type` - Elasticsearch 数据类型字符串
    /// 
    /// # 返回
    /// 对应的 MySQL 类型字符串
    /// 
    /// # 映射规则
    /// - long, integer, short, byte -> BIGINT
    /// - text -> TEXT
    /// - keyword -> VARCHAR(255)
    /// - date -> DATETIME
    /// - boolean -> BOOLEAN
    /// - double, float, half_float, scaled_float -> DOUBLE
    /// - object, nested -> JSON
    /// - binary -> BLOB
    /// - 其他 -> VARCHAR(255) (默认)
    pub fn es_to_mysql(es_type: &str) -> String {
        match es_type.to_lowercase().as_str() {
            // 整数类型
            "long" | "integer" | "short" | "byte" => "BIGINT".to_string(),
            
            // 字符串类型
            "text" => "TEXT".to_string(),
            "keyword" => "VARCHAR(255)".to_string(),
            
            // 日期类型
            "date" => "DATETIME".to_string(),
            
            // 布尔类型
            "boolean" => "BOOLEAN".to_string(),
            
            // 浮点数类型
            "double" | "float" | "half_float" | "scaled_float" => "DOUBLE".to_string(),
            
            // 对象类型
            "object" | "nested" => "JSON".to_string(),
            
            // 二进制类型
            "binary" => "BLOB".to_string(),
            
            // 默认类型
            _ => "VARCHAR(255)".to_string(),
        }
    }
    
    /// 将主键值映射为字符串（用于 ES 的 _id 字段）
    /// 
    /// # 参数
    /// * `pk_value` - 主键值（可以是任意 JSON 类型）
    /// 
    /// # 返回
    /// 主键值的字符串表示
    /// 
    /// # 说明
    /// - 如果是字符串，直接返回
    /// - 如果是数字，转换为字符串
    /// - 如果是布尔值，转换为 "true" 或 "false"
    /// - 如果是 null，返回空字符串
    /// - 其他类型使用 JSON 序列化
    pub fn map_primary_key(pk_value: &Value) -> String {
        match pk_value {
            Value::String(s) => s.clone(),
            Value::Number(n) => n.to_string(),
            Value::Bool(b) => b.to_string(),
            Value::Null => String::new(),
            _ => pk_value.to_string(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    
    #[test]
    fn test_mysql_to_es_integer_types() {
        assert_eq!(TypeMapper::mysql_to_es("INT"), "long");
        assert_eq!(TypeMapper::mysql_to_es("BIGINT"), "long");
        assert_eq!(TypeMapper::mysql_to_es("SMALLINT"), "long");
        assert_eq!(TypeMapper::mysql_to_es("TINYINT"), "long");
        assert_eq!(TypeMapper::mysql_to_es("MEDIUMINT"), "long");
    }
    
    #[test]
    fn test_mysql_to_es_string_types() {
        assert_eq!(TypeMapper::mysql_to_es("VARCHAR"), "keyword");
        assert_eq!(TypeMapper::mysql_to_es("VARCHAR(255)"), "keyword");
        assert_eq!(TypeMapper::mysql_to_es("CHAR"), "keyword");
        assert_eq!(TypeMapper::mysql_to_es("TEXT"), "text");
        assert_eq!(TypeMapper::mysql_to_es("MEDIUMTEXT"), "text");
        assert_eq!(TypeMapper::mysql_to_es("LONGTEXT"), "text");
    }
    
    #[test]
    fn test_mysql_to_es_datetime_types() {
        assert_eq!(TypeMapper::mysql_to_es("DATETIME"), "date");
        assert_eq!(TypeMapper::mysql_to_es("TIMESTAMP"), "date");
        assert_eq!(TypeMapper::mysql_to_es("DATE"), "date");
        assert_eq!(TypeMapper::mysql_to_es("TIME"), "keyword");
    }
    
    #[test]
    fn test_mysql_to_es_boolean_types() {
        assert_eq!(TypeMapper::mysql_to_es("BOOLEAN"), "boolean");
        assert_eq!(TypeMapper::mysql_to_es("BOOL"), "boolean");
        assert_eq!(TypeMapper::mysql_to_es("TINYINT(1)"), "boolean");
    }
    
    #[test]
    fn test_mysql_to_es_float_types() {
        assert_eq!(TypeMapper::mysql_to_es("FLOAT"), "double");
        assert_eq!(TypeMapper::mysql_to_es("DOUBLE"), "double");
        assert_eq!(TypeMapper::mysql_to_es("DECIMAL"), "double");
        assert_eq!(TypeMapper::mysql_to_es("DECIMAL(10,2)"), "double");
    }
    
    #[test]
    fn test_mysql_to_es_json_type() {
        assert_eq!(TypeMapper::mysql_to_es("JSON"), "object");
    }
    
    #[test]
    fn test_mysql_to_es_binary_types() {
        assert_eq!(TypeMapper::mysql_to_es("BLOB"), "binary");
        assert_eq!(TypeMapper::mysql_to_es("BINARY"), "binary");
    }
    
    #[test]
    fn test_mysql_to_es_unknown_type() {
        assert_eq!(TypeMapper::mysql_to_es("UNKNOWN"), "keyword");
    }
    
    #[test]
    fn test_es_to_mysql_integer_types() {
        assert_eq!(TypeMapper::es_to_mysql("long"), "BIGINT");
        assert_eq!(TypeMapper::es_to_mysql("integer"), "BIGINT");
        assert_eq!(TypeMapper::es_to_mysql("short"), "BIGINT");
        assert_eq!(TypeMapper::es_to_mysql("byte"), "BIGINT");
    }
    
    #[test]
    fn test_es_to_mysql_string_types() {
        assert_eq!(TypeMapper::es_to_mysql("text"), "TEXT");
        assert_eq!(TypeMapper::es_to_mysql("keyword"), "VARCHAR(255)");
    }
    
    #[test]
    fn test_es_to_mysql_date_type() {
        assert_eq!(TypeMapper::es_to_mysql("date"), "DATETIME");
    }
    
    #[test]
    fn test_es_to_mysql_boolean_type() {
        assert_eq!(TypeMapper::es_to_mysql("boolean"), "BOOLEAN");
    }
    
    #[test]
    fn test_es_to_mysql_float_types() {
        assert_eq!(TypeMapper::es_to_mysql("double"), "DOUBLE");
        assert_eq!(TypeMapper::es_to_mysql("float"), "DOUBLE");
        assert_eq!(TypeMapper::es_to_mysql("half_float"), "DOUBLE");
        assert_eq!(TypeMapper::es_to_mysql("scaled_float"), "DOUBLE");
    }
    
    #[test]
    fn test_es_to_mysql_object_types() {
        assert_eq!(TypeMapper::es_to_mysql("object"), "JSON");
        assert_eq!(TypeMapper::es_to_mysql("nested"), "JSON");
    }
    
    #[test]
    fn test_es_to_mysql_binary_type() {
        assert_eq!(TypeMapper::es_to_mysql("binary"), "BLOB");
    }
    
    #[test]
    fn test_es_to_mysql_unknown_type() {
        assert_eq!(TypeMapper::es_to_mysql("unknown"), "VARCHAR(255)");
    }
    
    #[test]
    fn test_map_primary_key_string() {
        let value = json!("test_id_123");
        assert_eq!(TypeMapper::map_primary_key(&value), "test_id_123");
    }
    
    #[test]
    fn test_map_primary_key_integer() {
        let value = json!(12345);
        assert_eq!(TypeMapper::map_primary_key(&value), "12345");
    }
    
    #[test]
    fn test_map_primary_key_float() {
        let value = json!(123.45);
        assert_eq!(TypeMapper::map_primary_key(&value), "123.45");
    }
    
    #[test]
    fn test_map_primary_key_boolean() {
        let value = json!(true);
        assert_eq!(TypeMapper::map_primary_key(&value), "true");
        
        let value = json!(false);
        assert_eq!(TypeMapper::map_primary_key(&value), "false");
    }
    
    #[test]
    fn test_map_primary_key_null() {
        let value = json!(null);
        assert_eq!(TypeMapper::map_primary_key(&value), "");
    }
}
