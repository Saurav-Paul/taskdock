// Package database sets up the GORM connection and runs Goose migrations.
package database

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Saurav-Paul/taskdock/internal/config"
)

// Migration SQL files are baked into the binary at compile time,
// so the scratch Docker image needs nothing but the binary itself.
//
//go:embed migrations
var migrations embed.FS

// Setup opens the SQLite database via GORM and runs any pending Goose migrations.
// Returns the *gorm.DB instance used throughout the app.
func Setup(cfg *config.Config) (*gorm.DB, error) {
	dbPath := filepath.Join(cfg.DataDir, "taskdock.db")

	db, err := gorm.Open(sqlite.Open(dbPath+"?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if err := runMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// runMigrations applies pending Goose migrations from the embedded filesystem.
// Runs automatically on every startup — like Alembic's "upgrade head".
func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	return goose.Up(db, "migrations")
}
