package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"time"
)

// IncrementalConsumer 增量消费引擎
type IncrementalConsumer struct {
	taskID     string
	queue      BinlogQueue
	targetDB   *sql.DB
	logService *TaskLogService
	sseService *TaskSSEService
	ctx        context.Context
	cancel     context.CancelFunc

	// 统计信息
	eventsProcessed int64
	eventsFailed    int64
	lastSaveTime    time.Time

	// 配置
	errorStrategy string // pause/skip
	saveInterval  time.Duration
}

// IncrementalConsumerConfig 消费者配置
type IncrementalConsumerConfig struct {
	TaskID        string
	Queue         BinlogQueue
	TargetDB      *sql.DB
	ErrorStrategy string        // pause/skip
	SaveInterval  time.Duration // 保存位置的间隔
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
		ctx:           ctx,
		cancel:        cancel,
		errorStrategy: config.ErrorStrategy,
		saveInterval:  config.SaveInterval,
		lastSaveTime:  time.Now(),
	}
}

// Start 启动消费
func (c *IncrementalConsumer) Start() error {
	c.logService.Info(c.taskID, "增量消费引擎启动")

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

				if c.errorStrategy == "pause" {
					c.logService.Error(c.taskID, "错误策略为 pause，停止消费")
					return
				}
				// skip 策略：记录错误但继续处理
			} else {
				c.eventsProcessed++
			}

			// ACK 确认
			if err := c.queue.Ack(event.ID); err != nil {
				c.logService.Error(c.taskID, fmt.Sprintf("ACK 事件失败: %v", err))
			}

			// 更新 Binlog 位置
			c.updatePosition(event.BinlogFile, event.BinlogPos)
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
	// 构建 INSERT 语句
	columns := make([]string, 0, len(event.Data))
	placeholders := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))

	for col, val := range event.Data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		event.Database,
		event.Table,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
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

	query := fmt.Sprintf(
		"UPDATE `%s`.`%s` SET %s WHERE %s",
		event.Database,
		event.Table,
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

	// 检查是否更新了行
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf("UPDATE 未影响任何行: %s.%s", event.Database, event.Table))
	}

	return nil
}

// applyDelete 应用 DELETE 事件
func (c *IncrementalConsumer) applyDelete(event *BinlogEvent) error {
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
		event.Database,
		event.Table,
		joinStrings(whereClauses, " AND "),
	)

	// 执行删除
	result, err := c.targetDB.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("执行 DELETE 失败: %v, SQL: %s", err, query)
	}

	// 检查是否删除了行
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf("DELETE 未影响任何行: %s.%s", event.Database, event.Table))
	}

	return nil
}

// updatePosition 更新 Binlog 位置
func (c *IncrementalConsumer) updatePosition(file string, pos uint32) {
	// 更新到数据库
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", c.taskID).
		Updates(map[string]interface{}{
			"current_binlog_file":        file,
			"current_binlog_pos":         pos,
			"incremental_events_applied": c.eventsProcessed,
			"updated_at":                 time.Now(),
		})
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
	// 计算延迟（简化版，实际应该比较 Binlog 位置）
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", c.taskID).Error; err != nil {
		return
	}

	// 更新统计信息
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", c.taskID).
		Updates(map[string]interface{}{
			"incremental_events_total":   c.eventsProcessed + c.eventsFailed,
			"incremental_events_applied": c.eventsProcessed,
			"updated_at":                 time.Now(),
		})

	// 推送进度更新
	c.sseService.BroadcastProgressUpdate(c.taskID)

	c.lastSaveTime = time.Now()
}

// Stop 停止消费
func (c *IncrementalConsumer) Stop() error {
	c.logService.Info(c.taskID, "正在停止增量消费引擎...")

	// 取消上下文
	c.cancel()

	// 最后保存一次统计信息
	c.saveStatistics()

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
