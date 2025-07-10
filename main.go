package main

import (
	"log"
	"simple_web_tool/config"
	"simple_web_tool/controllers"
	"simple_web_tool/services"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化数据库连接
	if err := services.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", controllers.IndexHandler)
	r.GET("/api/users", controllers.GetUserStats)
	r.GET("/api/partitions", controllers.GetPartitionDetails)
	r.GET("/api/configs", controllers.ListDBConfigs)
	r.POST("/api/configure-db", controllers.ConfigureDB)
	r.POST("/api/update-config", controllers.UpdateDBConfig)

	r.Run(":8080")
}
