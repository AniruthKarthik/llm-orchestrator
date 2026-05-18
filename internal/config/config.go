package config

import (
	"os"
)

// ServerConfig holds server-related settings.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds settings for the PostgreSQL database.
type DatabaseConfig struct {
	URL            string
	MigrationsPath string
}

// Config is the top-level configuration structure.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// Load populates the Config struct from environment variables with sensible defaults.
func Load() *Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}

	dbURL := os.Getenv("DATABASE_URL")
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	return &Config{
		Server: ServerConfig{
			Port: port,
		},
		Database: DatabaseConfig{
			URL:            dbURL,
			MigrationsPath: migrationsPath,
		},
	}
}
