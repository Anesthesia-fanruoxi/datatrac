package main

import (
	"datatrace/config"
	"datatrace/database"
	"datatrace/routers"
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

	// 3. 设置 Gin 模式
	gin.SetMode(config.GlobalConfig.Server.Mode)

	// 4. 设置路由
	r := routers.SetupRouter()

	// 5. 启动服务器
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	log.Println("========================================")
	log.Println("🚀 DataTrace 数据同步系统")
	log.Println("========================================")
	log.Printf("✅ 服务启动成功")
	log.Printf("📍 监听地址: http://localhost%s", addr)
	log.Printf("🏥 健康检查: http://localhost%s/health", addr)
	log.Printf("📚 API文档: http://localhost%s/api/v1", addr)
	log.Println("========================================")

	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ 启动服务器失败: %v", err)
	}
}
