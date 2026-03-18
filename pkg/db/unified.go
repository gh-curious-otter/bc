package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
)

// BCDBPath returns the path to the unified bc database for a workspace.
func BCDBPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".bc", "bc.db")
}

// shared holds the workspace-wide database connection.
// Set via SetShared() at app startup, used by all stores.
var (
	sharedDB     *sql.DB
	sharedDriver string // "sqlite" or "postgres"
	sharedMu     sync.RWMutex
)

// SetShared sets the shared workspace database connection.
// Call once at startup (in cmd/bcd or CLI init).
func SetShared(db *sql.DB, driver string) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	sharedDB = db
	sharedDriver = driver
}

// Shared returns the shared workspace database connection.
// Returns nil if not set (stores should fall back to opening their own).
func Shared() *sql.DB {
	sharedMu.RLock()
	defer sharedMu.RUnlock()
	return sharedDB
}

// SharedDriver returns "sqlite" or "postgres".
func SharedDriver() string {
	sharedMu.RLock()
	defer sharedMu.RUnlock()
	return sharedDriver
}

// OpenWorkspaceDB opens the workspace database based on configuration.
// If DATABASE_URL is set, connects to Postgres.
// Otherwise, opens SQLite at .bc/bc.db.
func OpenWorkspaceDB(workspaceRoot string) (*sql.DB, string, error) {
	if IsPostgresEnabled() {
		db, err := OpenPostgres(PostgresDSN())
		if err != nil {
			return nil, "", fmt.Errorf("open postgres: %w", err)
		}
		return db, "postgres", nil
	}

	path := BCDBPath(workspaceRoot)
	d, err := Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open sqlite %s: %w", path, err)
	}
	return d.DB, "sqlite", nil
}

// CloseShared closes the shared connection.
func CloseShared() error {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedDB != nil {
		err := sharedDB.Close()
		sharedDB = nil
		return err
	}
	return nil
}
