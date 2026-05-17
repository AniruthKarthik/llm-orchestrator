package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	os.Setenv("MIGRATIONS_PATH", "/tmp/migrations")

	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("MIGRATIONS_PATH")
	}()

	cfg := Load()

	if cfg.Server.Port != ":9090" {
		t.Errorf("expected port :9090, got %s", cfg.Server.Port)
	}

	if cfg.Database.URL != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("expected db url, got %s", cfg.Database.URL)
	}

	if cfg.Database.MigrationsPath != "/tmp/migrations" {
		t.Errorf("expected migrations path /tmp/migrations, got %s", cfg.Database.MigrationsPath)
	}
}

func TestConfigDefaults(t *testing.T) {
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("MIGRATIONS_PATH")

	cfg := Load()

	if cfg.Server.Port != ":8080" {
		t.Errorf("expected default port :8080, got %s", cfg.Server.Port)
	}

	if cfg.Database.MigrationsPath != "migrations" {
		t.Errorf("expected default migrations path 'migrations', got %s", cfg.Database.MigrationsPath)
	}
}
