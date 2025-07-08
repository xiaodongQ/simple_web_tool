package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"simple_web_tool/controllers"
	"simple_web_tool/services"
)

func main() {
	// 初始化默认数据库连接
	if err := services.InitDefaultDB(); err != nil {
		panic(fmt.Sprintf("初始化默认数据库失败: %v", err))
	}

	r := gin.Default()

	// 加载HTML模板
	r.LoadHTMLGlob("templates/*")
	// 提供静态文件服务
	r.Static("/static", "./static")

	// API路由
	r.POST("/api/configure-db", controllers.ConfigureDB)
	r.POST("/api/update-config", controllers.UpdateDBConfig)
	r.GET("/api/query", controllers.QueryData)

	// 页面路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "数据库查询工具",
		})
	})

	// 启动服务器
	r.Run(":8080")
}