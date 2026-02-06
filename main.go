package main

import (
	"ai-api-platform/backend/handlers"
	"ai-api-platform/backend/middleware"
	"ai-api-platform/backend/models"
	"ai-api-platform/backend/services"
	"ai-api-platform/backend/static"
	"ai-api-platform/backend/utils"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. 初始化配置
	if err := utils.InitConfig("config/config.yaml"); err != nil {
		log.Fatalf("Init config failed: %v", err)
	}

	// 2. 初始化数据库
	if err := models.InitDB(); err != nil {
		log.Fatalf("Init DB failed: %v", err)
	}

	// 3. 检查并创建默认管理员
	createDefaultAdmin()

	// 4. 初始化统计服务
	services.InitStats()

	// 5. 初始化 API 路径缓存
	if err := services.InitEndpointCache(); err != nil {
		log.Fatalf("Init endpoint cache failed: %v", err)
	}
	fmt.Println("API endpoint cache initialized successfully")

	// 6. 设置路由
	r := gin.Default()

	// 管理平台接口
	admin := r.Group("/admin")
	{
		admin.POST("/login", handlers.Login)

		// 需要授权的接口
		auth := admin.Group("/")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.GET("/providers", handlers.GetProviders)
			auth.POST("/providers", handlers.CreateProvider)
			auth.PUT("/providers/:id", handlers.UpdateProvider)
			auth.DELETE("/providers/:id", handlers.DeleteProvider)

			auth.GET("/endpoints", handlers.GetEndpoints)
			auth.POST("/endpoints", handlers.CreateEndpoint)
			auth.PUT("/endpoints/:id", handlers.UpdateEndpoint)
			auth.DELETE("/endpoints/:id", handlers.DeleteEndpoint)

			auth.GET("/stats", handlers.GetStats)

			// 用户管理接口
			auth.GET("/user/info", handlers.GetUserInfo)
			auth.PUT("/user/password", handlers.UpdatePassword)
			auth.PUT("/user/info", handlers.UpdateUserInfo)
		}
	}

	// 静态资源与代理逻辑
	// 注意：ProxyHandler 内部会检查路径是否存在于数据库中
	// 如果不匹配，则尝试作为静态资源服务
	static.Serve(r, utils.GlobalConfig.Server.FrontendPath)

	// 启动服务器
	port := utils.GlobalConfig.Server.Port
	if port == 0 {
		port = 8080
	}
	fmt.Printf("Server starting at :%d...\n", port)
	r.Run(fmt.Sprintf(":%d", port))
}

func createDefaultAdmin() {
	var count int64
	models.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Username: "admin",
			Password: string(hashedPassword),
		}
		models.DB.Create(&admin)
		fmt.Println("Created default admin user: admin / admin123")
	}
}
