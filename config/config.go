// Package config 使用 Viper 加载配置
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Server 服务端配置
type Server struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Database 数据库配置
type Database struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// JWT JWT 鉴权配置
type JWT struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// Log 日志配置
type Log struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// AI 大模型接入配置。
type AI struct {
	Enabled             bool    `mapstructure:"enabled"`
	BaseURL             string  `mapstructure:"base_url"`
	APIKey              string  `mapstructure:"api_key"`
	APIKeyEnv           string  `mapstructure:"api_key_env"`
	APIKeyEncryptKey    string  `mapstructure:"api_key_encrypt_key"`
	APIKeyEncryptKeyEnv string  `mapstructure:"api_key_encrypt_key_env"`
	DefaultModel        string  `mapstructure:"default_model"`
	DefaultBotUsername  string  `mapstructure:"default_bot_username"`
	DefaultBotNickname  string  `mapstructure:"default_bot_nickname"`
	SystemPrompt        string  `mapstructure:"system_prompt"`
	TimeoutSeconds      int     `mapstructure:"timeout_seconds"`
	MaxContextMessages  int     `mapstructure:"max_context_messages"`
	Temperature         float64 `mapstructure:"temperature"`
	MaxTokens           int     `mapstructure:"max_tokens"`
}

// Config 总配置
type Config struct {
	Server   Server   `mapstructure:"server"`
	Database Database `mapstructure:"database"`
	JWT      JWT      `mapstructure:"jwt"`
	Log      Log      `mapstructure:"log"`
	AI       AI       `mapstructure:"ai"`
}

// Load 加载配置
func Load(configDir string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	fillAIConfigFromEnv(&cfg)

	log.Printf("[config] 配置加载完成，端口: %d", cfg.Server.Port)
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("ai.enabled", true)
	v.SetDefault("ai.default_bot_username", "ai_assistant")
	v.SetDefault("ai.default_bot_nickname", "蓝妹")
	v.SetDefault("ai.timeout_seconds", 60)
	v.SetDefault("ai.max_context_messages", 12)
	v.SetDefault("ai.temperature", 0.7)
	v.SetDefault("ai.max_tokens", 1024)
	v.SetDefault("ai.api_key_encrypt_key_env", "AIM_AI_API_KEY_ENCRYPT_KEY")
}

func fillAIConfigFromEnv(cfg *Config) {
	if cfg.AI.APIKey == "" {
		if cfg.AI.APIKeyEnv != "" {
			cfg.AI.APIKey = os.Getenv(cfg.AI.APIKeyEnv)
		}
		if cfg.AI.APIKey == "" {
			cfg.AI.APIKey = os.Getenv("AIM_AI_API_KEY")
		}
	}

	if cfg.AI.APIKeyEncryptKey == "" {
		if cfg.AI.APIKeyEncryptKeyEnv != "" {
			cfg.AI.APIKeyEncryptKey = os.Getenv(cfg.AI.APIKeyEncryptKeyEnv)
		}
		if cfg.AI.APIKeyEncryptKey == "" {
			cfg.AI.APIKeyEncryptKey = os.Getenv("AIM_AI_API_KEY_ENCRYPT_KEY")
		}
	}
}
