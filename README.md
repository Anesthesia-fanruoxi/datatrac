# DataTrace 数据同步系统

DataTrace 是一个基于 Go + Gin 的 Web 数据同步工具，支持 MySQL 和 Elasticsearch 之间的双向数据同步。

## 🚀 快速开始

### 1. 环境要求

- Go 1.21+
- MySQL 5.7+

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置数据库

编辑 `config.yaml` 文件，配置您的 MySQL 连接信息：

```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  database: datatrace
```

### 4. 创建数据库

```sql
CREATE DATABASE datatrace CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 5. 运行项目

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动。

### 6. 健康检查

访问 `http://localhost:8080/health` 检查服务是否正常运行。

## 📁 项目结构

```
datatrace/
├── main.go                 # 程序入口
├── config.yaml             # 配置文件
├── go.mod                  # 依赖管理
├── api/                    # API处理器（待实现）
├── models/                 # 数据模型
│   ├── datasource.go       # 数据源模型
│   ├── sync_task.go        # 任务模型
│   ├── task_unit_config.go # 任务单元配置
│   └── task_unit_runtime.go # 任务单元运行记录
├── services/               # 业务逻辑层（待实现）
├── routers/                # 路由配置
│   └── router.go
├── common/                 # 中间件
│   ├── response.go         # 响应封装
│   └── logger.go           # 日志中间件
├── config/                 # 配置管理
│   └── config.go
├── database/               # 数据库初始化
│   └── mysql.go
└── docs/                   # 文档
```

## 🔧 配置说明

### config.yaml

```yaml
# 数据库配置
database:
  host: localhost           # 数据库主机
  port: 3306               # 数据库端口
  username: root           # 数据库用户名
  password: root           # 数据库密码
  database: datatrace      # 数据库名称
  max_open_conns: 100      # 最大打开连接数
  max_idle_conns: 10       # 最大空闲连接数

# 服务器配置
server:
  port: 8080               # 服务监听端口
  mode: debug              # 运行模式: debug/release

# 安全配置
security:
  encryption_key: "change-this-to-a-32-byte-key!!"  # 加密密钥（32字节）
```

⚠️ **重要**：请修改 `security.encryption_key` 为您自己的32字节密钥！

## 📊 数据库表

系统启动时会自动创建以下表：

- `data_sources` - 数据源表
- `sync_tasks` - 同步任务表
- `task_unit_configs` - 任务单元配置表
- `task_unit_runtimes` - 任务单元运行记录表

## 🎯 API 接口

### 健康检查

```
GET /health
```

### 数据源管理

```
GET    /api/v1/datasources      # 获取数据源列表
POST   /api/v1/datasources      # 创建数据源（待实现）
GET    /api/v1/datasources/:id  # 获取数据源详情（待实现）
PUT    /api/v1/datasources/:id  # 更新数据源（待实现）
DELETE /api/v1/datasources/:id  # 删除数据源（待实现）
```

### 任务管理

```
GET    /api/v1/tasks      # 获取任务列表
POST   /api/v1/tasks      # 创建任务（待实现）
GET    /api/v1/tasks/:id  # 获取任务详情（待实现）
PUT    /api/v1/tasks/:id  # 更新任务（待实现）
DELETE /api/v1/tasks/:id  # 删除任务（待实现）
```

## 📚 文档

详细设计文档请查看 `docs/` 目录：

- [系统设计文档](docs/requirements.md)
- [数据库设计](docs/数据库设计.md)
- [设计概览](docs/设计概览.md)

## 🔄 开发状态

- ✅ 项目基础框架
- ✅ 数据库连接和模型
- ✅ 路由和中间件
- ⏳ 数据源管理 API
- ⏳ 任务管理 API
- ⏳ 同步引擎核心
- ⏳ 任务调度器

## 📝 开发计划

### 阶段一：基础框架 ✅ 已完成

- [x] 项目初始化
- [x] 配置管理
- [x] 数据库连接
- [x] 数据模型定义
- [x] 路由配置
- [x] 响应封装

### 阶段二：数据源管理（进行中）

- [ ] 数据源 CRUD API
- [ ] 数据源连接测试
- [ ] 密码加密服务
- [ ] 数据库/表/索引查询

### 阶段三：任务管理

- [ ] 任务配置 API
- [ ] 任务单元生成
- [ ] 任务状态管理

### 阶段四：同步引擎

- [ ] Pipeline 框架
- [ ] Source 适配器
- [ ] Transformer 转换器
- [ ] Sink 适配器

### 阶段五：任务调度

- [ ] TaskManager 实现
- [ ] 并发控制
- [ ] 进度监控
- [ ] SSE 推送

## 📄 License

MIT License
