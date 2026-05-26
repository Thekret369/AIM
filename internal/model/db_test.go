package model

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabaseDialectorDefaultsToSQLite(t *testing.T) {
	_, driver, err := databaseDialector("", filepath.Join(t.TempDir(), "lanline.db"))
	if err != nil {
		t.Fatalf("databaseDialector failed: %v", err)
	}
	if driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %q", driver)
	}
}

func TestDatabaseDialectorAcceptsMySQL(t *testing.T) {
	_, driver, err := databaseDialector("mysql", "lanline:secret@tcp(localhost:3306)/lanline?parseTime=true&charset=utf8mb4")
	if err != nil {
		t.Fatalf("databaseDialector failed: %v", err)
	}
	if driver != "mysql" {
		t.Fatalf("expected mysql driver, got %q", driver)
	}
}

func TestDatabaseDialectorRejectsUnsupportedDriver(t *testing.T) {
	_, _, err := databaseDialector("postgres", "postgres://localhost/lanline")
	if err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("expected unsupported driver error, got %v", err)
	}
}

func TestInitDBSQLiteCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lanline.db")
	if err := InitDB("sqlite", dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	if DB == nil {
		t.Fatal("expected global DB to be initialized")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		DB = nil
	})
}

func TestSchemaMigrationsUseMySQLSpecificStatements(t *testing.T) {
	for _, migration := range schemaMigrations() {
		statements := strings.Join(migration.statements("mysql"), "\n")
		if strings.Contains(statements, "IF NOT EXISTS") {
			t.Fatalf("migration %s uses sqlite-style IF NOT EXISTS in mysql statements", migration.Version)
		}
	}
}
