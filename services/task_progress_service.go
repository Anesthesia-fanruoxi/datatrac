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

// TaskProgress 任务进度信息（根据同步模式返回不同结构）
type TaskProgress struct {
	TaskID      string `json:"task_id"`
	SyncMode    string `json:"sync_mode"`
	CurrentStep string `json:"current_step"`

	// 全量同步进度（仅 sync_mode=full 时有值）
	OverallProgress  *float64 `json:"overall_progress,omitempty"`
	SyncSpeed        *int64   `json:"sync_speed,omitempty"`
	ElapsedTime      *string  `json:"elapsed_time,omitempty"`
	EstimatedTime    *string  `json:"estimated_time,omitempty"`
	TotalTables      *int     `json:"total_tables,omitempty"`
	CompletedTables  *int     `json:"completed_tables,omitempty"`
	TotalRecords     *int64   `json:"total_records,omitempty"`
	ProcessedRecords *int64   `json:"processed_records,omitempty"`

	// 全量同步目标源级别统计（仅 sync_mode=full 时有值）
	TargetStats []*TargetProgress `json:"target_stats,omitempty"`

	// 增量同步数据库级别统计（仅 sync_mode=incremental 时有值，始终返回）
	DatabaseStats []*IncrementalDatabaseStats `json:"database_stats,omitempty"`

	// 增量同步表明细统计（仅 sync_mode=incremental 且指定 database 参数时有值）
	TableStats []*IncrementalTableStats `json:"table_stats,omitempty"`
}

// GetTaskProgress 获取任务进度
// dbName 参数可选，如果提供则同时返回该数据库的表明细
func (s *TaskProgressService) GetTaskProgress(taskID string, dbName string) (*TaskProgress, error) {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在")
	}

	// 构建进度信息
	progress := &TaskProgress{
		TaskID:      taskID,
		SyncMode:    task.SyncMode,
		CurrentStep: task.CurrentStep,
	}

	// 根据同步模式返回不同的数据
	if task.SyncMode == "incremental" {
		// 增量同步模式：始终返回数据库级别统计
		progress.DatabaseStats = s.getIncrementalDatabaseStats(taskID, &task)

		// 如果指定了 dbName 参数，则返回该数据库的表明细
		if dbName != "" {
			tableStats, _ := s.GetTaskProgressByDatabase(taskID, dbName)
			progress.TableStats = tableStats
		}
	} else {
		// 全量同步模式：返回整体进度
		s.fillFullSyncProgress(taskID, &task, progress)

		// 全量同步模式：获取目标源级别统计
		progress.TargetStats = progressManager.GetAllTargetStats(taskID)
	}

	return progress, nil
}

// getIncrementalDatabaseStats 获取增量同步的数据库级别统计（合计）
func (s *TaskProgressService) getIncrementalDatabaseStats(taskID string, task *models.SyncTask) []*IncrementalDatabaseStats {
	statsService := NewIncrementalStatsService()

	// 从Redis获取数据库级别统计
	dbStats, _ := statsService.GetDatabaseStatsList(taskID)

	// 如果 Redis 中没有数据，从任务配置中初始化数据库列表
	if len(dbStats) == 0 {
		dbStats = statsService.InitDatabaseStatsFromConfig(taskID, task)
	}

	// 从内存合并全量进度
	dbStats = s.mergeMemoryProgressToDatabaseStats(taskID, dbStats)

	return dbStats
}

// mergeMemoryProgressToDatabaseStats 将内存中的全量进度合并到数据库统计
func (s *TaskProgressService) mergeMemoryProgressToDatabaseStats(taskID string, dbStats []*IncrementalDatabaseStats) []*IncrementalDatabaseStats {
	progressManager := GetProgressManager()
	units := progressManager.GetUnits(taskID)

	if units == nil {
		return dbStats
	}

	// 按数据库分组统计全量进度
	dbProgressMap := make(map[string]struct {
		totalRecords     int64
		processedRecords int64
	})

	for _, unit := range units {
		// 从 unitName 中提取数据库名（格式：database.table）
		dbName := ""
		for i := 0; i < len(unit.UnitName); i++ {
			if unit.UnitName[i] == '.' {
				dbName = unit.UnitName[:i]
				break
			}
		}

		if dbName == "" {
			continue
		}

		progress := dbProgressMap[dbName]
		progress.totalRecords += unit.TotalRecords
		progress.processedRecords += unit.ProcessedRecords
		dbProgressMap[dbName] = progress
	}

	// 合并到数据库统计
	for _, stats := range dbStats {
		if progress, exists := dbProgressMap[stats.Database]; exists {
			stats.FullSyncTotalRecords = progress.totalRecords
			stats.FullSyncProcessedRecords = progress.processedRecords

			// 计算进度
			if progress.totalRecords > 0 {
				stats.FullSyncProgress = float64(progress.processedRecords) / float64(progress.totalRecords) * 100
			} else {
				stats.FullSyncProgress = 0
			}
		}
	}

	return dbStats
}

// getIncrementalTableStats 获取增量同步的表明细统计（用于展开数据库时调用）
func (s *TaskProgressService) getIncrementalTableStats(taskID string, task *models.SyncTask) []*IncrementalTableStats {
	statsService := NewIncrementalStatsService()

	// 先从Redis获取表统计
	tableStats, _ := statsService.GetTableStatsList(taskID)

	// 如果 Redis 中没有数据，从任务配置中初始化表列表
	if len(tableStats) == 0 {
		tableStats = statsService.InitTableStatsFromConfig(taskID, task)
	}

	// 始终从内存合并全量进度（全量阶段和增量阶段都需要）
	tableStats = s.mergeMemoryProgressToTableStats(taskID, tableStats)

	return tableStats
}

// mergeMemoryProgressToTableStats 将内存中的全量进度合并到表统计
func (s *TaskProgressService) mergeMemoryProgressToTableStats(taskID string, tableStats []*IncrementalTableStats) []*IncrementalTableStats {
	progressManager := GetProgressManager()
	units := progressManager.GetUnits(taskID)

	if units == nil {
		return tableStats
	}

	// 创建表名到运行时数据的映射
	unitMap := make(map[string]*TaskUnit)
	for _, unit := range units {
		unitMap[unit.UnitName] = unit
	}

	// 合并数据
	for _, stats := range tableStats {
		tableName := fmt.Sprintf("%s.%s", stats.Database, stats.Table)

		if unit, exists := unitMap[tableName]; exists {
			stats.FullSyncTotalRecords = unit.TotalRecords
			stats.FullSyncProcessedRecords = unit.ProcessedRecords

			// 计算进度
			if unit.TotalRecords > 0 {
				stats.FullSyncProgress = float64(unit.ProcessedRecords) / float64(unit.TotalRecords) * 100
			} else {
				stats.FullSyncProgress = 0
			}
		}
	}

	return tableStats
}

// fillFullSyncProgress 填充全量同步的整体进度
func (s *TaskProgressService) fillFullSyncProgress(taskID string, task *models.SyncTask, progress *TaskProgress) {
	progressManager := GetProgressManager()
	taskData := progressManager.GetTask(taskID)

	if taskData == nil {
		// 任务未在运行，返回空进度
		zero := 0
		zeroInt64 := int64(0)
		zeroFloat := 0.0
		emptyStr := "00:00:00"
		progress.OverallProgress = &zeroFloat
		progress.SyncSpeed = &zeroInt64
		progress.ElapsedTime = &emptyStr
		progress.EstimatedTime = &emptyStr
		progress.TotalTables = &zero
		progress.CompletedTables = &zero
		progress.TotalRecords = &zeroInt64
		progress.ProcessedRecords = &zeroInt64
		return
	}

	// 统计各状态表数
	total, completed, _, _, _, initialized := progressManager.GetTaskStats(taskID)

	// 如果当前是初始化阶段，使用initializedCount作为completedCount
	if task.CurrentStep == "initialize" {
		completed = initialized
	}

	// 获取总体进度
	totalRecords, processedRecords := progressManager.GetTotalProgress(taskID)

	// 计算总体进度（基于记录数）
	overallProgress := 0.0
	if totalRecords > 0 {
		overallProgress = float64(processedRecords) / float64(totalRecords) * 100
	}

	// 计算同步速度和时间
	syncSpeed := int64(0)
	elapsedTime := "00:00:00"
	estimatedTime := "00:00:00"

	earliestStartTime := progressManager.GetEarliestStartTime(taskID)
	if earliestStartTime != nil {
		elapsed := time.Since(*earliestStartTime)
		elapsedTime = formatDuration(elapsed)

		// 计算速度（条/秒）
		if elapsed.Seconds() > 0 {
			syncSpeed = int64(float64(processedRecords) / elapsed.Seconds())
		}

		// 估算剩余时间
		if syncSpeed > 0 && totalRecords > processedRecords {
			remaining := totalRecords - processedRecords
			estimatedSeconds := float64(remaining) / float64(syncSpeed)
			estimatedTime = formatDuration(time.Duration(estimatedSeconds) * time.Second)
		}
	}

	progress.OverallProgress = &overallProgress
	progress.SyncSpeed = &syncSpeed
	progress.ElapsedTime = &elapsedTime
	progress.EstimatedTime = &estimatedTime
	progress.TotalTables = &total
	progress.CompletedTables = &completed
	progress.TotalRecords = &totalRecords
	progress.ProcessedRecords = &processedRecords
}

// formatDuration 格式化时间间隔为 HH:MM:SS
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// GetTaskProgressByDatabase 获取指定数据库下的表明细进度
func (s *TaskProgressService) GetTaskProgressByDatabase(taskID, dbName string) ([]*IncrementalTableStats, error) {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在")
	}

	// 只支持增量模式
	if task.SyncMode != "incremental" {
		return nil, fmt.Errorf("只有增量模式支持按数据库查询表明细")
	}

	statsService := NewIncrementalStatsService()

	// 从Redis获取指定数据库的表统计
	tableStats, _ := statsService.GetTableStatsListByDatabase(taskID, dbName)

	// 如果 Redis 中没有数据，从任务配置中初始化
	if len(tableStats) == 0 {
		allTableStats := statsService.InitTableStatsFromConfig(taskID, &task)
		for _, stats := range allTableStats {
			if stats.Database == dbName {
				tableStats = append(tableStats, stats)
			}
		}
	}

	// 从内存合并全量进度
	tableStats = s.mergeMemoryProgressToTableStats(taskID, tableStats)

	return tableStats, nil
}
