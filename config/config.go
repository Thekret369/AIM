// Package config loads and validates LanLine runtime configuration.
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const insecureDefaultJWTSecret = "lanline-secret-key-change-in-production"

// Server holds HTTP server configuration.
type Server struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Database holds database connection configuration.
type Database struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// JWT holds token signing configuration.
type JWT struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// Log holds application log configuration.
type Log struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// AI holds large model integration configuration.
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

// Config is the complete application configuration.
type Config struct {
	Server   Server   `mapstructure:"server"`
	Database Database `mapstructure:"database"`
	JWT      JWT      `mapstructure:"jwt"`
	Log      Log      `mapstructure:"log"`
	AI       AI       `mapstructure:"ai"`
}

// Load reads config.yaml, applies LANLINE_* environment overrides, and validates it.
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
	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}
	fillAIConfigFromEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	log.Printf("[config] 配置加载完成，端口: %d", cfg.Server.Port)
	return &cfg, nil
}

// Validate checks startup-critical configuration and normalizes simple fields.
func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("配置校验失败: 配置不能为空")
	}

	var problems []string
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		problems = append(problems, "server.port 必须在 1-65535 之间")
	}

	cfg.Database.Driver = strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	switch cfg.Database.Driver {
	case "sqlite", "mysql":
	default:
		problems = append(problems, "database.driver 仅支持 sqlite/mysql")
	}
	cfg.Database.DSN = strings.TrimSpace(cfg.Database.DSN)
	if cfg.Database.DSN == "" {
		problems = append(problems, "database.dsn 不能为空")
	}

	cfg.JWT.Secret = strings.TrimSpace(cfg.JWT.Secret)
	switch {
	case cfg.JWT.Secret == "":
		problems = append(problems, "jwt.secret 不能为空")
	case cfg.JWT.Secret == insecureDefaultJWTSecret:
		problems = append(problems, "jwt.secret 不能使用默认示例密钥，请通过 LANLINE_JWT_SECRET 覆盖")
	case len(cfg.JWT.Secret) < 16:
		problems = append(problems, "jwt.secret 长度不能少于 16 个字符")
	}
	if cfg.JWT.ExpireHours <= 0 {
		problems = append(problems, "jwt.expire_hours 必须大于 0")
	}

	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	switch cfg.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, "log.level 仅支持 debug/info/warn/error")
	}

	if cfg.AI.Enabled {
		validateAIConfig(cfg, &problems)
	}

	if len(problems) > 0 {
		return fmt.Errorf("配置校验失败: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateAIConfig(cfg *Config, problems *[]string) {
	cfg.AI.DefaultBotUsername = strings.TrimSpace(cfg.AI.DefaultBotUsername)
	if cfg.AI.DefaultBotUsername == "" {
		*problems = append(*problems, "ai.default_bot_username 不能为空")
	}
	if cfg.AI.TimeoutSeconds <= 0 {
		*problems = append(*problems, "ai.timeout_seconds 必须大于 0")
	}
	if cfg.AI.MaxContextMessages <= 0 {
		*problems = append(*problems, "ai.max_context_messages 必须大于 0")
	}
	if cfg.AI.Temperature < 0 || cfg.AI.Temperature > 2 {
		*problems = append(*problems, "ai.temperature 必须在 0-2 之间")
	}
	if cfg.AI.MaxTokens <= 0 {
		*problems = append(*problems, "ai.max_tokens 必须大于 0")
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "lanline.db")
	v.SetDefault("jwt.expire_hours", 72)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.file", "logs/lanline.log")
	v.SetDefault("ai.enabled", true)
	v.SetDefault("ai.default_bot_username", "ai_assistant")
	v.SetDefault("ai.default_bot_nickname", "蓝妹")
	v.SetDefault("ai.timeout_seconds", 60)
	v.SetDefault("ai.max_context_messages", 12)
	v.SetDefault("ai.temperature", 0.7)
	v.SetDefault("ai.max_tokens", 1024)
	v.SetDefault("ai.api_key_encrypt_key_env", "LANLINE_AI_API_KEY_ENCRYPT_KEY")
}

func applyEnvOverrides(cfg *Config) error {
	overrideStringFromEnv(&cfg.Server.Host, "LANLINE_SERVER_HOST")
	if err := overrideIntFromEnv(&cfg.Server.Port, "LANLINE_SERVER_PORT"); err != nil {
		return err
	}
	overrideStringFromEnv(&cfg.Database.Driver, "LANLINE_DATABASE_DRIVER")
	overrideStringFromEnv(&cfg.Database.DSN, "LANLINE_DATABASE_DSN")
	overrideStringFromEnv(&cfg.JWT.Secret, "LANLINE_JWT_SECRET")
	if err := overrideIntFromEnv(&cfg.JWT.ExpireHours, "LANLINE_JWT_EXPIRE_HOURS"); err != nil {
		return err
	}
	overrideStringFromEnv(&cfg.Log.Level, "LANLINE_LOG_LEVEL")
	overrideStringFromEnv(&cfg.Log.File, "LANLINE_LOG_FILE")

	if err := overrideBoolFromEnv(&cfg.AI.Enabled, "LANLINE_AI_ENABLED"); err != nil {
		return err
	}
	overrideStringFromEnv(&cfg.AI.BaseURL, "LANLINE_AI_BASE_URL")
	overrideStringFromEnv(&cfg.AI.APIKey, "LANLINE_AI_API_KEY")
	overrideStringFromEnv(&cfg.AI.APIKeyEnv, "LANLINE_AI_API_KEY_ENV")
	overrideStringFromEnv(&cfg.AI.APIKeyEncryptKey, "LANLINE_AI_API_KEY_ENCRYPT_KEY")
	overrideStringFromEnv(&cfg.AI.APIKeyEncryptKeyEnv, "LANLINE_AI_API_KEY_ENCRYPT_KEY_ENV")
	overrideStringFromEnv(&cfg.AI.DefaultModel, "LANLINE_AI_DEFAULT_MODEL")
	overrideStringFromEnv(&cfg.AI.DefaultBotUsername, "LANLINE_AI_DEFAULT_BOT_USERNAME")
	overrideStringFromEnv(&cfg.AI.DefaultBotNickname, "LANLINE_AI_DEFAULT_BOT_NICKNAME")
	overrideStringFromEnv(&cfg.AI.SystemPrompt, "LANLINE_AI_SYSTEM_PROMPT")
	if err := overrideIntFromEnv(&cfg.AI.TimeoutSeconds, "LANLINE_AI_TIMEOUT_SECONDS"); err != nil {
		return err
	}
	if err := overrideIntFromEnv(&cfg.AI.MaxContextMessages, "LANLINE_AI_MAX_CONTEXT_MESSAGES"); err != nil {
		return err
	}
	if err := overrideFloatFromEnv(&cfg.AI.Temperature, "LANLINE_AI_TEMPERATURE"); err != nil {
		return err
	}
	return overrideIntFromEnv(&cfg.AI.MaxTokens, "LANLINE_AI_MAX_TOKENS")
}

func overrideStringFromEnv(target *string, key string) {
	if raw, ok := os.LookupEnv(key); ok && strings.TrimSpace(raw) != "" {
		*target = strings.TrimSpace(raw)
	}
}

func overrideIntFromEnv(target *int, key string) error {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s 必须是整数: %w", key, err)
	}
	*target = value
	return nil
}

func overrideFloatFromEnv(target *float64, key string) error {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fmt.Errorf("%s 必须是数字: %w", key, err)
	}
	*target = value
	return nil
}

func overrideBoolFromEnv(target *bool, key string) error {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s 必须是布尔值: %w", key, err)
	}
	*target = value
	return nil
}

func fillAIConfigFromEnv(cfg *Config) {
	if cfg.AI.APIKey == "" {
		if cfg.AI.APIKeyEnv != "" {
			cfg.AI.APIKey = os.Getenv(cfg.AI.APIKeyEnv)
		}
		if cfg.AI.APIKey == "" {
			cfg.AI.APIKey = os.Getenv("LANLINE_AI_API_KEY")
		}
	}

	if cfg.AI.APIKeyEncryptKey == "" {
		if cfg.AI.APIKeyEncryptKeyEnv != "" {
			cfg.AI.APIKeyEncryptKey = os.Getenv(cfg.AI.APIKeyEncryptKeyEnv)
		}
		if cfg.AI.APIKeyEncryptKey == "" {
			cfg.AI.APIKeyEncryptKey = os.Getenv("LANLINE_AI_API_KEY_ENCRYPT_KEY")
		}
	}
}
