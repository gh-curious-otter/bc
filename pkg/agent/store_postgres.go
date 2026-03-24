package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	bcdb "github.com/rpuneet/bc/pkg/db"
	"github.com/rpuneet/bc/pkg/log"
)

// PostgresStore provides Postgres-backed persistence for agent state.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a PostgresStore from an existing *sql.DB connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// InitSchema creates the agents and agent_stats tables in Postgres.
func (p *PostgresStore) InitSchema() error {
	ctx := context.Background()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			name            TEXT PRIMARY KEY,
			role            TEXT NOT NULL,
			state           TEXT NOT NULL DEFAULT 'idle',
			tool            TEXT,
			parent_id       TEXT,
			team            TEXT,
			task            TEXT,
			session         TEXT,
			workspace       TEXT NOT NULL,
			worktree_dir    TEXT,
			log_file        TEXT,
			hooked_work     TEXT,
			children        TEXT,
			is_root         BOOLEAN NOT NULL DEFAULT FALSE,
			crash_count     INTEGER NOT NULL DEFAULT 0,
			last_crash_time TIMESTAMPTZ,
			recovered_from  TEXT,
			runtime_backend TEXT,
			session_id      TEXT,
			ttl             INTEGER NOT NULL DEFAULT 0,
			created_at      TIMESTAMPTZ,
			stopped_at      TIMESTAMPTZ,
			deleted_at      TIMESTAMPTZ,
			started_at      TIMESTAMPTZ NOT NULL,
			updated_at      TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_state  ON agents(state)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_role   ON agents(role)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_parent ON agents(parent_id)`,

		`CREATE TABLE IF NOT EXISTS agent_stats (
			id              BIGSERIAL PRIMARY KEY,
			agent_name      TEXT    NOT NULL,
			collected_at    TIMESTAMPTZ NOT NULL,
			cpu_pct         DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_used_mb     DOUBLE PRECISION NOT NULL DEFAULT 0,
			mem_limit_mb    DOUBLE PRECISION NOT NULL DEFAULT 0,
			net_rx_mb       DOUBLE PRECISION NOT NULL DEFAULT 0,
			net_tx_mb       DOUBLE PRECISION NOT NULL DEFAULT 0,
			block_read_mb   DOUBLE PRECISION NOT NULL DEFAULT 0,
			block_write_mb  DOUBLE PRECISION NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_stats_agent ON agent_stats(agent_name)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_stats_time  ON agent_stats(collected_at)`,
	}

	for _, stmt := range stmts {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("postgres agent schema: %w", err)
		}
	}
	return nil
}

// Close closes the database connection.
func (p *PostgresStore) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Save persists a single agent (INSERT or UPDATE on conflict).
func (p *PostgresStore) Save(a *Agent) error {
	children, err := json.Marshal(a.Children)
	if err != nil {
		return fmt.Errorf("marshal children: %w", err)
	}

	now := time.Now()
	createdAt := a.CreatedAt
	if createdAt.IsZero() {
		createdAt = a.StartedAt
	}

	_, err = p.db.ExecContext(context.Background(), `
		INSERT INTO agents
		(name, role, state, tool, parent_id, team, task, session, workspace,
		 worktree_dir, log_file, hooked_work, children,
		 is_root, crash_count, last_crash_time, recovered_from,
		 runtime_backend, session_id, created_at, stopped_at, deleted_at,
		 started_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		        $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		ON CONFLICT(name) DO UPDATE SET
		 role=$2, state=$3, tool=$4, parent_id=$5, team=$6, task=$7, session=$8,
		 workspace=$9, worktree_dir=$10, log_file=$11, hooked_work=$12, children=$13,
		 is_root=$14, crash_count=$15, last_crash_time=$16, recovered_from=$17,
		 runtime_backend=$18, session_id=$19, created_at=$20, stopped_at=$21,
		 deleted_at=$22, started_at=$23, updated_at=$24`,
		a.Name, string(a.Role), string(a.State),
		pgNullStr(a.Tool), pgNullStr(a.ParentID), pgNullStr(a.Team), pgNullStr(a.Task),
		pgNullStr(a.Session), a.Workspace,
		pgNullStr(a.WorktreeDir), pgNullStr(a.LogFile),
		pgNullStr(a.HookedWork), string(children),
		a.IsRoot, a.CrashCount,
		pgNullTimestamp(a.LastCrashTime), pgNullStr(a.RecoveredFrom),
		pgNullStr(a.RuntimeBackend), pgNullStr(a.SessionID),
		pgTimestamp(createdAt), pgNullTimestamp(a.StoppedAt), pgNullTimestamp(a.DeletedAt),
		pgTimestamp(a.StartedAt), pgTimestamp(now),
	)
	return err
}

// Load reads a single agent by name. Returns nil, nil if not found.
func (p *PostgresStore) Load(name string) (*Agent, error) {
	row := p.db.QueryRowContext(context.Background(), pgAgentSelectCols+` FROM agents WHERE name = $1`, name)

	a, err := pgScanAgentRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

// LoadRoot reads the root agent (is_root=true). Returns nil, nil if not found.
func (p *PostgresStore) LoadRoot() (*Agent, error) {
	row := p.db.QueryRowContext(context.Background(), pgAgentSelectCols+` FROM agents WHERE is_root = TRUE LIMIT 1`)

	a, err := pgScanAgentRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

// SoftDelete marks an agent as deleted by setting deleted_at.
func (p *PostgresStore) SoftDelete(name string) error {
	now := time.Now()
	_, err := p.db.ExecContext(context.Background(),
		"UPDATE agents SET deleted_at = $1, updated_at = $2 WHERE name = $3",
		now, now, name,
	)
	return err
}

// Delete removes a single agent by name.
func (p *PostgresStore) Delete(name string) error {
	_, err := p.db.ExecContext(context.Background(), "DELETE FROM agents WHERE name = $1", name)
	return err
}

// LoadAll reads every non-deleted agent into a map keyed by name.
func (p *PostgresStore) LoadAll() (map[string]*Agent, error) {
	rows, err := p.db.QueryContext(context.Background(), pgAgentSelectCols+` FROM agents WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	agents := make(map[string]*Agent)
	for rows.Next() {
		a, scanErr := pgScanAgentRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		agents[a.Name] = a
	}
	return agents, rows.Err()
}

// SaveAll persists every agent in the map inside a single transaction.
func (p *PostgresStore) SaveAll(agents map[string]*Agent) error {
	ctx := context.Background()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // rollback after commit is no-op

	now := time.Now()
	for _, a := range agents {
		children, marshalErr := json.Marshal(a.Children)
		if marshalErr != nil {
			return fmt.Errorf("marshal children for %s: %w", a.Name, marshalErr)
		}
		createdAt := a.CreatedAt
		if createdAt.IsZero() {
			createdAt = a.StartedAt
		}
		_, execErr := tx.ExecContext(ctx, `
			INSERT INTO agents
			(name, role, state, tool, parent_id, team, task, session, workspace,
			 worktree_dir, log_file, hooked_work, children,
			 is_root, crash_count, last_crash_time, recovered_from,
			 runtime_backend, session_id, created_at, stopped_at, deleted_at,
			 started_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			        $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
			ON CONFLICT(name) DO UPDATE SET
			 role=$2, state=$3, tool=$4, parent_id=$5, team=$6, task=$7, session=$8,
			 workspace=$9, worktree_dir=$10, log_file=$11, hooked_work=$12, children=$13,
			 is_root=$14, crash_count=$15, last_crash_time=$16, recovered_from=$17,
			 runtime_backend=$18, session_id=$19, created_at=$20, stopped_at=$21,
			 deleted_at=$22, started_at=$23, updated_at=$24`,
			a.Name, string(a.Role), string(a.State),
			pgNullStr(a.Tool), pgNullStr(a.ParentID), pgNullStr(a.Team), pgNullStr(a.Task),
			pgNullStr(a.Session), a.Workspace,
			pgNullStr(a.WorktreeDir), pgNullStr(a.LogFile),
			pgNullStr(a.HookedWork), string(children),
			a.IsRoot, a.CrashCount,
			pgNullTimestamp(a.LastCrashTime), pgNullStr(a.RecoveredFrom),
			pgNullStr(a.RuntimeBackend), pgNullStr(a.SessionID),
			pgTimestamp(createdAt), pgNullTimestamp(a.StoppedAt), pgNullTimestamp(a.DeletedAt),
			pgTimestamp(a.StartedAt), pgTimestamp(now),
		)
		if execErr != nil {
			return fmt.Errorf("save agent %s: %w", a.Name, execErr)
		}
	}
	return tx.Commit()
}

// UpdateState updates only the state column for a given agent.
func (p *PostgresStore) UpdateState(name string, state State) error {
	res, err := p.db.ExecContext(context.Background(),
		"UPDATE agents SET state = $1, updated_at = $2 WHERE name = $3",
		string(state), time.Now(), name,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", name)
	}
	return nil
}

// UpdateField updates a single text column for a given agent.
func (p *PostgresStore) UpdateField(name, field, value string) error {
	allowed := map[string]bool{
		"tool": true, "parent_id": true, "team": true, "task": true,
		"session": true, "session_id": true, "worktree_dir": true,
		"log_file": true, "hooked_work": true, "children": true,
		"recovered_from": true, "runtime_backend": true,
	}
	if !allowed[field] {
		return fmt.Errorf("field %q is not updatable", field)
	}

	query := fmt.Sprintf("UPDATE agents SET %s = $1, updated_at = $2 WHERE name = $3", field) //nolint:gosec // field validated above
	res, err := p.db.ExecContext(context.Background(), query, value, time.Now(), name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", name)
	}
	return nil
}

// SaveStats inserts a single AgentStatsRecord.
func (p *PostgresStore) SaveStats(rec *AgentStatsRecord) error {
	_, err := p.db.ExecContext(context.Background(), `
		INSERT INTO agent_stats
		(agent_name, collected_at, cpu_pct, mem_used_mb, mem_limit_mb,
		 net_rx_mb, net_tx_mb, block_read_mb, block_write_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rec.AgentName, rec.CollectedAt,
		rec.CPUPct, rec.MemUsedMB, rec.MemLimitMB,
		rec.NetRxMB, rec.NetTxMB, rec.BlockReadMB, rec.BlockWriteMB,
	)
	return err
}

// QueryStats returns the most recent limit stats rows for an agent.
func (p *PostgresStore) QueryStats(agentName string, limit int) ([]*AgentStatsRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := p.db.QueryContext(context.Background(), `
		SELECT agent_name, collected_at, cpu_pct, mem_used_mb, mem_limit_mb,
		       net_rx_mb, net_tx_mb, block_read_mb, block_write_mb
		FROM agent_stats
		WHERE agent_name = $1
		ORDER BY collected_at DESC
		LIMIT $2`, agentName, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []*AgentStatsRecord
	for rows.Next() {
		var rec AgentStatsRecord
		var collectedAt time.Time
		if scanErr := rows.Scan(
			&rec.AgentName, &collectedAt, &rec.CPUPct, &rec.MemUsedMB, &rec.MemLimitMB,
			&rec.NetRxMB, &rec.NetTxMB, &rec.BlockReadMB, &rec.BlockWriteMB,
		); scanErr != nil {
			return nil, scanErr
		}
		rec.CollectedAt = collectedAt
		records = append(records, &rec)
	}
	return records, rows.Err()
}

// --- scan helpers ---

const pgAgentSelectCols = `SELECT name, role, state, tool, parent_id, team, task, session, workspace,
	       worktree_dir, log_file, hooked_work, children,
	       is_root, crash_count, last_crash_time, recovered_from,
	       runtime_backend, session_id, created_at, stopped_at, deleted_at,
	       started_at, updated_at`

func pgScanAgentRow(s interface{ Scan(...any) error }) (*Agent, error) {
	var a Agent
	var role, state string
	var tool, parentID, team, task, session, worktreeDir, logFile, hookedWork, childrenJSON *string
	var recoveredFrom, runtimeBackend, sessionID *string
	var isRoot bool
	var crashCount int
	var lastCrashTime, createdAt, stoppedAt, deletedAt *time.Time
	var startedAt, updatedAt time.Time

	err := s.Scan(
		&a.Name, &role, &state,
		&tool, &parentID, &team, &task, &session, &a.Workspace,
		&worktreeDir, &logFile, &hookedWork, &childrenJSON,
		&isRoot, &crashCount, &lastCrashTime, &recoveredFrom,
		&runtimeBackend, &sessionID, &createdAt, &stoppedAt, &deletedAt,
		&startedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	a.ID = a.Name
	a.Role = Role(role)
	a.State = State(state)
	a.Tool = pgDeref(tool)
	a.ParentID = pgDeref(parentID)
	a.Team = pgDeref(team)
	a.Task = pgDeref(task)
	a.Session = pgDeref(session)
	a.SessionID = pgDeref(sessionID)
	a.WorktreeDir = pgDeref(worktreeDir)
	a.LogFile = pgDeref(logFile)
	a.HookedWork = pgDeref(hookedWork)
	a.IsRoot = isRoot
	a.CrashCount = crashCount
	a.RecoveredFrom = pgDeref(recoveredFrom)
	a.RuntimeBackend = pgDeref(runtimeBackend)
	a.LastCrashTime = lastCrashTime
	a.StoppedAt = stoppedAt
	a.DeletedAt = deletedAt
	a.StartedAt = startedAt
	a.UpdatedAt = updatedAt

	if createdAt != nil {
		a.CreatedAt = *createdAt
	}

	if childrenJSON != nil && *childrenJSON != "" {
		_ = json.Unmarshal([]byte(*childrenJSON), &a.Children) //nolint:errcheck // best-effort
	}

	if a.CreatedAt.IsZero() {
		a.CreatedAt = a.StartedAt
	}

	return &a, nil
}

// --- value helpers ---

func pgNullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func pgNullTimestamp(t *time.Time) *time.Time {
	return t
}

func pgTimestamp(t time.Time) time.Time {
	return t
}

func pgDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// OpenStore opens the agent store for the workspace.
// Priority: DATABASE_URL (Postgres) > SQLite (.bc/bc.db).
func OpenStore(dbPath string) (interface{ Close() error }, error) {
	if bcdb.IsPostgresEnabled() {
		pgDB, err := bcdb.TryOpenPostgres()
		if err != nil {
			log.Warn("failed to connect to Postgres for agent store, falling back to SQLite", "error", err)
		} else if pgDB != nil {
			pg := NewPostgresStore(pgDB)
			if schemaErr := pg.InitSchema(); schemaErr != nil {
				_ = pg.Close()
				log.Warn("failed to init Postgres agent schema, falling back to SQLite", "error", schemaErr)
			} else {
				log.Debug("agent store: using Postgres backend")
				return pg, nil
			}
		}
	}

	// SQLite fallback
	return NewSQLiteStore(dbPath)
}
