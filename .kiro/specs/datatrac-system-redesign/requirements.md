# DataTrac 数据同步系统 - 系统重构设计文档

## 项目概述

DataTrac 是一个跨数据源的数据同步工具，支持 MySQL 和 Elasticsearch 之间的双向数据同步。本文档定义了系统重构的完整设计规范。

---

## 1. 全局规划

### 1.1 技术栈定型

**前端技术栈：**
- 框架：Vue 3 + TypeScript
- 构建工具：Vite
- UI 框架：Element Plus / Ant Design Vue（待定）
- 状态管理：Pinia
- 路由：Vue Router
- HTTP 客户端：Tauri Invoke API

**后端技术栈：**
- 语言：Rust
- 框架：Tauri 2.x
- 数据库：SQLite (sqlx)
- 异步运行时：Tokio
- 数据库驱动：
  - MySQL: sqlx-mysql
  - Elasticsearch: elasticsearch-rs

**命名规范：**
- 前后端统一使用 **camelCase**（驼峰命名）
- 数据库字段使用 **snake_case**（下划线命名）
- 文件名使用 **kebab-case**（短横线命名）

### 1.2 前后端统一数据结构

**统一响应体：**
```typescript
// 成功响应
interface SuccessResponse<T> {
  success: true;
  data: T;
}

// 错误响应
interface ErrorResponse {
  success: false;
  error: {
    code: string;
    message: string;
    details?: any;
  };
}

type ApiResponse<T> = SuccessResponse<T> | ErrorResponse;
```

**统一请求体：**
```typescript
// 分页请求
interface PageRequest {
  page: number;
  pageSize: number;
}

// 排序请求
interface SortRequest {
  field: string;
  order: 'asc' | 'desc';
}
```

### 1.3 目录规划

**前端目录结构：**
```
src/
├── api/                    # API 接口层
│   ├── datasource.ts      # 数据源 API
│   ├── task.ts            # 任务 API
│   └── monitor.ts         # 监控 API
├── types/                  # 类型定义
│   ├── datasource.ts
│   ├── task.ts
│   └── common.ts
├── stores/                 # 状态管理
│   ├── datasource.ts
│   ├── task.ts
│   └── monitor.ts
├── views/                  # 页面视图
│   ├── DataSource/        # 数据源管理
│   ├── TaskConfig/        # 任务配置
│   └── TaskMonitor/       # 任务监控
├── components/             # 公共组件
│   ├── common/            # 通用组件
│   └── business/          # 业务组件
└── utils/                  # 工具函数
    ├── request.ts         # 请求封装
    └── format.ts          # 格式化工具
```

**后端目录结构：**
```
src-tauri/src/
├── api/                    # API 层（Tauri Commands）
│   ├── datasource.rs      # 数据源 API
│   ├── task.rs            # 任务 API
│   └── monitor.rs         # 监控 API
├── service/                # 业务逻辑层
│   ├── datasource.rs      # 数据源服务
│   ├── task.rs            # 任务服务
│   └── sync.rs            # 同步服务
├── repository/             # 数据访问层
│   ├── datasource.rs      # 数据源仓储
│   ├── task.rs            # 任务仓储
│   └── unit.rs            # 任务单元仓储
├── domain/                 # 领域模型
│   ├── datasource.rs      # 数据源模型
│   ├── task.rs            # 任务模型
│   └── unit.rs            # 任务单元模型
├── sync/                   # 同步引擎
│   ├── executor/          # 执行器
│   │   ├── mysql.rs
│   │   └── elasticsearch.rs
│   ├── scheduler.rs       # 调度器
│   └── monitor.rs         # 监控器
└── infrastructure/         # 基础设施
    ├── database.rs        # 数据库连接
    ├── crypto.rs          # 加密服务
    └── logger.rs          # 日志服务
```

### 1.4 横向扩展设计

**插件化架构：**
- 数据源适配器接口（Trait）
- 同步策略接口
- 类型映射器接口

**扩展点：**
1. 新增数据源类型：实现 `DataSourceAdapter` trait
2. 新增同步策略：实现 `SyncStrategy` trait
3. 新增类型映射：实现 `TypeMapper` trait

---

## 2. 逻辑设计

### 2.1 创建数据源逻辑

**流程：**
1. 用户输入数据源信息（名称、类型、连接参数）
2. 前端验证必填字段
3. 调用后端 `create_datasource` API
4. 后端验证连接可用性
5. 加密敏感信息（密码）
6. 保存到数据库
7. 返回数据源 ID

**状态流转：**
```
[创建中] -> [测试连接] -> [连接成功] -> [保存] -> [已创建]
                      -> [连接失败] -> [创建失败]
```

### 2.2 配置任务逻辑

**流程：**
1. 选择源数据源和目标数据源
2. 系统自动识别同步方向（MySQL→ES, ES→MySQL, MySQL→MySQL, ES→ES）
3. 根据源数据源类型，加载可选的库/表/索引
4. 用户选择要同步的对象
5. 配置同步参数（批次大小、并发数、错误策略）
6. 配置数据库名称转换规则（可选）
7. 保存任务配置
8. 系统生成任务单元配置（每个表/索引一个单元）

**任务配置结构：**
```typescript
interface TaskConfig {
  id: string;
  name: string;
  sourceId: string;
  targetId: string;
  syncDirection: 'mysql_to_es' | 'es_to_mysql' | 'mysql_to_mysql' | 'es_to_es';
  syncConfig: {
    batchSize: number;        // 批次大小
    threadCount: number;      // 并发线程数
    errorStrategy: 'skip' | 'pause';  // 错误策略
    tableExistsStrategy: 'drop' | 'truncate' | 'backup';  // 表存在策略
    dbNameTransform?: {       // 数据库名称转换
      enabled: boolean;
      mode: 'prefix' | 'suffix';
      sourcePattern: string;
      targetPattern: string;
    };
  };
  mysqlConfig?: {
    databases: Array<{
      database: string;
      tables: string[];
    }>;
  };
  esConfig?: {
    indices: string[];
  };
}
```

### 2.3 同步执行逻辑

**核心流程：**
```
1. 启动任务
   ├─ 加载任务配置
   ├─ 初始化任务单元（从数据库加载或创建）
   ├─ 创建数据源连接池
   └─ 启动并发执行器

2. 并发执行
   ├─ 使用 Semaphore 控制并发数
   ├─ 每个任务单元独立执行
   │  ├─ 更新状态为 Running
   │  ├─ 创建目标表/索引
   │  ├─ 分批读取源数据
   │  ├─ 分批写入目标数据
   │  ├─ 更新进度
   │  └─ 更新状态为 Completed/Failed
   └─ 等待所有单元完成

3. 进度推送
   ├─ 每个批次后更新进度
   ├─ 通过 Tauri Event 推送到前端
   └─ 前端实时显示进度

4. 错误处理
   ├─ Skip 策略：记录错误，继续执行其他单元
   └─ Pause 策略：停止所有执行，保存断点
```

**断点续传：**
- 任务单元状态持久化到数据库
- 重启任务时，跳过已完成的单元
- 失败的单元可以重置后重试

### 2.4 前后端通讯逻辑

**通讯方式：**
1. **命令调用（Command）**：前端调用后端方法
   - 使用 Tauri 的 `invoke` API
   - 同步等待返回结果

2. **事件推送（Event）**：后端主动推送数据到前端
   - 使用 Tauri 的 `emit` API
   - 前端通过 `listen` 监听事件

**事件定义：**
```typescript
// 进度更新事件
interface TaskProgressEvent {
  taskId: string;
  status: 'running' | 'paused' | 'completed' | 'failed';
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  speed: number;
  estimatedTime: number;
  taskUnits: TaskUnit[];
}

// 错误事件
interface TaskErrorEvent {
  taskId: string;
  errorCode: string;
  errorMessage: string;
  unitName?: string;
}

// 日志事件
interface TaskLogEvent {
  taskId: string;
  level: 'info' | 'warn' | 'error';
  category: 'summary' | 'realtime' | 'verify';
  message: string;
  timestamp: string;
}
```

---

## 3. 数据库设计

### 3.1 表结构设计

#### 表 1：data_sources（数据源表）
```sql
CREATE TABLE data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- 'mysql' | 'elasticsearch'
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,  -- 加密存储
    database_name TEXT,      -- MySQL 专用
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

**作用：** 存储所有数据源的连接信息

#### 表 2：sync_tasks（同步任务表）
```sql
CREATE TABLE sync_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    sync_direction TEXT NOT NULL,
    config_json TEXT NOT NULL,  -- JSON 格式的完整配置
    status TEXT NOT NULL,  -- 'idle' | 'running' | 'paused' | 'completed' | 'failed'
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (source_id) REFERENCES data_sources(id),
    FOREIGN KEY (target_id) REFERENCES data_sources(id)
);
```

**作用：** 存储同步任务的基本信息和配置

#### 表 3：task_unit_configs（任务单元配置表）
```sql
CREATE TABLE task_unit_configs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    unit_name TEXT NOT NULL,  -- 表名或索引名（带数据库前缀）
    search_pattern TEXT,       -- 用于匹配的模式
    created_at TEXT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES sync_tasks(id) ON DELETE CASCADE,
    UNIQUE(task_id, unit_name)
);
```

**作用：** 存储任务的所有同步单元配置（哪些表/索引需要同步）

#### 表 4：task_unit_runtimes（任务单元运行记录表）
```sql
CREATE TABLE task_unit_runtimes (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    unit_name TEXT NOT NULL,
    status TEXT NOT NULL,  -- 'pending' | 'running' | 'completed' | 'failed' | 'paused'
    total_records INTEGER DEFAULT 0,
    processed_records INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TEXT,
    completed_at TEXT,
    FOREIGN KEY (task_id) REFERENCES sync_tasks(id) ON DELETE CASCADE,
    UNIQUE(task_id, unit_name)
);
```

**作用：** 存储任务单元的运行时状态，支持断点续传

#### 表 5：task_unit_histories（任务单元历史表）
```sql
CREATE TABLE task_unit_histories (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    unit_name TEXT NOT NULL,
    search_pattern TEXT,
    total_records INTEGER DEFAULT 0,
    duration INTEGER,  -- 执行时长（秒）
    completed_at TEXT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES sync_tasks(id) ON DELETE CASCADE
);
```

**作用：** 存储已完成的任务单元历史记录，用于去重和统计

### 3.2 数据库关系图

```
data_sources (1) ─────┐
                      │
                      ├─> sync_tasks (N)
                      │        │
data_sources (1) ─────┘        │
                               ├─> task_unit_configs (N)
                               │
                               ├─> task_unit_runtimes (N)
                               │
                               └─> task_unit_histories (N)
```

---

## 4. UI 设计

### 4.1 整体布局

**主布局：**
```
┌─────────────────────────────────────────────┐
│  Header (应用标题 + 导航)                    │
├──────────┬──────────────────────────────────┤
│          │                                  │
│  Sidebar │  Main Content Area               │
│          │                                  │
│  - 数据源 │  (根据路由显示不同页面)          │
│  - 任务   │                                  │
│  - 监控   │                                  │
│          │                                  │
└──────────┴──────────────────────────────────┘
```

### 4.2 数据源管理页面

**布局：**
```
┌─────────────────────────────────────────────┐
│  [+ 新建数据源]  [批量测试连接]              │
├─────────────────────────────────────────────┤
│  数据源列表 (Table)                          │
│  ┌───────────────────────────────────────┐  │
│  │ 名称 │ 类型 │ 地址 │ 状态 │ 操作     │  │
│  ├───────────────────────────────────────┤  │
│  │ MySQL-1 │ MySQL │ 192.168.1.1:3306 │  │  │
│  │ ES-1 │ ES │ 192.168.1.2:9200 │ ✓ │   │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**样式：**
- 卡片式布局
- 状态用颜色标识（绿色=正常，红色=异常）
- 操作按钮：编辑、删除、测试连接

### 4.3 任务配置页面

**布局（向导式）：**
```
Step 1: 选择数据源
┌─────────────────────────────────────────────┐
│  源数据源: [下拉选择]                        │
│  目标数据源: [下拉选择]                      │
│  同步方向: MySQL → Elasticsearch (自动识别)  │
└─────────────────────────────────────────────┘

Step 2: 选择同步对象
┌─────────────────────────────────────────────┐
│  可选表/索引 (左侧)    已选表/索引 (右侧)    │
│  ┌─────────────┐      ┌─────────────┐      │
│  │ □ table1    │  >>  │ ☑ table2    │      │
│  │ □ table3    │  <<  │ ☑ table4    │      │
│  └─────────────┘      └─────────────┘      │
└─────────────────────────────────────────────┘

Step 3: 配置参数
┌─────────────────────────────────────────────┐
│  批次大小: [1000]                            │
│  并发线程数: [4]                             │
│  错误策略: [跳过 ▼]                          │
│  表存在策略: [删除重建 ▼]                    │
└─────────────────────────────────────────────┘

Step 4: 确认
┌─────────────────────────────────────────────┐
│  任务名称: [输入框]                          │
│  配置摘要: (显示所有配置信息)                │
│  [取消]  [保存]                              │
└─────────────────────────────────────────────┘
```

### 4.4 任务监控页面

**布局：**
```
┌─────────────────────────────────────────────┐
│  任务列表 (左侧 30%)                         │
│  ┌─────────────────┐                        │
│  │ ☑ Task-1 (运行中)│                        │
│  │ □ Task-2 (空闲)  │                        │
│  └─────────────────┘                        │
├─────────────────────────────────────────────┤
│  任务详情 (右侧 70%)                         │
│  ┌─────────────────────────────────────┐    │
│  │ 控制按钮: [启动] [暂停] [停止]      │    │
│  ├─────────────────────────────────────┤    │
│  │ 进度概览:                            │    │
│  │  总进度: ████████░░ 80%              │    │
│  │  速度: 1000 条/秒                    │    │
│  │  预计剩余: 5 分钟                    │    │
│  ├─────────────────────────────────────┤    │
│  │ 表级进度 (Tab 1)                     │    │
│  │  table1: ██████████ 100% ✓          │    │
│  │  table2: ████░░░░░░ 40%  ⟳          │    │
│  │  table3: ░░░░░░░░░░ 0%   ⏸          │    │
│  ├─────────────────────────────────────┤    │
│  │ 实时日志 (Tab 2)                     │    │
│  │  [INFO] 开始同步 table1...           │    │
│  │  [INFO] table1 同步完成              │    │
│  ├─────────────────────────────────────┤    │
│  │ 错误日志 (Tab 3)                     │    │
│  │  [ERROR] table2 同步失败: ...        │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

**样式：**
- 进度条使用渐变色
- 状态图标：✓ (完成), ⟳ (进行中), ⏸ (暂停), ✗ (失败)
- 日志使用等宽字体，不同级别不同颜色

---

## 5. 交互设计

### 5.1 数据源管理交互

**创建数据源：**
1. 点击"新建数据源"按钮
2. 弹出对话框，选择数据源类型
3. 根据类型显示不同的表单字段
4. 填写完成后，点击"测试连接"
5. 连接成功后，"保存"按钮变为可用
6. 点击"保存"，关闭对话框，刷新列表

**测试连接：**
- 点击"测试连接"按钮
- 按钮显示加载状态
- 成功：显示绿色提示"连接成功"
- 失败：显示红色提示"连接失败：错误信息"

### 5.2 任务配置交互

**向导式配置：**
- 使用 Steps 组件显示当前步骤
- 每步完成后，"下一步"按钮变为可用
- 可以点击步骤标题返回上一步修改
- 最后一步显示"保存"按钮

**选择同步对象：**
- 左侧显示可选列表，右侧显示已选列表
- 支持搜索过滤
- 支持全选/取消全选
- 双击或点击箭头按钮移动项目

### 5.3 任务监控交互

**启动任务：**
1. 选择任务
2. 点击"启动"按钮
3. 按钮变为"暂停"
4. 进度条开始更新
5. 表级进度列表实时更新

**暂停/恢复：**
- 点击"暂停"，任务状态变为"已暂停"
- 点击"恢复"，任务继续执行

**实时更新：**
- 进度条每秒更新一次
- 表级进度实时更新
- 日志自动滚动到最新

**错误处理：**
- 失败的表显示红色
- 点击失败的表，显示错误详情
- 提供"重试"按钮

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 可以创建、编辑、删除数据源
- [ ] 可以测试数据源连接
- [ ] 可以创建、编辑、删除同步任务
- [ ] 可以启动、暂停、恢复任务
- [ ] 支持 4 种同步方向
- [ ] 支持并发执行多个表/索引
- [ ] 支持断点续传
- [ ] 实时显示进度和日志
- [ ] 错误处理符合配置的策略

### 6.2 性能验收

- [ ] 单表同步速度 > 1000 条/秒
- [ ] 并发执行 4 个表时，CPU 使用率 < 80%
- [ ] 内存使用 < 500MB
- [ ] 前端界面响应时间 < 100ms

### 6.3 用户体验验收

- [ ] 界面布局清晰，操作流程顺畅
- [ ] 所有操作有明确的反馈
- [ ] 错误提示清晰易懂
- [ ] 支持键盘快捷键
- [ ] 响应式布局，适配不同屏幕尺寸

---

## 7. 技术债务清理

### 7.1 需要重构的部分

1. **统一命名规范**
   - 前后端统一使用 camelCase
   - 移除所有 snake_case 的前端代码

2. **简化模块结构**
   - 合并重复的功能模块
   - 移除未使用的代码

3. **改进错误处理**
   - 统一错误码定义
   - 改进错误提示信息

4. **优化数据库设计**
   - 简化表结构
   - 添加必要的索引

5. **改进日志系统**
   - 统一日志格式
   - 添加日志级别控制

### 7.2 需要删除的部分

- [ ] 未使用的类型定义
- [ ] 重复的工具函数
- [ ] 废弃的 API 接口
- [ ] 测试代码中的临时文件

---

## 8. 实施计划

### Phase 1: 基础重构（1-2 周）
1. 统一命名规范
2. 重构目录结构
3. 统一数据结构
4. 重构数据库设计

### Phase 2: 核心功能（2-3 周）
1. 数据源管理
2. 任务配置
3. 同步执行引擎
4. 进度监控

### Phase 3: UI/UX 优化（1 周）
1. 界面重构
2. 交互优化
3. 响应式适配

### Phase 4: 测试和优化（1 周）
1. 功能测试
2. 性能测试
3. 用户体验测试
4. Bug 修复

---

## 附录

### A. 术语表

- **数据源（DataSource）**：MySQL 或 Elasticsearch 的连接配置
- **同步任务（SyncTask）**：一次完整的数据同步配置
- **任务单元（TaskUnit）**：任务中的一个表或索引
- **同步方向（SyncDirection）**：数据流向（MySQL→ES, ES→MySQL, MySQL→MySQL, ES→ES）
- **断点续传（Resume）**：任务中断后，从上次停止的地方继续执行

### B. 参考资料

- Tauri 官方文档：https://tauri.app/
- Vue 3 官方文档：https://vuejs.org/
- Rust 异步编程：https://rust-lang.github.io/async-book/
- SQLite 文档：https://www.sqlite.org/docs.html
