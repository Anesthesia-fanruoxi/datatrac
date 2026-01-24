# 需求文档 - 数据同步工具

## 简介

一个基于 Tauri + Vue3 + TypeScript 的桌面数据同步工具，支持 MySQL 和 Elasticsearch 之间的双向数据同步。

## 术语表

- **System**: 数据同步工具系统
- **DataSource**: 数据源（MySQL 或 Elasticsearch）
- **SyncTask**: 同步任务
- **Connection**: 数据库连接
- **SyncEngine**: 同步引擎
- **ProgressMonitor**: 进度监控器
- **ErrorLogger**: 错误日志记录器

## 需求

### 需求 1：数据源管理

**用户故事：** 作为用户，我想要管理多个数据源配置，以便在不同的数据库之间进行同步。

#### 验收标准

1. WHEN 用户添加新数据源 THEN THE System SHALL 保存数据源配置信息（名称、类型、主机、端口、用户名、密码、数据库名）
2. WHEN 用户查看密码字段 THEN THE System SHALL 显示明文密码（点击眼睛图标）
3. WHEN 数据源配置存储到本地 THEN THE System SHALL 加密存储敏感信息（密码）
4. WHEN 用户编辑数据源 THEN THE System SHALL 更新配置信息
5. WHEN 用户删除数据源 THEN THE System SHALL 从本地存储中移除该配置
6. WHEN 用户测试连接 THEN THE System SHALL 验证数据源连接是否成功并返回结果

### 需求 2：连接测试

**用户故事：** 作为用户，我想要测试数据源连接，以便确认配置是否正确。

#### 验收标准

1. WHEN 用户点击测试连接按钮 THEN THE System SHALL 尝试连接到指定的数据源
2. IF 连接成功 THEN THE System SHALL 显示成功提示信息
3. IF 连接失败 THEN THE System SHALL 显示详细的错误信息
4. WHEN 测试 MySQL 连接 THEN THE System SHALL 验证主机、端口、用户名、密码和数据库名
5. WHEN 测试 Elasticsearch 连接 THEN THE System SHALL 验证主机、端口和认证信息

### 需求 3：同步任务配置

**用户故事：** 作为用户，我想要创建同步任务，以便配置数据从源到目标的同步规则。

#### 验收标准

1. WHEN 用户创建同步任务 THEN THE System SHALL 要求选择源数据源和目标数据源
2. WHEN 源数据源是 MySQL THEN THE System SHALL 允许用户勾选要同步的数据库
3. WHEN 用户选择 MySQL 数据库后 THEN THE System SHALL 显示该数据库下的所有表供用户勾选
4. WHEN 源数据源是 Elasticsearch THEN THE System SHALL 允许用户勾选要同步的索引
5. WHEN 用户配置 ES 索引选择 THEN THE System SHALL 支持通配符匹配（例如：logs-*）
6. WHEN 用户使用通配符匹配 ES 索引 THEN THE System SHALL 显示匹配到的索引数量和示例
7. WHEN 用户配置 MySQL 到 ES 同步 THEN THE System SHALL 按照 ES 的数据类型规则自动映射字段
8. WHEN 用户配置 ES 到 MySQL 同步 THEN THE System SHALL 按照 MySQL 的数据类型规则自动映射字段
9. WHEN 用户保存任务配置 THEN THE System SHALL 验证配置完整性并保存
10. THE System SHALL 支持全量同步模式
11. WHEN 用户配置同步任务 THEN THE System SHALL 允许设置线程数和批量大小
12. WHEN 用户配置错误处理策略 THEN THE System SHALL 提供"跳过错误"或"遇错暂停"两个选项

### 需求 4：同步执行

**用户故事：** 作为用户，我想要手动执行同步任务，以便将数据从源同步到目标。

#### 验收标准

1. WHEN 用户启动同步任务 THEN THE SyncEngine SHALL 开始执行数据同步
2. WHEN 同步到目标数据库 THEN THE System SHALL 先删除目标表/索引，然后重新创建
3. IF 目标是 MySQL 且表存在 THEN THE System SHALL 删除表后重新创建
4. IF 目标是 MySQL 且表不存在 THEN THE System SHALL 直接创建表
5. IF 目标是 Elasticsearch 且索引存在 THEN THE System SHALL 删除索引后重新创建
6. IF 目标是 Elasticsearch 且索引不存在 THEN THE System SHALL 直接创建索引
7. WHEN 执行同步 THEN THE SyncEngine SHALL 根据配置的线程数并发处理数据
8. WHEN 执行同步 THEN THE SyncEngine SHALL 根据数据库类型和配置自适应批量大小
9. WHEN 同步过程中遇到错误且策略为"跳过" THEN THE System SHALL 记录错误并继续同步
10. WHEN 同步过程中遇到错误且策略为"暂停" THEN THE System SHALL 暂停同步并显示错误信息

### 需求 5：任务控制

**用户故事：** 作为用户，我想要控制同步任务的执行状态，以便在需要时暂停或恢复任务。

#### 验收标准

1. WHEN 同步任务正在执行 THEN THE System SHALL 提供暂停按钮
2. WHEN 用户点击暂停 THEN THE SyncEngine SHALL 停止当前批次后暂停任务
3. WHEN 任务处于暂停状态 THEN THE System SHALL 提供恢复按钮
4. WHEN 用户点击恢复 THEN THE SyncEngine SHALL 从暂停点继续执行同步
5. WHEN 任务完成或被停止 THEN THE System SHALL 更新任务状态

### 需求 6：进度监控

**用户故事：** 作为用户，我想要实时查看同步进度，以便了解任务执行情况。

#### 验收标准

1. WHEN 同步任务执行中 THEN THE ProgressMonitor SHALL 实时显示当前进度百分比
2. WHEN 同步任务执行中 THEN THE ProgressMonitor SHALL 显示已同步记录数和总记录数
3. WHEN 同步任务执行中 THEN THE ProgressMonitor SHALL 显示当前同步速度（记录/秒）
4. WHEN 同步任务执行中 THEN THE ProgressMonitor SHALL 显示预计剩余时间
5. WHEN 同步任务执行中 THEN THE ProgressMonitor SHALL 显示当前任务状态（运行中/暂停/完成/失败）

### 需求 7：错误日志

**用户故事：** 作为用户，我想要查看同步过程中的错误日志，以便排查问题。

#### 验收标准

1. WHEN 同步过程中发生错误 THEN THE ErrorLogger SHALL 记录错误信息
2. WHEN 记录错误 THEN THE ErrorLogger SHALL 包含时间戳、错误类型、错误消息和相关数据
3. WHEN 用户查看错误日志 THEN THE System SHALL 显示当前任务的所有错误记录
4. WHEN 任务完成 THEN THE System SHALL 显示错误总数
5. THE System SHALL 不保存历史任务的错误日志（仅显示当前任务）

### 需求 8：数据加密存储

**用户故事：** 作为用户，我想要安全地存储敏感信息，以便保护数据源的访问凭证。

#### 验收标准

1. WHEN 保存数据源配置 THEN THE System SHALL 使用加密算法加密密码字段
2. WHEN 读取数据源配置 THEN THE System SHALL 解密密码字段后使用
3. WHEN 用户点击查看密码 THEN THE System SHALL 显示解密后的明文密码
4. THE System SHALL 使用 AES-256 或更强的加密算法
5. THE System SHALL 将加密密钥安全存储在本地

### 需求 9：用户界面

**用户故事：** 作为用户，我想要一个直观的界面，以便轻松管理数据源和同步任务。

#### 验收标准

1. THE System SHALL 使用 Naive UI 组件库构建界面
2. THE System SHALL 提供数据源管理页面（列表、添加、编辑、删除、测试连接）
3. THE System SHALL 提供同步任务配置页面（创建、编辑任务）
4. THE System SHALL 提供任务执行监控页面（启动、暂停、恢复、进度显示、错误日志）
5. WHEN 用户操作成功 THEN THE System SHALL 显示成功提示
6. WHEN 用户操作失败 THEN THE System SHALL 显示错误提示

### 需求 10：数据类型映射

**用户故事：** 作为用户，我想要系统自动处理不同数据库之间的类型转换，以便无需手动配置映射规则。

#### 验收标准

1. WHEN 从 MySQL 同步到 ES THEN THE System SHALL 自动将 MySQL 类型映射到 ES 类型
2. WHEN 从 ES 同步到 MySQL THEN THE System SHALL 自动将 ES 类型映射到 MySQL 类型
3. THE System SHALL 支持常见数据类型的映射（INT、VARCHAR、TEXT、DATE、DATETIME、BOOLEAN 等）
4. IF 遇到无法自动映射的类型 THEN THE System SHALL 记录警告并使用默认类型
5. WHEN 同步 MySQL 到 ES THEN THE System SHALL 将主键字段映射为 ES 的 _id 字段

### 需求 11：性能优化

**用户故事：** 作为用户，我想要系统能够高效地同步大量数据，以便节省时间。

#### 验收标准

1. THE System SHALL 支持配置并发线程数（1-32 线程）
2. THE System SHALL 根据数据库类型自动调整批量大小
3. WHEN 同步到 MySQL THEN THE System SHALL 使用批量插入（默认 1000 条/批）
4. WHEN 同步到 ES THEN THE System SHALL 使用 bulk API（默认 500 条/批）
5. THE System SHALL 允许用户自定义批量大小
6. WHEN 系统资源不足 THEN THE System SHALL 自动降低并发度

### 需求 12：配置持久化

**用户故事：** 作为用户，我想要系统保存我的配置，以便下次启动时自动加载。

#### 验收标准

1. WHEN 用户添加或修改数据源 THEN THE System SHALL 持久化到本地存储
2. WHEN 用户创建或修改同步任务 THEN THE System SHALL 持久化到本地存储
3. WHEN 应用启动 THEN THE System SHALL 自动加载所有保存的配置
4. THE System SHALL 使用 SQLite 或 JSON 文件存储配置
5. WHEN 配置文件损坏 THEN THE System SHALL 显示错误并提供重置选项
