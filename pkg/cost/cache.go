package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gh-curious-otter/bc/pkg/db"
)

// Cache provides SQLite-backed caching for expensive cost queries (e.g. ccusage).
type Cache struct {
	db *db.DB
}

// NewCache opens (or creates) the cost cache at dbPath.
func NewCache(dbPath string) (*Cache, error) {
	d, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cost cache db: %w", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS cost_cache (
			key        TEXT PRIMARY KEY,
			data       TEXT NOT NULL,
			fetched_at TEXT NOT NULL
		);
	`
	if _, err := d.ExecContext(context.Background(), schema); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("create cost_cache table: %w", err)
	}

	return &Cache{db: d}, nil
}

// Load retrieves cached data by key. Returns nil data if not found.
func (c *Cache) Load(key string) (json.RawMessage, time.Time, error) {
	var dataStr string
	var fetchedAtStr string

	err := c.db.QueryRowContext(context.Background(),
		"SELECT data, fetched_at FROM cost_cache WHERE key = ?", key,
	).Scan(&dataStr, &fetchedAtStr)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}

	fetchedAt, _ := time.Parse(time.RFC3339, fetchedAtStr)
	return json.RawMessage(dataStr), fetchedAt, nil
}

// Save stores data under the given key, replacing any existing entry.
func (c *Cache) Save(key string, data json.RawMessage) error {
	_, err := c.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO cost_cache (key, data, fetched_at) VALUES (?, ?, ?)",
		key, string(data), time.Now().Format(time.RFC3339),
	)
	return err
}

// Close closes the cache database.
func (c *Cache) Close() error {
	return c.db.Close()
}
