# DataTrace 数据同步系统 - 系统设计文档

## 项目概述

DataTrace 是一个跨数据源的数据同步工具，支持 MySQL 和 Elasticsearch 之间的双向数据同步。采用 Go + Gin 框架开发的 Web 应用，前后端不分离架构。

---

## 1. 全局规划

### 1.1 技术栈定型

**前端技术栈：**
- 模板引擎：Go HTML Template
- UI 框架：Bootstrap 5 / Tailwind CSS
- JavaScript：原生 JS + Fetch API
- 实时通信：Server-Sent Events (SSE) / WebSocket

**后端技术栈：**
- 语言：Go 1.21+
- Web框架：Gin
- ORM：GORM
- 配置管理：Viper
- 日志：Zap
- 任务调度：robfig/cron
- 协程池：ants
- 数据库：MySQL (配置存储)
- 数据库驱动：
  - MySQL: go-sql-driver/mysql
  - Elasticsearch: olivere/elastic

**命名规范：**
- Go代码使用 **CamelCase**（大驼峰）和 **camelCase**（小驼峰）
- 数据库字段使用 **snake_case**（下划线命名）
- 文件名使用 **snake_case**（下划线命名）
- URL路径使用 **kebab-case**（短横线命名）

### 1.2 统一数据结构

**统一响应体：**
```go
// 成功响应
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

// 分页响应
type PageResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
    Total   int64       `json:"total"`
    Page    int         `json:"page"`
    PageSize int        `json:"page_size"`
}
```

**统一请求体：**
```go
// 分页请求
type PageRequest struct {
    Page     int    `form:"page" binding:"required,min=1"`
    PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
}

// 排序请求
type SortRequest struct {
    Field string `form:"field"`
    Order string `form:"order" binding:"oneof=asc desc"`
}
```

### 1.3 目录规划

**项目目录结构：**
```
datatrace/
├── main.go                 # 程序入口
├── config.yaml             # 配置文件
├── go.mod                  # 依赖管理
├── api/                    # API处理器(Controller层)
│   ├── datasource_api.go  # 数据源API
│   ├── task_api.go        # 任务API
│   └── monitor_api.go     # 监控API
├── models/                 # 数据模型(GORM)
│   ├── datasource.go      # 数据源模型
│   ├── task.go            # 任务模型
│   ├── task_unit.go       # 任务单元模型
│   └── task_runtime.go    # 运行时模型
├── services/               # 业务逻辑层
│   ├── datasource_service.go
│   ├── task_service.go
│   └── sync_service.go
├── pipeline/               # 同步管道(核心)
│   ├── pipeline.go        # 管道编排
│   ├── source/            # 数据源适配器
│   │   ├── interface.go
│   │   ├── mysql_source.go
│   │   └── es_source.go
│   ├── transformer/        # 数据转换器
│   │   ├── interface.go
│   │   └── json_transformer.go
│   └── sink/              # 数据写入器
│       ├── interface.go
│       ├── mysql_sink.go
│       └── es_sink.go
├── scheduler/              # 任务调度器
│   ├── scheduler.go       # 调度器
│   ├── executor.go        # 执行器
│   └── worker_pool.go     # 协程池
├── monitor/                # 监控模块
│   ├── progress.go        # 进度追踪
│   └── sse.go             # SSE推送
├── routers/                # 路由配置
│   └── router.go
├── common/                 # 中间件
│   ├── logger.go          # 日志中间件
│   └── cors.go            # CORS中间件
├── config/                 # 配置管理
│   └── config.go
├── database/               # 数据库初始化
│   └── mysql.go
├── utils/                  # 工具函数
│   ├── response.go        # 响应封装
│   ├── crypto.go          # 加密工具
│   └── validator.go       # 验证工具
├── templates/              # HTML模板
│   ├── layout/            # 布局模板
│   │   └── base.html
│   ├── datasource/        # 数据源页面
│   │   ├── list.html
│   │   └── form.html
│   ├── task/              # 任务页面
│   │   ├── list.html
│   │   ├── form.html
│   │   └── monitor.html
│   └── index.html         # 首页
├── static/                 # 静态资源
│   ├── css/
│   ├── js/
│   └── images/
└── docs/                   # 文档
```

### 1.4 横向扩展设计

**插件化架构：**
- 数据源适配器接口（Interface）
- 同步策略接口
- 类型映射器接口

**扩展点：**
1. 新增数据源类型：实现 `Source` 和 `Sink` 接口
2. 新增同步策略：实现 `SyncStrategy` 接口
3. 新增类型映射：实现 `TypeMapper` 接口

**接口定义示例：**
```go
// Source 数据源读取接口
type Source interface {
    Connect(config ConnectionConfig) error
    Read(query string, batchSize int) (<-chan []map[string]interface{}, error)
    Count(query string) (int64, error)
    Close() error
}

// Sink 数据写入接口
type Sink interface {
    Connect(config ConnectionConfig) error
    Write(table string, data []map[string]interface{}) error
    Close() error
}
```

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
```go
// 任务配置
type TaskConfig struct {
    ID            string         `json:"id"`
    Name          string         `json:"name"`
    SourceID      string         `json:"source_id"`
    TargetID      string         `json:"target_id"`
    SyncDirection string         `json:"sync_direction"` // mysql_to_es/es_to_mysql/mysql_to_mysql/es_to_es
    SyncConfig    SyncConfig     `json:"sync_config"`
    MySQLConfig   *MySQLConfig   `json:"mysql_config,omitempty"`
    ESConfig      *ESConfig      `json:"es_config,omitempty"`
}

// 同步配置
type SyncConfig struct {
    BatchSize           int                  `json:"batch_size"`            // 批次大小
    ThreadCount         int                  `json:"thread_count"`          // 并发数
    ErrorStrategy       string               `json:"error_strategy"`        // skip/pause
    TableExistsStrategy string               `json:"table_exists_strategy"` // drop/truncate/backup
    DBNameTransform     *DBNameTransform     `json:"db_name_transform,omitempty"`
}

// 数据库名称转换
type DBNameTransform struct {
    Enabled       bool   `json:"enabled"`
    Mode          string `json:"mode"`           // prefix/suffix
    SourcePattern string `json:"source_pattern"`
    TargetPattern string `json:"target_pattern"`
}

// MySQL配置
type MySQLConfig struct {
    Databases []DatabaseConfig `json:"databases"`
}

type DatabaseConfig struct {
    Database string   `json:"database"`
    Tables   []string `json:"tables"`
}

// ES配置
type ESConfig struct {
    Indices []string `json:"indices"`
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

2. 并发执行（使用goroutine + channel）
   ├─ 使用Semaphore控制并发数
   ├─ 每个任务单元独立执行
   │  ├─ 更新状态为Running
   │  ├─ 创建目标表/索引
   │  ├─ 分批读取源数据（channel流式处理）
   │  ├─ 分批写入目标数据
   │  ├─ 更新进度
   │  └─ 更新状态为Completed/Failed
   └─ 使用WaitGroup等待所有单元完成

3. 进度推送
   ├─ 每个批次后更新进度
   ├─ 通过SSE推送到前端
   └─ 前端实时显示进度

4. 错误处理
   ├─ Skip策略：记录错误，继续执行其他单元
   └─ Pause策略：停止所有执行，保存断点
```

**Go实现示例：**
```go
// 并发执行任务单元
func (e *Executor) ExecuteTask(taskID string) error {
    // 1. 加载任务单元
    units, err := e.loadTaskUnits(taskID)
    if err != nil {
        return err
    }
    
    // 2. 创建信号量控制并发
    sem := make(chan struct{}, e.config.ThreadCount)
    var wg sync.WaitGroup
    
    // 3. 并发执行
    for _, unit := range units {
        if unit.Status == "completed" {
            continue // 跳过已完成的单元
        }
        
        wg.Add(1)
        sem <- struct{}{} // 获取信号量
        
        go func(u TaskUnit) {
            defer wg.Done()
            defer func() { <-sem }() // 释放信号量
            
            // 执行单元同步
            if err := e.executeUnit(u); err != nil {
                e.handleError(u, err)
            }
        }(unit)
    }
    
    wg.Wait()
    return nil
}

// 执行单个任务单元
func (e *Executor) executeUnit(unit TaskUnit) error {
    // 1. 更新状态
    e.updateUnitStatus(unit.ID, "running")
    
    // 2. 创建Pipeline
    pipeline := e.createPipeline(unit)
    
    // 3. 流式读取和写入
    dataChan, err := pipeline.Source.ReadStream(unit.Query, e.config.BatchSize)
    if err != nil {
        return err
    }
    
    for batch := range dataChan {
        // 转换数据
        transformed := pipeline.Transformer.Transform(batch)
        
        // 写入目标
        if err := pipeline.Sink.Write(unit.TargetTable, transformed); err != nil {
            return err
        }
        
        // 更新进度
        e.updateProgress(unit.ID, len(batch))
        
        // 推送进度到前端
        e.pushProgress(unit.TaskID)
    }
    
    // 4. 更新状态为完成
    e.updateUnitStatus(unit.ID, "completed")
    return nil
}
```

**断点续传：**
- 任务单元状态持久化到MySQL
- 重启任务时，跳过已完成的单元
- 失败的单元可以重置后重试

### 2.4 前后端通讯逻辑

**通讯方式：**

1. **HTTP请求（标准RESTful API）**：
   - 用于CRUD操作
   - 前端使用Fetch API调用
   - 后端返回JSON响应

2. **Server-Sent Events (SSE)** - 实时进度推送：
   - 后端主动推送数据到前端
   - 单向通信，适合进度更新
   - 自动重连机制

3. **WebSocket（可选）** - 双向实时通信：
   - 用于复杂的实时交互
   - 支持双向通信

**SSE实现示例：**

**后端（Go）：**
```go
// SSE处理器
func (h *MonitorHandler) StreamProgress(c *gin.Context) {
    taskID := c.Param("id")
    
    // 设置SSE响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    
    // 创建事件通道
    eventChan := make(chan ProgressEvent)
    
    // 注册监听器
    h.monitor.RegisterListener(taskID, eventChan)
    defer h.monitor.UnregisterListener(taskID, eventChan)
    
    // 推送事件
    for {
        select {
        case event := <-eventChan:
            data, _ := json.Marshal(event)
            c.SSEvent("progress", string(data))
            c.Writer.Flush()
        case <-c.Request.Context().Done():
            return
        }
    }
}
```

**前端（JavaScript）：**
```javascript
// 监听任务进度
function watchTaskProgress(taskId) {
    const eventSource = new EventSource(`/api/v1/tasks/${taskId}/progress/stream`);
    
    eventSource.addEventListener('progress', (e) => {
        const data = JSON.parse(e.data);
        updateProgressUI(data);
    });
    
    eventSource.addEventListener('error', (e) => {
        console.error('SSE连接错误', e);
        eventSource.close();
    });
    
    return eventSource;
}
```

**事件数据结构：**
```go
// 进度事件
type ProgressEvent struct {
    TaskID           string      `json:"task_id"`
    Status           string      `json:"status"`           // running/paused/completed/failed
    TotalRecords     int64       `json:"total_records"`
    ProcessedRecords int64       `json:"processed_records"`
    Percentage       float64     `json:"percentage"`
    Speed            int64       `json:"speed"`            // 条/秒
    EstimatedTime    int64       `json:"estimated_time"`   // 秒
    TaskUnits        []TaskUnit  `json:"task_units"`
}

// 错误事件
type ErrorEvent struct {
    TaskID       string `json:"task_id"`
    ErrorCode    string `json:"error_code"`
    ErrorMessage string `json:"error_message"`
    UnitName     string `json:"unit_name,omitempty"`
}

// 日志事件
type LogEvent struct {
    TaskID    string `json:"task_id"`
    Level     string `json:"level"`     // info/warn/error
    Category  string `json:"category"`  // summary/realtime/verify
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}
```

---

## 3. 数据库设计

### 3.1 表结构设计

**数据库类型：MySQL 5.7+**

#### 表 1：data_sources（数据源表）
```sql
CREATE TABLE `data_sources` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT 'UUID主键',
    `name` VARCHAR(100) NOT NULL COMMENT '数据源名称',
    `type` VARCHAR(20) NOT NULL COMMENT '类型: mysql/elasticsearch',
    `host` VARCHAR(255) NOT NULL COMMENT '主机地址',
    `port` INT NOT NULL COMMENT '端口',
    `username` VARCHAR(100) NOT NULL COMMENT '用户名',
    `password` VARCHAR(255) NOT NULL COMMENT '密码(AES-256-GCM加密)',
    `database_name` VARCHAR(100) DEFAULT NULL COMMENT 'MySQL数据库名',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_type` (`type`),
    INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据源表';
```

**Go Model：**
```go
type DataSource struct {
    ID           string    `gorm:"primaryKey;size:36" json:"id"`
    Name         string    `gorm:"size:100;not null" json:"name"`
    Type         string    `gorm:"size:20;not null" json:"type"` // mysql/elasticsearch
    Host         string    `gorm:"size:255;not null" json:"host"`
    Port         int       `gorm:"not null" json:"port"`
    Username     string    `gorm:"size:100;not null" json:"username"`
    Password     string    `gorm:"size:255;not null" json:"password"` // 加密存储
    DatabaseName string    `gorm:"size:100" json:"database_name"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

#### 表 2：sync_tasks（同步任务表）
```sql
CREATE TABLE `sync_tasks` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT 'UUID主键',
    `name` VARCHAR(100) NOT NULL COMMENT '任务名称',
    `source_id` VARCHAR(36) NOT NULL COMMENT '源数据源ID',
    `target_id` VARCHAR(36) NOT NULL COMMENT '目标数据源ID',
    `source_type` VARCHAR(20) NOT NULL COMMENT '源类型',
    `target_type` VARCHAR(20) NOT NULL COMMENT '目标类型',
    `config` TEXT NOT NULL COMMENT 'JSON格式配置',
    `status` VARCHAR(20) NOT NULL DEFAULT 'idle' COMMENT '状态: idle/running/paused/completed/failed',
    `sync_mode` VARCHAR(20) NOT NULL DEFAULT 'auto' COMMENT '同步模式',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (`source_id`) REFERENCES `data_sources`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`target_id`) REFERENCES `data_sources`(`id`) ON DELETE CASCADE,
    INDEX `idx_status` (`status`),
    INDEX `idx_source_id` (`source_id`),
    INDEX `idx_target_id` (`target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='同步任务表';
```

**Go Model：**
```go
type SyncTask struct {
    ID         string    `gorm:"primaryKey;size:36" json:"id"`
    Name       string    `gorm:"size:100;not null" json:"name"`
    SourceID   string    `gorm:"size:36;not null" json:"source_id"`
    TargetID   string    `gorm:"size:36;not null" json:"target_id"`
    SourceType string    `gorm:"size:20;not null" json:"source_type"`
    TargetType string    `gorm:"size:20;not null" json:"target_type"`
    Config     string    `gorm:"type:text;not null" json:"config"` // JSON
    Status     string    `gorm:"size:20;not null;default:idle" json:"status"`
    SyncMode   string    `gorm:"size:20;not null;default:auto" json:"sync_mode"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    
    // 关联
    SourceConn DataSource `gorm:"foreignKey:SourceID" json:"source_conn,omitempty"`
    TargetConn DataSource `gorm:"foreignKey:TargetID" json:"target_conn,omitempty"`
}
```

#### 表 3：task_unit_configs（任务单元配置表）
```sql
CREATE TABLE `task_unit_configs` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT 'UUID主键',
    `task_id` VARCHAR(36) NOT NULL COMMENT '任务ID',
    `unit_name` VARCHAR(200) NOT NULL COMMENT '单元名称(表名或索引名)',
    `unit_type` VARCHAR(20) NOT NULL COMMENT '类型: table/index',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (`task_id`) REFERENCES `sync_tasks`(`id`) ON DELETE CASCADE,
    UNIQUE KEY `uk_task_unit` (`task_id`, `unit_name`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_unit_type` (`unit_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务单元配置表';
```

**Go Model：**
```go
type TaskUnitConfig struct {
    ID        string    `gorm:"primaryKey;size:36" json:"id"`
    TaskID    string    `gorm:"size:36;not null;uniqueIndex:uk_task_unit" json:"task_id"`
    UnitName  string    `gorm:"size:200;not null;uniqueIndex:uk_task_unit" json:"unit_name"`
    UnitType  string    `gorm:"size:20;not null" json:"unit_type"` // table/index
    CreatedAt time.Time `json:"created_at"`
}
```

#### 表 4：task_unit_runtimes（任务单元运行记录表）
```sql
CREATE TABLE `task_unit_runtimes` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT 'UUID主键',
    `task_id` VARCHAR(36) NOT NULL COMMENT '任务ID',
    `unit_name` VARCHAR(200) NOT NULL COMMENT '单元名称',
    `status` VARCHAR(20) NOT NULL COMMENT '状态: pending/running/completed/failed/paused',
    `total_records` BIGINT DEFAULT 0 COMMENT '总记录数',
    `processed_records` BIGINT DEFAULT 0 COMMENT '已处理记录数',
    `error_message` TEXT COMMENT '错误信息',
    `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `last_processed_batch` INT DEFAULT NULL COMMENT '最后处理批次号',
    FOREIGN KEY (`task_id`) REFERENCES `sync_tasks`(`id`) ON DELETE CASCADE,
    UNIQUE KEY `uk_task_unit` (`task_id`, `unit_name`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_task_status` (`task_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务单元运行记录表';
```

**Go Model：**
```go
type TaskUnitRuntime struct {
    ID                 string     `gorm:"primaryKey;size:36" json:"id"`
    TaskID             string     `gorm:"size:36;not null;uniqueIndex:uk_task_unit" json:"task_id"`
    UnitName           string     `gorm:"size:200;not null;uniqueIndex:uk_task_unit" json:"unit_name"`
    Status             string     `gorm:"size:20;not null" json:"status"`
    TotalRecords       int64      `gorm:"default:0" json:"total_records"`
    ProcessedRecords   int64      `gorm:"default:0" json:"processed_records"`
    ErrorMessage       string     `gorm:"type:text" json:"error_message"`
    StartedAt          *time.Time `json:"started_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
    LastProcessedBatch *int       `json:"last_processed_batch"`
}
```

### 3.2 数据库关系图

```
data_sources (1) ─────┐
                      │
                      ├─> sync_tasks (N)
                      │        │
data_sources (1) ─────┘        │
                               ├─> task_unit_configs (N)
                               │
                               └─> task_unit_runtimes (N)
```

**关系说明：**
- 一个数据源可以被多个任务使用（作为源或目标）
- 一个任务包含多个任务单元配置
- 一个任务单元配置对应一个运行时记录

---

## 4. UI设计（Go Template + Bootstrap）

### 4.1 整体布局

**主布局（base.html）：**
```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - DataTrace</title>
    <link href="/static/css/bootstrap.min.css" rel="stylesheet">
    <link href="/static/css/custom.css" rel="stylesheet">
</head>
<body>
    <!-- 导航栏 -->
    <nav class="navbar navbar-expand-lg navbar-dark bg-primary">
        <div class="container-fluid">
            <a class="navbar-brand" href="/">DataTrace</a>
            <div class="navbar-nav">
                <a class="nav-link" href="/datasources">数据源</a>
                <a class="nav-link" href="/tasks">任务管理</a>
                <a class="nav-link" href="/monitor">任务监控</a>
            </div>
        </div>
    </nav>
    
    <!-- 主内容区 -->
    <div class="container-fluid mt-4">
        <div class="row">
            <!-- 侧边栏（可选） -->
            {{if .ShowSidebar}}
            <div class="col-md-2">
                {{template "sidebar" .}}
            </div>
            {{end}}
            
            <!-- 内容区 -->
            <div class="{{if .ShowSidebar}}col-md-10{{else}}col-md-12{{end}}">
                {{template "content" .}}
            </div>
        </div>
    </div>
    
    <script src="/static/js/bootstrap.bundle.min.js"></script>
    <script src="/static/js/app.js"></script>
</body>
</html>
```

### 4.2 数据源管理页面

**模板（datasource/list.html）：**
```html
{{define "content"}}
<div class="card">
    <div class="card-header d-flex justify-content-between align-items-center">
        <h5>数据源管理</h5>
        <button class="btn btn-primary" onclick="showCreateModal()">
            <i class="bi bi-plus"></i> 新建数据源
        </button>
    </div>
    <div class="card-body">
        <table class="table table-hover">
            <thead>
                <tr>
                    <th>名称</th>
                    <th>类型</th>
                    <th>地址</th>
                    <th>状态</th>
                    <th>操作</th>
                </tr>
            </thead>
            <tbody>
                {{range .DataSources}}
                <tr>
                    <td>{{.Name}}</td>
                    <td>
                        <span class="badge bg-info">{{.Type}}</span>
                    </td>
                    <td>{{.Host}}:{{.Port}}</td>
                    <td>
                        <span class="badge bg-success">正常</span>
                    </td>
                    <td>
                        <button class="btn btn-sm btn-outline-primary" onclick="testConnection('{{.ID}}')">测试</button>
                        <button class="btn btn-sm btn-outline-warning" onclick="editDataSource('{{.ID}}')">编辑</button>
                        <button class="btn btn-sm btn-outline-danger" onclick="deleteDataSource('{{.ID}}')">删除</button>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>

<!-- 创建/编辑模态框 -->
<div class="modal fade" id="dataSourceModal" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">新建数据源</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <form id="dataSourceForm">
                    <div class="mb-3">
                        <label class="form-label">名称</label>
                        <input type="text" class="form-control" name="name" required>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">类型</label>
                        <select class="form-select" name="type" required>
                            <option value="mysql">MySQL</option>
                            <option value="elasticsearch">Elasticsearch</option>
                        </select>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">主机</label>
                        <input type="text" class="form-control" name="host" required>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">端口</label>
                        <input type="number" class="form-control" name="port" required>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">用户名</label>
                        <input type="text" class="form-control" name="username" required>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">密码</label>
                        <input type="password" class="form-control" name="password" required>
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
                <button type="button" class="btn btn-primary" onclick="testAndSave()">测试并保存</button>
            </div>
        </div>
    </div>
</div>
{{end}}
```

**JavaScript（datasource.js）：**
```javascript
// 显示创建模态框
function showCreateModal() {
    document.getElementById('dataSourceForm').reset();
    new bootstrap.Modal(document.getElementById('dataSourceModal')).show();
}

// 测试并保存
async function testAndSave() {
    const form = document.getElementById('dataSourceForm');
    const formData = new FormData(form);
    const data = Object.fromEntries(formData);
    
    try {
        // 先测试连接
        const testRes = await fetch('/api/v1/datasources/test', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        
        if (!testRes.ok) {
            alert('连接测试失败');
            return;
        }
        
        // 保存数据源
        const saveRes = await fetch('/api/v1/datasources', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        });
        
        if (saveRes.ok) {
            alert('保存成功');
            location.reload();
        }
    } catch (error) {
        alert('操作失败: ' + error.message);
    }
}
```

### 4.3 任务配置页面

**模板（task/form.html）：**
```html
{{define "content"}}
<div class="card">
    <div class="card-header">
        <h5>{{if .Task}}编辑任务{{else}}创建任务{{end}}</h5>
    </div>
    <div class="card-body">
        <!-- 步骤指示器 -->
        <ul class="nav nav-pills mb-4">
            <li class="nav-item">
                <a class="nav-link active" data-step="1">1. 选择数据源</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" data-step="2">2. 选择同步对象</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" data-step="3">3. 配置参数</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" data-step="4">4. 确认保存</a>
            </li>
        </ul>
        
        <!-- 步骤1: 选择数据源 -->
        <div class="step-content" data-step="1">
            <div class="row">
                <div class="col-md-6">
                    <label class="form-label">源数据源</label>
                    <select class="form-select" id="sourceId" onchange="loadSourceObjects()">
                        <option value="">请选择</option>
                        {{range .DataSources}}
                        <option value="{{.ID}}">{{.Name}} ({{.Type}})</option>
                        {{end}}
                    </select>
                </div>
                <div class="col-md-6">
                    <label class="form-label">目标数据源</label>
                    <select class="form-select" id="targetId">
                        <option value="">请选择</option>
                        {{range .DataSources}}
                        <option value="{{.ID}}">{{.Name}} ({{.Type}})</option>
                        {{end}}
                    </select>
                </div>
            </div>
            <div class="mt-3">
                <span class="badge bg-info">同步方向: <span id="syncDirection">-</span></span>
            </div>
        </div>
        
        <!-- 步骤2: 选择同步对象 -->
        <div class="step-content d-none" data-step="2">
            <div class="row">
                <div class="col-md-5">
                    <h6>可选对象</h6>
                    <input type="text" class="form-control mb-2" placeholder="搜索..." id="searchAvailable">
                    <div class="list-group" id="availableList" style="max-height: 400px; overflow-y: auto;">
                        <!-- 动态加载 -->
                    </div>
                </div>
                <div class="col-md-2 d-flex align-items-center justify-content-center">
                    <div>
                        <button class="btn btn-primary mb-2" onclick="moveToSelected()">>></button>
                        <button class="btn btn-secondary" onclick="moveToAvailable()"><<</button>
                    </div>
                </div>
                <div class="col-md-5">
                    <h6>已选对象 (<span id="selectedCount">0</span>)</h6>
                    <div class="list-group" id="selectedList" style="max-height: 400px; overflow-y: auto;">
                        <!-- 动态加载 -->
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 步骤3: 配置参数 -->
        <div class="step-content d-none" data-step="3">
            <div class="row">
                <div class="col-md-6">
                    <label class="form-label">批次大小</label>
                    <input type="number" class="form-control" id="batchSize" value="1000">
                </div>
                <div class="col-md-6">
                    <label class="form-label">并发数</label>
                    <input type="number" class="form-control" id="threadCount" value="4">
                </div>
            </div>
            <div class="row mt-3">
                <div class="col-md-6">
                    <label class="form-label">错误策略</label>
                    <select class="form-select" id="errorStrategy">
                        <option value="skip">跳过错误继续</option>
                        <option value="pause">遇错暂停</option>
                    </select>
                </div>
                <div class="col-md-6">
                    <label class="form-label">表存在策略</label>
                    <select class="form-select" id="tableExistsStrategy">
                        <option value="drop">删除重建</option>
                        <option value="truncate">清空数据</option>
                        <option value="backup">备份后重建</option>
                    </select>
                </div>
            </div>
        </div>
        
        <!-- 步骤4: 确认 -->
        <div class="step-content d-none" data-step="4">
            <div class="alert alert-info">
                <h6>任务配置摘要</h6>
                <div id="configSummary"></div>
            </div>
            <div class="mb-3">
                <label class="form-label">任务名称</label>
                <input type="text" class="form-control" id="taskName" placeholder="请输入任务名称">
            </div>
        </div>
        
        <!-- 导航按钮 -->
        <div class="d-flex justify-content-between mt-4">
            <button class="btn btn-secondary" id="prevBtn" onclick="prevStep()">上一步</button>
            <button class="btn btn-primary" id="nextBtn" onclick="nextStep()">下一步</button>
            <button class="btn btn-success d-none" id="saveBtn" onclick="saveTask()">保存任务</button>
        </div>
    </div>
</div>
{{end}}
```

### 4.4 任务监控页面

**模板（task/monitor.html）：**
```html
{{define "content"}}
<div class="row">
    <!-- 左侧任务列表 -->
    <div class="col-md-3">
        <div class="card">
            <div class="card-header">
                <h6>任务列表</h6>
            </div>
            <div class="list-group list-group-flush">
                {{range .Tasks}}
                <a href="#" class="list-group-item list-group-item-action" 
                   onclick="selectTask('{{.ID}}')" data-task-id="{{.ID}}">
                    <div class="d-flex justify-content-between align-items-center">
                        <span>{{.Name}}</span>
                        <span class="badge bg-{{if eq .Status "running"}}success{{else if eq .Status "failed"}}danger{{else}}secondary{{end}}">
                            {{.Status}}
                        </span>
                    </div>
                </a>
                {{end}}
            </div>
        </div>
    </div>
    
    <!-- 右侧任务详情 -->
    <div class="col-md-9">
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h6 id="taskTitle">请选择任务</h6>
                <div id="taskControls" class="d-none">
                    <button class="btn btn-sm btn-success" onclick="startTask()">
                        <i class="bi bi-play"></i> 启动
                    </button>
                    <button class="btn btn-sm btn-warning" onclick="pauseTask()">
                        <i class="bi bi-pause"></i> 暂停
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="stopTask()">
                        <i class="bi bi-stop"></i> 停止
                    </button>
                </div>
            </div>
            <div class="card-body">
                <!-- 进度概览 -->
                <div id="progressOverview" class="mb-4 d-none">
                    <h6>总体进度</h6>
                    <div class="progress mb-2" style="height: 30px;">
                        <div id="totalProgress" class="progress-bar progress-bar-striped progress-bar-animated" 
                             role="progressbar" style="width: 0%">0%</div>
                    </div>
                    <div class="row text-center">
                        <div class="col">
                            <small class="text-muted">速度</small>
                            <div id="speed">0 条/秒</div>
                        </div>
                        <div class="col">
                            <small class="text-muted">已处理</small>
                            <div id="processed">0</div>
                        </div>
                        <div class="col">
                            <small class="text-muted">总数</small>
                            <div id="total">0</div>
                        </div>
                        <div class="col">
                            <small class="text-muted">预计剩余</small>
                            <div id="estimated">-</div>
                        </div>
                    </div>
                </div>
                
                <!-- Tab导航 -->
                <ul class="nav nav-tabs" role="tablist">
                    <li class="nav-item">
                        <a class="nav-link active" data-bs-toggle="tab" href="#unitProgress">单元进度</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" data-bs-toggle="tab" href="#realtimeLog">实时日志</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" data-bs-toggle="tab" href="#errorLog">错误日志</a>
                    </li>
                </ul>
                
                <!-- Tab内容 -->
                <div class="tab-content mt-3">
                    <!-- 单元进度 -->
                    <div class="tab-pane fade show active" id="unitProgress">
                        <div class="table-responsive">
                            <table class="table table-sm">
                                <thead>
                                    <tr>
                                        <th>单元名称</th>
                                        <th>状态</th>
                                        <th>进度</th>
                                        <th>已处理/总数</th>
                                    </tr>
                                </thead>
                                <tbody id="unitProgressTable">
                                    <!-- 动态加载 -->
                                </tbody>
                            </table>
                        </div>
                    </div>
                    
                    <!-- 实时日志 -->
                    <div class="tab-pane fade" id="realtimeLog">
                        <div id="logContainer" style="height: 400px; overflow-y: auto; font-family: monospace; font-size: 12px; background: #f8f9fa; padding: 10px;">
                            <!-- 动态加载 -->
                        </div>
                    </div>
                    
                    <!-- 错误日志 -->
                    <div class="tab-pane fade" id="errorLog">
                        <div id="errorContainer" style="height: 400px; overflow-y: auto;">
                            <!-- 动态加载 -->
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>

<script>
let currentTaskId = null;
let eventSource = null;

// 选择任务
function selectTask(taskId) {
    currentTaskId = taskId;
    
    // 关闭旧的SSE连接
    if (eventSource) {
        eventSource.close();
    }
    
    // 显示控制按钮
    document.getElementById('taskControls').classList.remove('d-none');
    document.getElementById('progressOverview').classList.remove('d-none');
    
    // 加载任务详情
    loadTaskDetail(taskId);
    
    // 建立SSE连接
    eventSource = new EventSource(`/api/v1/tasks/${taskId}/progress/stream`);
    
    eventSource.addEventListener('progress', (e) => {
        const data = JSON.parse(e.data);
        updateProgressUI(data);
    });
    
    eventSource.addEventListener('log', (e) => {
        const data = JSON.parse(e.data);
        appendLog(data);
    });
    
    eventSource.onerror = (e) => {
        console.error('SSE连接错误', e);
    };
}

// 更新进度UI
function updateProgressUI(data) {
    const percentage = data.percentage.toFixed(2);
    document.getElementById('totalProgress').style.width = percentage + '%';
    document.getElementById('totalProgress').textContent = percentage + '%';
    document.getElementById('speed').textContent = data.speed + ' 条/秒';
    document.getElementById('processed').textContent = data.processed_records;
    document.getElementById('total').textContent = data.total_records;
    document.getElementById('estimated').textContent = formatTime(data.estimated_time);
    
    // 更新单元进度表
    updateUnitTable(data.task_units);
}

// 更新单元进度表
function updateUnitTable(units) {
    const tbody = document.getElementById('unitProgressTable');
    tbody.innerHTML = '';
    
    units.forEach(unit => {
        const percentage = (unit.processed_records / unit.total_records * 100).toFixed(2);
        const statusBadge = getStatusBadge(unit.status);
        
        tbody.innerHTML += `
            <tr>
                <td>${unit.unit_name}</td>
                <td>${statusBadge}</td>
                <td>
                    <div class="progress" style="height: 20px;">
                        <div class="progress-bar" style="width: ${percentage}%">${percentage}%</div>
                    </div>
                </td>
                <td>${unit.processed_records} / ${unit.total_records}</td>
            </tr>
        `;
    });
}

// 获取状态徽章
function getStatusBadge(status) {
    const badges = {
        'pending': '<span class="badge bg-secondary">待处理</span>',
        'running': '<span class="badge bg-primary">运行中</span>',
        'completed': '<span class="badge bg-success">已完成</span>',
        'failed': '<span class="badge bg-danger">失败</span>',
        'paused': '<span class="badge bg-warning">已暂停</span>'
    };
    return badges[status] || status;
}

// 追加日志
function appendLog(data) {
    const container = document.getElementById('logContainer');
    const time = new Date(data.timestamp).toLocaleTimeString();
    const levelColor = data.level === 'error' ? 'red' : data.level === 'warn' ? 'orange' : 'black';
    container.innerHTML += `<div style="color: ${levelColor}">[${time}] [${data.level.toUpperCase()}] ${data.message}</div>`;
    container.scrollTop = container.scrollHeight;
}

// 格式化时间
function formatTime(seconds) {
    if (seconds < 60) return seconds + '秒';
    if (seconds < 3600) return Math.floor(seconds / 60) + '分钟';
    return Math.floor(seconds / 3600) + '小时';
}
</script>
{{end}}
```

---

## 5. 交互设计

### 5.1 数据源管理交互

**创建数据源流程：**
1. 点击"新建数据源"按钮 → 弹出模态框
2. 选择数据源类型 → 显示对应表单
3. 填写连接信息 → 点击"测试并保存"
4. 后端验证连接 → 保存到数据库
5. 关闭模态框 → 刷新列表

**测试连接反馈：**
- 成功：绿色Toast提示
- 失败：红色Toast提示，显示错误信息

### 5.2 任务配置交互

**向导式配置流程：**
- 步骤1：选择源和目标数据源 → 自动识别同步方向
- 步骤2：选择同步对象 → 左右列表选择器，支持搜索
- 步骤3：配置参数 → 批次大小、并发数、错误策略等
- 步骤4：确认保存 → 显示配置摘要，输入任务名称

**交互特点：**
- 步骤指示器显示当前进度
- 每步验证通过后才能进入下一步
- 支持返回上一步修改
- 最后一步显示完整配置摘要

### 5.3 任务监控交互

**任务控制：**
- 启动：按钮显示加载状态 → 建立SSE连接 → 开始推送进度
- 暂停：保存当前状态 → 停止执行
- 恢复：从断点继续执行
- 停止：确认对话框 → 停止并关闭SSE连接

**实时更新机制：**
- 总体进度：进度条 + 速度 + 预计时间
- 单元进度：表格显示每个单元的状态和进度
- 实时日志：自动滚动，不同级别不同颜色
- 错误日志：单独Tab显示，可点击查看详情

**状态反馈：**
- 使用颜色徽章区分状态（待处理/运行中/已完成/失败/已暂停）
- 进度条使用动画效果
- 失败单元高亮显示，提供重试功能

---

## 6. 验收标准

### 6.1 功能验收

**数据源管理：**
- [ ] 可以创建、编辑、删除数据源
- [ ] 可以测试数据源连接
- [ ] 密码加密存储

**任务管理：**
- [ ] 可以创建、编辑、删除同步任务
- [ ] 支持4种同步方向（MySQL→ES, ES→MySQL, MySQL→MySQL, ES→ES）
- [ ] 可以选择要同步的表/索引
- [ ] 可以配置同步参数

**任务执行：**
- [ ] 可以启动、暂停、恢复、停止任务
- [ ] 支持并发执行多个表/索引
- [ ] 支持断点续传
- [ ] 错误处理符合配置的策略

**监控功能：**
- [ ] 实时显示任务进度
- [ ] 实时显示单元进度
- [ ] 实时显示日志
- [ ] 错误日志单独展示

### 6.2 性能验收

- [ ] 单表同步速度 > 1000 条/秒
- [ ] 并发执行4个表时，CPU使用率 < 80%
- [ ] 内存使用 < 500MB
- [ ] 页面响应时间 < 200ms
- [ ] SSE推送延迟 < 1秒

### 6.3 用户体验验收

- [ ] 界面布局清晰，操作流程顺畅
- [ ] 所有操作有明确的反馈（Toast/加载状态）
- [ ] 错误提示清晰易懂
- [ ] 响应式布局，适配不同屏幕尺寸
- [ ] 进度更新流畅，无卡顿

---

## 7. 技术债务清理

### 7.1 需要重构的部分

**代码规范统一：**
- Go代码遵循官方规范
- 统一命名风格（CamelCase/snake_case）
- 统一错误处理方式

**模块结构优化：**
- 合并重复的功能模块
- 移除未使用的代码
- 单个文件不超过300行

**数据库优化：**
- 添加必要的索引
- 优化查询语句
- 使用连接池

**日志系统改进：**
- 统一日志格式
- 添加日志级别控制
- 日志文件轮转

### 7.2 需要删除的部分

- [ ] 未使用的类型定义
- [ ] 重复的工具函数
- [ ] 废弃的API接口
- [ ] 测试代码中的临时文件

---

## 8. 实施计划

### Phase 1: 基础框架（1-2周）
**目标：** 搭建项目基础架构

**任务：**
- 初始化Go项目，配置依赖
- 实现配置管理（Viper）
- 实现数据库连接和模型（GORM）
- 搭建Gin Web框架
- 实现日志系统（Zap）
- 创建基础HTML模板

### Phase 2: 核心功能（2-3周）
**目标：** 实现数据源管理和任务配置

**任务：**
- 数据源管理（CRUD + 测试连接）
- 任务配置（向导式创建）
- Pipeline核心实现（Source-Transformer-Sink）
- MySQL和ES适配器实现
- 任务单元管理

### Phase 3: 同步执行（2-3周）
**目标：** 实现同步执行引擎

**任务：**
- 任务调度器实现
- 并发执行器实现（goroutine + channel）
- 进度追踪实现
- 断点续传实现
- 错误处理和重试机制

### Phase 4: 监控和UI（1-2周）
**目标：** 完善监控功能和用户界面

**任务：**
- SSE实时推送实现
- 任务监控页面
- 进度展示和日志展示
- UI优化和响应式适配

### Phase 5: 测试和优化（1周）
**目标：** 测试和性能优化

**任务：**
- 功能测试
- 性能测试和优化
- 用户体验测试
- Bug修复
- 文档完善

---

## 附录

### A. 术语表

- **数据源（DataSource）**：MySQL或Elasticsearch的连接配置
- **同步任务（SyncTask）**：一次完整的数据同步配置
- **任务单元（TaskUnit）**：任务中的一个表或索引
- **同步方向（SyncDirection）**：数据流向（MySQL→ES, ES→MySQL, MySQL→MySQL, ES→ES）
- **断点续传（Resume）**：任务中断后，从上次停止的地方继续执行
- **Pipeline**：Source-Transformer-Sink数据处理管道
- **SSE**：Server-Sent Events，服务器推送事件

### B. 技术选型理由

**为什么选择Go：**
- 高性能，适合并发处理
- 简单易学，开发效率高
- 丰富的标准库和生态
- 天然支持并发（goroutine）

**为什么选择Gin：**
- 轻量级，性能优秀
- 中间件支持完善
- 文档齐全，社区活跃

**为什么选择MySQL：**
- 成熟稳定，广泛使用
- 支持事务和外键
- 性能优秀

**为什么选择SSE而非WebSocket：**
- 单向推送足够满足需求
- 实现更简单
- 自动重连机制
- HTTP协议，无需额外端口

### C. 参考资料

- Go官方文档：https://golang.org/doc/
- Gin框架文档：https://gin-gonic.com/docs/
- GORM文档：https://gorm.io/docs/
- Bootstrap文档：https://getbootstrap.com/docs/
- SSE规范：https://html.spec.whatwg.org/multipage/server-sent-events.html

---

**文档版本：** v2.0  
**最后更新：** 2026-02-27  
**文档维护：** DataTrace开发团队
