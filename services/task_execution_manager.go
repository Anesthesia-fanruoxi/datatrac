package services

import (
	"context"
	"sync"
)

// TaskExecution 任务执行信息
type TaskExecution struct {
	TaskID    string
	Cancel    context.CancelFunc
	WaitGroup *sync.WaitGroup
}

// TaskExecutionManager 任务执行管理器
type TaskExecutionManager struct {
	executions       sync.Map // map[taskID]*TaskExecution
	incrementalSyncs sync.Map // map[taskID]*IncrementalSync
}

var (
	executionManager     *TaskExecutionManager
	executionManagerOnce sync.Once
)

// GetExecutionManager 获取执行管理器单例
func GetExecutionManager() *TaskExecutionManager {
	executionManagerOnce.Do(func() {
		executionManager = &TaskExecutionManager{}
	})
	return executionManager
}

// StoreExecution 存储任务执行信息
func (m *TaskExecutionManager) StoreExecution(taskID string, cancel context.CancelFunc, wg *sync.WaitGroup) {
	execution := &TaskExecution{
		TaskID:    taskID,
		Cancel:    cancel,
		WaitGroup: wg,
	}
	m.executions.Store(taskID, execution)
}

// GetExecution 获取任务执行信息
func (m *TaskExecutionManager) GetExecution(taskID string) (*TaskExecution, bool) {
	exec, ok := m.executions.Load(taskID)
	if !ok {
		return nil, false
	}
	return exec.(*TaskExecution), true
}

// DeleteExecution 删除任务执行信息
func (m *TaskExecutionManager) DeleteExecution(taskID string) {
	m.executions.Delete(taskID)
}

// StoreIncrementalSync 存储增量同步实例
func (m *TaskExecutionManager) StoreIncrementalSync(taskID string, sync *IncrementalSync) {
	m.incrementalSyncs.Store(taskID, sync)
}

// GetIncrementalSync 获取增量同步实例
func (m *TaskExecutionManager) GetIncrementalSync(taskID string) (*IncrementalSync, bool) {
	sync, ok := m.incrementalSyncs.Load(taskID)
	if !ok {
		return nil, false
	}
	return sync.(*IncrementalSync), true
}

// DeleteIncrementalSync 删除增量同步实例
func (m *TaskExecutionManager) DeleteIncrementalSync(taskID string) {
	m.incrementalSyncs.Delete(taskID)
}
