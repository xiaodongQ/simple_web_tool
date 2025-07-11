package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func loadConfig() error {
	// 先尝试读取配置文件
	file, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在时，使用并保存默认配置
			configs = []Config{{
				Host:     "192.168.1.150",
				Port:     "3306",
				User:     "test",
				Password: "test",
				DBName:   "testdb",
			}}
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
	if err := json.Unmarshal(file, &configs); err != nil {
		log.Printf("Error parsing config file: %v", err)
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func saveConfig() error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0644)
}

func connectDB(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.User, config.Password, config.Host, config.Port, config.DBName)
	fmt.Printf("dsn: %s\n", dsn)
	return sql.Open("mysql", dsn)
}
