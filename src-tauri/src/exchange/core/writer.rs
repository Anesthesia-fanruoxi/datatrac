// 目标写入器接口
// 定义如何将数据写入目标数据源

use super::data_record::{DataRecord, SchemaInfo};
use anyhow::Result;

/// 目标写入器 Trait
/// 所有数据源的 Writer 都需要实现这个接口
pub trait TargetWriter: Send + Sync {
    /// 初始化连接
    async fn open(&mut self) -> Result<()>;

    /// 根据源 Schema 准备目标表结构/索引 Mapping
    async fn prepare_target(&mut self, schema_info: &SchemaInfo) -> Result<()>;

    /// 写入一个批次的数据
    async fn write_batch(&mut self, batch: Vec<DataRecord>) -> Result<()>;

    /// 提交事务或刷新缓冲区
    async fn commit(&mut self) -> Result<()>;

    /// 释放资源
    async fn close(&mut self) -> Result<()>;
}
