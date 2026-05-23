package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEnvOverridesAndValidates(t *testing.T) {
	clearAIMEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, `
server:
  host: "127.0.0.1"
  port: 8080
database:
  driver: "sqlite"
  dsn: "aim.db"
jwt:
  secret: "unit-test-secret-123"
  expire_hours: 72
log:
  level: "debug"
  file: "logs/aim.log"
ai:
  enabled: true
  default_bot_username: "ai_assistant"
  timeout_seconds: 60
  max_context_messages: 12
  temperature: 0.7
  max_tokens: 1024
`)

	t.Setenv("AIM_SERVER_PORT", "9090")
	t.Setenv("AIM_DATABASE_DSN", "data/aim.db")
	t.Setenv("AIM_AI_ENABLED", "false")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected env port override, got %d", cfg.Server.Port)
	}
	if cfg.Database.DSN != "data/aim.db" {
		t.Fatalf("expected env dsn override, got %q", cfg.Database.DSN)
	}
	if cfg.AI.Enabled {
		t.Fatal("expected env AI enabled override")
	}
}

func TestValidateRejectsDangerousJWTSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWT.Secret = insecureDefaultJWTSecret

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "默认示例密钥") {
		t.Fatalf("expected dangerous secret error, got %v", err)
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 70000

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "server.port") {
		t.Fatalf("expected port validation error, got %v", err)
	}
}

func TestValidateRejectsUnsupportedDatabaseDriver(t *testing.T) {
	cfg := validConfig()
	cfg.Database.Driver = "mysql"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "database.driver") {
		t.Fatalf("expected database driver validation error, got %v", err)
	}
}

func TestLoadRejectsInvalidEnvType(t *testing.T) {
	clearAIMEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, `
jwt:
  secret: "unit-test-secret-123"
`)
	t.Setenv("AIM_SERVER_PORT", "invalid")

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "AIM_SERVER_PORT") {
		t.Fatalf("expected env parse error, got %v", err)
	}
}

func validConfig() Config {
	return Config{
		Server: Server{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: Database{
			Driver: "sqlite",
			DSN:    "aim.db",
		},
		JWT: JWT{
			Secret:      "unit-test-secret-123",
			ExpireHours: 72,
		},
		Log: Log{
			Level: "info",
			File:  "logs/aim.log",
		},
		AI: AI{
			Enabled:            true,
			DefaultBotUsername: "ai_assistant",
			TimeoutSeconds:     60,
			MaxContextMessages: 12,
			Temperature:        0.7,
			MaxTokens:          1024,
		},
	}
}

func writeConfig(t *testing.T, dir string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func clearAIMEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"AIM_SERVER_HOST",
		"AIM_SERVER_PORT",
		"AIM_DATABASE_DRIVER",
		"AIM_DATABASE_DSN",
		"AIM_JWT_SECRET",
		"AIM_JWT_EXPIRE_HOURS",
		"AIM_LOG_LEVEL",
		"AIM_LOG_FILE",
		"AIM_AI_ENABLED",
		"AIM_AI_BASE_URL",
		"AIM_AI_API_KEY",
		"AIM_AI_API_KEY_ENV",
		"AIM_AI_API_KEY_ENCRYPT_KEY",
		"AIM_AI_API_KEY_ENCRYPT_KEY_ENV",
		"AIM_AI_DEFAULT_MODEL",
		"AIM_AI_DEFAULT_BOT_USERNAME",
		"AIM_AI_DEFAULT_BOT_NICKNAME",
		"AIM_AI_SYSTEM_PROMPT",
		"AIM_AI_TIMEOUT_SECONDS",
		"AIM_AI_MAX_CONTEXT_MESSAGES",
		"AIM_AI_TEMPERATURE",
		"AIM_AI_MAX_TOKENS",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
