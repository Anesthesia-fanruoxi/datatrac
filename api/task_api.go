package api

import (
	"datatrace/common"
	"datatrace/services"

	"github.com/gin-gonic/gin"
)

// TaskAPI 任务API控制器
type TaskAPI struct {
	service *services.TaskService
}

// NewTaskAPI 创建任务API控制器
func NewTaskAPI() *TaskAPI {
	return &TaskAPI{
		service: services.NewTaskService(),
	}
}

// Create 创建任务
func (api *TaskAPI) Create(c *gin.Context) {
	var req services.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	task, err := api.service.Create(&req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, task)
}

// List 获取任务列表
func (api *TaskAPI) List(c *gin.Context) {
	list, err := api.service.List()
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, list)
}

// GetByID 获取任务详情
func (api *TaskAPI) GetByID(c *gin.Context) {
	id := c.Param("id")
	task, err := api.service.GetByID(id)
	if err != nil {
		common.NotFound(c, "任务不存在")
		return
	}

	common.Success(c, task)
}

// UpdateConfig 更新任务配置
func (api *TaskAPI) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	var req services.UpdateTaskConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	task, err := api.service.UpdateConfig(id, &req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, task)
}

// Delete 删除任务
func (api *TaskAPI) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := api.service.Delete(id); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "删除成功", nil)
}
