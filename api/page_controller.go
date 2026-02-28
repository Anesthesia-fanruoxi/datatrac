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

// Index 首页
func (pc *PageController) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "base.html", nil)
}

// DataSourceList 数据源列表页
func (pc *PageController) DataSourceList(c *gin.Context) {
	c.HTML(http.StatusOK, "datasource/list.html", gin.H{
		"Title":  "数据源管理",
		"Active": "datasources",
	})
}

// DataSourceNew 新建数据源页
func (pc *PageController) DataSourceNew(c *gin.Context) {
	c.HTML(http.StatusOK, "datasource/form.html", gin.H{
		"Title":  "新建数据源",
		"Active": "datasources",
	})
}

// DataSourceEdit 编辑数据源页
func (pc *PageController) DataSourceEdit(c *gin.Context) {
	// TODO: 从数据库加载数据源信息
	c.HTML(http.StatusOK, "datasource/form.html", gin.H{
		"Title":  "编辑数据源",
		"Active": "datasources",
		// "DataSource": dataSource,
	})
}

// TaskList 任务列表页
func (pc *PageController) TaskList(c *gin.Context) {
	c.HTML(http.StatusOK, "task/list.html", gin.H{
		"Title":  "任务管理",
		"Active": "tasks",
	})
}

// TaskNew 新建任务页
func (pc *PageController) TaskNew(c *gin.Context) {
	// TODO: 实现任务配置向导页面
	c.HTML(http.StatusOK, "task/list.html", gin.H{
		"Title":  "新建任务",
		"Active": "tasks",
	})
}

// TaskEdit 编辑任务页
func (pc *PageController) TaskEdit(c *gin.Context) {
	// TODO: 实现任务编辑页面
	c.HTML(http.StatusOK, "task/list.html", gin.H{
		"Title":  "编辑任务",
		"Active": "tasks",
	})
}

// TaskMonitor 任务监控页
func (pc *PageController) TaskMonitor(c *gin.Context) {
	// TODO: 待创建独立的 task/monitor.html 模板，当前暂用 list.html
	c.HTML(http.StatusOK, "task/list.html", gin.H{
		"Title":  "任务执行监控",
		"Active": "tasks",
	})
}
