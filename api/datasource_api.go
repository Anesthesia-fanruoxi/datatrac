package api

import (
	"datatrace/common"
	"datatrace/services"
	"datatrace/utils"

	"github.com/gin-gonic/gin"
)

// DataSourceAPI 数据源API控制器
type DataSourceAPI struct {
	service *services.DataSourceService
}

// NewDataSourceAPI 创建数据源API控制器
func NewDataSourceAPI() *DataSourceAPI {
	return &DataSourceAPI{
		service: services.NewDataSourceService(),
	}
}

// Create 创建数据源
func (api *DataSourceAPI) Create(c *gin.Context) {
	var req services.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	ds, err := api.service.Create(&req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, ds)
}

// List 获取数据源列表
func (api *DataSourceAPI) List(c *gin.Context) {
	list, err := api.service.List()
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, list)
}

// GetByID 获取数据源详情
func (api *DataSourceAPI) GetByID(c *gin.Context) {
	id := c.Param("id")
	ds, err := api.service.GetByID(id)
	if err != nil {
		common.NotFound(c, "数据源不存在")
		return
	}

	common.Success(c, ds)
}

// Update 更新数据源
func (api *DataSourceAPI) Update(c *gin.Context) {
	id := c.Param("id")
	var req services.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	ds, err := api.service.Update(id, &req)
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.Success(c, ds)
}

// Delete 删除数据源
func (api *DataSourceAPI) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := api.service.Delete(id); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	common.SuccessWithMessage(c, "删除成功", nil)
}

// TestConnection 测试数据源连接
func (api *DataSourceAPI) TestConnection(c *gin.Context) {
	var req services.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := api.service.TestConnection(&req)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	if result.Success {
		common.Success(c, result)
	} else {
		common.Error(c, 500, result.Message)
	}
}

// TestConnectionByID 根据ID测试数据源连接
func (api *DataSourceAPI) TestConnectionByID(c *gin.Context) {
	id := c.Param("id")

	result, err := api.service.TestConnectionByID(id)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	if result.Success {
		common.Success(c, result)
	} else {
		common.Error(c, 500, result.Message)
	}
}

// GetDatabases 获取数据源的数据库列表（仅MySQL）
func (api *DataSourceAPI) GetDatabases(c *gin.Context) {
	id := c.Param("id")

	// 获取数据源
	ds, err := api.service.GetByID(id)
	if err != nil {
		common.NotFound(c, "数据源不存在")
		return
	}

	if ds.Type != "mysql" {
		common.BadRequest(c, "只有MySQL数据源支持此操作")
		return
	}

	// 解密密码
	crypto := utils.NewCryptoService()
	password, err := crypto.Decrypt(ds.Password)
	if err != nil {
		common.Error(c, 500, "密码解密失败")
		return
	}

	// 获取数据库列表
	mysqlService := services.NewMySQLMetadataService()
	databases, err := mysqlService.GetDatabases(ds.Host, ds.Port, ds.Username, password)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, databases)
}

// GetTables 获取指定数据库的表列表（仅MySQL）
func (api *DataSourceAPI) GetTables(c *gin.Context) {
	id := c.Param("id")
	database := c.Query("database")

	if database == "" {
		common.BadRequest(c, "database参数不能为空")
		return
	}

	// 获取数据源
	ds, err := api.service.GetByID(id)
	if err != nil {
		common.NotFound(c, "数据源不存在")
		return
	}

	if ds.Type != "mysql" {
		common.BadRequest(c, "只有MySQL数据源支持此操作")
		return
	}

	// 解密密码
	crypto := utils.NewCryptoService()
	password, err := crypto.Decrypt(ds.Password)
	if err != nil {
		common.Error(c, 500, "密码解密失败")
		return
	}

	// 获取表列表
	mysqlService := services.NewMySQLMetadataService()
	tables, err := mysqlService.GetTables(ds.Host, ds.Port, ds.Username, password, database)
	if err != nil {
		common.Error(c, 500, err.Error())
		return
	}

	common.Success(c, tables)
}
