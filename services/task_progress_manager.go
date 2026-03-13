package services

import (
	"sync"
	"time"
)

// TaskUnit 任务单元（内存结构）
type TaskUnit struct {
	UnitName         string     `json:"unit_name"`
	Status           string     `json:"status"` // pending/running/completed/failed/paused/initialized
	TotalRecords     int64      `json:"total_records"`
	ProcessedRecords int64      `json:"processed_records"`
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
	mu    sync.RWMutex
	tasks map[string]*TaskProgressData // taskID -> progress data
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
			tasks: make(map[string]*TaskProgressData),
		}
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
		case "running":
			running++
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
		totalRecords += unit.TotalRecords
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
		// 获取第一个单元的信息作为目标源信息
		var targetName string
		var currentTable *TargetTableStat

		// 找出正在运行的表和已完成的表
		runningUnits := make([]*TargetTableStat, 0)
		completedCount := 0
		totalCount := len(targetUnits)

		for _, unit := range targetUnits {
			if targetName == "" {
				targetName = unit.TargetName
			}

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
				runningUnits = append(runningUnits, tableStat)
			} else if unit.Status == "completed" {
				completedCount++
			}
		}

		// 计算目标源整体进度
		progress := 0.0
		if totalCount > 0 {
			progress = float64(completedCount) / float64(totalCount) * 100
		}

		status := "syncing"
		if completedCount == totalCount && totalCount > 0 {
			status = "completed"
		}

		// 如果有正在运行的表，使用该表的进度
		if currentTable != nil && completedCount == 0 {
			progress = currentTable.Progress
		}

		stat := &TargetProgress{
			TargetID:   targetID,
			TargetName: targetName,
			Status:     status,
			Progress:   progress,
		}

		// 设置正在进行的表
		if currentTable != nil {
			stat.Tables = []*TargetTableStat{currentTable}
		}

		stats = append(stats, stat)
	}

	return stats
}

// TargetProgress 目标源进度（用于API返回）
type TargetProgress struct {
	TargetID   string             `json:"target_id"`
	TargetName string             `json:"target_name"`
	Status     string             `json:"status"` // syncing/completed
	Progress   float64            `json:"progress"`
	Tables     []*TargetTableStat `json:"tables,omitempty"`
}

// TargetTableStat 目标源的表进度
type TargetTableStat struct {
	TableName        string  `json:"table_name"`
	Status           string  `json:"status"`
	ProcessedRecords int64   `json:"processed_records"`
	TotalRecords     int64   `json:"total_records"`
	Progress         float64 `json:"progress"`
}
