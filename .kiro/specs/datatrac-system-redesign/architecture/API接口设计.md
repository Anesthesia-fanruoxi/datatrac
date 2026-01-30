# API 接口设计

## 📋 文档说明

本文档定义 DataTrac 系统的所有 API 接口，包括 Tauri Commands 和 Event 推送。

**文档版本**: v1.0  
**最后更新**: 2026-01-30  
**通讯方式**: Tauri IPC

---

## 🎯 接口概览

### 接口分类

| 分类 | 数量 | 说明 |
|------|------|------|
| 数据源管理 | 8 | 增删改查、测试连接、查询数据库/表/索引 |
| 任务管理 | 5 | 增删改查、获取详情 |
| 任务执行 | 4 | 启动、暂停、恢复、停止 |
| 任务监控 | 3 | 获取进度、获取单元、重置失败 |
| **总计** | **20** | - |

### 事件推送

| 事件名 | 说明 |
|--------|------|
| task-progress | 任务进度更新 |
| task-log | 任务日志推送 |
| task-units-update | 任务单元状态更新 |

---

## 📡 数据源管理接口

### 1. 获取数据源列表

**Command**: `list_data_sources`

**请求参数**: 无

**返回数据**:
```typescript
DataSource[]

interface DataSource {
  id: string;
  name: string;
  type: 'mysql' | 'elasticsearch';
  host: string;
  port: number;
  username: string;
  password: string;
  databaseName?: string;
  createdAt: number;
  updatedAt: number;
}
```

**前端调用**:
```typescript
const dataSources = await invoke<DataSource[]>('list_data_sources');
```

---

### 2. 创建数据源

**Command**: `create_data_source`

**请求参数**:
```typescript
{
  data: CreateDataSourceRequest
}

interface CreateDataSourceRequest {
  name: string;
  type: 'mysql' | 'elasticsearch';
  host: string;
  port: number;
  username: string;
  password: string;
  databaseName?: string;
}
```

**返回数据**:
```typescript
DataSource
```

**前端调用**:
```typescript
const newDataSource = await invoke<DataSource>('create_data_source', {
  data: {
    name: 'MySQL 测试',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: '123456',
    databaseName: 'test'
  }
});
```

---

### 3. 更新数据源

**Command**: `update_data_source`

**请求参数**:
```typescript
{
  id: string;
  data: UpdateDataSourceRequest;
}

interface UpdateDataSourceRequest {
  name?: string;
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  databaseName?: string;
}
```

**返回数据**:
```typescript
DataSource
```

**前端调用**:
```typescript
const updated = await invoke<DataSource>('update_data_source', {
  id: 'ds-123',
  data: {
    name: '新名称',
    port: 3307
  }
});
```

---

### 4. 删除数据源

**Command**: `delete_data_source`

**请求参数**:
```typescript
{
  id: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('delete_data_source', { id: 'ds-123' });
```

---

### 5. 测试数据源连接

**Command**: `test_data_source_connection`

**请求参数**:
```typescript
{
  id: string;
}
```

**返回数据**:
```typescript
TestConnectionResponse

interface TestConnectionResponse {
  success: boolean;
  message: string;
  version?: string;
}
```

**前端调用**:
```typescript
const result = await invoke<TestConnectionResponse>(
  'test_data_source_connection',
  { id: 'ds-123' }
);

if (result.success) {
  message.success(`连接成功，版本: ${result.version}`);
} else {
  message.error(`连接失败: ${result.message}`);
}
```

---

### 6. 获取数据库列表 (MySQL)

**Command**: `get_databases`

**请求参数**:
```typescript
{
  dataSourceId: string;
}
```

**返回数据**:
```typescript
string[]  // 数据库名称列表
```

**前端调用**:
```typescript
const databases = await invoke<string[]>('get_databases', {
  dataSourceId: 'ds-123'
});
// ['db1', 'db2', 'db3']
```

---

### 7. 获取表列表 (MySQL)

**Command**: `get_tables`

**请求参数**:
```typescript
{
  dataSourceId: string;
  database: string;
}
```

**返回数据**:
```typescript
string[]  // 表名列表
```

**前端调用**:
```typescript
const tables = await invoke<string[]>('get_tables', {
  dataSourceId: 'ds-123',
  database: 'test_db'
});
// ['table1', 'table2', 'table3']
```

---

### 8. 获取索引列表 (Elasticsearch)

**Command**: `get_indices`

**请求参数**:
```typescript
{
  dataSourceId: string;
}
```

**返回数据**:
```typescript
string[]  // 索引名称列表
```

**前端调用**:
```typescript
const indices = await invoke<string[]>('get_indices', {
  dataSourceId: 'ds-456'
});
// ['index-a', 'index-b', 'index-c']
```

---

## 📋 任务管理接口

### 9. 获取任务列表

**Command**: `list_sync_tasks`

**请求参数**: 无

**返回数据**:
```typescript
SyncTask[]

interface SyncTask {
  id: string;
  name: string;
  sourceId: string;
  targetId: string;
  syncDirection: 'mysql_to_es' | 'es_to_mysql' | 'mysql_to_mysql' | 'es_to_es';
  status: 'idle' | 'running' | 'paused' | 'completed' | 'failed';
  syncConfig: SyncConfig;
  mysqlConfig?: MySQLConfig;
  esConfig?: ESConfig;
  createdAt: number;
  updatedAt: number;
}
```

**前端调用**:
```typescript
const tasks = await invoke<SyncTask[]>('list_sync_tasks');
```

---

### 10. 创建任务

**Command**: `create_sync_task`

**请求参数**:
```typescript
{
  data: CreateSyncTaskRequest;
}

interface CreateSyncTaskRequest {
  name: string;
  sourceId: string;
  targetId: string;
  syncConfig: SyncConfig;
  mysqlConfig?: MySQLConfig;
  esConfig?: ESConfig;
}
```

**返回数据**:
```typescript
SyncTask
```

**前端调用**:
```typescript
const task = await invoke<SyncTask>('create_sync_task', {
  data: {
    name: 'ES 同步任务',
    sourceId: 'ds-source',
    targetId: 'ds-target',
    syncConfig: {
      batchSize: 2500,
      threadCount: 4,
      errorStrategy: 'skip',
      tableExistsStrategy: 'drop'
    },
    esConfig: {
      selectedIndices: ['index-a', 'index-b']
    }
  }
});
```

---

### 11. 更新任务

**Command**: `update_sync_task`

**请求参数**:
```typescript
{
  id: string;
  data: UpdateSyncTaskRequest;
}

interface UpdateSyncTaskRequest {
  name?: string;
  syncConfig?: SyncConfig;
  mysqlConfig?: MySQLConfig;
  esConfig?: ESConfig;
}
```

**返回数据**:
```typescript
SyncTask
```

---

### 12. 删除任务

**Command**: `delete_sync_task`

**请求参数**:
```typescript
{
  id: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('delete_sync_task', { id: 'task-123' });
```

---

### 13. 获取任务详情

**Command**: `get_sync_task`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**:
```typescript
SyncTask
```

**前端调用**:
```typescript
const task = await invoke<SyncTask>('get_sync_task', {
  taskId: 'task-123'
});
```

---

## ▶️ 任务执行接口

### 14. 启动任务

**Command**: `start_sync_task`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('start_sync_task', { taskId: 'task-123' });
```

**说明**: 
- 启动后会初始化任务单元
- 自动模式执行所有未完成单元
- 实时推送进度事件

---

### 15. 暂停任务

**Command**: `pause_sync_task`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('pause_sync_task', { taskId: 'task-123' });
```

**说明**: 
- 设置暂停标志
- 等待当前执行中的单元完成
- 不会立即停止

---

### 16. 恢复任务

**Command**: `resume_sync_task`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('resume_sync_task', { taskId: 'task-123' });
```

**说明**: 
- 清除暂停标志
- 继续执行未完成的单元

---

### 17. 停止任务

**Command**: `stop_sync_task`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**: `void`

**前端调用**:
```typescript
await invoke('stop_sync_task', { taskId: 'task-123' });
```

**说明**: 
- 设置停止标志
- 等待当前执行中的单元完成
- 任务状态变为 idle

---

## 📊 任务监控接口

### 18. 获取任务单元列表

**Command**: `get_task_units`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**:
```typescript
TaskUnit[]

interface TaskUnit {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  errorMessage?: string;
}
```

**前端调用**:
```typescript
const units = await invoke<TaskUnit[]>('get_task_units', {
  taskId: 'task-123'
});
```

---

### 19. 获取任务进度

**Command**: `get_task_progress`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**:
```typescript
TaskProgress

interface TaskProgress {
  taskId: string;
  status: 'idle' | 'running' | 'paused' | 'completed' | 'failed';
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  speed: number;
  estimatedTime: number;
  startTime: string;
  taskUnits: TaskUnit[];
}
```

**前端调用**:
```typescript
const progress = await invoke<TaskProgress>('get_task_progress', {
  taskId: 'task-123'
});
```

---

### 20. 重置失败的任务单元

**Command**: `reset_failed_units`

**请求参数**:
```typescript
{
  taskId: string;
}
```

**返回数据**:
```typescript
number  // 重置的单元数量
```

**前端调用**:
```typescript
const count = await invoke<number>('reset_failed_units', {
  taskId: 'task-123'
});
message.success(`已重置 ${count} 个失败单元`);
```

---

## 📢 事件推送

### 1. 任务进度更新事件

**事件名**: `task-progress`

**推送时机**: 
- 任务单元状态变化
- 进度更新
- 每秒推送一次

**数据格式**:
```typescript
TaskProgress

interface TaskProgress {
  taskId: string;
  status: string;
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  speed: number;
  estimatedTime: number;
  startTime: string;
  taskUnits: TaskUnit[];
}
```

**前端监听**:
```typescript
import { listen } from '@tauri-apps/api/event';

const unlisten = await listen<TaskProgress>('task-progress', (event) => {
  const progress = event.payload;
  console.log('进度更新:', progress);
  // 更新 UI
});

// 组件卸载时取消监听
onUnmounted(() => {
  unlisten();
});
```

---

### 2. 任务日志事件

**事件名**: `task-log`

**推送时机**: 
- 有新日志产生时立即推送

**数据格式**:
```typescript
LogEntry

interface LogEntry {
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  category: 'realtime' | 'summary' | 'verify' | 'error';
  message: string;
}
```

**前端监听**:
```typescript
const unlisten = await listen<LogEntry>('task-log', (event) => {
  const log = event.payload;
  console.log('新日志:', log);
  // 添加到日志列表
});
```

---

### 3. 任务单元状态更新事件

**事件名**: `task-units-update`

**推送时机**: 
- 任务单元状态变化时

**数据格式**:
```typescript
{
  taskId: string;
  units: TaskUnit[];
}
```

**前端监听**:
```typescript
const unlisten = await listen('task-units-update', (event) => {
  const { taskId, units } = event.payload;
  console.log('单元状态更新:', taskId, units);
  // 更新 UI
});
```

---

## 🔧 错误处理

### 错误格式

Tauri 会自动将 Rust 的 `Result::Err` 转换为 JavaScript 异常。

**后端返回错误**:
```rust
#[tauri::command]
pub async fn start_sync_task(task_id: String) -> Result<(), String> {
    if task_id.is_empty() {
        return Err("任务 ID 不能为空".to_string());
    }
    // ...
    Ok(())
}
```

**前端捕获错误**:
```typescript
try {
  await invoke('start_sync_task', { taskId: '' });
} catch (error) {
  // error 是字符串: "任务 ID 不能为空"
  message.error(`启动失败: ${error}`);
}
```

### 常见错误码

| 错误信息 | 说明 | 处理方式 |
|---------|------|---------|
| "任务不存在" | 任务 ID 无效 | 提示用户刷新列表 |
| "任务正在运行" | 重复启动 | 提示用户等待 |
| "数据源不存在" | 数据源 ID 无效 | 提示用户检查配置 |
| "连接失败" | 网络或配置错误 | 提示用户检查连接 |
| "配置表中没有找到任务单元" | 任务配置不完整 | 提示用户重新配置 |

---

## 📚 相关文档

- [技术规范](./技术规范.md) - 命名和数据格式规范
- [前后端通讯](./前后端通讯.md) - 通讯机制详解
- [实施指南](../implementation/实施指南.md) - Commands 实现步骤

---

**文档维护**: DataTrac 开发团队  
**最后更新**: 2026-01-30
