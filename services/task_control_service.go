package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskExecution 任务执行信息
type TaskExecution struct {
	TaskID    string
	Cancel    context.CancelFunc
	TaskQueue chan *models.TaskUnitRuntime
	WaitGroup *sync.WaitGroup
}

// TaskControlService 任务控制服务
type TaskControlService struct {
	executions sync.Map // map[taskID]*TaskExecution
}

var (
	taskControlInstance *TaskControlService
	taskControlOnce     sync.Once
)

// NewTaskControlService 获取任务控制服务单例
func NewTaskControlService() *TaskControlService {
	taskControlOnce.Do(func() {
		taskControlInstance = &TaskControlService{}
	})
	return taskControlInstance
}

// StartTask 启动任务
func (s *TaskControlService) StartTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 检查任务状态
	if task.IsRunning {
		return fmt.Errorf("任务已在运行中")
	}

	if task.Status != "configured" {
		return fmt.Errorf("任务未配置，无法启动")
	}

	// 3. 检查是否所有任务单元都已完成（允许重新启动）
	var completedCount int64
	var totalCount int64
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ?", taskID).Count(&totalCount)
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ? AND status = ?", taskID, "completed").Count(&completedCount)

	// 如果所有任务单元都已完成，重置它们的状态为pending
	if totalCount > 0 && completedCount == totalCount {
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ?", taskID).
			Updates(map[string]interface{}{
				"status":            "pending",
				"processed_records": 0,
				"total_records":     0,
			})
	}

	// 4. 清理之前的日志，避免数据串台
	logService := NewTaskLogService()
	logService.mu.Lock()
	delete(logService.logs, taskID)
	logService.mu.Unlock()

	// 5. 解析配置获取线程数
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("解析任务配置失败: %w", err)
	}
	threadCount := config.SyncConfig.ThreadCount
	if threadCount <= 0 {
		threadCount = 1 // 默认1个线程
	}

	// 6. 初始化任务单元运行记录（如果是首次启动）
	if err := s.initTaskUnits(taskID); err != nil {
		return fmt.Errorf("初始化任务单元失败: %w", err)
	}

	// 7. 查询所有待处理的任务单元
	var units []models.TaskUnitRuntime
	database.DB.Where("task_id = ? AND status IN ?", taskID, []string{"pending", "failed"}).Find(&units)

	if len(units) == 0 {
		return fmt.Errorf("没有待处理的任务单元")
	}

	// 6. 更新任务为运行状态
	task.IsRunning = true
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 7. 创建任务队列和context
	ctx, cancel := context.WithCancel(context.Background())
	taskQueue := make(chan *models.TaskUnitRuntime, len(units))

	execution := &TaskExecution{
		TaskID:    taskID,
		Cancel:    cancel,
		TaskQueue: taskQueue,
		WaitGroup: &sync.WaitGroup{},
	}
	s.executions.Store(taskID, execution)

	// 8. 将所有任务单元放入队列
	for i := range units {
		taskQueue <- &units[i]
	}
	close(taskQueue) // 关闭队列，表示没有更多任务

	// 9. 启动Worker Pool
	engine := NewSyncEngine()
	for i := 0; i < threadCount; i++ {
		execution.WaitGroup.Add(1)
		go func(workerID int) {
			defer execution.WaitGroup.Done()
			engine.Worker(ctx, taskID, taskQueue, workerID)
		}(i)
	}

	// 10. 启动监控goroutine，等待所有Worker完成
	go func() {
		execution.WaitGroup.Wait()
		s.executions.Delete(taskID)

		// 所有Worker完成，更新is_running为false
		database.DB.Model(&models.SyncTask{}).
			Where("id = ?", taskID).
			Update("is_running", false)
	}()

	return nil
}

// PauseTask 暂停任务
func (s *TaskControlService) PauseTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 检查任务状态
	if !task.IsRunning {
		return fmt.Errorf("任务未在运行中")
	}

	// 3. 发送取消信号给所有Worker
	if exec, ok := s.executions.Load(taskID); ok {
		execution := exec.(*TaskExecution)
		execution.Cancel()         // 发送取消信号
		execution.WaitGroup.Wait() // 等待所有Worker退出
	}

	// 4. 更新任务状态
	task.IsRunning = false
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 5. 更新运行中的任务单元状态为paused
	database.DB.Model(&models.TaskUnitRuntime{}).
		Where("task_id = ? AND status = ?", taskID, "running").
		Update("status", "paused")

	return nil
}

// StopTask 停止任务
func (s *TaskControlService) StopTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 发送取消信号给所有Worker（如果任务正在运行）
	if task.IsRunning {
		if exec, ok := s.executions.Load(taskID); ok {
			execution := exec.(*TaskExecution)
			execution.Cancel()         // 发送取消信号
			execution.WaitGroup.Wait() // 等待所有Worker退出
		}
	}

	// 3. 更新任务状态
	task.IsRunning = false
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 4. 清除任务单元的状态，而不是删除它们
	database.DB.Model(&models.TaskUnitRuntime{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":            "pending",
			"processed_records": 0,
			"total_records":     0,
			"started_at":        nil,
			"updated_at":        time.Now(),
		})

	return nil
}

// initTaskUnits 初始化任务单元运行记录
func (s *TaskControlService) initTaskUnits(taskID string) error {
	// 检查是否已有运行记录
	var count int64
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ?", taskID).Count(&count)
	if count > 0 {
		// 已有记录，更新状态为pending
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ? AND status IN ?", taskID, []string{"paused", "failed"}).
			Update("status", "pending")
		return nil
	}

	// 查询任务单元配置
	var configs []models.TaskUnitConfig
	if err := database.DB.Where("task_id = ?", taskID).Find(&configs).Error; err != nil {
		return err
	}

	if len(configs) == 0 {
		return fmt.Errorf("任务没有配置单元")
	}

	// 创建运行记录（pending状态不设置StartedAt，等实际开始运行时再设置）
	now := time.Now()
	var runtimes []models.TaskUnitRuntime
	for _, config := range configs {
		runtime := models.TaskUnitRuntime{
			ID:               uuid.New().String(),
			TaskID:           taskID,
			UnitName:         config.UnitName,
			Status:           "pending",
			TotalRecords:     0,
			ProcessedRecords: 0,
			StartedAt:        nil,
			UpdatedAt:        now,
		}
		runtimes = append(runtimes, runtime)
	}

	// 批量插入
	if err := database.DB.Create(&runtimes).Error; err != nil {
		return err
	}

	return nil
}
