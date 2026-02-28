package routers

import (
	"datatrace/api"
	"datatrace/common"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.New()

	// 使用中间件
	r.Use(gin.Recovery())
	// 只在 debug 模式下使用日志中间件
	// r.Use(common.Logger())

	// 加载 HTML 模板
	r.LoadHTMLGlob("templates/**/*.html")

	// 静态资源
	r.Static("/static", "./static")

	// 页面控制器
	pageCtrl := api.NewPageController()

	// 页面路由
	r.GET("/", pageCtrl.Index)
	r.GET("/datasources", pageCtrl.DataSourceList)
	r.GET("/datasources/new", pageCtrl.DataSourceNew)
	r.GET("/datasources/:id/edit", pageCtrl.DataSourceEdit)
	r.GET("/tasks", pageCtrl.TaskList)
	r.GET("/tasks/new", pageCtrl.TaskNew)
	r.GET("/tasks/:id/edit", pageCtrl.TaskEdit)
	r.GET("/tasks/monitor", pageCtrl.TaskMonitor)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		common.Success(c, gin.H{
			"status": "ok",
		})
	})

	// API 路由组
	apiGroup := r.Group("/api/v1")
	{
		// 数据源管理
		dsAPI := api.NewDataSourceAPI()
		dsSSEAPI := api.NewDataSourceSSEAPI()
		datasources := apiGroup.Group("/datasources")
		{
			datasources.GET("", dsAPI.List)
			datasources.POST("", dsAPI.Create)
			datasources.POST("/test", dsAPI.TestConnection)
			datasources.GET("/:id", dsAPI.GetByID)
			datasources.POST("/:id/test", dsAPI.TestConnectionByID)
			datasources.GET("/:id/databases", dsAPI.GetDatabases)
			datasources.GET("/:id/tables", dsAPI.GetTables)
			datasources.PUT("/:id", dsAPI.Update)
			datasources.DELETE("/:id", dsAPI.Delete)

			// SSE流式推送测试结果
			datasources.GET("/test/stream", dsSSEAPI.StreamTestResults)
		}

		// 任务管理
		taskAPI := api.NewTaskAPI()
		taskMonitorAPI := api.NewTaskMonitorAPI()
		taskControlAPI := api.NewTaskControlAPI()
		taskSSEAPI := api.NewTaskSSEAPI()
		tasks := apiGroup.Group("/tasks")
		{
			tasks.GET("", taskAPI.List)
			tasks.POST("", taskAPI.Create)
			tasks.GET("/:id", taskAPI.GetByID)
			tasks.PUT("/:id/config", taskAPI.UpdateConfig)
			tasks.DELETE("/:id", taskAPI.Delete)

			// 任务监控
			tasks.GET("/:id/progress", taskMonitorAPI.GetProgress)
			tasks.GET("/:id/logs", taskMonitorAPI.GetLogs)

			// 任务控制
			tasks.POST("/:id/start", taskControlAPI.Start)
			tasks.POST("/:id/pause", taskControlAPI.Pause)
			tasks.POST("/:id/stop", taskControlAPI.Stop)

			// SSE流式推送
			tasks.GET("/:id/stream", taskSSEAPI.StreamTaskUpdates)
		}
	}

	return r
}
