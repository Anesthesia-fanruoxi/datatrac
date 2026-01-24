// ProgressMonitor 使用示例
// 
// 这个示例展示了如何使用 ProgressMonitor 来跟踪任务进度

use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;

// 注意：这个示例需要在实际的 Tauri 应用中运行
// 因为它需要 AppHandle 来发送事件

#[tokio::main]
async fn main() {
    // 创建进度监控器
    let monitor = Arc::new(app_lib::progress::ProgressMonitor::new());
    
    // 在实际应用中，你需要设置 AppHandle：
    // monitor.set_app_handle(app_handle);
    
    // 启动任务
    let task_id = "example-task";
    let total_records = 1000;
    monitor.start_task(task_id, total_records);
    
    println!("任务已启动，总记录数: {}", total_records);
    
    // 模拟处理进度
    for i in 1..=10 {
        sleep(Duration::from_millis(500)).await;
        
        let processed = i * 100;
        monitor.update_progress(task_id, processed);
        
        if let Some(progress) = monitor.get_progress(task_id) {
            println!(
                "进度: {}% ({}/{}), 速度: {:.2} 记录/秒, 预计剩余: {} 秒",
                progress.percentage,
                progress.processed_records,
                progress.total_records,
                progress.speed,
                progress.estimated_time
            );
        }
        
        // 模拟暂停和恢复
        if i == 5 {
            monitor.pause_task(task_id);
            println!("任务已暂停");
            sleep(Duration::from_secs(1)).await;
            monitor.resume_task(task_id);
            println!("任务已恢复");
        }
    }
    
    // 完成任务
    monitor.complete_task(task_id);
    println!("任务已完成！");
    
    if let Some(progress) = monitor.get_progress(task_id) {
        println!("最终状态: {:?}", progress.status);
    }
}
