// Package tool provides persistent storage and management for AI tool providers.
package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Tool represents a configured AI tool provider stored in the workspace.
type Tool struct {
	CreatedAt  time.Time      `json:"created_at"`
	Name       string         `json:"name"`
	Command    string         `json:"command"`
	InstallCmd string         `json:"install_cmd,omitempty"`
	UpgradeCmd string         `json:"upgrade_cmd,omitempty"`
	SlashCmds  []string       `json:"slash_cmds,omitempty"`
	MCPServers []string       `json:"mcp_servers,omitempty"`
	Config     map[string]any `json:"config,omitempty"`
	Builtin    bool           `json:"builtin,omitempty"`
	Enabled    bool           `json:"enabled"`
}

// builtinTools contains default configurations for popular AI tools.
var builtinTools = []Tool{
	{
		Name:       "claude",
		Command:    "claude --dangerously-skip-permissions",
		InstallCmd: "npm install -g @anthropic-ai/claude-code",
		UpgradeCmd: "npm update -g @anthropic-ai/claude-code",
		SlashCmds:  []string{"/clear", "/compact", "/help", "/mcp", "/cost", "/quit"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "opencode",
		Command:    "opencode",
		InstallCmd: "go install github.com/opencode-ai/opencode@latest",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "cursor",
		Command:    "cursor-agent",
		InstallCmd: "npm install -g cursor-agent",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "aider",
		Command:    "aider --yes",
		InstallCmd: "pip install aider-chat",
		UpgradeCmd: "pip install --upgrade aider-chat",
		SlashCmds:  []string{"/help", "/quit", "/clear"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "openclaw",
		Command:    "openclaw",
		InstallCmd: "npm install -g openclaw",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "gemini",
		Command:    "gemini",
		InstallCmd: "npm install -g @google/gemini-cli",
		SlashCmds:  []string{"/help", "/quit"},
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "codex",
		Command:    "codex --full-auto",
		InstallCmd: "npm install -g @openai/codex",
		SlashCmds:  []string{"/help", "/quit"},
		Enabled:    true,
		Builtin:    true,
	},
}

// Store provides SQLite-backed tool management.
type Store struct {
	db   *sql.DB
	path string
}

// NewStore creates a new tool store for the given workspace state directory.
func NewStore(stateDir string) *Store {
	return &Store{
		path: filepath.Join(stateDir, "bc.db"),
	}
}

// Open initializes the SQLite database and seeds built-in tools.
func (s *Store) Open() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	s.db = db

	if err := s.seedBuiltins(context.Background()); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to seed built-in tools: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tools (
			name        TEXT PRIMARY KEY,
			command     TEXT NOT NULL,
			install_cmd TEXT,
			upgrade_cmd TEXT,
			slash_cmds  TEXT,
			mcp_servers TEXT,
			config      TEXT,
			builtin     INTEGER NOT NULL DEFAULT 0,
			enabled     INTEGER NOT NULL DEFAULT 1,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *Store) seedBuiltins(ctx context.Context) error {
	for _, t := range builtinTools {
		t := t
		existing, err := s.Get(ctx, t.Name)
		if err != nil {
			return fmt.Errorf("failed to check %s: %w", t.Name, err)
		}
		if existing != nil {
			continue // already seeded
		}
		if err := s.add(ctx, &t); err != nil {
			return fmt.Errorf("failed to seed %s: %w", t.Name, err)
		}
	}
	return nil
}

func marshalJSON(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStrings(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

func unmarshalMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

func (s *Store) add(ctx context.Context, t *Tool) error {
	slashCmds, err := marshalJSON(t.SlashCmds)
	if err != nil {
		return err
	}
	mcpServers, err := marshalJSON(t.MCPServers)
	if err != nil {
		return err
	}
	config, err := marshalJSON(t.Config)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tools (name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.Command, t.InstallCmd, t.UpgradeCmd,
		slashCmds, mcpServers, config, t.Builtin, t.Enabled,
	)
	return err
}

// Add inserts a new tool. Returns an error if a tool with that name already exists.
func (s *Store) Add(ctx context.Context, t *Tool) error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if t.Command == "" {
		return fmt.Errorf("tool command is required")
	}
	existing, err := s.Get(ctx, t.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("tool %q already exists", t.Name)
	}
	return s.add(ctx, t)
}

// Get returns a tool by name. Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, name string) (*Tool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled, created_at
		 FROM tools WHERE name = ?`, name)
	return scanTool(row)
}

func scanTool(row *sql.Row) (*Tool, error) {
	var t Tool
	var installCmd, upgradeCmd, slashCmds, mcpServers, config sql.NullString
	if err := row.Scan(
		&t.Name, &t.Command,
		&installCmd, &upgradeCmd,
		&slashCmds, &mcpServers, &config,
		&t.Builtin, &t.Enabled, &t.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.InstallCmd = installCmd.String
	t.UpgradeCmd = upgradeCmd.String
	t.SlashCmds = unmarshalStrings(slashCmds.String)
	t.MCPServers = unmarshalStrings(mcpServers.String)
	t.Config = unmarshalMap(config.String)
	return &t, nil
}

// List returns all tools.
func (s *Store) List(ctx context.Context) ([]*Tool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled, created_at
		 FROM tools ORDER BY builtin DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort close

	var tools []*Tool
	for rows.Next() {
		var t Tool
		var installCmd, upgradeCmd, slashCmds, mcpServers, config sql.NullString
		if err := rows.Scan(
			&t.Name, &t.Command,
			&installCmd, &upgradeCmd,
			&slashCmds, &mcpServers, &config,
			&t.Builtin, &t.Enabled, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.InstallCmd = installCmd.String
		t.UpgradeCmd = upgradeCmd.String
		t.SlashCmds = unmarshalStrings(slashCmds.String)
		t.MCPServers = unmarshalStrings(mcpServers.String)
		t.Config = unmarshalMap(config.String)
		tools = append(tools, &t)
	}
	return tools, rows.Err()
}

// Update replaces a tool's mutable fields (command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, enabled).
func (s *Store) Update(ctx context.Context, t *Tool) error {
	slashCmds, err := marshalJSON(t.SlashCmds)
	if err != nil {
		return err
	}
	mcpServers, err := marshalJSON(t.MCPServers)
	if err != nil {
		return err
	}
	config, err := marshalJSON(t.Config)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE tools SET command=?, install_cmd=?, upgrade_cmd=?, slash_cmds=?, mcp_servers=?, config=?, enabled=?
		 WHERE name=?`,
		t.Command, t.InstallCmd, t.UpgradeCmd,
		slashCmds, mcpServers, config, t.Enabled,
		t.Name,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("tool %q not found", t.Name)
	}
	return nil
}

// Delete removes a tool by name.
func (s *Store) Delete(ctx context.Context, name string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tools WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("tool %q not found", name)
	}
	return nil
}

// SetEnabled enables or disables a tool.
func (s *Store) SetEnabled(ctx context.Context, name string, enabled bool) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tools SET enabled=? WHERE name=?`, enabled, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("tool %q not found", name)
	}
	return nil
}
