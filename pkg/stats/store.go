package stats

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver via pgx
)

// DefaultStatsDSN is the connection string for the bcstats TimescaleDB container.
const DefaultStatsDSN = "postgres://bc:bc@localhost:5433/bcstats"

// StatsDSN returns the TimescaleDB connection string from STATS_DATABASE_URL env var,
// or the default bcstats DSN if not set.
func StatsDSN() string {
	if dsn := os.Getenv("STATS_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return DefaultStatsDSN
}

// Store provides time-series metrics storage backed by TimescaleDB.
type Store struct {
	db *sql.DB
}

// NewStore connects to TimescaleDB and ensures schema exists.
func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open timescaledb: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping timescaledb at %s: %w", dsn, err)
	}

	s := &Store{db: db}
	if err := s.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for advanced queries.
func (s *Store) DB() *sql.DB {
	return s.db
}

// ensureSchema creates hypertables if they don't exist.
func (s *Store) ensureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS system_metrics (
			time        TIMESTAMPTZ NOT NULL,
			cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_bytes   BIGINT NOT NULL DEFAULT 0,
			mem_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
			disk_bytes  BIGINT NOT NULL DEFAULT 0,
			goroutines  INT NOT NULL DEFAULT 0,
			hostname    TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS agent_metrics (
			time       TIMESTAMPTZ NOT NULL,
			agent_name TEXT NOT NULL,
			agent_id   TEXT NOT NULL DEFAULT '',
			role       TEXT NOT NULL DEFAULT '',
			state      TEXT NOT NULL DEFAULT '',
			cpu_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_bytes  BIGINT NOT NULL DEFAULT 0,
			uptime_sec BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS token_metrics (
			time         TIMESTAMPTZ NOT NULL,
			agent_id     TEXT NOT NULL DEFAULT '',
			agent_name   TEXT NOT NULL DEFAULT '',
			provider     TEXT NOT NULL DEFAULT '',
			model        TEXT NOT NULL DEFAULT '',
			input_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			cost_usd     DOUBLE PRECISION NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS channel_metrics (
			time          TIMESTAMPTZ NOT NULL,
			channel_name  TEXT NOT NULL,
			messages_sent BIGINT NOT NULL DEFAULT 0,
			messages_read BIGINT NOT NULL DEFAULT 0,
			participants  INT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS daemon_metrics (
			time        TIMESTAMPTZ NOT NULL,
			daemon_name TEXT NOT NULL,
			state       TEXT NOT NULL DEFAULT '',
			pid         INT NOT NULL DEFAULT 0,
			cpu_pct     DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_bytes   BIGINT NOT NULL DEFAULT 0,
			restarts    INT NOT NULL DEFAULT 0
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}

	// Convert to hypertables (idempotent via if_not_exists).
	hypertables := []string{
		`SELECT create_hypertable('system_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('agent_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('token_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('channel_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('daemon_metrics', 'time', if_not_exists => TRUE)`,
	}

	for _, stmt := range hypertables {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create hypertable: %w", err)
		}
	}

	return nil
}

// TimeRange specifies a query window and aggregation interval.
type TimeRange struct {
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Interval string    `json:"interval"` // e.g. "1m", "5m", "1h"
}

// PGInterval converts short interval notation (e.g. "5m", "1h", "30s") to
// Postgres interval format (e.g. "5 minutes", "1 hours", "30 seconds").
func (tr TimeRange) PGInterval() string {
	s := tr.Interval
	if s == "" {
		return "5 minutes"
	}
	// Already in Postgres format
	if len(s) > 2 && (s[len(s)-1] < '0' || s[len(s)-1] > '9') && s[len(s)-2] >= 'a' {
		return s
	}
	unit := s[len(s)-1]
	val := s[:len(s)-1]
	switch unit {
	case 's':
		return val + " seconds"
	case 'm':
		return val + " minutes"
	case 'h':
		return val + " hours"
	case 'd':
		return val + " days"
	default:
		return s
	}
}

// SystemMetric represents a system-level resource sample.
type SystemMetric struct {
	Time       time.Time `json:"time"`
	Hostname   string    `json:"hostname"`
	MemBytes   int64     `json:"mem_bytes"`
	DiskBytes  int64     `json:"disk_bytes"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float64   `json:"mem_percent"`
	Goroutines int       `json:"goroutines"`
}

// AgentMetric represents an agent-level resource sample.
type AgentMetric struct {
	Time      time.Time `json:"time"`
	AgentName string    `json:"agent_name"`
	AgentID   string    `json:"agent_id"`
	Role      string    `json:"role"`
	State     string    `json:"state"`
	CPUPct    float64   `json:"cpu_pct"`
	MemBytes  int64     `json:"mem_bytes"`
	UptimeSec int64     `json:"uptime_sec"`
}

// TokenMetric represents a token usage sample.
type TokenMetric struct {
	Time         time.Time `json:"time"`
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
}

// ChannelMetric represents channel activity at a point in time.
type ChannelMetric struct {
	Time         time.Time `json:"time"`
	ChannelName  string    `json:"channel_name"`
	MessagesSent int64     `json:"messages_sent"`
	MessagesRead int64     `json:"messages_read"`
	Participants int       `json:"participants"`
}

// DaemonMetric represents a daemon process sample.
type DaemonMetric struct {
	Time       time.Time `json:"time"`
	DaemonName string    `json:"daemon_name"`
	State      string    `json:"state"`
	PID        int       `json:"pid"`
	CPUPct     float64   `json:"cpu_pct"`
	MemBytes   int64     `json:"mem_bytes"`
	Restarts   int       `json:"restarts"`
}

// StatsSummary provides current aggregate totals across all metrics.
type StatsSummary struct {
	TotalAgents   int     `json:"total_agents"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	TotalMessages int64   `json:"total_messages"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemBytes      int64   `json:"mem_bytes"`
}
