package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of the YAML file.
type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Redis    RedisConfig     `yaml:"redis"`
	Services []ServiceConfig `yaml:"services"`
}

// ServerConfig stores the gateway server settings.
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// RedisConfig stores Redis connection and strategy settings.
type RedisConfig struct {
	Addr        string `yaml:"addr"`
	Password    string `yaml:"password"`
	FailureMode string `yaml:"failure_mode"` // "open" or "closed"
}

// ServiceConfig stores routing and rate-limit rules for each AI service.
type ServiceConfig struct {
	Name      string           `yaml:"name"`
	Path      string           `yaml:"path"`
	TargetURL string           `yaml:"target_url"`
	RateLimit RateLimitDetails `yaml:"rate_limit"`
}

// RateLimitDetails stores concrete rate-limit values.
type RateLimitDetails struct {
	GlobalRate     float64 `yaml:"global_rate"`
	GlobalCapacity float64 `yaml:"global_capacity"`
	IPRate         float64 `yaml:"ip_rate"`
	IPCapacity     float64 `yaml:"ip_capacity"`
	UserRate       float64 `yaml:"user_rate"`
	UserCapacity   float64 `yaml:"user_capacity"`
}

// LoadConfig reads and parses the YAML file.
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	// 1. Read file
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// 2. Parse YAML (Unmarshal)
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}
