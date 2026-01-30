// 插件枚举
// 使用枚举代替 trait object，避免 dyn-compatibility 问题

use super::data_record::{DataRecord, SchemaInfo};
use crate::exchange::readers::{elasticsearch::ElasticsearchReader, mysql::MySqlReader};
use crate::exchange::writers::{elasticsearch::ElasticsearchWriter, mysql::MySqlWriter};
use anyhow::Result;

/// Reader 枚举
/// 包含所有支持的 Reader 实现
pub enum ReaderPlugin {
    MySql(MySqlReader),
    Elasticsearch(ElasticsearchReader),
}

impl ReaderPlugin {
    pub async fn open(&mut self) -> Result<()> {
        match self {
            ReaderPlugin::MySql(r) => r.open().await,
            ReaderPlugin::Elasticsearch(r) => r.open().await,
        }
    }

    pub async fn get_schema(&self) -> Result<SchemaInfo> {
        match self {
            ReaderPlugin::MySql(r) => r.get_schema().await,
            ReaderPlugin::Elasticsearch(r) => r.get_schema().await,
        }
    }

    pub async fn get_total_count(&mut self) -> Result<u64> {
        match self {
            ReaderPlugin::MySql(r) => r.get_total_count().await,
            ReaderPlugin::Elasticsearch(r) => r.get_total_count().await,
        }
    }

    pub async fn read_batch(&mut self, batch_size: usize) -> Result<Vec<DataRecord>> {
        match self {
            ReaderPlugin::MySql(r) => r.read_batch(batch_size).await,
            ReaderPlugin::Elasticsearch(r) => r.read_batch(batch_size).await,
        }
    }

    pub fn has_next(&self) -> bool {
        match self {
            ReaderPlugin::MySql(r) => r.has_next(),
            ReaderPlugin::Elasticsearch(r) => r.has_next(),
        }
    }

    pub async fn close(&mut self) -> Result<()> {
        match self {
            ReaderPlugin::MySql(r) => r.close().await,
            ReaderPlugin::Elasticsearch(r) => r.close().await,
        }
    }
}

/// Writer 枚举
/// 包含所有支持的 Writer 实现
pub enum WriterPlugin {
    MySql(MySqlWriter),
    Elasticsearch(ElasticsearchWriter),
}

impl WriterPlugin {
    pub async fn open(&mut self) -> Result<()> {
        match self {
            WriterPlugin::MySql(w) => w.open().await,
            WriterPlugin::Elasticsearch(w) => w.open().await,
        }
    }

    pub async fn prepare_target(&mut self, schema_info: &SchemaInfo) -> Result<()> {
        match self {
            WriterPlugin::MySql(w) => w.prepare_target(schema_info).await,
            WriterPlugin::Elasticsearch(w) => w.prepare_target(schema_info).await,
        }
    }

    pub async fn write_batch(&mut self, batch: Vec<DataRecord>) -> Result<()> {
        match self {
            WriterPlugin::MySql(w) => w.write_batch(batch).await,
            WriterPlugin::Elasticsearch(w) => w.write_batch(batch).await,
        }
    }

    pub async fn commit(&mut self) -> Result<()> {
        match self {
            WriterPlugin::MySql(w) => w.commit().await,
            WriterPlugin::Elasticsearch(w) => w.commit().await,
        }
    }

    pub async fn close(&mut self) -> Result<()> {
        match self {
            WriterPlugin::MySql(w) => w.close().await,
            WriterPlugin::Elasticsearch(w) => w.close().await,
        }
    }
}
