package api

import (
	"datatrace/common"
	"datatrace/services"
	"time"

	"github.com/gin-gonic/gin"
)

// IncrementalDebugAPI 增量同步调试API
type IncrementalDebugAPI struct{}

// NewIncrementalDebugAPI 创建增量同步调试API
func NewIncrementalDebugAPI() *IncrementalDebugAPI {
	return &IncrementalDebugAPI{}
}

// RegisterRoutes 注册路由
func (api *IncrementalDebugAPI) RegisterRoutes(r *gin.RouterGroup) {
	debug := r.Group("/debug")
	{
		debug.POST("/incremental/:taskId/diagnose", api.DiagnoseIncrementalSync)
		debug.POST("/incremental/:taskId/test-binlog", api.TestBinlogListener)
		debug.GET("/incremental/:taskId/status", api.GetIncrementalSyncStatus)
	}
}

// DiagnoseIncrementalSync 诊断增量同步
func (api *IncrementalDebugAPI) DiagnoseIncrementalSync(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		common.BadRequest(c, "任务ID不能为空")
		return
	}

	// 创建调试器
	debugger := services.NewIncrementalSyncDebugger(taskID)

	// 执行诊断
	if err := debugger.DiagnoseIncrementalSync(); err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.SuccessWithMessage(c, "增量同步诊断完成，请查看任务日志", nil)
}

// TestBinlogListenerRequest 测试binlog监听器请求
type TestBinlogListenerRequest struct {
	Duration int `json:"duration" binding:"required"` // 测试持续时间（秒）
}

// TestBinlogListener 测试binlog监听器
func (api *IncrementalDebugAPI) TestBinlogListener(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		common.BadRequest(c, "任务ID不能为空")
		return
	}

	var req TestBinlogListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if req.Duration <= 0 || req.Duration > 300 {
		common.BadRequest(c, "测试持续时间必须在1-300秒之间")
		return
	}

	// 创建调试器
	debugger := services.NewIncrementalSyncDebugger(taskID)

	// 异步执行测试
	go func() {
		duration := time.Duration(req.Duration) * time.Second
		if err := debugger.TestBinlogListener(duration); err != nil {
			// 错误会记录到日志中
		}
	}()

	common.SuccessWithMessage(c, "binlog监听器测试已启动，请查看任务日志", nil)
}

// GetIncrementalSyncStatus 获取增量同步状态
func (api *IncrementalDebugAPI) GetIncrementalSyncStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		common.BadRequest(c, "任务ID不能为空")
		return
	}

	// 获取任务控制服务
	taskControlService := services.NewTaskControlService()

	// 获取增量同步状态
	status, err := taskControlService.GetIncrementalSyncStatus(taskID)
	if err != nil {
		common.NotFound(c, err.Error())
		return
	}

	common.Success(c, status)
}
