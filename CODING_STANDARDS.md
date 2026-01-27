# 数据同步工具 - 编码规范

## 命名规范

### 前端 (TypeScript/JavaScript)

1. **变量和函数名**：使用驼峰命名法 (camelCase)
   ```typescript
   const userName = 'John';
   function getUserInfo() { }
   ```

2. **类型和接口名**：使用帕斯卡命名法 (PascalCase)
   ```typescript
   interface UserInfo { }
   type DataSourceType = 'mysql' | 'elasticsearch';
   ```

3. **常量**：使用大写下划线命名法 (UPPER_SNAKE_CASE)
   ```typescript
   const MAX_RETRY_COUNT = 3;
   const API_BASE_URL = 'http://localhost';
   ```

4. **文件名**：使用驼峰命名法
   ```
   userService.ts
   dataSourceStore.ts
   ```

5. **组件名**：使用帕斯卡命名法
   ```
   UserProfile.vue
   DataSourceList.vue
   ```

### 后端 (Rust)

1. **变量和函数名**：使用蛇形命名法 (snake_case)
   ```rust
   let user_name = "John";
   fn get_user_info() { }
   ```

2. **结构体和枚举名**：使用帕斯卡命名法 (PascalCase)
   ```rust
   struct UserInfo { }
   enum DataSourceType { }
   ```

3. **常量**：使用大写下划线命名法 (UPPER_SNAKE_CASE)
   ```rust
   const MAX_RETRY_COUNT: u32 = 3;
   ```

4. **模块名**：使用蛇形命名法
   ```rust
   mod user_service;
   mod data_source;
   ```

5. **文件名**：使用蛇形命名法
   ```
   user_service.rs
   data_source_manager.rs
   ```

## 前后端数据交互规范

### JSON 字段命名

**统一使用驼峰命名法 (camelCase)**

前端发送和接收的 JSON 数据使用驼峰命名：
```json
{
  "taskId": "123",
  "taskName": "测试任务",
  "sourceId": "456",
  "syncConfig": {
    "threadCount": 4,
    "batchSize": 1000,
    "errorStrategy": "skip"
  }
}
```

后端使用 serde 自动转换：
```rust
#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncTaskConfig {
    pub task_id: String,        // JSON: taskId
    pub task_name: String,       // JSON: taskName
    pub source_id: String,       // JSON: sourceId
    pub sync_config: SyncConfig, // JSON: syncConfig
}
```

### Tauri Command 命名

**使用蛇形命名法 (snake_case)**

```rust
#[tauri::command]
pub async fn create_data_source() { }

#[tauri::command]
pub async fn start_sync() { }
```

前端调用：
```typescript
await invoke('create_data_source', { ... });
await invoke('start_sync', { ... });
```

## 代码组织规范

### 前端

1. **Store 文件**：每个 store 负责一个功能模块
   - `dataSource.ts` - 数据源管理
   - `syncTask.ts` - 同步任务管理
   - `taskMonitor.ts` - 任务监控

2. **组件文件**：单个文件不超过 300 行，超过则拆分
   - 使用子组件拆分复杂组件
   - 提取可复用的逻辑到 composables

3. **类型定义**：集中在 `types/index.ts`

### 后端

1. **模块划分**：按功能模块组织
   - `datasource.rs` - 数据源管理
   - `sync_engine/` - 同步引擎
   - `storage/` - 数据存储
   - `commands.rs` - Tauri 命令

2. **文件大小**：单个文件不超过 500 行，超过则拆分

3. **类型定义**：每个模块的类型定义在对应的 `types.rs`

## Serde 配置规范

### 结构体序列化配置

所有需要与前端交互的结构体都添加 `rename_all = "camelCase"`：

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncTaskConfig {
    pub task_id: String,
    pub task_name: String,
    pub sync_config: SyncConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SyncConfig {
    pub thread_count: usize,
    pub batch_size: usize,
    pub error_strategy: ErrorStrategy,
}
```

### 枚举序列化配置

枚举值使用小写：

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ErrorStrategy {
    Skip,   // JSON: "skip"
    Pause,  // JSON: "pause"
}
```

## 日志规范

### 日志级别

- `error!` - 错误信息，需要立即处理
- `warn!` - 警告信息，可能导致问题
- `info!` - 重要的业务流程信息
- `debug!` - 调试信息
- `trace!` - 详细的追踪信息

### 日志格式

```rust
// 任务开始
log::info!("========================================");
log::info!("开始启动同步任务: {}", task_id);
log::info!("========================================");

// 处理过程
log::info!("处理数据库: {}", database);
log::info!("  处理表: {}", table);
log::info!("    批次 {}-{}: 读取 {} 条记录", offset, offset + count, count);

// 错误处理
log::error!("同步失败: {}", error);
```

## Git 提交规范

### 提交信息格式

```
<type>(<scope>): <subject>

<body>
```

### Type 类型

- `feat` - 新功能
- `fix` - 修复 bug
- `refactor` - 重构代码
- `style` - 代码格式调整
- `docs` - 文档更新
- `test` - 测试相关
- `chore` - 构建/工具相关

### 示例

```
feat(sync): 添加数据库名称转换功能

- 支持前缀和后缀替换
- 在步骤2中配置
- 实时显示转换效果
```

## 错误处理规范

### 前端

使用统一的错误处理工具：
```typescript
import { handleApiError } from '../utils/message';

try {
  await someOperation();
} catch (error) {
  handleApiError(error, '操作失败');
}
```

### 后端

使用 `anyhow::Result` 和 `context`：
```rust
use anyhow::{Context, Result};

pub async fn some_operation() -> Result<()> {
    let data = fetch_data()
        .await
        .context("获取数据失败")?;
    
    Ok(())
}
```

## 测试规范

### 前端

- 组件测试使用 Vitest
- E2E 测试使用 Playwright

### 后端

- 单元测试放在 `tests/` 目录
- 集成测试使用 `#[tokio::test]`

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_sync_task() {
        // 测试代码
    }
}
```

## 文档规范

### 代码注释

- 公共 API 必须有文档注释
- 复杂逻辑需要添加说明注释
- 使用中文注释

```rust
/// 启动同步任务
/// 
/// # 参数
/// - `config`: 同步任务配置
/// 
/// # 返回
/// - `Ok(())`: 同步成功
/// - `Err`: 同步失败
pub async fn start_sync(&self, config: SyncTaskConfig) -> Result<()> {
    // 实现代码
}
```

```typescript
/**
 * 启动同步任务
 * @param taskId 任务ID
 * @throws 任务不存在或启动失败
 */
async function startTask(taskId: string): Promise<void> {
  // 实现代码
}
```
