package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"simple_web_tool/services"
	"simple_web_tool/models"
)

// ConfigureDB 处理数据库配置
func ConfigureDB(c *gin.Context) {
	type ConfigRequest struct {
		ConfigName string `json:"config_name" binding:"required"`
	}

	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的配置参数"})
		return
	}

	if err := services.InitDB(req.ConfigName); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("数据库连接失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("数据库 %s 连接成功", req.ConfigName)})
}

// UpdateDBConfig 更新数据库配置
func UpdateDBConfig(c *gin.Context) {
	type UpdateConfigRequest struct {
		ConfigName string `json:"config_name" binding:"required"`
		Config    config.DBConfig `json:"config" binding:"required"`
	}

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的配置参数"})
		return
	}

	if err := config.UpdateDBConfig(req.ConfigName, req.Config); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("更新配置失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("数据库配置 %s 更新成功", req.ConfigName)})
}

// QueryData 处理数据查询
func QueryData(c *gin.Context) {
	configName := c.DefaultQuery("db", "default")
	db, err := services.GetDB(configName)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("数据库连接错误: %v", err)})
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