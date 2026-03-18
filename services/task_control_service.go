package services

import (
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"sync"
	"time"
)

// TaskControlService 任务控制服务
type TaskControlService struct{}

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

	// 3. 根据同步模式选择执行路径
	switch task.SyncMode {
	case "incremental":
		return s.startIncrementalTask(taskID)
	case "structure":
		// 结构同步：对比两边表结构，使用 ALTER 增删改（功能开发中）
		return fmt.Errorf("结构同步功能开发中，敬请期待")
	default:
		// 全量同步
		return s.startFullSyncTask(taskID)
	}
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

	// 3. 根据同步模式处理
	if task.SyncMode == "incremental" {
		// 增量同步不支持暂停
		return fmt.Errorf("增量同步不支持暂停，请使用停止功能")
	}

	// 4. 处理全量同步暂停
	execManager := GetExecutionManager()
	if exec, ok := execManager.GetExecution(taskID); ok {
		exec.Cancel()         // 发送取消信号
		exec.WaitGroup.Wait() // 等待所有Worker退出
	}

	// 5. 更新任务状态
	task.IsRunning = false
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	return nil
}

// StopTask 停止任务
func (s *TaskControlService) StopTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	execManager := GetExecutionManager()
	progressManager := GetProgressManager()
	statsService := NewIncrementalStatsService()

	// 2. 根据同步模式处理
	if task.SyncMode == "incremental" {
		// 处理增量同步停止
		if incrementalSync, ok := execManager.GetIncrementalSync(taskID); ok {
			// 停止增量同步
			if err := incrementalSync.Stop(); err != nil {
				return fmt.Errorf("停止增量同步失败: %v", err)
			}
			// 等待增量同步停止
			time.Sleep(1 * time.Second)
			// 清理增量同步实例
			execManager.DeleteIncrementalSync(taskID)
		}

		// 清理Redis增量统计数据
		statsService.ClearTaskStats(taskID)

		// 注意：不清除内存中的全量进度数据，因为增量阶段还需要显示全量进度
		// progressManager.ClearTask(taskID) // 已删除
	} else {
		// 处理全量同步停止
		if task.IsRunning {
			if exec, ok := execManager.GetExecution(taskID); ok {
				exec.Cancel()         // 发送取消信号
				exec.WaitGroup.Wait() // 等待所有Worker退出
			}
		}

		// 清除内存中的进度数据
		progressManager.ClearTask(taskID)
	}

	// 3. 更新任务状态
	task.IsRunning = false
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	return nil
}

// GetIncrementalSyncStatus 获取增量同步状态
func (s *TaskControlService) GetIncrementalSyncStatus(taskID string) (map[string]interface{}, error) {
	execManager := GetExecutionManager()
	if incrementalSync, ok := execManager.GetIncrementalSync(taskID); ok {
		return incrementalSync.GetStatus(), nil
	}
	return nil, fmt.Errorf("增量同步实例不存在")
}
