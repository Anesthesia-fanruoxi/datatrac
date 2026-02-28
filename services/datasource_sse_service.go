package services

import (
	"encoding/json"
	"fmt"
	"sync"
)

// DataSourceSSEService 数据源SSE服务
type DataSourceSSEService struct {
	dsService *DataSourceService
	clients   map[chan DataSourceTestResult]bool
	mu        sync.RWMutex
}

// DataSourceTestResult 数据源测试结果
type DataSourceTestResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`  // testing/success/failed
	Message string `json:"message"` // 错误信息或版本信息
}

var (
	dsSSEInstance *DataSourceSSEService
	dsSSEOnce     sync.Once
)

// NewDataSourceSSEService 获取数据源SSE服务单例
func NewDataSourceSSEService() *DataSourceSSEService {
	dsSSEOnce.Do(func() {
		dsSSEInstance = &DataSourceSSEService{
			dsService: NewDataSourceService(),
			clients:   make(map[chan DataSourceTestResult]bool),
		}
	})
	return dsSSEInstance
}

// AddClient 添加客户端
func (s *DataSourceSSEService) AddClient(client chan DataSourceTestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client] = true
}

// RemoveClient 移除客户端
func (s *DataSourceSSEService) RemoveClient(client chan DataSourceTestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
	close(client)
}

// TestAllDataSources 测试所有数据源并推送结果
func (s *DataSourceSSEService) TestAllDataSources(client chan DataSourceTestResult) {
	// 获取所有数据源
	dataSources, err := s.dsService.List()
	if err != nil {
		return
	}

	// 逐个测试
	for _, ds := range dataSources {
		// 发送测试中状态
		select {
		case client <- DataSourceTestResult{
			ID:      ds.ID,
			Status:  "testing",
			Message: "正在测试连接...",
		}:
		default:
		}

		// 测试连接
		result, err := s.dsService.TestConnectionByID(ds.ID)
		if err != nil {
			// 发送失败结果
			select {
			case client <- DataSourceTestResult{
				ID:      ds.ID,
				Status:  "failed",
				Message: err.Error(),
			}:
			default:
			}
			continue
		}

		// 发送测试结果
		status := "failed"
		message := "连接失败"
		if result.Success {
			status = "success"
			if result.Version != "" {
				message = fmt.Sprintf("连接成功 (版本: %s)", result.Version)
			} else {
				message = "连接成功"
			}
		} else {
			message = result.Message
		}

		select {
		case client <- DataSourceTestResult{
			ID:      ds.ID,
			Status:  status,
			Message: message,
		}:
		default:
		}
	}
}

// FormatDataSourceSSEMessage 格式化SSE消息
func FormatDataSourceSSEMessage(result DataSourceTestResult) string {
	data, _ := json.Marshal(result)
	return fmt.Sprintf("event: test\ndata: %s\n\n", string(data))
}
