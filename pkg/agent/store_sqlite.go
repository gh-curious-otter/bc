package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/db"
)

// SQLiteStore provides SQLite-backed persistence for agent state.
// It replaces the JSON file-based storage (agents.json, root.json, per-agent JSONs)
// with a single state.db using WAL mode for safe concurrent access.
type SQLiteStore struct {
	db *db.DB
}

// NewSQLiteStore opens (or creates) the state database at dbPath and
// ensures the agents table exists.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	d, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open state db: %w", err)
	}

	if err := createAgentsTable(d); err != nil {
		_ = d.Close()
		return nil, err
	}

	return &SQLiteStore{db: d}, nil
}

func createAgentsTable(d *db.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS agents (
			name          TEXT PRIMARY KEY,
			role          TEXT NOT NULL,
			state         TEXT NOT NULL DEFAULT 'idle',
			tool          TEXT,
			parent_id     TEXT,
			team          TEXT,
			task          TEXT,
			session       TEXT,
			workspace     TEXT NOT NULL,
			worktree_dir  TEXT,
			memory_dir    TEXT,
			log_file      TEXT,
			hooked_work   TEXT,
			children      TEXT,
			is_root       INTEGER NOT NULL DEFAULT 0,
			crash_count   INTEGER NOT NULL DEFAULT 0,
			last_crash_time TEXT,
			recovered_from  TEXT,
			started_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_agents_state ON agents(state);
		CREATE INDEX IF NOT EXISTS idx_agents_role ON agents(role);
		CREATE INDEX IF NOT EXISTS idx_agents_parent ON agents(parent_id);
	`
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("create agents table: %w", err)
	}
	return nil
}

// Save persists a single agent (INSERT OR REPLACE).
func (s *SQLiteStore) Save(a *Agent) error {
	children, err := json.Marshal(a.Children)
	if err != nil {
		return fmt.Errorf("marshal children: %w", err)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO agents
		(name, role, state, tool, parent_id, team, task, session, workspace,
		 worktree_dir, memory_dir, log_file, hooked_work, children,
		 is_root, crash_count, last_crash_time, recovered_from,
		 started_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Name, string(a.Role), string(a.State),
		nullStr(a.Tool), nullStr(a.ParentID), nullStr(a.Team), nullStr(a.Task),
		nullStr(a.Session), a.Workspace,
		nullStr(a.WorktreeDir), nullStr(a.MemoryDir), nullStr(a.LogFile),
		nullStr(a.HookedWork), string(children),
		boolToInt(a.IsRoot), a.CrashCount,
		nullTime(a.LastCrashTime), nullStr(a.RecoveredFrom),
		formatTime(a.StartedAt), formatTime(now),
	)
	return err
}

// Load reads a single agent by name. Returns nil, nil if not found.
func (s *SQLiteStore) Load(name string) (*Agent, error) {
	row := s.db.QueryRow(`
		SELECT name, role, state, tool, parent_id, team, task, session, workspace,
		       worktree_dir, memory_dir, log_file, hooked_work, children,
		       is_root, crash_count, last_crash_time, recovered_from,
		       started_at, updated_at
		FROM agents WHERE name = ?`, name)

	a, err := scanAgentRow(row)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

// Delete removes a single agent by name.
func (s *SQLiteStore) Delete(name string) error {
	_, err := s.db.Exec("DELETE FROM agents WHERE name = ?", name)
	return err
}

// LoadAll reads every agent into a map keyed by name.
func (s *SQLiteStore) LoadAll() (map[string]*Agent, error) {
	rows, err := s.db.Query(`
		SELECT name, role, state, tool, parent_id, team, task, session, workspace,
		       worktree_dir, memory_dir, log_file, hooked_work, children,
		       is_root, crash_count, last_crash_time, recovered_from,
		       started_at, updated_at
		FROM agents`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	agents := make(map[string]*Agent)
	for rows.Next() {
		a, err := scanAgentRow(rows)
		if err != nil {
			return nil, err
		}
		agents[a.Name] = a
	}
	return agents, rows.Err()
}

// SaveAll persists every agent in the map inside a single transaction.
func (s *SQLiteStore) SaveAll(agents map[string]*Agent) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // rollback after commit is no-op

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO agents
		(name, role, state, tool, parent_id, team, task, session, workspace,
		 worktree_dir, memory_dir, log_file, hooked_work, children,
		 is_root, crash_count, last_crash_time, recovered_from,
		 started_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now()
	for _, a := range agents {
		children, err := json.Marshal(a.Children)
		if err != nil {
			return fmt.Errorf("marshal children for %s: %w", a.Name, err)
		}
		_, err = stmt.Exec(
			a.Name, string(a.Role), string(a.State),
			nullStr(a.Tool), nullStr(a.ParentID), nullStr(a.Team), nullStr(a.Task),
			nullStr(a.Session), a.Workspace,
			nullStr(a.WorktreeDir), nullStr(a.MemoryDir), nullStr(a.LogFile),
			nullStr(a.HookedWork), string(children),
			boolToInt(a.IsRoot), a.CrashCount,
			nullTime(a.LastCrashTime), nullStr(a.RecoveredFrom),
			formatTime(a.StartedAt), formatTime(now),
		)
		if err != nil {
			return fmt.Errorf("save agent %s: %w", a.Name, err)
		}
	}
	return tx.Commit()
}

// UpdateState updates only the state column for a given agent.
func (s *SQLiteStore) UpdateState(name string, state State) error {
	res, err := s.db.Exec(
		"UPDATE agents SET state = ?, updated_at = ? WHERE name = ?",
		string(state), formatTime(time.Now()), name,
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
func (s *SQLiteStore) UpdateField(name, field, value string) error {
	// Allowlist of updatable columns to prevent SQL injection.
	allowed := map[string]bool{
		"tool": true, "parent_id": true, "team": true, "task": true,
		"session": true, "worktree_dir": true, "memory_dir": true,
		"log_file": true, "hooked_work": true, "children": true,
		"recovered_from": true,
	}
	if !allowed[field] {
		return fmt.Errorf("field %q is not updatable", field)
	}

	query := fmt.Sprintf("UPDATE agents SET %s = ?, updated_at = ? WHERE name = ?", field) //nolint:gosec // field validated above
	res, err := s.db.Exec(query, value, formatTime(time.Now()), name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", name)
	}
	return nil
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- scan helpers ---

func scanAgentRow(s interface{ Scan(...any) error }) (*Agent, error) {
	var a Agent
	var role, state string
	var tool, parentID, team, task, session, worktreeDir, memoryDir, logFile, hookedWork, childrenJSON *string
	var lastCrashTime, recoveredFrom *string
	var startedAt, updatedAt string
	var isRoot, crashCount int

	err := s.Scan(
		&a.Name, &role, &state,
		&tool, &parentID, &team, &task, &session, &a.Workspace,
		&worktreeDir, &memoryDir, &logFile, &hookedWork, &childrenJSON,
		&isRoot, &crashCount, &lastCrashTime, &recoveredFrom,
		&startedAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	a.ID = a.Name
	a.Role = Role(role)
	a.State = State(state)
	a.Tool = deref(tool)
	a.ParentID = deref(parentID)
	a.Team = deref(team)
	a.Task = deref(task)
	a.Session = deref(session)
	a.WorktreeDir = deref(worktreeDir)
	a.MemoryDir = deref(memoryDir)
	a.LogFile = deref(logFile)
	a.HookedWork = deref(hookedWork)
	a.IsRoot = isRoot != 0
	a.CrashCount = crashCount
	a.RecoveredFrom = deref(recoveredFrom)

	if childrenJSON != nil && *childrenJSON != "" {
		_ = json.Unmarshal([]byte(*childrenJSON), &a.Children) //nolint:errcheck // best-effort
	}
	if lastCrashTime != nil && *lastCrashTime != "" {
		if t, err := time.Parse(time.RFC3339, *lastCrashTime); err == nil {
			a.LastCrashTime = &t
		}
	}
	a.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &a, nil
}

// --- value helpers ---

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
