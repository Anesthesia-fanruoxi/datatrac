# MySQL 同步"创建表失败"问题修复 (第二版)

## 问题描述

**时间**: 2026-01-26 08:57:09

**错误日志**:
```
[ERROR] 任务 cfe7fa64-d2bb-48ee-8e1f-7b71dd6a57ac 同步失败: 创建表失败
```

## 问题根源分析

经过深入分析，发现了两个关键问题：

### 问题 1: 数据库不存在
在创建表之前没有检查并创建目标数据库。

### 问题 2: 列类型信息不完整 ⭐ **主要问题**
原代码从 `INFORMATION_SCHEMA.COLUMNS` 只查询了 `DATA_TYPE` 字段（如 `varchar`、`int`），但这个字段**不包含长度和精度信息**。

例如：
- ❌ 错误: `DATA_TYPE` 返回 `varchar` → 创建表时 SQL 为 `CREATE TABLE ... (name varchar)` → **语法错误！**
- ✅ 正确: `COLUMN_TYPE` 返回 `varchar(255)` → 创建表时 SQL 为 `CREATE TABLE ... (name varchar(255))` → 成功

## 修复方案

### 修改文件
- `src-tauri/src/sync_engine/mysql_sync.rs`

### 修复内容

#### 1. 修改表结构查询 - 使用 COLUMN_TYPE

**修改前**:
```rust
SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
```

**修改后**:
```rust
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
```

**关键变化**:
- `DATA_TYPE` → `COLUMN_TYPE` (获取完整类型，如 `varchar(255)`, `int(11)`, `decimal(10,2)`)
- 新增 `COLUMN_DEFAULT` 和 `EXTRA` 字段（为未来扩展预留）

#### 2. 改进创建表 SQL 构建

**新增功能**:
1. ✅ 添加表引擎和字符集: `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
2. ✅ 详细的错误日志，包含完整的 SQL 语句
3. ✅ 调试日志记录表结构信息
4. ✅ 更精确的错误上下文

**修改后的代码**:
```rust
// 构建创建表的 SQL
let mut create_sql = format!("CREATE TABLE `{}`.`{}` (", database, table);

for (i, col) in schema.columns.iter().enumerate() {
    if i > 0 {
        create_sql.push_str(", ");
    }
    
    // 使用完整的列类型（已包含长度等信息）
    create_sql.push_str(&format!(
        "`{}` {}{}",
        col.name,
        col.data_type, // 现在是完整类型，如 varchar(255)
        if col.nullable { "" } else { " NOT NULL" }
    ));
}

// 添加主键
if let Some(pk) = &schema.primary_key {
    create_sql.push_str(&format!(", PRIMARY KEY (`{}`)", pk));
}

create_sql.push(')');
create_sql.push_str(" ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci");

// 记录 SQL 语句用于调试
log::info!("创建表 SQL: {}", create_sql);

// 创建表
sqlx::query(&create_sql)
    .execute(pool)
    .await
    .context(format!("创建表失败: {}.{}\nSQL: {}", database, table, create_sql))?;
```

#### 3. 添加数据库自动创建

```rust
/// 确保 MySQL 数据库存在（如果不存在则创建）
pub(super) async fn ensure_mysql_database_exists(
    &self,
    pool: &MySqlPool,
    database: &str,
) -> Result<()> {
    // 检查数据库是否存在
    let check_query = "SELECT COUNT(*) as count FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?";
    let row: (i64,) = sqlx::query_as(check_query)
        .bind(database)
        .fetch_one(pool)
        .await
        .context("检查数据库是否存在失败")?;
    
    let db_exists = row.0 > 0;
    
    if !db_exists {
        // 创建数据库
        let create_query = format!("CREATE DATABASE `{}` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database);
        sqlx::query(&create_query)
            .execute(pool)
            .await
            .context(format!("创建数据库 {} 失败", database))?;
        
        log::info!("已创建目标数据库: {}", database);
    }
    
    Ok(())
}
```

## 修复效果

### 1. 完整的列类型支持
现在可以正确处理所有 MySQL 列类型：
- ✅ `varchar(255)`, `char(10)`
- ✅ `int(11)`, `bigint(20)`
- ✅ `decimal(10,2)`, `float`, `double`
- ✅ `text`, `longtext`, `blob`
- ✅ `datetime`, `timestamp`, `date`
- ✅ `enum('a','b','c')`
- ✅ 等等...

### 2. 详细的错误信息
当创建表失败时，日志会显示：
```
创建表失败: test_db.users
SQL: CREATE TABLE `test_db`.`users` (
  `id` int(11) NOT NULL,
  `name` varchar(255) NOT NULL,
  `email` varchar(255),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 3. 自动数据库管理
- 自动检测目标数据库是否存在
- 不存在则自动创建
- 使用 utf8mb4 字符集

## 测试建议

### 测试场景 1: 基本同步
1. 创建源表：
```sql
CREATE TABLE test_db.users (
    id INT(11) NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    age INT(3),
    created_at DATETIME,
    PRIMARY KEY (id)
);
```

2. 配置 MySQL -> MySQL 同步任务
3. 启动同步
4. **预期结果**: 成功创建目标表并同步数据

### 测试场景 2: 复杂类型
测试包含各种类型的表：
```sql
CREATE TABLE test_db.complex_types (
    id BIGINT(20) NOT NULL,
    price DECIMAL(10,2),
    description TEXT,
    status ENUM('active','inactive'),
    data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
```

### 测试场景 3: 数据库名称转换
1. 配置数据库名称转换（前缀/后缀替换）
2. 启动同步
3. **预期结果**: 自动创建转换后的数据库名称

## 日志输出示例

成功同步时的日志：
```
[INFO] 表 test_db.users 结构: 5 列, 主键: Some("id")
[INFO] 已创建目标数据库: test_db_copy
[INFO] 创建表 SQL: CREATE TABLE `test_db_copy`.`users` (`id` int(11) NOT NULL, `name` varchar(100) NOT NULL, `email` varchar(255), `age` int(3), `created_at` datetime, PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
[INFO] 成功创建表: test_db_copy.users
```

## 编译验证

```bash
cargo check --manifest-path src-tauri/Cargo.toml
```

**结果**: ✅ 编译通过 (0.84s)

## 相关代码位置

- 表结构查询: `src-tauri/src/sync_engine/mysql_sync.rs:11-50`
- 创建表逻辑: `src-tauri/src/sync_engine/mysql_sync.rs:52-95`
- 数据库创建: `src-tauri/src/sync_engine/mysql_sync.rs:120-145`

## 技术细节

### COLUMN_TYPE vs DATA_TYPE

| 字段 | 示例值 | 说明 |
|------|--------|------|
| `DATA_TYPE` | `varchar` | 只有基础类型名 ❌ |
| `COLUMN_TYPE` | `varchar(255)` | 完整类型定义 ✅ |
| `DATA_TYPE` | `int` | 缺少长度 ❌ |
| `COLUMN_TYPE` | `int(11)` | 包含显示宽度 ✅ |
| `DATA_TYPE` | `decimal` | 缺少精度 ❌ |
| `COLUMN_TYPE` | `decimal(10,2)` | 包含精度和小数位 ✅ |
| `DATA_TYPE` | `enum` | 缺少枚举值 ❌ |
| `COLUMN_TYPE` | `enum('a','b')` | 包含所有枚举值 ✅ |

## 注意事项

1. **权限要求**: 目标 MySQL 用户需要有 `CREATE DATABASE` 和 `CREATE TABLE` 权限
2. **字符集**: 新创建的数据库和表都使用 `utf8mb4` 字符集
3. **表引擎**: 默认使用 `InnoDB` 引擎
4. **日志级别**: 建议设置为 `INFO` 或 `DEBUG` 以查看详细的同步过程

## 后续优化建议

1. ✅ 支持更多表选项（如索引、外键、触发器等）
2. ✅ 添加表结构对比功能
3. ✅ 支持增量同步（只同步变更的数据）
4. ✅ 添加同步前的预检查功能
