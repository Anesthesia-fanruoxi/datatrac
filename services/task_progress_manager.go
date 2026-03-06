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

// TaskProgressManager 任务进度管理器（内存）
type TaskProgressManager struct {
	mu    sync.RWMutex
	tasks map[string]*TaskProgressData // taskID -> progress data
}

// TaskProgressData 任务进度数据
type TaskProgressData struct {
	TaskID      string               `json:"task_id"`
	Units       map[string]*TaskUnit `json:"units"` // unitName -> unit
	StartTime   time.Time            `json:"start_time"`
	CurrentStep string               `json:"current_step"`
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
