package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"time"
)

// IncrementalSync 增量同步引擎总控
type IncrementalSync struct {
	taskID     string
	task       *models.SyncTask
	config     *TaskConfig
	logService *TaskLogService
	sseService *TaskSSEService
	dsService  *DataSourceService

	// 组件
	queue    BinlogQueue
	listener *BinlogListener
	consumer *IncrementalConsumer
	fullSync *SyncEngine

	// 数据库连接
	sourceDB *sql.DB
	targetDB *sql.DB

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// NewIncrementalSync 创建增量同步引擎
func NewIncrementalSync(taskID string) (*IncrementalSync, error) {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").
		First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("查询任务失败: %v", err)
	}

	// 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IncrementalSync{
		taskID:     taskID,
		task:       &task,
		config:     &config,
		logService: NewTaskLogService(),
		sseService: NewTaskSSEService(),
		dsService:  NewDataSourceService(),
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Start 启动增量同步（使用内部 context）
func (s *IncrementalSync) Start() error {
	return s.StartWithContext(s.ctx)
}

// StartWithContext 启动增量同步（使用外部 context）
func (s *IncrementalSync) StartWithContext(ctx context.Context) error {
	// 使用传入的 context
	s.ctx = ctx

	s.logService.Info(s.taskID, "========== 增量同步开始 ==========")

	// 1. 初始化数据库连接
	if err := s.initDatabaseConnections(); err != nil {
		return fmt.Errorf("初始化数据库连接失败: %v", err)
	}
	defer s.closeDatabaseConnections()

	// 2. 检查源库 Binlog 配置
	if err := s.checkBinlogConfig(); err != nil {
		return fmt.Errorf("检查 Binlog 配置失败: %v", err)
	}

	// 3. 捕获快照点
	snapshot, err := s.captureSnapshot()
	if err != nil {
		return fmt.Errorf("捕获快照点失败: %v", err)
	}

	// 4. 创建队列
	if err := s.createQueue(); err != nil {
		return fmt.Errorf("创建队列失败: %v", err)
	}
	defer s.queue.Close()

	// 5. 启动 Binlog 监听器（异步）
	if err := s.startBinlogListener(snapshot); err != nil {
		return fmt.Errorf("启动 Binlog 监听器失败: %v", err)
	}
	defer s.listener.Stop()

	// 6. 执行全量同步
	s.logService.Info(s.taskID, "开始执行全量同步...")
	if err := s.executeFullSync(); err != nil {
		return fmt.Errorf("全量同步失败: %v", err)
	}

	// 7. 标记全量完成
	s.markFullSyncCompleted()

	// 8. 启动增量消费
	s.logService.Info(s.taskID, "全量同步完成，开始增量消费...")
	if err := s.startIncrementalConsumer(); err != nil {
		return fmt.Errorf("启动增量消费失败: %v", err)
	}
	defer s.consumer.Stop()

	// 9. 持续运行，直到收到停止信号
	s.logService.Info(s.taskID, "增量同步运行中...")
	<-s.ctx.Done()

	s.logService.Info(s.taskID, "========== 增量同步结束 ==========")
	return nil
}

// BinlogSnapshot Binlog 快照
type BinlogSnapshot struct {
	File      string
	Position  uint32
	Timestamp time.Time
}

// 辅助方法已拆分到 incremental_sync_helpers.go

// Stop 停止增量同步
func (s *IncrementalSync) Stop() error {
	s.logService.Info(s.taskID, "正在停止增量同步...")

	// 取消上下文
	s.cancel()

	// 停止各个组件
	if s.consumer != nil {
		s.consumer.Stop()
	}

	if s.listener != nil {
		s.listener.Stop()
	}

	if s.queue != nil {
		s.queue.Close()
	}

	s.logService.Info(s.taskID, "增量同步已停止")
	return nil
}

// GetStatus 获取增量同步状态
func (s *IncrementalSync) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"task_id": s.taskID,
	}

	// 获取队列大小
	if s.queue != nil {
		if size, err := s.queue.Size(); err == nil {
			status["queue_size"] = size
		}
	}

	// 获取 Binlog 位置
	if s.listener != nil {
		file, pos := s.listener.GetCurrentPosition()
		status["binlog_file"] = file
		status["binlog_pos"] = pos
	}

	// 获取消费统计
	if s.consumer != nil {
		stats := s.consumer.GetStatistics()
		status["consumer_stats"] = stats
	}

	return status
}
