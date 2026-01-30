// 转换器接口
// 定义如何在读写之间转换数据

use super::data_record::DataRecord;
use anyhow::Result;

/// 转换器 Trait
/// 用于在读写之间处理数据（字段改名、类型转换、脱敏等）
pub trait Transformer: Send + Sync {
    /// 转换单条记录
    fn transform(&self, record: DataRecord) -> Result<DataRecord>;

    /// 批量转换记录
    fn transform_batch(&self, batch: Vec<DataRecord>) -> Result<Vec<DataRecord>> {
        batch.into_iter().map(|r| self.transform(r)).collect()
    }
}

/// 默认转换器
/// 不做任何转换，直接透传
pub struct DefaultTransformer;

impl Transformer for DefaultTransformer {
    fn transform(&self, record: DataRecord) -> Result<DataRecord> {
        Ok(record)
    }
}
