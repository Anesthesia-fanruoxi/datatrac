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
	clients         map[string]map[chan SSEMessage]struct{} // taskID -> clients
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
// NewTaskSSEService 获取SSE服务单例
func NewTaskSSEService() *TaskSSEService {
	taskSSEOnce.Do(func() {
		taskSSEInstance = &TaskSSEService{
			progressService: NewTaskProgressService(),
			logService:      NewTaskLogService(),
			clients:         make(map[string]map[chan SSEMessage]struct{}),
		}
	})
	return taskSSEInstance
}

// AddClient 添加客户端
// AddClient 添加客户端
func (s *TaskSSEService) AddClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clients[taskID] == nil {
		s.clients[taskID] = make(map[chan SSEMessage]struct{})
	}
	s.clients[taskID][client] = struct{}{}
}

// RemoveClient 移除客户端
// RemoveClient 移除客户端（不关闭 channel，由调用方管理）
func (s *TaskSSEService) RemoveClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.clients[taskID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(s.clients, taskID)
		}
	}
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

// StreamInitialize 流式推送初始化步骤更新
func (s *TaskSSEService) StreamInitialize(taskID string, client chan SSEMessage, done <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second) // 初始化阶段更新更频繁
	defer ticker.Stop()

	// 立即发送一次
	s.sendInitializeUpdate(taskID, client)

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
				// 如果已经不在初始化阶段，停止推送
				if task.CurrentStep != "initialize" && task.CurrentStep != "" {
					return
				}
			}
			s.sendInitializeUpdate(taskID, client)
		}
	}
}

// StreamSync 流式推送数据同步步骤更新
func (s *TaskSSEService) StreamSync(taskID string, client chan SSEMessage, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 立即发送一次
	s.sendSyncUpdate(taskID, client)

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
				// 如果不在数据同步阶段，停止推送
				if task.CurrentStep != "sync_data" {
					return
				}
			}
			s.sendSyncUpdate(taskID, client)
		}
	}
}

// StreamIncremental 流式推送增量消费步骤更新
func (s *TaskSSEService) StreamIncremental(taskID string, client chan SSEMessage, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 立即发送一次
	s.sendIncrementalUpdate(taskID, client)

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
				// 如果不在增量消费阶段，停止推送
				if task.CurrentStep != "incremental" {
					return
				}
			}
			s.sendIncrementalUpdate(taskID, client)
		}
	}
}

// StreamLogs 流式推送任务日志（事件驱动）
func (s *TaskSSEService) StreamLogs(taskID string, client chan SSEMessage, done <-chan struct{}) {
	// 立即发送一次初始数据
	s.sendLogsUpdate(taskID, client)

	// 保持连接，等待广播或客户端断开
	<-done
}

// StreamTaskDetail 流式推送任务详情（事件驱动）
func (s *TaskSSEService) StreamTaskDetail(taskID string, client chan SSEMessage, done <-chan struct{}) {
	// 立即发送一次初始数据
	s.sendTaskDetailUpdate(taskID, client)

	// 保持连接，等待广播或客户端断开
	<-done
}

// StreamProgress 流式推送任务进度（事件驱动）
func (s *TaskSSEService) StreamProgress(taskID string, client chan SSEMessage, done <-chan struct{}) {
	// 立即发送一次初始数据
	s.sendProgressUpdate(taskID, client)

	// 保持连接，等待广播或客户端断开
	<-done
}

// sendInitializeUpdate 发送初始化步骤更新（只推送进度）
func (s *TaskSSEService) sendInitializeUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
		}
	}
}

// sendSyncUpdate 发送数据同步步骤更新（只推送进度）
func (s *TaskSSEService) sendSyncUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
		}
	}
}

// sendIncrementalUpdate 发送增量消费步骤更新（只推送进度）
func (s *TaskSSEService) sendIncrementalUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
		}
	}
}

// sendLogsUpdate 发送日志更新（推送所有日志）
func (s *TaskSSEService) sendLogsUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取所有日志
	logs, err := s.logService.GetTaskLogs(taskID, 200)
	if err == nil && len(logs) > 0 {
		select {
		case client <- SSEMessage{
			Event: "log",
			Data:  logs,
		}:
		default:
		}
	}
}

// sendTaskDetailUpdate 发送任务详情更新
func (s *TaskSSEService) sendTaskDetailUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取任务详情
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return
	}

	// 构建任务详情数据
	taskDetail := map[string]interface{}{
		"id":           task.ID,
		"name":         task.Name,
		"status":       task.Status,
		"is_running":   task.IsRunning,
		"current_step": task.CurrentStep,
		"sync_mode":    task.SyncMode,
	}

	select {
	case client <- SSEMessage{
		Event: "task_detail",
		Data:  taskDetail,
	}:
	default:
	}
}

// sendProgressUpdate 发送统一的进度更新
func (s *TaskSSEService) sendProgressUpdate(taskID string, client chan SSEMessage) {
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

	// 获取进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err == nil {
		select {
		case client <- SSEMessage{
			Event: "progress",
			Data:  progress,
		}:
		default:
		}
	}
}

// sendUpdate 发送更新
// sendUpdate 发送更新（安全发送，防止 panic）
func (s *TaskSSEService) sendUpdate(taskID string, client chan SSEMessage) {
	// 使用 defer + recover 防止 panic
	defer func() {
		if r := recover(); r != nil {
			// channel 已关闭，忽略错误
		}
	}()

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
	s.mu.RLock()
	clients, ok := s.clients[taskID]
	s.mu.RUnlock()

	if !ok || len(clients) == 0 {
		return
	}

	// 获取最新进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err != nil {
		return
	}

	// 向所有客户端发送更新
	for client := range clients {
		func(c chan SSEMessage) {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略错误
				}
			}()
			select {
			case c <- SSEMessage{
				Event: "progress",
				Data:  progress,
			}:
			default:
				// 客户端缓冲区满，跳过
			}
		}(client)
	}
}

// BroadcastTaskDetailUpdate 广播任务详情更新
func (s *TaskSSEService) BroadcastTaskDetailUpdate(taskID string) {
	s.mu.RLock()
	clients, ok := s.clients[taskID]
	s.mu.RUnlock()

	if !ok || len(clients) == 0 {
		return
	}

	// 获取任务详情
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return
	}

	// 构建任务详情数据
	taskDetail := map[string]interface{}{
		"id":           task.ID,
		"name":         task.Name,
		"status":       task.Status,
		"is_running":   task.IsRunning,
		"current_step": task.CurrentStep,
		"sync_mode":    task.SyncMode,
	}

	// 向所有客户端发送更新
	for client := range clients {
		func(c chan SSEMessage) {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略错误
				}
			}()
			select {
			case c <- SSEMessage{
				Event: "task_detail",
				Data:  taskDetail,
			}:
			default:
				// 客户端缓冲区满，跳过
			}
		}(client)
	}
}

// BroadcastLogUpdate 广播日志更新
func (s *TaskSSEService) BroadcastLogUpdate(taskID string, log TaskLog) {
	s.mu.RLock()
	clients, ok := s.clients[taskID]
	s.mu.RUnlock()

	if !ok || len(clients) == 0 {
		return
	}

	// 向所有客户端发送单条日志
	for client := range clients {
		func(c chan SSEMessage) {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略错误
				}
			}()
			select {
			case c <- SSEMessage{
				Event: "log",
				Data:  []TaskLog{log}, // 包装成数组，保持前端兼容
			}:
			default:
				// 客户端缓冲区满，跳过
			}
		}(client)
	}
}

// FormatSSEMessage 格式化SSE消息
func FormatSSEMessage(msg SSEMessage) string {
	data, _ := json.Marshal(msg.Data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", msg.Event, string(data))
}
