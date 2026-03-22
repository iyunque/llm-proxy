package models

import (
	"ai-api-platform/backend/utils"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
}

type AIProvider struct {
	gorm.Model
	Name       string `gorm:"not null"`
	APIAddress string `gorm:"not null"`
	APIKey     string `gorm:"not null"`
	ModelName  string `gorm:"not null"` // 模型名称，如 gpt-4, deepseek-chat 等
}

type APIEndpoint struct {
	gorm.Model
	Path                string `gorm:"uniqueIndex;not null"` // 如 /api/translate
	SystemPrompt        string `gorm:"type:text"`
	ApiKey              string `gorm:"size:32;not null"` // 客户端调用此接口的Key
	ProviderID          uint
	Provider            AIProvider `gorm:"foreignKey:ProviderID"`
	SelectedModel       string     // 选择的大模型名称
	StreamOutput        bool       `gorm:"default:false"` // 是否启用流式输出
	EnableThinking      bool       `gorm:"default:false"` // 是否启用思考模式
	Temperature         float64    `gorm:"default:0.7"`   // 温度参数，控制随机性
	FallbackProviderID1 uint       // 第一个备用供应商ID
	FallbackModel1      string     // 第一个备用模型名称
	FallbackProviderID2 uint       // 第二个备用供应商ID
	FallbackModel2      string     // 第二个备用模型名称
}

type APIStats struct {
	ID              uint   `gorm:"primaryKey"`
	APIEndpointID   uint   `gorm:"index:idx_endpoint_date"`
	Date            string `gorm:"index:idx_endpoint_date"` // YYYY-MM-DD
	CallCount       int64
	InputTokens     int64
	OutputTokens    int64
	CacheHitTokens  int64
	FailedCallCount int64  // 失败调用次数
	FailedModels    string `gorm:"type:text"` // JSON格式的失败模型统计 {"model_name": count, ...}
	LastFailedModel string // 最后失败的模型名称
	LastUpdated     time.Time
}

func InitDB() error {
	var err error
	config := utils.GlobalConfig.Database

	if config.Type == "sqlite" {
		DB, err = gorm.Open(sqlite.Open(config.Sqlite.Path), &gorm.Config{})
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.Mysql.User, config.Mysql.Password, config.Mysql.Host, config.Mysql.Port, config.Mysql.Dbname)
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 自动迁移
	err = DB.AutoMigrate(&User{}, &AIProvider{}, &APIEndpoint{}, &APIStats{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	return nil
}
