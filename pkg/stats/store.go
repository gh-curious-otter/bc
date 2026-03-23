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

// StatsDSN returns the TimescaleDB connection string.
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
func (s *Store) Close() error { return s.db.Close() }

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) ensureSchema(ctx context.Context) error {
	stmts := []string{
		// System metrics — bc-daemon, bc-sql, bc-stats containers
		`CREATE TABLE IF NOT EXISTS system_metrics (
			time            TIMESTAMPTZ NOT NULL,
			system_name     TEXT NOT NULL,
			cpu_percent     DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_used_bytes  BIGINT NOT NULL DEFAULT 0,
			mem_limit_bytes BIGINT NOT NULL DEFAULT 0,
			mem_percent     DOUBLE PRECISION NOT NULL DEFAULT 0,
			net_rx_bytes    BIGINT NOT NULL DEFAULT 0,
			net_tx_bytes    BIGINT NOT NULL DEFAULT 0,
			disk_read_bytes BIGINT NOT NULL DEFAULT 0,
			disk_write_bytes BIGINT NOT NULL DEFAULT 0
		)`,
		// Agent metrics — per-agent container stats
		`CREATE TABLE IF NOT EXISTS agent_metrics (
			time            TIMESTAMPTZ NOT NULL,
			agent_name      TEXT NOT NULL,
			role            TEXT NOT NULL DEFAULT '',
			tool            TEXT NOT NULL DEFAULT '',
			runtime         TEXT NOT NULL DEFAULT 'docker',
			state           TEXT NOT NULL DEFAULT '',
			cpu_percent     DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_used_bytes  BIGINT NOT NULL DEFAULT 0,
			mem_limit_bytes BIGINT NOT NULL DEFAULT 0,
			mem_percent     DOUBLE PRECISION NOT NULL DEFAULT 0,
			net_rx_bytes    BIGINT NOT NULL DEFAULT 0,
			net_tx_bytes    BIGINT NOT NULL DEFAULT 0,
			disk_read_bytes BIGINT NOT NULL DEFAULT 0,
			disk_write_bytes BIGINT NOT NULL DEFAULT 0
		)`,
		// Token metrics — per-agent token consumption from JSONL
		`CREATE TABLE IF NOT EXISTS token_metrics (
			time          TIMESTAMPTZ NOT NULL,
			agent_name    TEXT NOT NULL DEFAULT '',
			model         TEXT NOT NULL DEFAULT '',
			input_tokens  BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			cache_read    BIGINT NOT NULL DEFAULT 0,
			cache_create  BIGINT NOT NULL DEFAULT 0,
			cost_usd      DOUBLE PRECISION NOT NULL DEFAULT 0
		)`,
		// Channel metrics — message/member/reaction counts
		`CREATE TABLE IF NOT EXISTS channel_metrics (
			time           TIMESTAMPTZ NOT NULL,
			channel_name   TEXT NOT NULL,
			message_count  BIGINT NOT NULL DEFAULT 0,
			member_count   INT NOT NULL DEFAULT 0,
			reaction_count BIGINT NOT NULL DEFAULT 0
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}

	hypertables := []string{
		`SELECT create_hypertable('system_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('agent_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('token_metrics', 'time', if_not_exists => TRUE)`,
		`SELECT create_hypertable('channel_metrics', 'time', if_not_exists => TRUE)`,
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
	From     time.Time
	To       time.Time
	Interval string // e.g. "5m", "1h" — converted to Postgres interval via PGInterval()
}

// PGInterval converts short notation (5m, 1h) to Postgres format (5 minutes, 1 hours).
func (tr TimeRange) PGInterval() string {
	s := tr.Interval
	if s == "" {
		return "5 minutes"
	}
	if len(s) < 2 {
		return s
	}
	val := s[:len(s)-1]
	switch s[len(s)-1] {
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

// ── Types ──────────────────────────────────────────────────────────────────────

// SystemMetric represents a system container resource sample.
type SystemMetric struct {
	Time           time.Time `json:"time"`
	SystemName     string    `json:"system_name"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemUsedBytes   int64     `json:"mem_used_bytes"`
	MemLimitBytes  int64     `json:"mem_limit_bytes"`
	MemPercent     float64   `json:"mem_percent"`
	NetRxBytes     int64     `json:"net_rx_bytes"`
	NetTxBytes     int64     `json:"net_tx_bytes"`
	DiskReadBytes  int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64     `json:"disk_write_bytes"`
}

// AgentMetric represents an agent container resource sample.
type AgentMetric struct {
	Time           time.Time `json:"time"`
	AgentName      string    `json:"agent_name"`
	Role           string    `json:"role"`
	Tool           string    `json:"tool"`
	Runtime        string    `json:"runtime"`
	State          string    `json:"state"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemUsedBytes   int64     `json:"mem_used_bytes"`
	MemLimitBytes  int64     `json:"mem_limit_bytes"`
	MemPercent     float64   `json:"mem_percent"`
	NetRxBytes     int64     `json:"net_rx_bytes"`
	NetTxBytes     int64     `json:"net_tx_bytes"`
	DiskReadBytes  int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64     `json:"disk_write_bytes"`
}

// TokenMetric represents token consumption at a point in time.
type TokenMetric struct {
	Time         time.Time `json:"time"`
	AgentName    string    `json:"agent_name"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	CacheRead    int64     `json:"cache_read"`
	CacheCreate  int64     `json:"cache_create"`
	CostUSD      float64   `json:"cost_usd"`
}

// ChannelMetric represents channel activity at a point in time.
type ChannelMetric struct {
	Time          time.Time `json:"time"`
	ChannelName   string    `json:"channel_name"`
	MessageCount  int64     `json:"message_count"`
	MemberCount   int       `json:"member_count"`
	ReactionCount int64     `json:"reaction_count"`
}
