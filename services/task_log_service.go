package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskLogService 任务日志服务
type TaskLogService struct {
	mu sync.RWMutex
}

// NewTaskLogService 创建任务日志服务
func NewTaskLogService() *TaskLogService {
	return &TaskLogService{}
}

// TaskLog 任务日志
type TaskLog struct {
	Time     string `json:"time"`     // 时间
	Level    string `json:"level"`    // 日志级别：info/success/warning/error
	Message  string `json:"message"`  // 日志消息
	Category string `json:"category"` // 日志分类：all/initialize/complete
}

// getLogFilePath 获取日志文件路径
func (s *TaskLogService) getLogFilePath(taskID string, category string) string {
	logDir := filepath.Join("logs", taskID)
	return filepath.Join(logDir, category+".log")
}

// ensureLogDir 确保日志目录存在
func (s *TaskLogService) ensureLogDir(taskID string) error {
	logDir := filepath.Join("logs", taskID)
	return os.MkdirAll(logDir, 0755)
}

// GetTaskLogs 获取任务日志（从文件读取最后N条）
func (s *TaskLogService) GetTaskLogs(taskID string, category string, limit int) ([]TaskLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getLogFilePath(taskID, category)

	// 如果文件不存在，返回空数组
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []TaskLog{}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 解析日志行
	lines := splitLines(string(content))
	logs := make([]TaskLog, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		log := parseLogLine(line)
		if log != nil {
			logs = append(logs, *log)
		}
	}

	// 限制返回数量（返回最后N条）
	if limit > 0 && len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}

	return logs, nil
}

// AddLog 添加日志（写入文件）
func (s *TaskLogService) AddLog(taskID string, level string, message string, category string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保日志目录存在
	if err := s.ensureLogDir(taskID); err != nil {
		return
	}

	log := TaskLog{
		Time:     formatLogTime(time.Now()),
		Level:    level,
		Message:  message,
		Category: category,
	}

	// 格式化日志行
	logLine := formatLogLine(log)

	// 写入 all.log
	s.appendToFile(taskID, "all", logLine)

	// 根据分类写入对应文件
	if category == "initialize" {
		s.appendToFile(taskID, "initialize", logLine)
	} else if category == "complete" {
		s.appendToFile(taskID, "complete", logLine)
	}

	// 广播新日志到SSE客户端
	sseService := NewTaskSSEService()
	sseService.BroadcastLogUpdate(taskID, log)
}

// appendToFile 追加内容到文件
func (s *TaskLogService) appendToFile(taskID string, category string, content string) {
	filePath := s.getLogFilePath(taskID, category)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(content + "\n")
}

// formatLogTime 格式化日志时间
func formatLogTime(t time.Time) string {
	return t.Format("15:04:05")
}

// formatLogLine 格式化日志行
func formatLogLine(log TaskLog) string {
	return fmt.Sprintf("[%s] [%s] %s", log.Time, log.Level, log.Message)
}

// parseLogLine 解析日志行
func parseLogLine(line string) *TaskLog {
	// 简单解析：[时间] [级别] 消息
	// 实际可以使用正则表达式
	if len(line) < 20 {
		return nil
	}

	// 提取时间
	timeEnd := 10 // [HH:MM:SS]
	if len(line) < timeEnd {
		return nil
	}
	timeStr := line[1:timeEnd]

	// 提取级别
	levelStart := timeEnd + 3 // ] [
	levelEnd := levelStart + 10
	if len(line) < levelEnd {
		return nil
	}

	var level string
	for i := levelStart; i < len(line) && i < levelEnd; i++ {
		if line[i] == ']' {
			level = line[levelStart:i]
			break
		}
	}

	// 提取消息
	msgStart := levelStart + len(level) + 2 // ]
	if len(line) <= msgStart {
		return nil
	}
	message := line[msgStart:]

	return &TaskLog{
		Time:    timeStr,
		Level:   level,
		Message: message,
	}
}

// splitLines 分割行
func splitLines(content string) []string {
	lines := make([]string, 0)
	start := 0

	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}

	if start < len(content) {
		lines = append(lines, content[start:])
	}

	return lines
}

// Info 记录信息日志
func (s *TaskLogService) Info(taskID string, message string) {
	s.AddLog(taskID, "info", message, "all")
}

// Error 记录错误日志
func (s *TaskLogService) Error(taskID string, message string) {
	s.AddLog(taskID, "error", message, "all")
}

// Warning 记录警告日志
func (s *TaskLogService) Warning(taskID string, message string) {
	s.AddLog(taskID, "warning", message, "all")
}

// Success 记录成功日志
func (s *TaskLogService) Success(taskID string, message string) {
	s.AddLog(taskID, "success", message, "complete")
}
