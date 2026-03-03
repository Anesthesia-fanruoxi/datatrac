package api

import (
	"datatrace/services"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TaskSSEAPI SSE API控制器
type TaskSSEAPI struct {
	sseService *services.TaskSSEService
}

// NewTaskSSEAPI 创建SSE API控制器
func NewTaskSSEAPI() *TaskSSEAPI {
	return &TaskSSEAPI{
		sseService: services.NewTaskSSEService(),
	}
}

// StreamTaskUpdates 流式推送任务更新（已废弃，保留向后兼容）
func (api *TaskSSEAPI) StreamTaskUpdates(c *gin.Context) {
	taskID := c.Param("id")

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	client := make(chan services.SSEMessage, 10)

	// 添加客户端
	api.sseService.AddClient(taskID, client)

	// 确保清理资源
	defer func() {
		api.sseService.RemoveClient(taskID, client)
		close(client) // 在移除后关闭 channel
	}()

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// 创建退出信号通道
	done := make(chan struct{})

	// 启动推送协程
	go api.sseService.StreamTaskUpdates(taskID, client, done)

	// 监听客户端断开
	notify := c.Request.Context().Done()

	// 发送消息
	for {
		select {
		case <-notify:
			// 客户端断开连接，通知推送协程退出
			close(done)
			return
		case msg, ok := <-client:
			if !ok {
				// 通道关闭
				return
			}
			// 发送SSE消息
			_, err := io.WriteString(w, services.FormatSSEMessage(msg))
			if err != nil {
				close(done)
				return
			}
			flusher.Flush()
		}
	}
}

// StreamInitialize 流式推送初始化步骤更新
func (api *TaskSSEAPI) StreamInitialize(c *gin.Context) {
	taskID := c.Param("id")

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	client := make(chan services.SSEMessage, 10)

	// 添加客户端
	api.sseService.AddClient(taskID, client)

	// 确保清理资源
	defer func() {
		api.sseService.RemoveClient(taskID, client)
		close(client)
	}()

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// 创建退出信号通道
	done := make(chan struct{})

	// 启动推送协程（只推送初始化进度，不推送日志）
	go api.sseService.StreamInitialize(taskID, client, done)

	// 监听客户端断开
	notify := c.Request.Context().Done()

	// 发送消息
	for {
		select {
		case <-notify:
			close(done)
			return
		case msg, ok := <-client:
			if !ok {
				return
			}
			_, err := io.WriteString(w, services.FormatSSEMessage(msg))
			if err != nil {
				close(done)
				return
			}
			flusher.Flush()
		}
	}
}

// StreamSync 流式推送数据同步步骤更新
func (api *TaskSSEAPI) StreamSync(c *gin.Context) {
	taskID := c.Param("id")

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	client := make(chan services.SSEMessage, 10)

	// 添加客户端
	api.sseService.AddClient(taskID, client)

	// 确保清理资源
	defer func() {
		api.sseService.RemoveClient(taskID, client)
		close(client)
	}()

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// 创建退出信号通道
	done := make(chan struct{})

	// 启动推送协程（只推送数据同步进度，不推送日志）
	go api.sseService.StreamSync(taskID, client, done)

	// 监听客户端断开
	notify := c.Request.Context().Done()

	// 发送消息
	for {
		select {
		case <-notify:
			close(done)
			return
		case msg, ok := <-client:
			if !ok {
				return
			}
			_, err := io.WriteString(w, services.FormatSSEMessage(msg))
			if err != nil {
				close(done)
				return
			}
			flusher.Flush()
		}
	}
}

// StreamLogs 流式推送任务日志
func (api *TaskSSEAPI) StreamLogs(c *gin.Context) {
	taskID := c.Param("id")

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	client := make(chan services.SSEMessage, 10)

	// 添加客户端
	api.sseService.AddClient(taskID, client)

	// 确保清理资源
	defer func() {
		api.sseService.RemoveClient(taskID, client)
		close(client)
	}()

	// 获取响应写入器
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// 创建退出信号通道
	done := make(chan struct{})

	// 启动推送协程（只推送日志）
	go api.sseService.StreamLogs(taskID, client, done)

	// 监听客户端断开
	notify := c.Request.Context().Done()

	// 发送消息
	for {
		select {
		case <-notify:
			close(done)
			return
		case msg, ok := <-client:
			if !ok {
				return
			}
			_, err := io.WriteString(w, services.FormatSSEMessage(msg))
			if err != nil {
				close(done)
				return
			}
			flusher.Flush()
		}
	}
}
