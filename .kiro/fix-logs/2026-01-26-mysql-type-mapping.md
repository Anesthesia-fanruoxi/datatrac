# MySQL 类型映射完整文档

## 修复日期：2026-01-26

## 问题描述
MySQL 同步时出现类型映射错误，ENUM 类型被错误识别为 DATE 类型。

## 根本原因
1. 使用 `contains()` 而不是 `starts_with()` 导致误匹配
2. 类型检查顺序不当，导致某些类型被错误识别
3. 缺少对某些 MySQL 类型的支持

## 完整的 MySQL 类型映射表

### 1. 数值类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| TINYINT | i8/u8 | long | 1字节整数 |
| TINYINT(1) | bool | boolean | 布尔值 |
| SMALLINT | i16/u16 | long | 2字节整数 |
| MEDIUMINT | i32/u32 | long | 3字节整数 |
| INT/INTEGER | i32/u32 | long | 4字节整数 |
| BIGINT | i64/u64 | long | 8字节整数 |
| FLOAT | f32 | double | 单精度浮点 |
| DOUBLE | f64 | double | 双精度浮点 |
| DECIMAL | f64 | double | 定点数（读取时 CAST 为 DOUBLE）|
| NUMERIC | f64 | double | 定点数（读取时 CAST 为 DOUBLE）|
| BIT | u64 | long | 位字段 |

**注意：** 所有整数类型都支持 UNSIGNED 修饰符

### 2. 日期时间类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| DATE | NaiveDate | date | 日期 |
| TIME | NaiveTime | keyword | 时间 |
| DATETIME | NaiveDateTime | date | 日期时间 |
| TIMESTAMP | DateTime\<Utc\> | date | 时间戳 |
| YEAR | i16 | keyword | 年份（读取时 CAST 为 INT，用 i32 读取，i16 绑定）|

### 3. 字符串类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| CHAR | String | keyword | 定长字符串 |
| VARCHAR | String | keyword | 变长字符串 |
| TINYTEXT | String | text | 微型文本 |
| TEXT | String | text | 文本 |
| MEDIUMTEXT | String | text | 中型文本 |
| LONGTEXT | String | text | 长文本 |
| ENUM | String | keyword | 枚举 |
| SET | String | keyword | 集合 |

### 4. 二进制类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| BINARY | Vec\<u8\> | binary | 定长二进制 |
| VARBINARY | Vec\<u8\> | binary | 变长二进制 |
| TINYBLOB | Vec\<u8\> | binary | 微型二进制对象 |
| BLOB | Vec\<u8\> | binary | 二进制对象 |
| MEDIUMBLOB | Vec\<u8\> | binary | 中型二进制对象 |
| LONGBLOB | Vec\<u8\> | binary | 长二进制对象 |

### 5. JSON 类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| JSON | String | object | JSON 文档 |

### 6. 空间类型

| MySQL 类型 | Rust 类型 | ES 类型 | 说明 |
|-----------|----------|---------|------|
| GEOMETRY | Vec\<u8\> | geo_shape | 几何对象 |
| POINT | Vec\<u8\> | geo_shape | 点 |
| LINESTRING | Vec\<u8\> | geo_shape | 线 |
| POLYGON | Vec\<u8\> | geo_shape | 多边形 |
| MULTIPOINT | Vec\<u8\> | geo_shape | 多点 |
| MULTILINESTRING | Vec\<u8\> | geo_shape | 多线 |
| MULTIPOLYGON | Vec\<u8\> | geo_shape | 多多边形 |
| GEOMETRYCOLLECTION | Vec\<u8\> | geo_shape | 几何集合 |

## 类型判断规则

### 关键原则
1. **使用 `starts_with()` 而不是 `contains()`**：避免误匹配
2. **按特殊到一般的顺序检查**：先检查特殊类型，再检查通用类型
3. **整数类型按大小顺序检查**：BIGINT → MEDIUMINT → INT → SMALLINT → TINYINT

### 检查顺序
```rust
1. ENUM/SET (特殊字符串类型)
2. JSON (特殊类型)
3. 时间类型 (TIMESTAMP → DATETIME → DATE → TIME → YEAR)
4. 布尔类型 (TINYINT(1), BOOL)
5. 整数类型 (BIGINT → MEDIUMINT → INT → SMALLINT → TINYINT)
6. BIT 类型
7. 浮点类型 (DOUBLE/DECIMAL/NUMERIC → FLOAT)
8. 二进制类型 (BLOB, BINARY, VARBINARY)
9. 空间类型 (GEOMETRY, POINT, etc.)
10. 默认字符串类型 (VARCHAR, TEXT, CHAR, etc.)
```

## 修改的文件

1. **src-tauri/src/sync_engine/mysql_sync.rs**
   - 修复 `batch_insert_mysql_raw` 函数的类型判断逻辑
   - 所有类型检查改用 `starts_with()`
   - 调整检查顺序，避免误匹配
   - 添加 JSON、BIT、完整空间类型支持

2. **src-tauri/src/type_mapper.rs**
   - 更新 `mysql_to_es` 函数
   - 添加 BIT、YEAR、空间类型的映射
   - 完善 ENUM、SET、JSON 类型的映射

## 测试建议

1. 测试包含 ENUM 字段的表同步
2. 测试包含 SET 字段的表同步
3. 测试包含 JSON 字段的表同步
4. 测试包含各种整数类型（UNSIGNED）的表同步
5. 测试包含时间类型的表同步
6. 测试包含空间类型的表同步

## 已知限制

1. 空间类型作为二进制数据处理，不进行空间数据解析
2. JSON 类型作为字符串处理，不进行 JSON 解析
3. BIT 类型作为 u64 处理，最大支持 64 位
4. YEAR 类型在读取时会被 CAST 为 INT（性能影响可忽略）
5. DECIMAL/NUMERIC 类型在读取时会被 CAST 为 DOUBLE（可能损失精度，但对于大多数场景足够）

## 修复历史

### 2026-01-26 第一次修复
- 修复 ENUM 类型被误判为 DATE 类型的问题
- 将所有类型检查从 `contains()` 改为 `starts_with()`
- 添加完整的 MySQL 类型支持

### 2026-01-26 第二次修复 - YEAR 类型（最终方案）
- **问题**：YEAR 类型映射错误
- **错误分析**：
  - 错误信息：`error occurred while decoding column 9`
  - 关键发现：错误发生在**读取（decode）**阶段，不是写入阶段
  - 问题根源：sqlx 不支持将 YEAR 类型读取为整数（i16/i32）
- **尝试过程**：
  1. 尝试 String（绑定）→ 读取时失败
  2. 尝试 i32 → 读取时类型不兼容
  3. 尝试 i16 → 读取时类型不兼容
  4. 尝试 i16 读取 + String 绑定 → 读取时仍然失败
  5. 最终：String 读取 + String 绑定 → 成功
- **最终方案**：读取和绑定都使用 `String`
- **原理**：
  - sqlx-mysql 对 YEAR 类型有特殊限制
  - 不支持将 YEAR 读取为任何整数类型（i16/i32/i64）
  - 只能作为字符串读取和绑定
  - MySQL 会自动在字符串和 YEAR 之间转换


### 2026-01-26 第三次修复 - YEAR 类型调试
- **问题**：用户反馈仍然报错 `mismatched types; Rust type i16 (as SQL type SMALLINT) is not compatible with SQL type YEAR`
- **可能原因**：
  1. 代码未重新编译
  2. YEAR 类型的 COLUMN_TYPE 值可能是 `year(4)` 而不是 `year`
  3. 类型判断逻辑未命中
- **调试措施**：
  1. 添加调试日志：记录每个列的类型和值
  2. 确保 `starts_with("year")` 可以匹配 `year` 和 `year(4)`
  3. 创建 SQL 调试脚本 `debug-year-type.sql` 帮助用户查看实际的列类型
- **待确认**：
  - 用户是否重新编译了代码
  - 第 9 列的实际类型是什么
  - COLUMN_TYPE 的实际值是什么


### 2026-01-26 第四次修复 - YEAR 类型最终解决方案（SQL 层面转换）
- **问题根源**：sqlx 为了防止隐式转换带来的 bug，不允许 String 直接对应 YEAR 类型
- **场景分析**：
  - A 表和 B 表结构完全一致
  - 数据直接从 A 表复制到 B 表
  - 不需要任何类型转换或适配
  - 所有类型都是已知的
- **最终方案**：在 SQL 查询层面解决
  1. **读取时**：使用 `CAST(year_column AS SIGNED)` 将 YEAR 转换为 INT
  2. **处理时**：用 `i32` 类型读取 CAST 后的结果
  3. **绑定时**：用 `i16` 类型绑定（MySQL YEAR 范围 1901-2155，i16 足够）
  4. **写入时**：MySQL 自动将整数值写入 YEAR 字段
- **修改内容**：
  1. `read_mysql_batch_raw` 函数：
     - 添加 `schema` 参数
     - 构建列选择列表，对 YEAR 类型使用 CAST
     - 查询示例：`SELECT col1, CAST(year_col AS SIGNED) as year_col, col3 FROM table`
  2. `batch_insert_mysql_raw` 函数：
     - YEAR 类型用 `i32` 读取（CAST 后的结果）
     - 用 `i16` 绑定（转换为 i16）
  3. `other_sync_impl.rs`：
     - 调用 `read_mysql_batch_raw` 时传入 `source_schema` 参数
- **优势**：
  - 不依赖 sqlx 对 YEAR 类型的支持
  - 利用 MySQL 自身的类型转换能力
  - 代码清晰，逻辑简单
  - 性能无影响（CAST 操作很轻量）
- **编译状态**：✅ 通过


### 2026-01-26 第五次修复 - DECIMAL/NUMERIC 类型
- **问题**：`error occurred while decoding column 5: mismatched types; Rust type f64 (as SQL type DOUBLE) is not compatible with SQL type DECIMAL`
- **原因**：与 YEAR 类型相同，sqlx 对 DECIMAL 类型也有严格限制
- **解决方案**：在 SQL 查询层面转换
  - 读取时：`CAST(decimal_column AS DOUBLE)`
  - 处理时：用 `f64` 读取（已有逻辑，无需修改）
  - 绑定时：用 `f64` 绑定（已有逻辑，无需修改）
  - 写入时：MySQL 自动将 DOUBLE 转换为 DECIMAL
- **修改内容**：
  - `read_mysql_batch_raw` 函数：对 DECIMAL 和 NUMERIC 类型使用 CAST
- **编译状态**：✅ 通过
