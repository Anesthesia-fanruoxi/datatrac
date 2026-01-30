# DataTrac 重构任务列表

## 📋 说明
本列表记录 DataTrac v2.0 重构计划的关键任务。

---

## 🏗️ 第一阶段：核心底座 (The Base)

- [x] 1. 数据库三表结构重构 (Tasks, UnitConfigs, UnitRuntimes)
- [x] 2. 任务管理器 (TaskManager) 自动模式调度实现
- [ ] 3. 数据交换框架 (Exchange Framework)
    - [ ] 3.1 定义 DataRecord 标准中间格式
    - [ ] 3.2 定义 SourceReader / TargetWriter 异步 Traits
    - [ ] 3.3 实现 SyncPipeline 核心调度循环
    - [ ] 3.4 实现字段映射 Transformer

## 🔌 第二阶段：插件化适配 (Plugins)

- [ ] 4. MySQL 适配器
    - [ ] 4.1 实现 MySqlReader (基于分页/游标)
    - [ ] 4.2 实现 MySqlWriter (基于批量 Insert)
- [ ] 5. Elasticsearch 适配器
    - [ ] 5.1 实现 ElasticReader (基于 Scroll API)
    - [ ] 5.2 实现 ElasticWriter (基于 Bulk API)
- [ ] 6. MongoDB 适配器 (扩展目标)
    - [ ] 6.1 实现 MongoReader

## 🎨 第三阶段：UI/UX 增强

- [ ] 7. 任务监控页面优化
    - [ ] 7.1 实时日志分类展示
    - [ ] 7.2 进度条平滑动画
- [ ] 8. 任务配置向导重构
    - [ ] 8.1 动态 Reader/Writer 配置表单

## ✅ 第四阶段：验证与发布

- [ ] 9. 端到端集成测试 (MySQL <-> ES)
- [ ] 10. 性能基准测试
