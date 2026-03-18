package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver
)

// Driver constants.
const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

// DatabaseConfig matches the generated config.DatabaseConfig fields.
// Defined here to avoid a circular import between pkg/db and config/.
type DatabaseConfig struct {
	Driver       string
	URL          string
	MaxOpenConns int
	MaxIdleConns int
}

// OpenFromConfig opens a database using the provided configuration.
// For sqlite, dbName is appended to the workspace .bc/ directory (e.g. "channels.db").
// For postgres, the URL from config is used directly and dbName is ignored.
func OpenFromConfig(cfg DatabaseConfig, workspacePath, dbName string) (*DB, error) {
	switch cfg.Driver {
	case DriverPostgres:
		return openPostgres(cfg)
	case DriverSQLite, "":
		return openSQLiteFromConfig(cfg, workspacePath, dbName)
	default:
		return nil, fmt.Errorf("unsupported database driver %q (use %q or %q)", cfg.Driver, DriverSQLite, DriverPostgres)
	}
}

// openSQLiteFromConfig opens a SQLite database, using the URL as path if set,
// otherwise falling back to .bc/<dbName> in the workspace.
func openSQLiteFromConfig(cfg DatabaseConfig, workspacePath, dbName string) (*DB, error) {
	path := cfg.URL
	if path == "" {
		path = filepath.Join(workspacePath, ".bc", dbName)
	}
	return Open(path)
}

// openPostgres opens a Postgres connection using pgx via database/sql.
func openPostgres(cfg DatabaseConfig) (*DB, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database.url is required for postgres driver")
	}

	sqlDB, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Verify connectivity
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &DB{
		DB:   sqlDB,
		path: cfg.URL,
	}, nil
}

// DetectDriver returns the configured driver, defaulting to sqlite.
// Reads from BC_DATABASE_DRIVER env var first, then falls back to the provided driver string.
func DetectDriver(configDriver string) string {
	if d := os.Getenv("BC_DATABASE_DRIVER"); d != "" {
		return d
	}
	if configDriver == "" {
		return DriverSQLite
	}
	return configDriver
}

// DetectURL returns the configured URL.
// Reads from BC_DATABASE_URL env var first, then falls back to the provided URL string.
func DetectURL(configURL string) string {
	if u := os.Getenv("BC_DATABASE_URL"); u != "" {
		return u
	}
	return configURL
}
