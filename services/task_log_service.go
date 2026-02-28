package services

import (
	"fmt"
	"sync"
	"time"
)

// TaskLogService 任务日志服务
type TaskLogService struct {
	logs map[string][]TaskLog // taskID -> logs
	mu   sync.RWMutex
}

// NewTaskLogService 创建任务日志服务
func NewTaskLogService() *TaskLogService {
	return &TaskLogService{
		logs: make(map[string][]TaskLog),
	}
}

// TaskLog 任务日志
type TaskLog struct {
	Time     string `json:"time"`     // 时间
	Level    string `json:"level"`    // 日志级别：info/success/warning/error
	Message  string `json:"message"`  // 日志消息
	Category string `json:"category"` // 日志分类：all/create/sync/complete
}

// GetTaskLogs 获取任务日志
func (s *TaskLogService) GetTaskLogs(taskID string, limit int) ([]TaskLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, exists := s.logs[taskID]
	if !exists {
		return []TaskLog{}, nil
	}

	// 限制日志数量
	if limit > 0 && len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}

	return logs, nil
}

// AddLog 添加日志
func (s *TaskLogService) AddLog(taskID string, level string, message string, category string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.logs[taskID] == nil {
		s.logs[taskID] = []TaskLog{}
	}

	log := TaskLog{
		Time:     formatLogTime(time.Now()),
		Level:    level,
		Message:  message,
		Category: category,
	}

	s.logs[taskID] = append(s.logs[taskID], log)

	// 限制日志数量（最多1000条）
	if len(s.logs[taskID]) > 1000 {
		s.logs[taskID] = s.logs[taskID][len(s.logs[taskID])-1000:]
	}
}

// formatLogTime 格式化日志时间
func formatLogTime(t time.Time) string {
	return t.Format("15:04:05")
}

// Info 记录信息日志
func (s *TaskLogService) Info(taskID string, message string) {
	// 添加到内存日志
	s.AddLog(taskID, "info", message, "all")
	// 打印到控制台
	fmt.Printf("[INFO] [Task:%s] %s\n", taskID, message)
}

// Error 记录错误日志
func (s *TaskLogService) Error(taskID string, message string) {
	// 添加到内存日志
	s.AddLog(taskID, "error", message, "all")
	// 打印到控制台
	fmt.Printf("[ERROR] [Task:%s] %s\n", taskID, message)
}

// Warning 记录警告日志
func (s *TaskLogService) Warning(taskID string, message string) {
	// 添加到内存日志
	s.AddLog(taskID, "warning", message, "all")
	// 打印到控制台
	fmt.Printf("[WARNING] [Task:%s] %s\n", taskID, message)
}

// Success 记录成功日志
func (s *TaskLogService) Success(taskID string, message string) {
	// 添加到内存日志
	s.AddLog(taskID, "success", message, "complete")
	// 打印到控制台
	fmt.Printf("[SUCCESS] [Task:%s] %s\n", taskID, message)
}
