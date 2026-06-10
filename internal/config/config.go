// Package config loads application configuration from environment variables.
package config

import (
	"os"
)

// Config holds all application configuration.
type Config struct {
	DataDir string // Root directory for persistent data (SQLite db lives here)
	Port    string // HTTP port the server listens on
}

// Load reads configuration from environment variables and creates required directories.
// Called once at startup from main.go.
func Load() (*Config, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	port := os.Getenv("TASKDOCK_PORT")
	if port == "" {
		port = "8860"
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &Config{
		DataDir: dataDir,
		Port:    port,
	}, nil
}
