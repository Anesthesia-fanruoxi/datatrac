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
	detailClients   map[string]map[chan SSEMessage]struct{}            // taskID -> detail clients
	progressClients map[string]map[chan SSEMessage]struct{}            // taskID -> progress clients
	logClients      map[string]map[string]map[chan SSEMessage]struct{} // taskID -> category -> clients (for logs)
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
			detailClients:   make(map[string]map[chan SSEMessage]struct{}),
			progressClients: make(map[string]map[chan SSEMessage]struct{}),
			logClients:      make(map[string]map[string]map[chan SSEMessage]struct{}),
		}
	})
	return taskSSEInstance
}

// AddDetailClient 添加任务详情客户端
func (s *TaskSSEService) AddDetailClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.detailClients[taskID] == nil {
		s.detailClients[taskID] = make(map[chan SSEMessage]struct{})
	}
	s.detailClients[taskID][client] = struct{}{}
}

// RemoveDetailClient 移除任务详情客户端
func (s *TaskSSEService) RemoveDetailClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.detailClients[taskID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(s.detailClients, taskID)
		}
	}
}

// AddProgressClient 添加进度客户端
func (s *TaskSSEService) AddProgressClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.progressClients[taskID] == nil {
		s.progressClients[taskID] = make(map[chan SSEMessage]struct{})
	}
	s.progressClients[taskID][client] = struct{}{}
}

// RemoveProgressClient 移除进度客户端
func (s *TaskSSEService) RemoveProgressClient(taskID string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.progressClients[taskID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(s.progressClients, taskID)
		}
	}
}

// AddClient 添加客户端（已废弃，保留向后兼容）
func (s *TaskSSEService) AddClient(taskID string, client chan SSEMessage) {
	// 默认添加到进度客户端
	s.AddProgressClient(taskID, client)
}

// RemoveClient 移除客户端（已废弃，保留向后兼容）
func (s *TaskSSEService) RemoveClient(taskID string, client chan SSEMessage) {
	// 尝试从两个列表中移除
	s.RemoveDetailClient(taskID, client)
	s.RemoveProgressClient(taskID, client)
}

// AddLogClient 添加日志客户端（带category）
func (s *TaskSSEService) AddLogClient(taskID string, category string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.logClients[taskID] == nil {
		s.logClients[taskID] = make(map[string]map[chan SSEMessage]struct{})
	}
	if s.logClients[taskID][category] == nil {
		s.logClients[taskID][category] = make(map[chan SSEMessage]struct{})
	}
	s.logClients[taskID][category][client] = struct{}{}
}

// RemoveLogClient 移除日志客户端
func (s *TaskSSEService) RemoveLogClient(taskID string, category string, client chan SSEMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if categories, ok := s.logClients[taskID]; ok {
		if clients, ok := categories[category]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(categories, category)
			}
		}
		if len(categories) == 0 {
			delete(s.logClients, taskID)
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

// StreamLogs 流式推送任务日志（事件驱动 + 文件监听）
func (s *TaskSSEService) StreamLogs(taskID string, category string, client chan SSEMessage, done <-chan struct{}) {
	// 创建日志文件监听器
	watcher, err := NewLogFileWatcher(taskID, category, client, done)
	if err != nil {
		fmt.Printf("创建日志文件监听器失败: %v\n", err)
		return
	}

	// 启动监听
	if err := watcher.Start(); err != nil {
		fmt.Printf("启动日志文件监听失败: %v\n", err)
		return
	}

	// 保持连接，等待客户端断开
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

	// 获取所有日志（默认all分类）
	logs, err := s.logService.GetTaskLogs(taskID, "all", 200)
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

	// 获取日志（默认all分类）
	logs, err := s.logService.GetTaskLogs(taskID, "all", 50)
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
	clients, ok := s.progressClients[taskID]
	s.mu.RUnlock()

	fmt.Printf("[DEBUG] BroadcastProgressUpdate - taskID: %s, 客户端数量: %d\n", taskID, len(clients))

	if !ok || len(clients) == 0 {
		fmt.Printf("[DEBUG] BroadcastProgressUpdate - 没有进度客户端,跳过推送\n")
		return
	}

	// 获取最新进度
	progress, err := s.progressService.GetTaskProgress(taskID)
	if err != nil {
		fmt.Printf("[DEBUG] BroadcastProgressUpdate - 获取进度失败: %v\n", err)
		return
	}

	fmt.Printf("[DEBUG] BroadcastProgressUpdate - 进度数据: sync_mode=%s, current_step=%s\n",
		progress.SyncMode, progress.CurrentStep)

	// 向所有进度客户端发送更新
	sentCount := 0
	for client := range clients {
		func(c chan SSEMessage) {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略错误
					fmt.Printf("[DEBUG] BroadcastProgressUpdate - 发送失败(channel已关闭)\n")
				}
			}()
			select {
			case c <- SSEMessage{
				Event: "progress",
				Data:  progress,
			}:
				sentCount++
			default:
				// 客户端缓冲区满，跳过
				fmt.Printf("[DEBUG] BroadcastProgressUpdate - 客户端缓冲区满,跳过\n")
			}
		}(client)
	}

	fmt.Printf("[DEBUG] BroadcastProgressUpdate - 成功发送给 %d 个客户端\n", sentCount)
}

// BroadcastTaskDetailUpdate 广播任务详情更新
func (s *TaskSSEService) BroadcastTaskDetailUpdate(taskID string) {
	s.mu.RLock()
	clients, ok := s.detailClients[taskID]
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

	// 向所有详情客户端发送更新
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

// BroadcastLogUpdate 广播日志更新（支持category过滤）
func (s *TaskSSEService) BroadcastLogUpdate(taskID string, log TaskLog) {
	s.mu.RLock()
	categories, ok := s.logClients[taskID]
	s.mu.RUnlock()

	if !ok || len(categories) == 0 {
		return
	}

	// 向订阅了对应category的客户端发送日志
	for category, clients := range categories {
		// 过滤：只向订阅了对应category的客户端发送
		// all: 接收所有日志
		// initialize: 只接收category=initialize的日志
		// complete: 只接收category=complete的日志
		shouldSend := false
		if category == "all" {
			shouldSend = true
		} else if category == log.Category {
			shouldSend = true
		}

		if !shouldSend {
			continue
		}

		// 向该category的所有客户端发送日志
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
}

// FormatSSEMessage 格式化SSE消息
func FormatSSEMessage(msg SSEMessage) string {
	data, _ := json.Marshal(msg.Data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", msg.Event, string(data))
}
