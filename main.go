package main

import (
	"asily_blog/internal/routes"
	"asily_blog/internal/utils"
	"asily_blog/pkg/config"
	"asily_blog/pkg/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化日志
	logger.InitLogger()

	// 加载配置
	config.LoadConfig()

	// 初始化路由
	r := gin.Default()
	routes.SetupRoutes(r)

	// 连接数据库
	utils.ConnectDB()

	// 启动服务器
	r.Run(":" + config.C.Server.Port)
}
