package api

import (
	"datatrace/common"
	"datatrace/services"

	"github.com/gin-gonic/gin"
)

// CredentialAPI 凭据API控制器
type CredentialAPI struct {
	service *services.CredentialService
}

// NewCredentialAPI 创建凭据API控制器
func NewCredentialAPI() *CredentialAPI {
	return &CredentialAPI{
		service: services.NewCredentialService(),
	}
}

// Create 创建凭据
func (api *CredentialAPI) Create(c *gin.Context) {
	var req services.CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	credential, err := api.service.Create(&req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, credential)
}

// List 获取凭据列表
func (api *CredentialAPI) List(c *gin.Context) {
	list, err := api.service.List()
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, list)
}

// GetByID 获取凭据详情
func (api *CredentialAPI) GetByID(c *gin.Context) {
	id := c.Param("id")
	credential, err := api.service.GetByID(id)
	if err != nil {
		common.NotFound(c, "凭据不存在")
		return
	}

	common.Success(c, credential)
}

// Update 更新凭据
func (api *CredentialAPI) Update(c *gin.Context) {
	id := c.Param("id")
	var req services.CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	credential, err := api.service.Update(id, &req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, credential)
}

// Delete 删除凭据
func (api *CredentialAPI) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := api.service.Delete(id); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "删除成功", nil)
}
