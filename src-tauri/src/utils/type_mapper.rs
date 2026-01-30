// 类型映射器
// 负责在 MySQL 和 Elasticsearch 之间进行数据类型转换

pub struct TypeMapper;

impl TypeMapper {
    /// 将 MySQL 数据类型映射到 Elasticsearch 类型
    pub fn mysql_to_es(mysql_type: &str) -> String {
        let normalized = mysql_type.to_uppercase();

        let base_type = if let Some(pos) = normalized.find('(') {
            &normalized[..pos]
        } else {
            &normalized
        };

        match base_type {
            "INT" | "INTEGER" | "TINYINT" | "SMALLINT" | "MEDIUMINT" | "BIGINT" => {
                if normalized.starts_with("TINYINT(1)") {
                    "boolean".to_string()
                } else {
                    "long".to_string()
                }
            }
            "BIT" => "long".to_string(),
            "VARCHAR" | "CHAR" => "keyword".to_string(),
            "TEXT" | "MEDIUMTEXT" | "LONGTEXT" | "TINYTEXT" => "text".to_string(),
            "ENUM" | "SET" => "keyword".to_string(),
            "DATETIME" | "TIMESTAMP" => "date".to_string(),
            "DATE" => "date".to_string(),
            "TIME" => "keyword".to_string(),
            "YEAR" => "keyword".to_string(),
            "BOOLEAN" | "BOOL" => "boolean".to_string(),
            "FLOAT" | "DOUBLE" | "DECIMAL" | "NUMERIC" | "REAL" => "double".to_string(),
            "JSON" => "object".to_string(),
            "BLOB" | "MEDIUMBLOB" | "LONGBLOB" | "TINYBLOB" | "BINARY" | "VARBINARY" => {
                "binary".to_string()
            }
            _ => "keyword".to_string(),
        }
    }

    /// 将 Elasticsearch 数据类型映射到 MySQL 类型
    pub fn es_to_mysql(es_type: &str) -> String {
        match es_type.to_lowercase().as_str() {
            "long" | "integer" | "short" | "byte" => "BIGINT".to_string(),
            "text" => "TEXT".to_string(),
            "keyword" => "VARCHAR(255)".to_string(),
            "date" => "DATETIME".to_string(),
            "boolean" => "BOOLEAN".to_string(),
            "double" | "float" | "half_float" | "scaled_float" => "DOUBLE".to_string(),
            "object" | "nested" => "JSON".to_string(),
            "binary" => "BLOB".to_string(),
            _ => "VARCHAR(255)".to_string(),
        }
    }
}
