package services

import (
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TaskSSEService SSE服务
type TaskSSEService struct {
	progressService *TaskProgressService
	logService      *TaskLogService
	clients         map[string]map[chan SSEMessage]bool // taskID -> clients
	mu              sync.RWMutex
}

// SSEMessage SSE消息
type SSEMessage struct {
	Event string      `json:"event"` // progress, log, error
	Data  interface{} `json:"data"`
}

var (
	taskSSEInstance *TaskSSEService
	taskSSEOnce     sync.Once
)

// NewTaskSSEService 获取SSE服务单例
func NewTaskSSEService() *TaskSSEService {
	taskSSEOnce.Do(func() {
		taskSSEInstance = &TaskSSEService{
			progressService: NewTaskProgressService(),
			logService:      NewTaskLogService(),
			clients:         make(map[string]map[chan SSEMessage]bool),
		}
	})
	return taskSSEInstance
}

// AddClient 添加客户端
func (s *TaskSSEService) AddClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clients[taskID] == nil {
		s.clients[taskID] = make(map[chan SSEMessage]bool)
	}
	s.clients[taskID][client] = true
}

// RemoveClient 移除客户端
func (s *TaskSSEService) RemoveClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.clients[taskID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(s.clients, taskID)
		}
	}
	close(client)
}

// StreamTaskUpdates 流式推送任务更新
func (s *TaskSSEService) StreamTaskUpdates(taskID string, client chan SSEMessage, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 立即发送一次
	s.sendUpdate(taskID, client)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// 检查任务是否正在运行
			var task models.SyncTask
			if err := database.DB.First(&task, "id = ?", taskID).Error; err == nil {
				// 如果任务不在运行，停止推送
				if !task.IsRunning {
					return
				}
			}
			s.sendUpdate(taskID, client)
		}
	}
}

// sendUpdate 发送更新
func (s *TaskSSEService) sendUpdate(taskID string, client chan SSEMessage) {
	// 获取进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
			// 客户端缓冲区满，跳过
		}
	}

	// 获取日志
	logs, err := s.logService.GetTaskLogs(taskID, 50)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "log",
			Data:  logs,
		}:
		default:
			// 客户端缓冲区满，跳过
		}
	}
}

// BroadcastProgressUpdate 广播进度更新
func (s *TaskSSEService) BroadcastProgressUpdate(taskID string) {
	// 检查任务是否正在运行
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err == nil {
		// 如果任务不在运行，停止推送
		if !task.IsRunning {
			return
		}
	}

	s.mu.RLock()
	clients, ok := s.clients[taskID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	// 获取最新进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err != nil {
		return
	}

	// 向所有客户端发送更新
	for client := range clients {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
			// 客户端缓冲区满，跳过
		}
	}
}

// BroadcastLogUpdate 广播日志更新
func (s *TaskSSEService) BroadcastLogUpdate(taskID string, logs []TaskLog) {
	// 检查任务是否正在运行
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err == nil {
		// 如果任务不在运行，停止推送
		if !task.IsRunning {
			return
		}
	}

	s.mu.RLock()
	clients, ok := s.clients[taskID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	// 向所有客户端发送更新
	for client := range clients {
		select {
		case client <- SSEMessage{
			Event: "log",
			Data:  logs,
		}:
		default:
			// 客户端缓冲区满，跳过
		}
	}
}

// FormatSSEMessage 格式化SSE消息
func FormatSSEMessage(msg SSEMessage) string {
	data, _ := json.Marshal(msg.Data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", msg.Event, string(data))
}
