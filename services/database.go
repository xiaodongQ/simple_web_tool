package services

import (
	"fmt"
	"hash/crc32"
	"simple_web_tool/config"
	"simple_web_tool/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/driver/mysql"
)

var (
	dbConnections = make(map[string]*gorm.DB)
)

func UpdateDatabaseConfig(configName string, cfg config.DBConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	dbConnections[configName] = db
	return config.SaveConfig(config.GetCurrentConfig())
}

func GetDatabaseStats() map[string]interface{} {
	stats := make(map[string]interface{})
	for name, db := range dbConnections {
		sqlDB, _ := db.DB()
		stats[name] = gin.H{
			"max_open_conns": sqlDB.Stats().MaxOpenConnections,
			"in_use":        sqlDB.Stats().InUse,
		}
	}
	return stats
}

func GetUserStatistics() map[string]interface{} {
	var count int64
	dbConnections["default"].Model(&models.User{}).Count(&count)
	return gin.H{
		"total_users": count,
		"db_status":   GetDatabaseStats(),
	}
}

func CalculatePartition(bucketID string) string {
	var bucket models.Bucket
	dbConnections["default"].First(&bucket, "bid = ?", bucketID)
	return bucket.Partition
}

func QueryPartitionFiles(bid string, fname string) []models.BucketFile {
	query := dbConnections["default"].Table(CalculatePartition(bid))
	
	if bid != "" {
		query = query.Where("bucket_id = ?", bid)
	}
	if fname != "" {
		query = query.Where("file_name LIKE ?", "%"+fname+"%")
	}
	
	var files []models.BucketFile
	query.Find(&files)
	return files
}

func GetAllConfigs() map[string]config.DBConfig {
	return config.GetCurrentConfig().Databases
}