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

	// 初始化所有表的统计（使用目标数据库和目标表名）
	if c.config != nil {
		for _, dbSel := range c.config.SelectedDatabases {
			targetDB := dbSel.Database

			for _, tbl := range dbSel.Tables {
				targetTable := tbl.TargetTable
				if targetTable == "" {
					targetTable = tbl.SourceTable
				}
				// 为每个表创建初始统计（使用目标表名）
				c.statsService.InitTableStats(c.taskID, targetDB, targetTable)
			}
		}
	}

	// 启动消费循环
	go c.consumeLoop()

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

				// 映射到目标表名
				targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
				if !skip {
					// 更新统计（使用目标表名）
					eventTime := time.Unix(int64(event.Timestamp), 0)
					c.statsService.UpdateTaskStats(c.taskID, event.Type, event.BinlogFile, event.BinlogPos, eventTime, false)
					c.statsService.UpdateTableStats(c.taskID, targetDB, targetTable, event.Type, eventTime)

					// 事件驱动推送：每处理一个事件立即推送进度更新
					c.sseService.BroadcastProgressUpdate(c.taskID)
				}
			}

			// ACK 确认
			if err := c.queue.Ack(event.ID); err != nil {
				c.logService.Error(c.taskID, fmt.Sprintf("ACK 事件失败: %v", err))
			}
		}
	}
}

// Stop 停止消费
func (c *IncrementalConsumer) Stop() error {
	c.logService.Info(c.taskID, "正在停止增量消费引擎...")

	// 取消上下文
	c.cancel()

	c.logService.Info(c.taskID, fmt.Sprintf(
		"增量消费引擎已停止，共处理 %d 个事件，失败 %d 个",
		c.eventsProcessed,
		c.eventsFailed,
	))

	return nil
}
