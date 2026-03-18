package services

import (
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
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
	OverallProgress  *float64 `json:"overall_progress,omitempty"`  // 已处理记录/总记录，取2位小数
	SyncSpeed        *int64   `json:"sync_speed,omitempty"`        // 每秒处理速度
	ElapsedTime      *string  `json:"elapsed_time,omitempty"`      // 已运行时间
	EstimatedTime    *string  `json:"estimated_time,omitempty"`    // 预计剩余时间
	TotalTables      *int     `json:"total_tables,omitempty"`      // 总表数（固定，创建任务时选定）
	InitTables       *int     `json:"init_tables,omitempty"`       // 已初始化表数（建表完成后更新）
	CompletedTables  *int     `json:"completed_tables,omitempty"`  // 已完成表数（数据同步完成后更新）
	TotalRecords     *int64   `json:"total_records,omitempty"`     // 总记录数（固定，获取元数据时统计）
	ProcessedRecords *int64   `json:"processed_records,omitempty"` // 已处理记录数

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
		// 任务未在运行：从任务配置解析总表数，使初始状态显示 0/N
		zero := 0
		zeroInt64 := int64(0)
		zeroFloat := 0.0
		emptyStr := "00:00:00"
		progress.OverallProgress = &zeroFloat
		progress.SyncSpeed = &zeroInt64
		progress.ElapsedTime = &emptyStr
		progress.EstimatedTime = &emptyStr
		progress.CompletedTables = &zero
		progress.InitTables = &zero
		progress.TotalRecords = &zeroInt64
		progress.ProcessedRecords = &zeroInt64
		totalFromConfig := countTotalTablesFromConfig(task)
		progress.TotalTables = &totalFromConfig
		return
	}

	// 获取目标源级别统计
	// 全量同步：总表数必须固定为“配置表数 × 目标数”，不能跟随 TargetUnits 动态增长。
	// 这里优先使用 Redis 配置（带降级），避免 DB 中 task.Config 与运行时配置不一致导致 target_stats 总表数异常。
	cfg, _ := NewConfigCacheService().GetTaskConfigWithFallback(taskID)
	baseTargetStats := s.buildTargetStatsFromTaskConfig(cfg)
	runtimeTargetStats := progressManager.GetAllTargetStats(taskID)
	targetStats := mergeFullSyncTargetStats(baseTargetStats, runtimeTargetStats)

	// 汇总所有目标的表数、记录数作为总体进度
	total := countTotalTablesFromTaskConfig(cfg)
	completed := 0
	initCompleted := 0
	var totalRecords, processedRecords int64
	isFullSync := task.SyncMode == "full"

	// 注释掉源端统计获取，改为使用目标端累加
	// _, completedSrc, _, _, _, _ := progressManager.GetTaskStats(taskID)

	for _, ts := range targetStats {
		completed += ts.CompletedTables // 累加各目标的完成表数
		totalRecords += ts.TotalRecords
		processedRecords += ts.ProcessedRecords
		// 全量模式不返回单表详情，只返回整体进度
		if isFullSync {
			ts.Tables = nil
		}
	}

	// init_tables 从源端 Units 的 initialized 状态统计
	if isFullSync {
		// 初始化阶段：用 Units 的 initialized 状态折算（×目标数）
		// 同步数据阶段：init_tables 表示“结构已初始化完成的表”，不应因为单位状态变成 running/completed 而回退，直接固定为 total
		if task.CurrentStep == "initialize" {
			_, _, _, _, _, initialized := progressManager.GetTaskStats(taskID)
			targets := countTargetsFromTaskConfig(cfg)
			initCompleted = initialized * targets
			if initCompleted > total {
				initCompleted = total
			}
		} else {
			initCompleted = total // 同步数据阶段：init_tables 直接固定为总量（结构已在初始化阶段完成）
		}
		// 如果还在初始化阶段，completed 还未产生，设为 0
		if task.CurrentStep == "initialize" {
			completed = 0
		}
	}

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
	progress.InitTables = &initCompleted
	// 全量同步：completed_tables 使用目标端累加统计（各目标完成表数之和）
	progress.CompletedTables = &completed
	progress.TotalRecords = &totalRecords
	progress.ProcessedRecords = &processedRecords
	// 修正 target_stats 的口径：根据当前阶段决定 init_tables 逻辑
	targetStats = fixFullSyncTargetStats(targetStats, total, task.CurrentStep)
	progress.TargetStats = targetStats
}

// fixFullSyncTargetStats 修正全量同步的 target_stats 口径：
// - completed_tables: 应为"该目标已完成同步的表数"，从运行时统计获取
func fixFullSyncTargetStats(stats []*TargetProgress, totalSrcTables int, currentStep string) []*TargetProgress {
	if len(stats) == 0 {
		return stats
	}
	for _, s := range stats {
		if s == nil {
			continue
		}
		// completed_tables 保持运行时统计（已完成同步的表数）
	}
	return stats
}

// countTotalTablesFromConfig 从任务配置中统计总表数（用于未运行时展示 0/N）
// 返回值为：源端选中表数 × 目标数（每个目标都会同步这些表）
func countTotalTablesFromConfig(task *models.SyncTask) int {
	if task == nil || task.Config == "" {
		return 0
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
		return 0
	}
	n := 0
	for _, db := range cfg.SelectedDatabases {
		n += len(db.Tables)
	}
	targets := len(cfg.TargetIDs)
	if targets == 0 && cfg.TargetID != "" {
		targets = 1
	}
	if targets == 0 {
		targets = 1
	}
	return n * targets
}

// countTargetsFromConfig 统计目标源数量（用于多目标源折算）
func countTargetsFromConfig(task *models.SyncTask) int {
	if task == nil || task.Config == "" {
		return 1
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
		return 1
	}
	targets := len(cfg.TargetIDs)
	if targets == 0 && cfg.TargetID != "" {
		targets = 1
	}
	if targets <= 0 {
		targets = 1
	}
	return targets
}

func countTotalTablesFromTaskConfig(cfg *TaskConfig) int {
	if cfg == nil {
		return 0
	}
	n := 0
	for _, db := range cfg.SelectedDatabases {
		n += len(db.Tables)
	}
	return n * countTargetsFromTaskConfig(cfg)
}

func countTargetsFromTaskConfig(cfg *TaskConfig) int {
	if cfg == nil {
		return 1
	}
	targets := len(cfg.TargetIDs)
	if targets == 0 && cfg.TargetID != "" {
		targets = 1
	}
	if targets <= 0 {
		targets = 1
	}
	return targets
}

func (s *TaskProgressService) buildTargetStatsFromTaskConfig(cfg *TaskConfig) []*TargetProgress {
	if cfg == nil {
		return nil
	}
	targetIDs := cfg.TargetIDs
	if len(targetIDs) == 0 && cfg.TargetID != "" {
		targetIDs = []string{cfg.TargetID}
	}
	if len(targetIDs) == 0 {
		return nil
	}
	n := 0
	for _, db := range cfg.SelectedDatabases {
		n += len(db.Tables)
	}
	if n < 0 {
		n = 0
	}
	dsService := NewDataSourceService()
	list := make([]*TargetProgress, 0, len(targetIDs))
	for _, id := range targetIDs {
		name := id
		if ds, err := dsService.GetByID(id); err == nil && ds != nil {
			name = ds.Name
		}
		list = append(list, &TargetProgress{
			TargetID:         id,
			TargetName:       name,
			Status:           "pending",
			Progress:         0,
			TotalTables:      n, // 每个目标都需要同步 n 张表
			CompletedTables:  0,
			TotalRecords:     0,
			ProcessedRecords: 0,
		})
	}
	return list
}

// mergeFullSyncTargetStats 将运行时统计合并到配置占位统计中（固定 TotalTables，不随 TargetUnits 动态增长）
func mergeFullSyncTargetStats(base []*TargetProgress, runtime []*TargetProgress) []*TargetProgress {
	if len(base) == 0 && len(runtime) == 0 {
		return nil
	}
	if len(base) == 0 {
		// 没有配置占位时（极端情况），直接返回运行时
		return runtime
	}

	// target_id -> stat
	rt := make(map[string]*TargetProgress, len(runtime))
	for _, s := range runtime {
		if s == nil || s.TargetID == "" {
			continue
		}
		rt[s.TargetID] = s
	}

	out := make([]*TargetProgress, 0, len(base))
	for _, b := range base {
		if b == nil {
			continue
		}
		if r, ok := rt[b.TargetID]; ok && r != nil {
			// 固定 b.TotalTables，其余用运行时数据覆盖
			// Progress 改为按表完成度计算（而非按数据量，避免跳动）
			merged := *b
			merged.Status = r.Status
			if r.TotalTables > 0 {
				merged.Progress = float64(r.CompletedTables) / float64(r.TotalTables) * 100
			} else {
				merged.Progress = 0
			}
			merged.CompletedTables = r.CompletedTables
			merged.TotalRecords = r.TotalRecords
			merged.ProcessedRecords = r.ProcessedRecords
			merged.Tables = r.Tables
			out = append(out, &merged)
		} else {
			out = append(out, b)
		}
	}

	// 运行时出现但配置没有的目标（理论上不应该），追加在末尾
	for id, r := range rt {
		found := false
		for _, b := range base {
			if b != nil && b.TargetID == id {
				found = true
				break
			}
		}
		if !found {
			out = append(out, r)
		}
	}

	return out
}

// buildTargetStatsFromConfig 任务未运行时，从配置构建各目标数据源占位进度（每个目标 0/N 表、0 记录）
func (s *TaskProgressService) buildTargetStatsFromConfig(task *models.SyncTask) []*TargetProgress {
	if task == nil || task.Config == "" {
		return nil
	}
	var cfg TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &cfg); err != nil {
		return nil
	}
	targetIDs := cfg.TargetIDs
	if len(targetIDs) == 0 && cfg.TargetID != "" {
		targetIDs = []string{cfg.TargetID}
	}
	if len(targetIDs) == 0 {
		return nil
	}
	totalUnits := countTotalTablesFromConfig(task)
	tablesPerTarget := totalUnits / len(targetIDs)
	dsService := NewDataSourceService()
	list := make([]*TargetProgress, 0, len(targetIDs))
	for _, id := range targetIDs {
		name := id
		if ds, err := dsService.GetByID(id); err == nil && ds != nil {
			name = ds.Name
		}
		list = append(list, &TargetProgress{
			TargetID:         id,
			TargetName:       name,
			Status:           "pending",
			Progress:         0,
			TotalTables:      tablesPerTarget,
			CompletedTables:  0,
			TotalRecords:     0,
			ProcessedRecords: 0,
		})
	}
	return list
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
