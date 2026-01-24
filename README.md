# DataTrac - 数据同步工具

一个基于 Tauri + Vue3 + TypeScript 的桌面数据同步工具，支持 MySQL 和 Elasticsearch 之间的双向数据同步。

## 技术栈

### 前端
- Vue 3.5+
- TypeScript 5.0+
- Naive UI 2.43+
- Pinia 3.0+ (状态管理)
- Vite 7.3+ (构建工具)

### 后端
- Rust 1.77+
- Tauri 2.9+
- sqlx 0.7+ (MySQL/SQLite 客户端)
- elasticsearch 8.19+ (ES 客户端)
- tokio 1.49+ (异步运行时)
- serde (序列化)
- aes-gcm (加密)

## 项目结构

```
datatrac/
├── src/                      # 前端源代码
│   ├── components/          # 可复用组件
│   ├── stores/              # Pinia 状态管理
│   ├── views/               # 页面组件
│   ├── types/               # TypeScript 类型定义
│   ├── utils/               # 工具函数
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
├── src-tauri/               # 后端源代码
│   ├── src/
│   │   ├── crypto.rs        # 加密服务
│   │   ├── storage.rs       # 存储层
│   │   ├── datasource.rs    # 数据源管理器
│   │   ├── type_mapper.rs   # 类型映射器
│   │   ├── progress.rs      # 进度监控器
│   │   ├── error_logger.rs  # 错误日志器
│   │   ├── sync_engine.rs   # 同步引擎
│   │   ├── commands.rs      # Tauri Commands
│   │   ├── lib.rs           # 库入口
│   │   └── main.rs          # 应用入口
│   ├── tests/
│   │   ├── properties/      # 属性测试
│   │   └── integration/     # 集成测试
│   ├── Cargo.toml           # Rust 依赖配置
│   └── tauri.conf.json      # Tauri 配置
├── .kiro/                   # Kiro 规范文档
│   └── specs/
│       └── data-sync-tool/
│           ├── requirements.md  # 需求文档
│           ├── design.md        # 设计文档
│           └── tasks.md         # 任务列表
├── index.html               # HTML 入口
├── vite.config.ts           # Vite 配置
├── tsconfig.json            # TypeScript 配置
└── package.json             # NPM 依赖配置
```

## 开发命令

### 安装依赖
```bash
npm install
```

### 开发模式
```bash
npm run tauri:dev
```

### 构建应用
```bash
npm run tauri:build
```

### 前端开发服务器
```bash
npm run dev
```

### 前端构建
```bash
npm run build
```

## 功能特性

- ✅ 数据源管理（MySQL、Elasticsearch）
- ✅ 双向数据同步
- ✅ 实时进度监控
- ✅ 错误日志记录
- ✅ 密码加密存储
- ✅ 类型自动映射
- ✅ 并发处理优化
- ✅ 批量数据处理

## 开发进度

当前已完成：
- [x] 项目初始化和结构搭建

待实现：
- [ ] 加密服务模块
- [ ] 存储层（SQLite）
- [ ] 数据源管理器
- [ ] 类型映射器
- [ ] 进度监控器
- [ ] 错误日志器
- [ ] 同步引擎
- [ ] Tauri Commands
- [ ] 前端状态管理
- [ ] 前端 UI 组件

详细任务列表请查看：`.kiro/specs/data-sync-tool/tasks.md`

## 许可证

ISC
