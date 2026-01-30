// 插件工厂
// 根据配置动态创建 Reader 和 Writer 实例

use super::plugin::{ReaderPlugin, WriterPlugin};
use crate::exchange::readers::{elasticsearch::ElasticsearchReader, mysql::MySqlReader};
use crate::exchange::writers::{elasticsearch::ElasticsearchWriter, mysql::MySqlWriter};
use crate::storage::models::{DataSource, DataSourceType};
use anyhow::Result;

/// Reader 工厂
pub struct ReaderFactory;

impl ReaderFactory {
    /// 根据数据源创建 Reader
    pub fn create(datasource: &DataSource, source_name: &str) -> Result<ReaderPlugin> {
        match datasource.source_type {
            DataSourceType::Mysql => {
                let reader = MySqlReader::new(datasource.clone(), source_name.to_string());
                Ok(ReaderPlugin::MySql(reader))
            }
            DataSourceType::Elasticsearch => {
                let reader =
                    ElasticsearchReader::new(datasource.clone(), source_name.to_string());
                Ok(ReaderPlugin::Elasticsearch(reader))
            }
        }
    }
}

/// Writer 工厂
pub struct WriterFactory;

impl WriterFactory {
    /// 根据数据源创建 Writer
    pub fn create(datasource: &DataSource, target_name: &str) -> Result<WriterPlugin> {
        match datasource.source_type {
            DataSourceType::Mysql => {
                let writer = MySqlWriter::new(datasource.clone(), target_name.to_string());
                Ok(WriterPlugin::MySql(writer))
            }
            DataSourceType::Elasticsearch => {
                let writer =
                    ElasticsearchWriter::new(datasource.clone(), target_name.to_string());
                Ok(WriterPlugin::Elasticsearch(writer))
            }
        }
    }
}
