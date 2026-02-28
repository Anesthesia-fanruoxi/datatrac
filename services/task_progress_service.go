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

// TaskProgress 任务进度信息
type TaskProgress struct {
	TaskID           string      `json:"task_id"`
	TaskName         string      `json:"task_name"`
	Status           string      `json:"status"`
	SyncMode         string      `json:"sync_mode"`
	OverallProgress  float64     `json:"overall_progress"`  // 总体进度百分比
	TotalTables      int         `json:"total_tables"`      // 总表数
	CompletedTables  int         `json:"completed_tables"`  // 已完成表数
	CurrentTable     string      `json:"current_table"`     // 当前同步表
	CurrentProgress  float64     `json:"current_progress"`  // 当前表进度百分比
	TotalRecords     int64       `json:"total_records"`     // 当前表总记录数
	ProcessedRecords int64       `json:"processed_records"` // 当前表已处理记录数
	SyncSpeed        int64       `json:"sync_speed"`        // 同步速度（条/秒）
	ElapsedTime      string      `json:"elapsed_time"`      // 已用时间
	EstimatedTime    string      `json:"estimated_time"`    // 预计剩余时间
	StartedAt        *time.Time  `json:"started_at"`        // 开始时间
	TableUnits       []TableUnit `json:"table_units"`       // 任务单元列表
}

// TableUnit 表单元信息
type TableUnit struct {
	Name             string  `json:"name"`              // 表名
	Status           string  `json:"status"`            // pending/running/completed/failed/paused
	TotalRecords     int64   `json:"total_records"`     // 总记录数
	ProcessedRecords int64   `json:"processed_records"` // 已处理记录数
	Progress         float64 `json:"progress"`          // 进度百分比
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

	// 计算进度
	progress := &TaskProgress{
		TaskID:      taskID,
		TaskName:    task.Name,
		Status:      task.Status,
		SyncMode:    task.SyncMode,
		TotalTables: totalTables,
	}

	// 统计已完成表数
	completedCount := 0
	var currentUnit *models.TaskUnitRuntime
	var totalProcessed int64
	var totalRecords int64

	for i := range units {
		unit := &units[i]
		if unit.Status == "completed" {
			completedCount++
		} else if unit.Status == "running" && currentUnit == nil {
			currentUnit = unit
		}
		totalProcessed += unit.ProcessedRecords
		totalRecords += unit.TotalRecords
	}

	progress.CompletedTables = completedCount

	// 计算总体进度
	if totalTables > 0 {
		progress.OverallProgress = float64(completedCount) / float64(totalTables) * 100
	}

	// 构建任务单元列表（基于实际的运行记录）
	tableUnits := make([]TableUnit, 0, len(units))
	for _, runtime := range units {
		// 计算进度百分比
		unitProgress := 0.0
		if runtime.TotalRecords > 0 {
			unitProgress = float64(runtime.ProcessedRecords) / float64(runtime.TotalRecords) * 100
		}

		unit := TableUnit{
			Name:             runtime.UnitName,
			Status:           runtime.Status,
			TotalRecords:     runtime.TotalRecords,
			ProcessedRecords: runtime.ProcessedRecords,
			Progress:         unitProgress,
		}

		tableUnits = append(tableUnits, unit)
	}
	progress.TableUnits = tableUnits

	// 当前表信息
	if currentUnit != nil {
		progress.CurrentTable = currentUnit.UnitName
		progress.TotalRecords = currentUnit.TotalRecords
		progress.ProcessedRecords = currentUnit.ProcessedRecords

		if currentUnit.TotalRecords > 0 {
			progress.CurrentProgress = float64(currentUnit.ProcessedRecords) / float64(currentUnit.TotalRecords) * 100
		}

		// 计算同步速度和时间
		if currentUnit.StartedAt != nil {
			progress.StartedAt = currentUnit.StartedAt
			elapsed := time.Since(*currentUnit.StartedAt)
			progress.ElapsedTime = formatDuration(elapsed)

			// 计算速度（条/秒）
			if elapsed.Seconds() > 0 {
				progress.SyncSpeed = int64(float64(totalProcessed) / elapsed.Seconds())
			}

			// 估算剩余时间
			if progress.SyncSpeed > 0 && totalRecords > totalProcessed {
				remaining := totalRecords - totalProcessed
				estimatedSeconds := float64(remaining) / float64(progress.SyncSpeed)
				progress.EstimatedTime = formatDuration(time.Duration(estimatedSeconds) * time.Second)
			}
		}
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
