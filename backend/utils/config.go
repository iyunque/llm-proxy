package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port         int    `yaml:"port"`
		JwtSecret    string `yaml:"jwt_secret"`
		FrontendPath string `yaml:"frontend_path"`
	} `yaml:"server"`
	Database struct {
		Type   string `yaml:"type"`
		Sqlite struct {
			Path string `yaml:"path"`
		} `yaml:"sqlite"`
		Mysql struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Dbname   string `yaml:"dbname"`
		} `yaml:"mysql"`
	} `yaml:"database"`
	Stats struct {
		SyncInterval int `yaml:"sync_interval"`
	} `yaml:"stats"`
}

var GlobalConfig Config

func InitConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config file error: %v", err)
	}

	err = yaml.Unmarshal(data, &GlobalConfig)
	if err != nil {
		return fmt.Errorf("unmarshal config error: %v", err)
	}

	return nil
}
