// Exchange Framework 模块
// 数据交换框架 - 实现跨异构数据源的数据传输

pub mod core;
pub mod readers;
pub mod transformers;
pub mod writers;

// 重新导出核心类型
pub use core::{
    data_record::{DataRecord, FieldValue},
    pipeline::SyncPipeline,
    reader::SourceReader,
    transformer::Transformer,
    writer::TargetWriter,
};
