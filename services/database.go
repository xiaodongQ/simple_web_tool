package services

import (
	"fmt"
	"sync"
	"simple_web_tool/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DBConnections 存储多个数据库连接
var DBConnections = make(map[string]*gorm.DB)
var dbMutex sync.RWMutex

// InitDB 初始化指定配置的数据库连接
func InitDB(configName string) error {
	conf, exists := config.GetDBConfig(configName)
	if !exists {
		return fmt.Errorf("database config '%s' not found", configName)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	dbMutex.Lock()
	DBConnections[configName] = db
	dbMutex.Unlock()

	return nil
}

// GetDB 获取指定配置的数据库连接
func GetDB(configName string) (*gorm.DB, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	db, exists := DBConnections[configName]
	if !exists {
		return nil, fmt.Errorf("database connection '%s' not initialized", configName)
	}

	return db, nil
}

// GetDefaultDB 获取默认数据库连接
func GetDefaultDB() (*gorm.DB, error) {
	return GetDB("default")
}

// InitDefaultDB 初始化默认数据库连接
func InitDefaultDB() error {
	return InitDB("default")
}