// Package config loads application configuration from environment variables.
package config

import (
	"os"
	"path/filepath"
)

// Config holds all application configuration.
type Config struct {
	DataDir      string // Root directory for persistent data (SQLite db lives here)
	FilesDir     string // Subdirectory for uploaded attachments (pasted images)
	Port         string // HTTP port the server listens on
	BranchPrefix string // Prepended to generated git branch names (e.g. "feature/")
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

	filesDir := filepath.Join(dataDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return nil, err
	}

	return &Config{
		DataDir:      dataDir,
		FilesDir:     filesDir,
		Port:         port,
		BranchPrefix: os.Getenv("TASKDOCK_BRANCH_PREFIX"),
	}, nil
}
