package services

import (
	"fmt"
	"sync"
	"time"
)

// ProgressMessage 进度上报消息（Work 线程发送给 Process 线程）
type ProgressMessage struct {
	TaskID       string
	TargetID     string
	TargetName   string
	UnitName     string
	Status       string // "initialized" / "running" / "completed" / "failed"
	TotalRecords int64  // 总记录数
	Processed    int64  // 已处理记录数
	IsNew        bool   // 是否是新发现的表（首次上报）
}

// TaskUnit 任务单元（内存结构）
type TaskUnit struct {
	UnitName         string     `json:"unit_name"`
	Status           string     `json:"status"` // pending/running/completed/failed/paused/initialized
	TotalRecords     int64      `json:"total_records"`
	ProcessedRecords int64      `json:"processed_records"`
	ApproxRows       int64      `json:"approx_rows"` // 初始化阶段预获取的近似行数（来自 SHOW TABLE STATUS）
	ErrorMessage     string     `json:"error_message,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TargetUnit 目标源单元进度（多目标源时使用）
type TargetUnit struct {
	TargetID         string     `json:"target_id"`
	TargetName       string     `json:"target_name"`
	UnitName         string     `json:"unit_name"`
	Status           string     `json:"status"` // pending/running/completed/failed/paused
	TotalRecords     int64      `json:"total_records"`
	ProcessedRecords int64      `json:"processed_records"`
	BatchNum         int        `json:"batch_num"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TaskProgressManager 任务进度管理器（内存）
type TaskProgressManager struct {
	mu           sync.RWMutex
	tasks        map[string]*TaskProgressData // taskID -> progress data
	progressChan chan ProgressMessage         // 进度上报通道
	ssePusher    SSEPusher                    // SSE 推送器
}

// SSEPusher SSE 推送接口
type SSEPusher interface {
	BroadcastProgress(taskID string, progress *TaskProgress)
}

// SSEDirectPusher 直接推送实现
type SSEDirectPusher struct{}

func (p *SSEDirectPusher) BroadcastProgress(taskID string, progress *TaskProgress) {
	sseService := NewTaskSSEService()
	if sseService != nil {
		sseService.BroadcastProgressUpdate(taskID)
	}
}

// TaskProgressData 任务进度数据
type TaskProgressData struct {
	TaskID      string                            `json:"task_id"`
	Units       map[string]*TaskUnit              `json:"units"`        // unitName -> unit (整体进度，不乘以目标数)
	TargetUnits map[string]map[string]*TargetUnit `json:"target_units"` // targetID -> unitName -> TargetUnit (目标源级别进度)
	StartTime   time.Time                         `json:"start_time"`
	CurrentStep string                            `json:"current_step"`
}

var (
	progressManager     *TaskProgressManager
	progressManagerOnce sync.Once
)

// GetProgressManager 获取进度管理器单例
func GetProgressManager() *TaskProgressManager {
	progressManagerOnce.Do(func() {
		progressManager = &TaskProgressManager{
			tasks:        make(map[string]*TaskProgressData),
			progressChan: make(chan ProgressMessage, 1000), // 带缓冲的 channel
			ssePusher:    &SSEDirectPusher{},
		}
		// 启动进度处理线程
		go progressManager.startProgressProcessor()
	})
	return progressManager
}

// InitTask 初始化任务进度
func (m *TaskProgressManager) InitTask(taskID string, unitNames []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	units := make(map[string]*TaskUnit)
	now := time.Now()

	for _, name := range unitNames {
		units[name] = &TaskUnit{
			UnitName:         name,
			Status:           "pending",
			TotalRecords:     0,
			ProcessedRecords: 0,
			ApproxRows:       0,
			UpdatedAt:        now,
		}
	}

	m.tasks[taskID] = &TaskProgressData{
		TaskID:      taskID,
		Units:       units,
		TargetUnits: make(map[string]map[string]*TargetUnit),
		StartTime:   now,
		CurrentStep: "initialize",
	}
}

// SetApproxRows 设置单元的近似行数（从 SHOW TABLE STATUS 获取）
func (m *TaskProgressManager) SetApproxRows(taskID string, unitName string, approxRows int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, ok := m.tasks[taskID]; ok {
		if unit, ok := task.Units[unitName]; ok {
			unit.ApproxRows = approxRows
		}
	}
}

// GetApproxRows 获取单元的近似行数
func (m *TaskProgressManager) GetApproxRows(taskID string, unitName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if task, ok := m.tasks[taskID]; ok {
		if unit, ok := task.Units[unitName]; ok {
			return unit.ApproxRows
		}
	}
	return 0
}

// SendProgress 发送进度消息（Work 线程调用）
func (m *TaskProgressManager) SendProgress(msg ProgressMessage) {
	select {
	case m.progressChan <- msg:
		// 消息已发送
	default:
		// channel 满，记录警告日志
	}
}

// startProgressProcessor 启动进度处理线程（Process 线程）
func (m *TaskProgressManager) startProgressProcessor() {
	for msg := range m.progressChan {
		m.processProgressMessage(msg)
	}
}

// processProgressMessage 处理进度消息并推送
func (m *TaskProgressManager) processProgressMessage(msg ProgressMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[msg.TaskID]
	if !ok {
		return
	}

	// 初始化目标单元映射
	if task.TargetUnits == nil {
		task.TargetUnits = make(map[string]map[string]*TargetUnit)
	}
	if task.TargetUnits[msg.TargetID] == nil {
		task.TargetUnits[msg.TargetID] = make(map[string]*TargetUnit)
	}

	// 更新或创建单元
	now := time.Now()
	if unit, exists := task.TargetUnits[msg.TargetID][msg.UnitName]; exists {
		// 已存在，更新
		unit.Status = msg.Status
		unit.TotalRecords = msg.TotalRecords
		unit.ProcessedRecords = msg.Processed
		unit.UpdatedAt = now
		if msg.Status == "running" && unit.StartedAt == nil {
			unit.StartedAt = &now
		}
	} else {
		// 新建
		task.TargetUnits[msg.TargetID][msg.UnitName] = &TargetUnit{
			TargetID:         msg.TargetID,
			TargetName:       msg.TargetName,
			UnitName:         msg.UnitName,
			Status:           msg.Status,
			TotalRecords:     msg.TotalRecords,
			ProcessedRecords: msg.Processed,
			UpdatedAt:        now,
		}
		if msg.Status == "running" {
			task.TargetUnits[msg.TargetID][msg.UnitName].StartedAt = &now
		}
	}

	// 计算并推送进度（需要释放锁，因为推送可能耗时）
	m.mu.Unlock()

	// 构建进度并推送
	progress := m.buildProgress(msg.TaskID)
	if progress != nil && m.ssePusher != nil {
		m.ssePusher.BroadcastProgress(msg.TaskID, progress)
	}

	m.mu.Lock()
}

// GetTask 获取任务进度
func (m *TaskProgressManager) GetTask(taskID string) *TaskProgressData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[taskID]
}

// UpdateUnitStatus 更新单元状态
func (m *TaskProgressManager) UpdateUnitStatus(taskID, unitName, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	unit, ok := task.Units[unitName]
	if !ok {
		return
	}

	unit.Status = status
	unit.UpdatedAt = time.Now()

	// 如果是开始运行，设置开始时间
	if status == "running" && unit.StartedAt == nil {
		now := time.Now()
		unit.StartedAt = &now
	}
}

// UpdateUnitProgress 更新单元进度
func (m *TaskProgressManager) UpdateUnitProgress(taskID, unitName string, total, processed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	unit, ok := task.Units[unitName]
	if !ok {
		return
	}

	// 如果传入的 total 为 0，但有预获取的近似行数，则使用近似行数
	if total == 0 && unit.ApproxRows > 0 {
		total = unit.ApproxRows
	}

	unit.TotalRecords = total
	unit.ProcessedRecords = processed
	unit.UpdatedAt = time.Now()
}

// UpdateUnitError 更新单元错误信息
func (m *TaskProgressManager) UpdateUnitError(taskID, unitName, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	unit, ok := task.Units[unitName]
	if !ok {
		return
	}

	unit.Status = "failed"
	unit.ErrorMessage = errorMsg
	unit.UpdatedAt = time.Now()
}

// UpdateTaskStep 更新任务步骤
func (m *TaskProgressManager) UpdateTaskStep(taskID, step string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	task.CurrentStep = step
}

// GetUnits 获取任务的所有单元
func (m *TaskProgressManager) GetUnits(taskID string) []*TaskUnit {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil
	}

	units := make([]*TaskUnit, 0, len(task.Units))
	for _, unit := range task.Units {
		units = append(units, unit)
	}

	return units
}

// GetUnit 获取指定单元
func (m *TaskProgressManager) GetUnit(taskID, unitName string) *TaskUnit {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil
	}

	return task.Units[unitName]
}

// ClearTask 清除任务进度
func (m *TaskProgressManager) ClearTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tasks, taskID)
}

// GetTaskStats 获取任务统计信息
func (m *TaskProgressManager) GetTaskStats(taskID string) (total, completed, running, failed, pending, initialized int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	total = len(task.Units)
	for _, unit := range task.Units {
		switch unit.Status {
		case "completed":
			completed++
			initialized++ // 已完成的表也算已初始化
		case "running":
			running++
			initialized++ // 运行中的表也算已初始化
		case "failed":
			failed++
		case "pending":
			pending++
		case "initialized":
			initialized++
		}
	}

	return
}

// GetTotalProgress 获取总体进度
func (m *TaskProgressManager) GetTotalProgress(taskID string) (totalRecords, processedRecords int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	for _, unit := range task.Units {
		// 优先使用 TotalRecords，如果没有则使用 ApproxRows
		if unit.TotalRecords > 0 {
			totalRecords += unit.TotalRecords
		} else if unit.ApproxRows > 0 {
			totalRecords += unit.ApproxRows
		}
		processedRecords += unit.ProcessedRecords
	}

	return
}

// GetEarliestStartTime 获取最早的开始时间
func (m *TaskProgressManager) GetEarliestStartTime(taskID string) *time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil
	}

	var earliest *time.Time
	for _, unit := range task.Units {
		if unit.StartedAt != nil {
			if earliest == nil || unit.StartedAt.Before(*earliest) {
				earliest = unit.StartedAt
			}
		}
	}

	return earliest
}

// ========== 目标源级别进度管理 ==========

// InitTargetUnit 初始化目标源单元
func (m *TaskProgressManager) InitTargetUnit(taskID, targetID, targetName, unitName string, totalRecords int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	if task.TargetUnits == nil {
		task.TargetUnits = make(map[string]map[string]*TargetUnit)
	}
	if task.TargetUnits[targetID] == nil {
		task.TargetUnits[targetID] = make(map[string]*TargetUnit)
	}

	now := time.Now()
	task.TargetUnits[targetID][unitName] = &TargetUnit{
		TargetID:         targetID,
		TargetName:       targetName,
		UnitName:         unitName,
		Status:           "pending",
		TotalRecords:     totalRecords,
		ProcessedRecords: 0,
		BatchNum:         0,
		UpdatedAt:        now,
	}
}

// UpdateTargetUnitStatus 更新目标源单元状态
func (m *TaskProgressManager) UpdateTargetUnitStatus(taskID, targetID, unitName, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	if targetUnits, ok := task.TargetUnits[targetID]; ok {
		if unit, ok := targetUnits[unitName]; ok {
			unit.Status = status
			unit.UpdatedAt = time.Now()

			if status == "running" && unit.StartedAt == nil {
				now := time.Now()
				unit.StartedAt = &now
			}
		}
	}
}

// UpdateTargetUnitProgress 更新目标源单元进度
func (m *TaskProgressManager) UpdateTargetUnitProgress(taskID, targetID, unitName string, processed int64, batchNum int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return
	}

	if targetUnits, ok := task.TargetUnits[targetID]; ok {
		if unit, ok := targetUnits[unitName]; ok {
			unit.ProcessedRecords = processed
			unit.BatchNum = batchNum
			unit.UpdatedAt = time.Now()
		}
	}
}

// GetTargetUnits 获取指定目标源的所有单元进度
func (m *TaskProgressManager) GetTargetUnits(taskID, targetID string) []*TargetUnit {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil
	}

	if targetUnits, ok := task.TargetUnits[targetID]; ok {
		units := make([]*TargetUnit, 0, len(targetUnits))
		for _, unit := range targetUnits {
			units = append(units, unit)
		}
		return units
	}

	return nil
}

// GetAllTargetStats 获取所有目标源的进度统计
func (m *TaskProgressManager) GetAllTargetStats(taskID string) []*TargetProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok || task.TargetUnits == nil {
		return nil
	}

	stats := make([]*TargetProgress, 0, len(task.TargetUnits))

	for targetID, targetUnits := range task.TargetUnits {
		var targetName string
		var currentTable *TargetTableStat
		completedCount := 0
		initCount := 0
		totalCount := len(targetUnits)
		var totalRecords, processedRecords int64

		for _, unit := range targetUnits {
			if targetName == "" {
				targetName = unit.TargetName
			}
			totalRecords += unit.TotalRecords
			processedRecords += unit.ProcessedRecords

			tableStat := &TargetTableStat{
				TableName:        unit.UnitName,
				Status:           unit.Status,
				ProcessedRecords: unit.ProcessedRecords,
				TotalRecords:     unit.TotalRecords,
			}
			if unit.TotalRecords > 0 {
				tableStat.Progress = float64(unit.ProcessedRecords) / float64(unit.TotalRecords) * 100
			}

			if unit.Status == "running" {
				currentTable = tableStat
			} else if unit.Status == "completed" {
				completedCount++
			} else if unit.Status == "initialized" {
				initCount++
			}
		}

		// progress 按记录数计算（与 overall_progress 一致）
		progress := 0.0
		if totalRecords > 0 {
			progress = float64(processedRecords) / float64(totalRecords) * 100
		}
		status := "syncing"
		if completedCount == totalCount && totalCount > 0 {
			status = "completed"
		}
		if currentTable != nil && completedCount == 0 {
			progress = currentTable.Progress
		}

		stat := &TargetProgress{
			TargetID:         targetID,
			TargetName:       targetName,
			Status:           status,
			Progress:         progress,
			TotalTables:      totalCount,
			InitTables:       initCount,
			CompletedTables:  completedCount,
			TotalRecords:     totalRecords,
			ProcessedRecords: processedRecords,
		}
		// 只在增量模式下返回单表详情，全量模式不返回
		// if currentTable != nil {
		// 	stat.Tables = []*TargetTableStat{currentTable}
		// }
		stats = append(stats, stat)
	}

	return stats
}

// buildProgress 构建进度对象（供 Process 线程使用）
func (m *TaskProgressManager) buildProgress(taskID string) *TaskProgress {
	m.mu.RLock()
	task, _ := m.tasks[taskID]
	m.mu.RUnlock()

	if task == nil {
		return nil
	}

	// 获取所有目标统计
	targetStats := m.buildTargetStats(task)

	// 汇总计算
	total := 0
	initialized := 0
	completed := 0
	var totalRecords, processedRecords int64

	for _, ts := range targetStats {
		total += ts.TotalTables
		initialized += ts.InitTables
		completed += ts.CompletedTables
		totalRecords += ts.TotalRecords
		processedRecords += ts.ProcessedRecords
	}

	// 计算总体进度
	overallProgress := 0.0
	if totalRecords > 0 {
		overallProgress = float64(processedRecords) / float64(totalRecords) * 100
	}

	// 计算同步速度和时间
	var syncSpeed int64
	elapsedTime := "00:00:00"
	estimatedTime := "00:00:00"

	m.mu.RLock()
	startTime := task.StartTime
	currentStep := task.CurrentStep
	m.mu.RUnlock()

	if !startTime.IsZero() {
		elapsed := time.Since(startTime)
		elapsedTime = formatDurationM(elapsed)

		if elapsed.Seconds() > 0 {
			syncSpeed = int64(float64(processedRecords) / elapsed.Seconds())
		}

		if syncSpeed > 0 && totalRecords > processedRecords {
			remaining := totalRecords - processedRecords
			estimatedSeconds := float64(remaining) / float64(syncSpeed)
			estimatedTime = formatDurationM(time.Duration(estimatedSeconds) * time.Second)
		}
	}

	// 构建返回对象（使用指针）
	progress := &TaskProgress{
		TaskID:           taskID,
		OverallProgress:  &overallProgress,
		SyncSpeed:        &syncSpeed,
		ElapsedTime:      &elapsedTime,
		EstimatedTime:    &estimatedTime,
		TotalTables:      &total,
		InitTables:       &initialized,
		CompletedTables:  &completed,
		TotalRecords:     &totalRecords,
		ProcessedRecords: &processedRecords,
		TargetStats:      targetStats,
		CurrentStep:      currentStep,
		SyncMode:         "full", // 全量同步模式
	}

	return progress
}

// buildTargetStats 构建目标源进度统计
func (m *TaskProgressManager) buildTargetStats(task *TaskProgressData) []*TargetProgress {
	if task == nil || task.TargetUnits == nil {
		return nil
	}

	stats := make([]*TargetProgress, 0, len(task.TargetUnits))

	for targetID, targetUnits := range task.TargetUnits {
		var targetName string
		completedCount := 0
		initCount := 0
		totalCount := len(targetUnits)
		var totalRecords, processedRecords int64

		for _, unit := range targetUnits {
			if targetName == "" {
				targetName = unit.TargetName
			}
			totalRecords += unit.TotalRecords
			processedRecords += unit.ProcessedRecords

			if unit.Status == "completed" {
				completedCount++
				initCount++ // 已完成的表也算已初始化
			} else if unit.Status == "initialized" || unit.Status == "running" {
				initCount++ // 初始化状态或运行中的表都算已初始化
			}
		}

		progress := 0.0
		if totalRecords > 0 {
			progress = float64(processedRecords) / float64(totalRecords) * 100
		}
		status := "syncing"
		if completedCount == totalCount && totalCount > 0 {
			status = "completed"
		}

		stats = append(stats, &TargetProgress{
			TargetID:         targetID,
			TargetName:       targetName,
			Status:           status,
			Progress:         progress,
			TotalTables:      totalCount,
			InitTables:       initCount,
			CompletedTables:  completedCount,
			TotalRecords:     totalRecords,
			ProcessedRecords: processedRecords,
		})
	}

	return stats
}

// formatDurationM 格式化时间间隔为 HH:MM:SS
func formatDurationM(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// TargetProgress 目标源进度（用于API返回）
type TargetProgress struct {
	TargetID         string             `json:"target_id"`
	TargetName       string             `json:"target_name"`
	Status           string             `json:"status"` // syncing/completed
	Progress         float64            `json:"progress"`
	TotalTables      int                `json:"total_tables"`
	InitTables       int                `json:"init_tables"`       // 已初始化表数（建表完成后更新）
	CompletedTables  int                `json:"completed_tables"`  // 已完成表数（数据同步完成后更新）
	TotalRecords     int64              `json:"total_records"`     // 总记录数（固定）
	ProcessedRecords int64              `json:"processed_records"` // 已处理记录数
	Tables           []*TargetTableStat `json:"tables,omitempty"`  // 增量模式使用，全量模式不返回
}

// TargetTableStat 目标源的表进度
type TargetTableStat struct {
	TableName        string  `json:"table_name"`
	Status           string  `json:"status"`
	ProcessedRecords int64   `json:"processed_records"`
	TotalRecords     int64   `json:"total_records"`
	Progress         float64 `json:"progress"`
}
