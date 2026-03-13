package api

import (
	"datatrace/services"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DataSourceSSEAPI 数据源SSE API控制器
type DataSourceSSEAPI struct {
	sseService *services.DataSourceSSEService
}

// NewDataSourceSSEAPI 创建数据源SSE API控制器
func NewDataSourceSSEAPI() *DataSourceSSEAPI {
	return &DataSourceSSEAPI{
		sseService: services.NewDataSourceSSEService(),
	}
}

// StreamTestResults 流式推送测试结果
func (api *DataSourceSSEAPI) StreamTestResults(c *gin.Context) {
	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建客户端通道（接收数组）
	client := make(chan []services.DataSourceTestResult, 10)

	// 添加客户端
	api.sseService.AddClient(client)

	// 确保清理资源
	defer func() {
		api.sseService.RemoveClient(client)
		close(client)
	}()

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// 立即推送当前缓存的测试结果
	api.sseService.SendCachedResults(client)

	// 监听客户端断开
	notify := c.Request.Context().Done()

	// 发送消息
	for {
		select {
		case <-notify:
			return
		case results, ok := <-client:
			if !ok {
				return
			}
			// 发送SSE消息（数组格式）
			_, err := io.WriteString(w, services.FormatDataSourceSSEMessage(results))
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
