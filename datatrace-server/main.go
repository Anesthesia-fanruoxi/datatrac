package main

import (
	"datatrace/config"
	"datatrace/database"
	"datatrace/routers"
	"datatrace/services"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	if err := config.LoadConfig("config.yaml"); err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 2. 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("❌ 初始化数据库失败: %v", err)
	}
	defer database.CloseDB()

	// 3. 初始化 Redis
	if err := database.InitRedis(); err != nil {
		log.Printf("⚠️  初始化 Redis 失败: %v", err)
		// Redis 失败不阻止程序启动，增量统计功能将不可用
	} else {
		log.Println("✅ Redis 连接成功")

		// 4. 加载所有任务配置到Redis
		configCache := services.NewConfigCacheService()
		if err := configCache.InitAllTaskConfigs(); err != nil {
			log.Printf("⚠️  加载任务配置到Redis失败: %v", err)
		}
	}
	defer database.CloseRedis()

	// 5. 设置 Gin 模式
	gin.SetMode(config.GlobalConfig.Server.Mode)

	// 6. 启动数据源健康检查
	dsSSE := services.NewDataSourceSSEService()
	dsSSE.StartHealthCheck()
	log.Println("✅ 数据源健康检查已启动")

	// 7. 设置路由
	r := routers.SetupRouter()

	// 7. 启动服务器
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	log.Println("========================================")
	log.Println("🚀 DataTrace 数据同步系统")
	log.Println("========================================")
	log.Printf("✅ 服务启动成功")
	log.Printf("📍 监听地址: http://localhost%s", addr)
	log.Printf("🏥 健康检查: http://localhost%s/health", addr)
	log.Println("========================================")

	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ 启动服务器失败: %v", err)
	}
}
