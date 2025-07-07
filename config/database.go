package config

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

var DefaultConfig = DBConfig{
	Host:     "localhost",
	Port:     "3306",
	User:     "root",
	Password: "",
	DBName:   "test",
}