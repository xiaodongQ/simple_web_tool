package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

func (c *DBConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName)
}

type Config struct {
	Databases map[string]DBConfig `yaml:"databases"`
}

var currentConfig = &Config{}

func LoadConfig() error {
	if _, err := os.Stat("config/database.yaml"); os.IsNotExist(err) {
		defaultConfig := &Config{
			Databases: map[string]DBConfig{
				"default": {
					Host:     "localhost",
					Port:     3306,
					User:     "root",
					Password: "",
					DBName:   "test_db",
				},
			},
		}
		if err := SaveConfig(defaultConfig); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(filepath.Join("config", "database.yaml"))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, currentConfig)
}

func SaveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile("config/database.yaml", data, 0644)
}

func GetCurrentConfig() *Config {
	return currentConfig
}

func UpdateDBConfig(name string, cfg DBConfig) {
	currentConfig.Databases[name] = cfg
}
