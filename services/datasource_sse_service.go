package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const (
	// DataSourceCheckInterval 数据源健康检查间隔（秒）
	DataSourceCheckInterval = 10
)

// DataSourceSSEService 数据源SSE服务
type DataSourceSSEService struct {
	dsService   *DataSourceService
	clients     map[chan []DataSourceTestResult]struct{}
	mu          sync.RWMutex
	stopCh      chan struct{}
	running     bool
	lastResults map[string]DataSourceTestResult // 数据源ID -> 最新测试结果
	resultMutex sync.RWMutex
}

// DataSourceTestResult 数据源测试结果
type DataSourceTestResult struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`    // testing/success/failed
	Message   string    `json:"message"`   // 错误信息或版本信息
	Timestamp time.Time `json:"timestamp"` // 测试时间
}

var (
	dsSSEInstance *DataSourceSSEService
	dsSSEOnce     sync.Once
)

// NewDataSourceSSEService 获取数据源SSE服务单例
func NewDataSourceSSEService() *DataSourceSSEService {
	dsSSEOnce.Do(func() {
		dsSSEInstance = &DataSourceSSEService{
			dsService:   NewDataSourceService(),
			clients:     make(map[chan []DataSourceTestResult]struct{}),
			stopCh:      make(chan struct{}),
			running:     false,
			lastResults: make(map[string]DataSourceTestResult),
		}
	})
	return dsSSEInstance
}

// StartHealthCheck 启动后台健康检查（只启动一次）
func (s *DataSourceSSEService) StartHealthCheck() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.healthCheckLoop()
}

// healthCheckLoop 后台健康检查循环
func (s *DataSourceSSEService) healthCheckLoop() {
	ticker := time.NewTicker(DataSourceCheckInterval * time.Second)
	defer func() {
		ticker.Stop()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// 启动后立即检查一次
	s.checkAllDataSources()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAllDataSources()
		}
	}
}

// checkAllDataSources 检查所有数据源，返回状态变化的结果
func (s *DataSourceSSEService) checkAllDataSources() []DataSourceTestResult {
	// 获取所有数据源
	dataSources, err := s.dsService.List()
	if err != nil {
		return nil
	}

	// 并发测试所有数据源
	var wg sync.WaitGroup
	resultsCh := make(chan DataSourceTestResult, len(dataSources))

	for _, ds := range dataSources {
		wg.Add(1)
		go func(dsID string) {
			defer wg.Done()
			result := s.testSingleDataSource(dsID)

			// 检查状态是否变化
			s.resultMutex.RLock()
			lastResult, exists := s.lastResults[dsID]
			s.resultMutex.RUnlock()

			// 更新缓存
			s.resultMutex.Lock()
			s.lastResults[dsID] = result
			s.resultMutex.Unlock()

			// 首次推送或状态变化时发送
			if !exists || lastResult.Status != result.Status {
				resultsCh <- result
			}
		}(ds.ID)
	}
	wg.Wait()
	close(resultsCh)

	// 收集变化的结果
	var changedResults []DataSourceTestResult
	for result := range resultsCh {
		changedResults = append(changedResults, result)
	}

	// 有变化时广播
	if len(changedResults) > 0 {
		s.broadcast(changedResults)
	}

	return changedResults
}

// broadcast 广播给所有客户端
func (s *DataSourceSSEService) broadcast(results []DataSourceTestResult) {
	s.mu.RLock()
	clients := make([]chan []DataSourceTestResult, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.RUnlock()

	for _, client := range clients {
		func(c chan []DataSourceTestResult) {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略错误
				}
			}()
			select {
			case c <- results:
			default:
			}
		}(client)
	}
}

// testSingleDataSource 测试单个数据源
func (s *DataSourceSSEService) testSingleDataSource(dsID string) DataSourceTestResult {
	testResult, err := s.dsService.TestConnectionByID(dsID)

	status := "failed"
	message := "连接失败"
	if err != nil {
		message = err.Error()
	} else if testResult.Success {
		status = "success"
		if testResult.Version != "" {
			message = fmt.Sprintf("连接成功 (版本: %s)", testResult.Version)
		} else {
			message = "连接成功"
		}
	} else {
		message = testResult.Message
	}

	return DataSourceTestResult{
		ID:        dsID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// SendCachedResults 发送缓存的测试结果给新客户端（批量发送）
func (s *DataSourceSSEService) SendCachedResults(client chan []DataSourceTestResult) {
	s.resultMutex.RLock()
	results := make([]DataSourceTestResult, 0, len(s.lastResults))
	for _, result := range s.lastResults {
		results = append(results, result)
	}
	s.resultMutex.RUnlock()

	if len(results) > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// channel 已关闭，忽略
				}
			}()
			select {
			case client <- results:
			default:
			}
		}()
	}
}

// AddClient 添加客户端
func (s *DataSourceSSEService) AddClient(client chan []DataSourceTestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client] = struct{}{}
}

// RemoveClient 移除客户端（不关闭 channel，由调用方管理）
func (s *DataSourceSSEService) RemoveClient(client chan []DataSourceTestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
}

// FormatDataSourceSSEMessage 格式化SSE消息（数组格式）
func FormatDataSourceSSEMessage(results []DataSourceTestResult) string {
	data, _ := json.Marshal(results)
	return fmt.Sprintf("event: test\ndata: %s\n\n", string(data))
}
