package services

import (
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"time"
)

// TaskProgressService 任务进度服务
type TaskProgressService struct{}

// NewTaskProgressService 创建任务进度服务
func NewTaskProgressService() *TaskProgressService {
	return &TaskProgressService{}
}

// TaskProgress 任务进度信息（简化版，减少前端渲染压力）
type TaskProgress struct {
	TaskID           string     `json:"task_id"`
	TaskName         string     `json:"task_name"`
	Status           string     `json:"status"`
	SyncMode         string     `json:"sync_mode"`
	CurrentStep      string     `json:"current_step"`      // 当前步骤: initialize/sync_data/validate
	TotalTables      int        `json:"total_tables"`      // 总表数
	CompletedTables  int        `json:"completed_tables"`  // 已完成表数
	RunningTables    int        `json:"running_tables"`    // 运行中表数
	FailedTables     int        `json:"failed_tables"`     // 失败表数
	TotalRecords     int64      `json:"total_records"`     // 总记录数
	ProcessedRecords int64      `json:"processed_records"` // 已处理记录数
	OverallProgress  float64    `json:"overall_progress"`  // 总体进度百分比
	SyncSpeed        int64      `json:"sync_speed"`        // 同步速度（条/秒）
	ElapsedTime      string     `json:"elapsed_time"`      // 已用时间
	EstimatedTime    string     `json:"estimated_time"`    // 预计剩余时间
	StartedAt        *time.Time `json:"started_at"`        // 开始时间
}

// TableUnit 表单元信息（已废弃，不再使用）
// 保留此结构体定义以保持向后兼容，但不再填充数据
type TableUnit struct {
	Name             string  `json:"name"`
	Status           string  `json:"status"`
	TotalRecords     int64   `json:"total_records"`
	ProcessedRecords int64   `json:"processed_records"`
	Progress         float64 `json:"progress"`
}

// GetTaskProgress 获取任务进度
func (s *TaskProgressService) GetTaskProgress(taskID string) (*TaskProgress, error) {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在")
	}

	// 查询任务单元配置
	var unitConfigs []models.TaskUnitConfig
	database.DB.Where("task_id = ?", taskID).Find(&unitConfigs)

	// 查询任务单元运行记录
	var units []models.TaskUnitRuntime
	database.DB.Where("task_id = ?", taskID).Find(&units)

	// 使用任务单元运行记录的数量作为总表数
	totalTables := len(units)

	// 创建运行记录映射
	unitRuntimeMap := make(map[string]*models.TaskUnitRuntime)
	for i := range units {
		unitRuntimeMap[units[i].UnitName] = &units[i]
	}

	// 统计各状态表数和总记录数
	completedCount := 0
	runningCount := 0
	failedCount := 0
	initializedCount := 0 // 已初始化的表数
	var totalProcessed int64
	var totalRecords int64
	var earliestStartTime *time.Time

	for i := range units {
		unit := &units[i]
		switch unit.Status {
		case "completed":
			completedCount++
		case "running":
			runningCount++
		case "failed":
			failedCount++
		case "initialized":
			initializedCount++
		}

		totalProcessed += unit.ProcessedRecords
		totalRecords += unit.TotalRecords

		// 找到最早的开始时间
		if unit.StartedAt != nil {
			if earliestStartTime == nil || unit.StartedAt.Before(*earliestStartTime) {
				earliestStartTime = unit.StartedAt
			}
		}
	}

	// 如果当前是初始化阶段，使用initializedCount作为completedCount
	if task.CurrentStep == "initialize" {
		completedCount = initializedCount
	}

	// 计算总体进度（基于记录数）
	overallProgress := 0.0
	if totalRecords > 0 {
		overallProgress = float64(totalProcessed) / float64(totalRecords) * 100
	}

	// 计算同步速度和时间
	syncSpeed := int64(0)
	elapsedTime := ""
	estimatedTime := ""

	if earliestStartTime != nil {
		elapsed := time.Since(*earliestStartTime)
		elapsedTime = formatDuration(elapsed)

		// 计算速度（条/秒）
		if elapsed.Seconds() > 0 {
			syncSpeed = int64(float64(totalProcessed) / elapsed.Seconds())
		}

		// 估算剩余时间
		if syncSpeed > 0 && totalRecords > totalProcessed {
			remaining := totalRecords - totalProcessed
			estimatedSeconds := float64(remaining) / float64(syncSpeed)
			estimatedTime = formatDuration(time.Duration(estimatedSeconds) * time.Second)
		}
	}

	// 构建简化的进度信息
	progress := &TaskProgress{
		TaskID:           taskID,
		TaskName:         task.Name,
		Status:           task.Status,
		SyncMode:         task.SyncMode,
		CurrentStep:      task.CurrentStep,
		TotalTables:      totalTables,
		CompletedTables:  completedCount,
		RunningTables:    runningCount,
		FailedTables:     failedCount,
		TotalRecords:     totalRecords,
		ProcessedRecords: totalProcessed,
		OverallProgress:  overallProgress,
		SyncSpeed:        syncSpeed,
		ElapsedTime:      elapsedTime,
		EstimatedTime:    estimatedTime,
		StartedAt:        earliestStartTime,
	}

	return progress, nil
}

// formatDuration 格式化时间间隔为 HH:MM:SS
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
