// 源读取器接口
// 定义如何从数据源读取数据

use super::data_record::{DataRecord, SchemaInfo};
use anyhow::Result;

/// 源读取器 Trait
/// 所有数据源的 Reader 都需要实现这个接口
pub trait SourceReader: Send + Sync {
    /// 初始化连接和准备工作
    async fn open(&mut self) -> Result<()>;

    /// 获取 Schema 信息
    async fn get_schema(&self) -> Result<SchemaInfo>;

    /// 获取总记录数（用于进度条）
    async fn get_total_count(&mut self) -> Result<u64>;

    /// 读取一个批次的数据
    /// 返回空 Vec 表示没有更多数据
    async fn read_batch(&mut self, batch_size: usize) -> Result<Vec<DataRecord>>;

    /// 检查是否还有更多数据
    fn has_next(&self) -> bool;

    /// 释放资源
    async fn close(&mut self) -> Result<()>;
}
