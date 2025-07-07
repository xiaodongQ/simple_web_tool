package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"simple_web_tool/config"
	"simple_web_tool/services"
	"simple_web_tool/models"
)

// ConfigureDB 处理数据库配置
func ConfigureDB(c *gin.Context) {
	var conf config.DBConfig
	if err := c.ShouldBindJSON(&conf); err != nil {
		c.JSON(400, gin.H{"error": "无效的配置参数"})
		return
	}

	if err := services.InitDB(conf); err != nil {
		c.JSON(500, gin.H{"error": "数据库连接失败"})
		return
	}

	c.JSON(200, gin.H{"message": "数据库连接成功"})
}

// QueryData 处理数据查询
func QueryData(c *gin.Context) {
	db := services.GetDB()
	if db == nil {
		c.JSON(500, gin.H{"error": "数据库未连接"})
		return
	}

	bid := c.Query("bid")
	bname := c.Query("bname")

	var results []models.TableA
	query := db
	if bid != "" {
		query = query.Where("bid = ?", bid)
	}
	if bname != "" {
		query = query.Where("bname = ?", bname)
	}

	if err := query.Find(&results).Error; err != nil {
		c.JSON(500, gin.H{"error": "查询失败"})
		return
	}

	// 获取详细信息
	var details []models.TableADetail
	for _, result := range results {
		var tableDetails []models.TableADetail
		tableName := fmt.Sprintf("A_%s", result.Partition)
		if err := db.Table(tableName).Where("bid = ?", result.Bid).Find(&tableDetails).Error; err != nil {
			continue
		}
		details = append(details, tableDetails...)
	}

	c.JSON(200, gin.H{
		"main_data": results,
		"details": details,
	})
}