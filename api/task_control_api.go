package api

import (
	"datatrace/common"
	"datatrace/services"

	"github.com/gin-gonic/gin"
)

// TaskControlAPI 任务控制API控制器
type TaskControlAPI struct {
	service *services.TaskControlService
}

// NewTaskControlAPI 创建任务控制API控制器
func NewTaskControlAPI() *TaskControlAPI {
	return &TaskControlAPI{
		service: services.NewTaskControlService(),
	}
}

// Start 启动任务
func (api *TaskControlAPI) Start(c *gin.Context) {
	taskID := c.Param("id")

	if err := api.service.StartTask(taskID); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "任务启动成功", nil)
}

// Pause 暂停任务
func (api *TaskControlAPI) Pause(c *gin.Context) {
	taskID := c.Param("id")

	if err := api.service.PauseTask(taskID); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "任务暂停成功", nil)
}

// Stop 停止任务
func (api *TaskControlAPI) Stop(c *gin.Context) {
	taskID := c.Param("id")

	if err := api.service.StopTask(taskID); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "任务停止成功", nil)
}
