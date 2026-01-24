// ErrorLogger 使用示例
use app_lib::error_logger::{ErrorLogger, ErrorLog};
use serde_json::json;

fn main() {
    println!("=== ErrorLogger 使用示例 ===\n");

    // 1. 创建 ErrorLogger 实例
    let logger = ErrorLogger::new();
    println!("✓ 创建 ErrorLogger 实例");

    // 2. 记录不同类型的错误
    let task_id = "task-001";

    // 连接错误
    let error1 = ErrorLog::new(
        "ConnectionError".to_string(),
        "无法连接到 MySQL 数据库".to_string(),
        Some(json!({
            "host": "localhost",
            "port": 3306,
            "database": "test_db"
        }))
    );
    logger.log_error(task_id, error1);
    println!("✓ 记录连接错误");

    // 数据转换错误
    let error2 = ErrorLog::new(
        "DataConversionError".to_string(),
        "无法将 VARCHAR 转换为 INTEGER".to_string(),
        Some(json!({
            "table": "users",
            "column": "age",
            "value": "invalid"
        }))
    );
    logger.log_error(task_id, error2);
    println!("✓ 记录数据转换错误");

    // 批量插入错误
    let error3 = ErrorLog::new(
        "BatchInsertError".to_string(),
        "批量插入失败：主键冲突".to_string(),
        Some(json!({
            "batch_size": 1000,
            "failed_records": 5
        }))
    );
    logger.log_error(task_id, error3);
    println!("✓ 记录批量插入错误");

    // 3. 获取错误列表
    let errors = logger.get_errors(task_id);
    println!("\n获取到 {} 条错误日志:", errors.len());
    for (i, error) in errors.iter().enumerate() {
        println!("  {}. [{}] {} - {}", 
            i + 1, 
            error.error_type, 
            error.timestamp.format("%Y-%m-%d %H:%M:%S"),
            error.message
        );
        if let Some(data) = &error.data {
            println!("     数据: {}", data);
        }
    }

    // 4. 获取错误数量
    let count = logger.get_error_count(task_id);
    println!("\n错误总数: {}", count);

    // 5. 清空错误日志
    logger.clear_errors(task_id);
    println!("✓ 清空错误日志");

    let count_after_clear = logger.get_error_count(task_id);
    println!("清空后错误数量: {}", count_after_clear);

    // 6. 多任务测试
    println!("\n=== 多任务测试 ===");
    let task2_id = "task-002";
    let error4 = ErrorLog::new(
        "TimeoutError".to_string(),
        "查询超时".to_string(),
        None
    );
    logger.log_error(task2_id, error4);

    println!("任务 {} 错误数: {}", task_id, logger.get_error_count(task_id));
    println!("任务 {} 错误数: {}", task2_id, logger.get_error_count(task2_id));

    println!("\n✓ 所有测试完成！");
}
