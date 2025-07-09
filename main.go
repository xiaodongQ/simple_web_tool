package main

import (
	"simple_web_tool/controllers"
	"simple_web_tool/config"
	
	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig()

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