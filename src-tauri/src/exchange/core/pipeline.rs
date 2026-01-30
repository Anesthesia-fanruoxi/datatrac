// 同步管道实现
// 负责组装 Reader、Transformer、Writer 并驱动数据流转

use super::data_record::DataRecord;
use super::plugin::{ReaderPlugin, WriterPlugin};
use super::transformer::Transformer;
use anyhow::Result;
use std::sync::Arc;
use tokio::sync::Mutex;

/// 同步管道
/// 管道是底座的运行实体，负责组装并运行 Reader、Transformer、Writer
pub struct SyncPipeline {
    reader: Arc<Mutex<ReaderPlugin>>,
    writer: Arc<Mutex<WriterPlugin>>,
    transformers: Vec<Box<dyn Transformer>>,
    batch_size: usize,
}

impl SyncPipeline {
    /// 创建新的同步管道
    pub fn new(reader: ReaderPlugin, writer: WriterPlugin, batch_size: usize) -> Self {
        Self {
            reader: Arc::new(Mutex::new(reader)),
            writer: Arc::new(Mutex::new(writer)),
            transformers: Vec::new(),
            batch_size,
        }
    }

    /// 添加转换器
    pub fn add_transformer(&mut self, transformer: Box<dyn Transformer>) {
        self.transformers.push(transformer);
    }

    /// 初始化管道
    pub async fn open(&self) -> Result<()> {
        // 初始化 Reader
        let mut reader = self.reader.lock().await;
        reader.open().await?;

        // 初始化 Writer
        let mut writer = self.writer.lock().await;
        writer.open().await?;

        Ok(())
    }

    /// 准备目标环境
    pub async fn prepare(&self) -> Result<()> {
        let reader = self.reader.lock().await;
        let schema = reader.get_schema().await?;

        let mut writer = self.writer.lock().await;
        writer.prepare_target(&schema).await?;

        Ok(())
    }

    /// 执行同步
    /// 返回处理的总记录数
    pub async fn execute<F>(&self, mut progress_callback: F) -> Result<u64>
    where
        F: FnMut(u64, u64) + Send,
    {
        let mut total_processed = 0u64;

        // 获取总记录数
        let mut reader = self.reader.lock().await;
        let total_count = reader.get_total_count().await?;
        drop(reader);

        // 循环读取并写入数据
        loop {
            // 读取一批数据
            let mut reader = self.reader.lock().await;
            let batch = reader.read_batch(self.batch_size).await?;
            let has_next = reader.has_next();
            drop(reader);

            if batch.is_empty() {
                break;
            }

            // 应用转换器
            let transformed_batch = self.apply_transformers(batch)?;

            // 写入数据
            let batch_len = transformed_batch.len() as u64;
            let mut writer = self.writer.lock().await;
            writer.write_batch(transformed_batch).await?;
            writer.commit().await?;
            drop(writer);

            // 更新进度
            total_processed += batch_len;
            progress_callback(total_processed, total_count);

            if !has_next {
                break;
            }
        }

        Ok(total_processed)
    }

    /// 应用所有转换器
    fn apply_transformers(&self, batch: Vec<DataRecord>) -> Result<Vec<DataRecord>> {
        let mut result = batch;

        for transformer in &self.transformers {
            result = transformer.transform_batch(result)?;
        }

        Ok(result)
    }

    /// 关闭管道
    pub async fn close(&self) -> Result<()> {
        let mut reader = self.reader.lock().await;
        reader.close().await?;

        let mut writer = self.writer.lock().await;
        writer.close().await?;

        Ok(())
    }
}
