package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

type AppConfig struct {
	Configs        []Config `json:"configs"`
	DefaultDBIndex int      `json:"default_db_index"`
}

func loadConfig() error {
	// 先尝试读取配置文件
	file, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在时，使用并保存默认配置
			appConfig = AppConfig{
				Configs: []Config{{
					Host:     "192.168.1.150",
					Port:     "3306",
					User:     "test",
					Password: "test",
					DBName:   "testdb",
				}},
				DefaultDBIndex: 0,
			}
			if err = saveConfig(); err != nil {
				log.Printf("Error saving default config: %v", err)
				return fmt.Errorf("failed to save default config: %w", err)
			}
			log.Println("Created default config file")
			return nil
		}
		// 其他读取错误
		log.Printf("Error reading config file: %v", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 文件存在，解析配置
	if err := json.Unmarshal(file, &appConfig); err != nil {
		log.Printf("Error parsing config file: %v", err)
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 确保DefaultDBIndex在有效范围内
	if appConfig.DefaultDBIndex < 0 || appConfig.DefaultDBIndex >= len(appConfig.Configs) {
		if len(appConfig.Configs) > 0 {
			appConfig.DefaultDBIndex = 0
		} else {
			appConfig.DefaultDBIndex = -1 // No configs available
		}
	}

	return nil
}

func saveConfig() error {
	data, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0644)
}

func connectDB(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.User, config.Password, config.Host, config.Port, config.DBName)
	log.Printf("dsn: %s\n", dsn)
	return sql.Open("mysql", dsn)
}

// testDBConnection 尝试连接到给定的数据库配置，并返回错误（如果连接失败）
func testDBConnection(config Config) error {
	db, err := connectDB(config)
	if err != nil {
		return fmt.Errorf("无法连接到数据库: %w", err)
	}
	defer db.Close()

	// 尝试ping数据库以验证连接
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("无法ping数据库: %w", err)
	}
	return nil
}
