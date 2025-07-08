package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type Config struct {
	Databases map[string]DBConfig `yaml:"databases"`
}

var config *Config

// LoadConfig 加载数据库配置文件
func LoadConfig() error {
	configPath := filepath.Join("config", "database.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	config = &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	return nil
}

// GetDBConfig 根据配置名获取数据库配置
func GetDBConfig(configName string) (DBConfig, bool) {
	if config == nil {
		if err := LoadConfig(); err != nil {
			return DBConfig{}, false
		}
	}

	dbConfig, exists := config.Databases[configName]
	return dbConfig, exists
}

// DefaultConfig 获取默认配置
func DefaultConfig() (DBConfig, error) {
	config, exists := GetDBConfig("default")
	if !exists {
		return DBConfig{}, fmt.Errorf("默认配置不存在")
	}
	return config, nil
}

// SaveConfig 保存数据库配置到文件
func SaveConfig() error {
	if config == nil {
		return fmt.Errorf("配置未初始化")
	}

	configPath := filepath.Join("config", "database.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// UpdateDBConfig 更新数据库配置
func UpdateDBConfig(name string, dbConfig DBConfig) error {
	if config == nil {
		if err := LoadConfig(); err != nil {
			return err
		}
	}

	config.Databases[name] = dbConfig
	return SaveConfig()
}