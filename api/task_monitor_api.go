package api

import (
	"datatrace/common"
	"datatrace/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TaskMonitorAPI 任务监控API控制器
type TaskMonitorAPI struct {
	progressService *services.TaskProgressService
	logService      *services.TaskLogService
}

// NewTaskMonitorAPI 创建任务监控API控制器
func NewTaskMonitorAPI() *TaskMonitorAPI {
	return &TaskMonitorAPI{
		progressService: services.NewTaskProgressService(),
		logService:      services.NewTaskLogService(),
	}
}

// GetProgress 获取任务进度
func (api *TaskMonitorAPI) GetProgress(c *gin.Context) {
	taskID := c.Param("id")

	progress, err := api.progressService.GetTaskProgress(taskID)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, progress)
}

// GetLogs 获取任务日志
func (api *TaskMonitorAPI) GetLogs(c *gin.Context) {
	taskID := c.Param("id")

	// 获取limit参数，默认100条
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := api.logService.GetTaskLogs(taskID, limit)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, logs)
}
