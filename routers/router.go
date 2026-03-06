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
	// 使用 Gin 自带的日志中间件
	r.Use(gin.Logger())

	// 加载 HTML 模板
	r.LoadHTMLGlob("templates/**/*.html")

	// 静态资源
	r.Static("/static", "./static")

	// 页面控制器
	pageCtrl := api.NewPageController()

	// 页面路由
	r.GET("/", pageCtrl.Index)
	r.GET("/datasources", pageCtrl.DataSource)
	r.GET("/task-config", pageCtrl.TaskConfig)
	r.GET("/task-monitor", pageCtrl.TaskMonitor)
	r.GET("/task-monitor/:id", pageCtrl.TaskMonitorWithID)

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
			datasources.GET("/:id/database-tables", dsAPI.GetDatabasesWithTables) // 新增：获取完整树形结构
			datasources.PUT("/:id", dsAPI.Update)
			datasources.DELETE("/:id", dsAPI.Delete)

			// SSE流式推送测试结果
			datasources.GET("/test/stream", dsSSEAPI.StreamTestResults)
		}

		// 任务管理
		taskAPI := api.NewTaskAPI()
		taskControlAPI := api.NewTaskControlAPI()
		taskSSEAPI := api.NewTaskSSEAPI()
		tasks := apiGroup.Group("/tasks")
		{
			tasks.GET("", taskAPI.List)
			tasks.POST("", taskAPI.Create)
			tasks.GET("/:id", taskAPI.GetByID)
			tasks.PUT("/:id/config", taskAPI.UpdateConfig)
			tasks.DELETE("/:id", taskAPI.Delete)

			// 任务控制
			tasks.POST("/:id/start", taskControlAPI.Start)
			tasks.POST("/:id/pause", taskControlAPI.Pause)
			tasks.POST("/:id/stop", taskControlAPI.Stop)

			// SSE流式推送（只保留3个SSE接口）
			tasks.GET("/:id/stream/detail", taskSSEAPI.StreamTaskDetail) // 任务详情SSE
			tasks.GET("/:id/stream/progress", taskSSEAPI.StreamProgress) // 统一进度SSE
			tasks.GET("/:id/stream/logs", taskSSEAPI.StreamLogs)         // 日志SSE
		}
	}

	return r
}
