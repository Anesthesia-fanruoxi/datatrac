# DataTrace Monorepo

当前仓库已拆分为：

- `datatrace-server/`：Go 后端
- `datatrace-ui/`：Vue 前端

## 启动方式

### 1. 启动后端

```bash
cd datatrace-server
go run main.go
```

默认端口：`http://127.0.0.1:8090`

### 2. 启动前端

```bash
cd datatrace-ui
npm install
npm run dev
```

默认端口：`http://127.0.0.1:3000`

前端已配置开发代理：

- `/api/*` → `http://127.0.0.1:8090`
- `/health` → `http://127.0.0.1:8090`

## 迁移策略

当前 `datatrace-server/static` 与 `datatrace-server/templates` 先保留，作为 Vue 页面迁移参考。

优先迁移顺序建议：

1. 任务监控
2. 任务配置
3. 数据源管理
4. 凭据管理
