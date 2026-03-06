package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// IncrementalConsumer 增量消费引擎
type IncrementalConsumer struct {
	taskID       string
	queue        BinlogQueue
	targetDB     *sql.DB
	logService   *TaskLogService
	sseService   *TaskSSEService
	statsService *IncrementalStatsService
	ctx          context.Context
	cancel       context.CancelFunc

	// 统计信息
	eventsProcessed int64
	eventsFailed    int64
	lastSaveTime    time.Time

	// 配置
	errorStrategy string // pause/skip
	saveInterval  time.Duration
	config        *TaskConfig // 任务配置，用于数据库和表名映射
}

// IncrementalConsumerConfig 消费者配置
type IncrementalConsumerConfig struct {
	TaskID        string
	Queue         BinlogQueue
	TargetDB      *sql.DB
	ErrorStrategy string        // pause/skip
	SaveInterval  time.Duration // 保存位置的间隔
	TaskConfig    *TaskConfig   // 任务配置
}

// NewIncrementalConsumer 创建增量消费引擎
func NewIncrementalConsumer(config *IncrementalConsumerConfig) *IncrementalConsumer {
	if config.SaveInterval == 0 {
		config.SaveInterval = 5 * time.Second // 默认 5 秒保存一次
	}

	if config.ErrorStrategy == "" {
		config.ErrorStrategy = "pause" // 默认遇到错误暂停
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IncrementalConsumer{
		taskID:        config.TaskID,
		queue:         config.Queue,
		targetDB:      config.TargetDB,
		logService:    NewTaskLogService(),
		sseService:    NewTaskSSEService(),
		statsService:  NewIncrementalStatsService(),
		ctx:           ctx,
		cancel:        cancel,
		errorStrategy: config.ErrorStrategy,
		saveInterval:  config.SaveInterval,
		lastSaveTime:  time.Now(),
		config:        config.TaskConfig,
	}
}

// Start 启动消费
func (c *IncrementalConsumer) Start() error {
	c.logService.Info(c.taskID, "增量消费引擎启动")

	// 初始化任务统计
	c.statsService.InitTaskStats(c.taskID)

	// 初始化所有表的统计（从配置中获取表列表）
	if c.config != nil {
		for _, dbSel := range c.config.SelectedDatabases {
			sourceDB := dbSel.SourceDatabase
			if sourceDB == "" {
				sourceDB = dbSel.Database
			}

			for _, tbl := range dbSel.Tables {
				// 为每个表创建初始统计
				c.statsService.InitTableStats(c.taskID, sourceDB, tbl.SourceTable)
			}
		}
	}

	// 启动消费循环
	go c.consumeLoop()

	// 启动定期保存位置的协程
	go c.periodicSave()

	return nil
}

// consumeLoop 消费循环
func (c *IncrementalConsumer) consumeLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.logService.Error(c.taskID, fmt.Sprintf("增量消费引擎 panic: %v", r))
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.logService.Info(c.taskID, "增量消费引擎收到停止信号")
			return

		default:
			// 从队列获取事件
			event, err := c.queue.Pop(c.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.logService.Error(c.taskID, fmt.Sprintf("从队列获取事件失败: %v", err))
				time.Sleep(1 * time.Second)
				continue
			}

			if event == nil {
				// 没有新事件，继续等待
				continue
			}

			// 处理事件
			if err := c.processEvent(event); err != nil {
				c.eventsFailed++
				c.logService.Error(c.taskID, fmt.Sprintf("处理事件失败: %v", err))

				// 更新失败统计
				eventTime := time.Unix(int64(event.Timestamp), 0)
				c.statsService.UpdateTaskStats(c.taskID, event.Type, event.BinlogFile, event.BinlogPos, eventTime, true)

				if c.errorStrategy == "pause" {
					c.logService.Error(c.taskID, "错误策略为 pause，停止消费")
					return
				}
				// skip 策略：记录错误但继续处理
			} else {
				c.eventsProcessed++

				// 更新统计
				eventTime := time.Unix(int64(event.Timestamp), 0)
				c.statsService.UpdateTaskStats(c.taskID, event.Type, event.BinlogFile, event.BinlogPos, eventTime, false)
				c.statsService.UpdateTableStats(c.taskID, event.Database, event.Table, event.Type, eventTime)
			}

			// ACK 确认
			if err := c.queue.Ack(event.ID); err != nil {
				c.logService.Error(c.taskID, fmt.Sprintf("ACK 事件失败: %v", err))
			}
		}
	}
}

// processEvent 处理单个事件
func (c *IncrementalConsumer) processEvent(event *BinlogEvent) error {
	switch event.Type {
	case "INSERT":
		return c.applyInsert(event)
	case "UPDATE":
		return c.applyUpdate(event)
	case "DELETE":
		return c.applyDelete(event)
	default:
		return fmt.Errorf("未知事件类型: %s", event.Type)
	}
}

// applyInsert 应用 INSERT 事件
func (c *IncrementalConsumer) applyInsert(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"INSERT 事件数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 INSERT 语句
	columns := make([]string, 0, len(event.Data))
	placeholders := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))
	updateClauses := make([]string, 0, len(event.Data))

	for col, val := range event.Data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
		// 构建 ON DUPLICATE KEY UPDATE 子句
		updateClauses = append(updateClauses, fmt.Sprintf("`%s`=VALUES(`%s`)", col, col))
	}

	// 使用 INSERT ... ON DUPLICATE KEY UPDATE 处理主键冲突
	query := fmt.Sprintf(
		"INSERT INTO `%s`.`%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		targetDB,
		targetTable,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
		joinStrings(updateClauses, ", "),
	)

	// 执行插入
	_, err := c.targetDB.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("执行 INSERT 失败: %v, SQL: %s", err, query)
	}

	return nil
}

// applyUpdate 应用 UPDATE 事件
func (c *IncrementalConsumer) applyUpdate(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"UPDATE 事件新数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 UPDATE 语句
	setClauses := make([]string, 0, len(event.Data))
	setValues := make([]interface{}, 0, len(event.Data))

	for col, val := range event.Data {
		setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col))
		setValues = append(setValues, val)
	}

	// 构建 WHERE 条件（使用旧值）
	whereClauses := make([]string, 0, len(event.OldData))
	whereValues := make([]interface{}, 0, len(event.OldData))

	for col, val := range event.OldData {
		if val == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
			whereValues = append(whereValues, val)
		}
	}

	// 如果没有 WHERE 条件，记录警告但仍然执行（使用新数据作为条件）
	if len(whereClauses) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"UPDATE 事件缺少旧数据，使用新数据作为 WHERE 条件: %s.%s",
			targetDB, targetTable,
		))
		for col, val := range event.Data {
			if val == nil {
				whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
				whereValues = append(whereValues, val)
			}
		}
	}

	query := fmt.Sprintf(
		"UPDATE `%s`.`%s` SET %s WHERE %s",
		targetDB,
		targetTable,
		joinStrings(setClauses, ", "),
		joinStrings(whereClauses, " AND "),
	)

	// 合并参数
	allValues := append(setValues, whereValues...)

	// 执行更新
	result, err := c.targetDB.Exec(query, allValues...)
	if err != nil {
		return fmt.Errorf("执行 UPDATE 失败: %v, SQL: %s", err, query)
	}

	// 检查是否更新了行（只在调试模式记录）
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 降低日志级别，这种情况在增量同步中很常见
		// c.logService.Warning(c.taskID, fmt.Sprintf("UPDATE 未影响任何行: %s.%s", targetDB, targetTable))
	}

	return nil
}

// applyDelete 应用 DELETE 事件
func (c *IncrementalConsumer) applyDelete(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"DELETE 事件数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 WHERE 条件
	whereClauses := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))

	for col, val := range event.Data {
		if val == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
			values = append(values, val)
		}
	}

	query := fmt.Sprintf(
		"DELETE FROM `%s`.`%s` WHERE %s",
		targetDB,
		targetTable,
		joinStrings(whereClauses, " AND "),
	)

	// 执行删除
	result, err := c.targetDB.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("执行 DELETE 失败: %v, SQL: %s", err, query)
	}

	// 检查是否删除了行（只在调试模式记录）
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 降低日志级别
		// c.logService.Warning(c.taskID, fmt.Sprintf("DELETE 未影响任何行: %s.%s", targetDB, targetTable))
	}

	return nil
}

// mapDatabaseAndTable 映射源数据库和表名到目标数据库和表名
// 返回: targetDB, targetTable, skip
// skip=true 表示该表不在同步范围内，应该跳过
func (c *IncrementalConsumer) mapDatabaseAndTable(sourceDB, sourceTable string) (string, string, bool) {
	if c.config == nil {
		// 没有配置，直接使用原名
		return sourceDB, sourceTable, false
	}

	// 遍历配置，查找匹配的数据库和表
	for _, dbSel := range c.config.SelectedDatabases {
		// 检查源数据库是否匹配
		if dbSel.SourceDatabase != sourceDB {
			continue
		}

		// 找到匹配的数据库，获取目标数据库名
		targetDB := dbSel.Database

		// 查找匹配的表
		for _, tbl := range dbSel.Tables {
			if tbl.SourceTable == sourceTable {
				// 找到匹配的表
				targetTable := tbl.TargetTable
				if targetTable == "" {
					targetTable = sourceTable // 如果没有指定目标表名，使用源表名
				}
				return targetDB, targetTable, false
			}
		}

		// 数据库匹配但表不匹配，跳过
		return "", "", true
	}

	// 数据库不在同步范围内，跳过
	return "", "", true
}

// periodicSave 定期保存位置和统计信息
func (c *IncrementalConsumer) periodicSave() {
	ticker := time.NewTicker(c.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.saveStatistics()
		}
	}
}

// saveStatistics 保存统计信息
func (c *IncrementalConsumer) saveStatistics() {
	// 只在有新事件时才推送进度更新
	// 避免无意义的 SSE 推送
	if c.eventsProcessed > 0 || c.eventsFailed > 0 {
		c.sseService.BroadcastProgressUpdate(c.taskID)
	}
	c.lastSaveTime = time.Now()
}

// Stop 停止消费
func (c *IncrementalConsumer) Stop() error {
	c.logService.Info(c.taskID, "正在停止增量消费引擎...")

	// 取消上下文
	c.cancel()

	// 最后保存一次统计信息
	c.saveStatistics()

	// 清理 Redis 统计数据
	c.statsService.ClearTaskStats(c.taskID)

	c.logService.Info(c.taskID, fmt.Sprintf(
		"增量消费引擎已停止，共处理 %d 个事件，失败 %d 个",
		c.eventsProcessed,
		c.eventsFailed,
	))

	return nil
}

// GetStatistics 获取统计信息
func (c *IncrementalConsumer) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"events_processed": c.eventsProcessed,
		"events_failed":    c.eventsFailed,
		"last_save_time":   c.lastSaveTime,
	}
}

// joinStrings 连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}
