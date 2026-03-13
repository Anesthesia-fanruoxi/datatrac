package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PageController 页面控制器
type PageController struct{}

// NewPageController 创建页面控制器
func NewPageController() *PageController {
	return &PageController{}
}

// Index 首页 - 重定向到数据源页面
func (pc *PageController) Index(c *gin.Context) {
	c.Redirect(http.StatusFound, "/datasources")
}

// DataSource 数据源页面
func (pc *PageController) DataSource(c *gin.Context) {
	c.HTML(http.StatusOK, "datasource/index.html", gin.H{
		"Title":  "数据源配置",
		"Active": "datasources",
	})
}

// TaskConfig 任务配置页面
func (pc *PageController) TaskConfig(c *gin.Context) {
	c.HTML(http.StatusOK, "task-config/index.html", gin.H{
		"Title":  "任务配置",
		"Active": "task-config",
	})
}

// TaskMonitor 任务监控页面
func (pc *PageController) TaskMonitor(c *gin.Context) {
	c.HTML(http.StatusOK, "task-monitor/index.html", gin.H{
		"Title":  "任务执行监控",
		"Active": "task-monitor",
		"TaskID": "",
	})
}

// TaskMonitorWithID 任务监控页面（带任务ID）
func (pc *PageController) TaskMonitorWithID(c *gin.Context) {
	taskID := c.Param("id")
	c.HTML(http.StatusOK, "task-monitor/index.html", gin.H{
		"Title":  "任务执行监控",
		"Active": "task-monitor",
		"TaskID": taskID,
	})
}

// Credential 凭据管理页面
func (pc *PageController) Credential(c *gin.Context) {
	c.HTML(http.StatusOK, "credential/index.html", gin.H{
		"Title":  "凭据管理",
		"Active": "credentials",
	})
}
