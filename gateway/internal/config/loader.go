package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 是整個 YAML 檔案的最外層結構
type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Redis    RedisConfig     `yaml:"redis"`
	Services []ServiceConfig `yaml:"services"`
}

// ServerConfig 儲存 Gateway 自身的伺服器設定
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// RedisConfig 儲存 Redis 連線與策略設定
type RedisConfig struct {
	Addr        string `yaml:"addr"`
	Password    string `yaml:"password"`
	FailureMode string `yaml:"failure_mode"` // "open" 或 "closed"
}

// ServiceConfig 儲存每個 AI 服務的轉發與限流規則
type ServiceConfig struct {
	Name      string           `yaml:"name"`
	Path      string           `yaml:"path"`
	TargetURL string           `yaml:"target_url"`
	RateLimit RateLimitDetails `yaml:"rate_limit"`
}

// RateLimitDetails 儲存具體的限流數值
type RateLimitDetails struct {
	GlobalRate     float64 `yaml:"global_rate"`
	GlobalCapacity float64 `yaml:"global_capacity"`
	IPRate         float64 `yaml:"ip_rate"`
	IPCapacity     float64 `yaml:"ip_capacity"`
	UserRate       float64 `yaml:"user_rate"`
	UserCapacity   float64 `yaml:"user_capacity"`
}

// LoadConfig 負責讀取 YAML 檔案並解析
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	// 1. 讀取檔案
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// 2. 解析 YAML (Unmarshal)
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}
