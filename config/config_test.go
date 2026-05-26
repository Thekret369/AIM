package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEnvOverridesAndValidates(t *testing.T) {
	clearLanLineEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, `
server:
  host: "127.0.0.1"
  port: 8080
database:
  driver: "sqlite"
  dsn: "lanline.db"
jwt:
  secret: "unit-test-secret-123"
  expire_hours: 72
log:
  level: "debug"
  file: "logs/lanline.log"
ai:
  enabled: true
  default_bot_username: "ai_assistant"
  timeout_seconds: 60
  max_context_messages: 12
  temperature: 0.7
  max_tokens: 1024
`)

	t.Setenv("LANLINE_SERVER_PORT", "9090")
	t.Setenv("LANLINE_DATABASE_DSN", "data/lanline.db")
	t.Setenv("LANLINE_AI_ENABLED", "false")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected env port override, got %d", cfg.Server.Port)
	}
	if cfg.Database.DSN != "data/lanline.db" {
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

func TestValidateAcceptsMySQLDatabaseDriver(t *testing.T) {
	cfg := validConfig()
	cfg.Database.Driver = "mysql"

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected mysql database driver to pass, got %v", err)
	}
}

func TestValidateRejectsUnsupportedDatabaseDriver(t *testing.T) {
	cfg := validConfig()
	cfg.Database.Driver = "postgres"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "database.driver") {
		t.Fatalf("expected database driver validation error, got %v", err)
	}
}

func TestLoadRejectsInvalidEnvType(t *testing.T) {
	clearLanLineEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, `
jwt:
  secret: "unit-test-secret-123"
`)
	t.Setenv("LANLINE_SERVER_PORT", "invalid")

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "LANLINE_SERVER_PORT") {
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
			DSN:    "lanline.db",
		},
		JWT: JWT{
			Secret:      "unit-test-secret-123",
			ExpireHours: 72,
		},
		Log: Log{
			Level: "info",
			File:  "logs/lanline.log",
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

func clearLanLineEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"LANLINE_SERVER_HOST",
		"LANLINE_SERVER_PORT",
		"LANLINE_DATABASE_DRIVER",
		"LANLINE_DATABASE_DSN",
		"LANLINE_JWT_SECRET",
		"LANLINE_JWT_EXPIRE_HOURS",
		"LANLINE_LOG_LEVEL",
		"LANLINE_LOG_FILE",
		"LANLINE_AI_ENABLED",
		"LANLINE_AI_BASE_URL",
		"LANLINE_AI_API_KEY",
		"LANLINE_AI_API_KEY_ENV",
		"LANLINE_AI_API_KEY_ENCRYPT_KEY",
		"LANLINE_AI_API_KEY_ENCRYPT_KEY_ENV",
		"LANLINE_AI_DEFAULT_MODEL",
		"LANLINE_AI_DEFAULT_BOT_USERNAME",
		"LANLINE_AI_DEFAULT_BOT_NICKNAME",
		"LANLINE_AI_SYSTEM_PROMPT",
		"LANLINE_AI_TIMEOUT_SECONDS",
		"LANLINE_AI_MAX_CONTEXT_MESSAGES",
		"LANLINE_AI_TEMPERATURE",
		"LANLINE_AI_MAX_TOKENS",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
