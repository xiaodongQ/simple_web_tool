package main

import (
	"github.com/gin-gonic/gin"
	"simple_web_tool/controllers"
)

func main() {
	r := gin.Default()

	// 加载HTML模板
	r.LoadHTMLGlob("templates/*")
	// 提供静态文件服务
	r.Static("/static", "./static")

	// API路由
	r.POST("/api/configure-db", controllers.ConfigureDB)
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