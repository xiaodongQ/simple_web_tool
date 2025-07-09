package controllers

import (
	"simple_web_tool/config"
	"simple_web_tool/services"
	
	"github.com/gin-gonic/gin"
)

func IndexHandler(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{
		"Title": "文件存储管理系统",
	})
}

func ConfigureDB(c *gin.Context) {
	var initConfig config.DBConfig
	if err := c.ShouldBindJSON(&initConfig); err != nil {
		c.JSON(400, gin.H{"error": "无效的配置格式"})
		return
	}
	
	config.UpdateDBConfig("default", initConfig)
	c.JSON(200, gin.H{"status": "数据库配置已初始化"})
}

func UpdateDBConfig(c *gin.Context) {
	configName := c.PostForm("config_name")
	var newConfig config.DBConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := services.UpdateDatabaseConfig(configName, newConfig); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}
	c.JSON(200, gin.H{"status": "updated"})
}

func GetUserStats(c *gin.Context) {
	stats := services.GetUserStatistics()
	c.JSON(200, stats)
}

func GetPartitionDetails(c *gin.Context) {
	bid := c.Query("bid")
	fname := c.Query("fname")
	details := services.QueryPartitionFiles(bid, fname)
	c.JSON(200, details)
}

func ListDBConfigs(c *gin.Context) {
	current := config.GetCurrentConfig()
	c.JSON(200, current.Databases)
}