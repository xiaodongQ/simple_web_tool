package services

import (
	"fmt"
	"log"
	"simple_web_tool/config"
	"simple_web_tool/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
			"in_use":         sqlDB.Stats().InUse,
		}
	}
	return stats
}

func GetDBConnection(name string) *gorm.DB {
	if db, ok := dbConnections[name]; ok && db != nil {
		return db
	}
	fmt.Printf("数据库连接 %s 未初始化或已关闭", name)
	return nil
}

type UserStats struct {
	TotalUsers int64  `json:"total_users"`
	Error      string `json:"error,omitempty"`
}

func GetUserStatistics() *UserStats {
	if db := GetDBConnection("default"); db != nil {
		var count int64
		if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
			fmt.Printf("用户统计查询失败: %v\n", err)
		}
		return &UserStats{TotalUsers: count}
	}
	return &UserStats{Error: "数据库连接不可用"}
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

func InitDatabase() error {
	cfg := config.GetCurrentConfig().Databases["default"]
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		// 打印配置信息
		log.Printf("数据库配置: %+v", cfg.GetDSN())
		return fmt.Errorf("%v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	dbConnections["default"] = db
	return nil
}
