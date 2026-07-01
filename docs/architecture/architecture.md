# DataTrace 架构设计文档

## 1. 整体架构

```
┌──────────────────────────────────────────────────┐
│                  Frontend (Vue + TS)              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────┐ │
│  │ 任务监控 │ │ 任务配置 │ │ 数据源   │ │凭据 │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──┬──┘ │
│       │            │            │           │     │
│       └────────────┼────────────┼───────────┘     │
│                    │ REST API / SSE               │
└────────────────────┼──────────────────────────────┘
                     │
┌────────────────────┼──────────────────────────────┐
│            Backend (Go + Gin)                      │
│                    │                               │
│  ┌─────────────────┼─────────────────────────┐    │
│  │         API Layer                         │    │
│  │  task_api / task_control_api / sse_api    │    │
│  └─────────────────┬─────────────────────────┘    │
│                    │                               │
│  ┌─────────────────┼─────────────────────────┐    │
│  │         Service Layer                     │    │
│  │  ┌───────────┐  ┌──────────────┐          │    │
│  │  │ 全量同步  │  │ 增量同步     │          │    │
│  │  │ (一次性)  │  │ (持续运行)   │          │    │
│  │  └─────┬─────┘  └──────┬───────┘          │    │
│  │        │               │                  │    │
│  │  ┌─────┴─────┐  ┌──────┴───────┐          │    │
│  │  │ SyncEngine│  │ BinlogListener│          │    │
│  │  │ Reader    │  │ BinlogQueue   │          │    │
│  │  │ Writer    │  │ Consumer      │          │    │
│  │  └───────────┘  └──────────────┘          │    │
│  └────────────────────────────────────────────┘    │
│                    │                               │
│  ┌─────────────────┼─────────────────────────┐    │
│  │      Infrastructure Layer                 │    │
│  │  MySQL / Redis / File System              │    │
│  └───────────────────────────────────────────┘    │
└───────────────────────────────────────────────────┘
```

---

## 2. 核心组件

### 2.1 组件清单

| 组件 | 文件 | 职责 |
|---|---|---|
| **TaskControlService** | `task_control_service.go` | 任务启停控制 |
| **SyncEngine** | `sync_engine.go` | 全量同步核心引擎 |
| **IncrementalSync** | `incremental_sync.go` | 增量同步总控 |
| **BinlogListener** | `binlog_listener.go` | Binlog 事件监听 |
| **BinlogQueue** | `binlog_queue.go` | 事件队列（内存/Redis） |
| **IncrementalConsumer** | `incremental_consumer.go` | 事件消费 → 写入目标 |
| **TaskProgressManager** | `task_progress_manager.go` | 内存进度管理 |
| **TaskSSEService** | `task_sse_service.go` | SSE 推送 |
| **ConfigCacheService** | `config_cache_service.go` | Redis 配置缓存 |

### 2.2 组件交互关系

```
TaskControlService
  │
  ├── startFullSyncTask()
  │     ├── ConfigCacheService → 获取配置
  │     ├── TaskProgressManager → 初始化进度
  │     └── SyncEngine.Worker() × N 并发
  │           ├── MySQLReader → 分批读取
  │           └── MySQLWriter → 批量写入
  │
  └── startIncrementalTask()
        └── IncrementalSync.Start()
              ├── initDatabaseConnections()
              ├── checkBinlogConfig()
              ├── captureSnapshot()
              ├── createQueue()
              ├── startBinlogListener() → 异步监听
              ├── executeFullSync() → SyncEngine
              ├── markFullSyncCompleted()
              └── startIncrementalConsumer() → 持续消费
                    └── ← ctx.Done() 阻塞
```

---

## 3. 关键设计决策

### 3.1 增量同步中的表变更

**方案：运行时配置热更新 + 事件队列过滤**

```
用户操作（API 更新配置）
  │
  ├── 1. 更新 Redis 配置缓存
  ├── 2. 如果是新增表：
  │     ├── 对新表执行全量同步
  │     └── 将新表加入监听过滤列表
  ├── 3. 如果是删除表：
  │     └── 从监听过滤列表中移除
  └── 4. BinlogListener 在下一个周期刷新过滤列表
```

**BinlogListener 需要的改造**：
- 当前：过滤列表在启动时固定
- 目标：支持运行时动态更新过滤列表（线程安全的 map + 定期刷新）

**Consumer 需要的改造**：
- 当前：表映射在启动时固定
- 目标：每次消费事件时，从 Redis/内存读取最新的映射配置

### 3.2 全量同步复用

增量模式下的全量同步阶段，复用 `SyncEngine`，但需要：

1. 增量模式只同步**新表**（已同步过的表跳过）
2. 通过 Redis 记录每张表的全量同步状态
3. 增量同步阶段的全量同步结果上报到 SSE

### 3.3 队列选择

| 队列类型 | 优势 | 劣势 | 适用场景 |
|---|---|---|---|
| **内存队列** | 零延迟 | 进程重启丢数据 | 开发/测试 |
| **Redis 队列** | 持久化 | 网络开销 | 生产环境 |

当前支持 `memory` / `redis` 两种，通过 `queue_type` 配置。

### 3.4 多目标源

当前系统已支持一个源 → 多个目标。增量模式下：

- 每个目标独立维护一套 Binlog 消费
- 每个目标独立上报进度
- SSE 按 `target_id` 过滤推送

---

## 4. 数据模型

### 4.1 SyncTask

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | 主键 |
| name | string | 任务名 |
| source_id | string | 源数据源 |
| target_id | string | 主目标源（兼容） |
| config | text(JSON) | 完整配置 |
| status | string | idle / configured / running / completed |
| is_running | bool | 是否运行中 |
| sync_mode | string | full / incremental |
| current_step | string | 当前步骤 |

### 4.2 TaskConfig（JSON in config 字段）

```json
{
  "source_id": "...",
  "target_id": "...",
  "target_ids": ["...", "..."],
  "selected_databases": [
    {
      "database": "target_db",
      "source_database": "source_db",
      "is_database_modified": false,
      "tables": [
        {
          "source_table": "users",
          "target_table": "users",
          "is_modified": false,
          "selected_fields": ["id", "name", "email"]
        }
      ]
    }
  ],
  "sync_config": {
    "sync_mode": "full",
    "error_strategy": "skip",
    "table_exists_strategy": "truncate",
    "sync_structure_only": false
  }
}
```

---

## 5. Redis 数据结构

| Key 模式 | 说明 | 过期 |
|---|---|---|
| `config:task:{taskID}` | 任务配置缓存 | 不过期 |
| `incremental:task:{taskID}:target:{targetID}:tables` | 表列表 Set | 不过期 |
| `incremental:task:{taskID}:target:{targetID}:table:{db}.{table}` | 单表统计 JSON | 不过期 |
| `progress:task:{taskID}:units` | 全量同步进度 Units | 任务结束后清除 |

---

## 6. SSE 推送机制

### 6.1 推送架构

```
业务层广播                SSE 服务层                    前端
  │                         │                           │
  ├── BroadcastProgress ──→ ├── 按 targetID 分发 ──→ ── │ EventSource
  ├── BroadcastDetail ────→ ├── 按 taskID 广播 ────→ ── │ EventSource
  └── BroadcastLog ───────→ ├── 按 category 分发 ──→ ── │ EventSource
```

### 6.2 推送原则

- **只推送增量变化**，不做轮询
- **按客户端参数过滤**（`target_id`、`database`、`category`）
- **新连接立即推送缓存快照**，确保页面刷新后能立刻看到数据

---

## 7. 部署架构

```
┌──────────────┐     ┌──────────────┐
│  Vue Frontend│     │  Go Backend  │
│  :3000       │────→│  :8090       │
└──────────────┘     └──────┬───────┘
                            │
                     ┌──────┴───────┐
                     │   MySQL      │
                     │  (数据同步)  │
                     └──────────────┘
                            │
                     ┌──────┴───────┐
                     │   Redis      │
                     │  (配置/队列) │
                     └──────────────┘
```

### 构建产物

```
datatrace/
├── datatrace-server/
│   ├── main.go
│   ├── go.mod / go.sum
│   ├── config.yaml
│   └── ...（Go 后端代码）
└── datatrace-ui/
    ├── src/
    ├── package.json
    └── ...（Vue + TS 前端代码）
```

### 启动方式

```bash
# 后端
cd datatrace-server && go run main.go

# 前端
cd datatrace-ui && npm install && npm run dev
```

前端端口：`3000`，后端端口：`8090`，开发代理：`/api/* → localhost:8090`
