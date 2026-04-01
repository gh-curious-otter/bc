// Package tool provides persistent storage and management for AI tool providers.
package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-curious-otter/bc/pkg/db"
)

// ToolType classifies a tool.
const (
	ToolTypeCLI      = "cli"      // CLI binary (gh, aws, wrangler)
	ToolTypeMCP      = "mcp"      // MCP server (bc, playwright, github)
	ToolTypeProvider = "provider" // AI provider (claude, gemini, cursor)
)

// Tool represents a configured tool in the workspace (CLI, MCP server, or AI provider).
type Tool struct {
	CreatedAt    time.Time         `json:"created_at"`
	Config       map[string]any    `json:"config,omitempty"`
	Env          map[string]string `json:"env,omitempty"`           // env vars, supports ${secret:NAME}
	Name         string            `json:"name"`
	Type         string            `json:"type"`                    // "cli", "mcp", "provider"
	Command      string            `json:"command"`
	InstallCmd   string            `json:"install_cmd,omitempty"`
	UpgradeCmd   string            `json:"upgrade_cmd,omitempty"`
	VersionCmd   string            `json:"version_cmd,omitempty"`   // e.g., "gh --version"
	Transport    string            `json:"transport,omitempty"`     // "stdio", "sse" (MCP only)
	URL          string            `json:"url,omitempty"`           // SSE endpoint (MCP only)
	HealthStatus string            `json:"health_status,omitempty"` // connected/installed/not_installed/error
	LastChecked  string            `json:"last_checked,omitempty"`  // ISO timestamp
	SlashCmds    []string          `json:"slash_cmds,omitempty"`
	Args         []string          `json:"args,omitempty"`          // stdio args (MCP only)
	MCPServers   []string          `json:"mcp_servers,omitempty"`   // associated MCP server names
	Builtin      bool              `json:"builtin,omitempty"`
	Enabled      bool              `json:"enabled"`
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
		Type:       ToolTypeProvider,
	},
	{
		Name:       "opencode",
		Command:    "opencode",
		InstallCmd: "go install github.com/opencode-ai/opencode@latest",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
	{
		Name:       "cursor",
		Command:    "cursor-agent",
		InstallCmd: "npm install -g cursor-agent",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
	{
		Name:       "aider",
		Command:    "aider --yes",
		InstallCmd: "pip install aider-chat",
		UpgradeCmd: "pip install --upgrade aider-chat",
		SlashCmds:  []string{"/help", "/quit", "/clear"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
	{
		Name:       "openclaw",
		Command:    "openclaw",
		InstallCmd: "npm install -g openclaw",
		SlashCmds:  []string{"/exit", "/help"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
	{
		Name:       "gemini",
		Command:    "gemini",
		InstallCmd: "npm install -g @google/gemini-cli",
		SlashCmds:  []string{"/help", "/quit"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
	{
		Name:       "codex",
		Command:    "codex --full-auto",
		InstallCmd: "npm install -g @openai/codex",
		SlashCmds:  []string{"/help", "/quit"},
		Enabled:    true,
		Builtin:    true,
		Type:       ToolTypeProvider,
	},
}

// builtinMCPServers contains default MCP server definitions.
var builtinMCPServers = []Tool{
	{
		Name:      "bc",
		Type:      ToolTypeMCP,
		Transport: "sse",
		URL:       "http://host.docker.internal:9374/_mcp/sse",
		Enabled:   true,
		Builtin:   true,
	},
	{
		Name:       "playwright",
		Type:       ToolTypeMCP,
		Transport:  "sse",
		URL:        "http://host.docker.internal:3000/sse",
		InstallCmd: "npx -y @playwright/mcp@latest",
		Enabled:    true,
		Builtin:    true,
	},
	{
		Name:       "github",
		Type:       ToolTypeMCP,
		Transport:  "stdio",
		Command:    "github-mcp-server",
		InstallCmd: "go install github.com/github/github-mcp-server@latest",
		Env:        map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${secret:GITHUB_PERSONAL_ACCESS_TOKEN}"},
		Enabled:    true,
		Builtin:    true,
	},
}

// Store provides tool management backed by SQLite or Postgres.
type Store struct {
	db   *db.DB
	pg   *PostgresStore // non-nil when using Postgres via OpenStore
	path string
}

// NewStore creates a new tool store for the given workspace state directory.
func NewStore(stateDir string) *Store {
	return &Store{
		path: filepath.Join(stateDir, "tools.db"),
	}
}

// Open initializes the database and seeds built-in tools.
// Uses shared bc.db if available, falls back to tools.db.
func (s *Store) Open() error {
	var database *db.DB
	var err error
	if shared := db.SharedWrapped(); shared != nil {
		database = shared
	} else {
		database, err = db.Open(s.path)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
	}

	// Skip schema init on Postgres — init.sql handles table creation.
	if db.SharedDriver() != "postgres" {
		if err := initSchema(database.DB); err != nil {
			if db.SharedWrapped() == nil {
				_ = database.Close()
			}
			return fmt.Errorf("failed to initialize schema: %w", err)
		}
	}

	s.db = database

	if err := s.seedBuiltins(context.Background()); err != nil {
		_ = database.Close()
		return fmt.Errorf("failed to seed built-in tools: %w", err)
	}

	// Migrate MCP server configs from mcp_servers table into unified tools table.
	s.migrateMCPServers()

	return nil
}

// migrateMCPServers reads the old mcp_servers table and inserts entries into tools
// with type=mcp. Idempotent — skips entries that already exist by name.
func (s *Store) migrateMCPServers() {
	if s.db == nil {
		return
	}
	rows, err := s.db.QueryContext(context.Background(),
		"SELECT name, transport, command, url, args, env, enabled FROM mcp_servers")
	if err != nil {
		return // table may not exist — that's fine
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var name, transport, command, url, argsJSON, envJSON string
		var enabled int
		if err := rows.Scan(&name, &transport, &command, &url, &argsJSON, &envJSON, &enabled); err != nil {
			continue
		}

		// Skip if already in tools table
		var count int
		_ = s.db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM tools WHERE name = ?", name).Scan(&count)
		if count > 0 {
			continue
		}

		// Insert as MCP tool
		_, _ = s.db.ExecContext(context.Background(), `
			INSERT INTO tools (name, type, command, transport, url, args, env, enabled, created_at)
			VALUES (?, 'mcp', ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			name, command, transport, url, argsJSON, envJSON, enabled) //nolint:errcheck
	}
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.pg != nil {
		return s.pg.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func initSchema(db *sql.DB) error {
	_, err := db.ExecContext(context.TODO(), `
		CREATE TABLE IF NOT EXISTS tools (
			name          TEXT PRIMARY KEY,
			type          TEXT NOT NULL DEFAULT 'provider',
			command       TEXT NOT NULL DEFAULT '',
			install_cmd   TEXT,
			upgrade_cmd   TEXT,
			version_cmd   TEXT,
			transport     TEXT DEFAULT '',
			url           TEXT,
			args          TEXT DEFAULT '[]',
			env           TEXT DEFAULT '{}',
			slash_cmds    TEXT,
			mcp_servers   TEXT,
			config        TEXT,
			health_status TEXT DEFAULT 'unknown',
			last_checked  TEXT,
			builtin       BOOLEAN DEFAULT FALSE,
			enabled       BOOLEAN DEFAULT TRUE,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Migration: add new columns to existing tables
	for _, col := range []string{
		"ALTER TABLE tools ADD COLUMN type TEXT NOT NULL DEFAULT 'provider'",
		"ALTER TABLE tools ADD COLUMN transport TEXT DEFAULT ''",
		"ALTER TABLE tools ADD COLUMN url TEXT",
		"ALTER TABLE tools ADD COLUMN args TEXT DEFAULT '[]'",
		"ALTER TABLE tools ADD COLUMN env TEXT DEFAULT '{}'",
		"ALTER TABLE tools ADD COLUMN version_cmd TEXT",
		"ALTER TABLE tools ADD COLUMN health_status TEXT DEFAULT 'unknown'",
		"ALTER TABLE tools ADD COLUMN last_checked TEXT",
	} {
		_, _ = db.ExecContext(context.TODO(), col) //nolint:errcheck // ignore if columns exist
	}

	return nil
}

func (s *Store) seedBuiltins(ctx context.Context) error {
	// Seed provider tools (claude, gemini, cursor, etc.)
	for _, t := range builtinTools {
		t := t
		existing, err := s.Get(ctx, t.Name)
		if err != nil {
			return fmt.Errorf("failed to check %s: %w", t.Name, err)
		}
		if existing != nil {
			continue
		}
		if err := s.add(ctx, &t); err != nil {
			return fmt.Errorf("failed to seed %s: %w", t.Name, err)
		}
	}
	// Seed MCP server tools (bc, playwright, github)
	for _, t := range builtinMCPServers {
		t := t
		existing, err := s.Get(ctx, t.Name)
		if err != nil {
			return fmt.Errorf("failed to check MCP %s: %w", t.Name, err)
		}
		if existing != nil {
			continue
		}
		if err := s.add(ctx, &t); err != nil {
			return fmt.Errorf("failed to seed MCP %s: %w", t.Name, err)
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

func unmarshalStringMap(s string) map[string]string {
	if s == "" || s == "{}" {
		return nil
	}
	var result map[string]string
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
	if s.pg != nil {
		return s.pg.Add(ctx, t)
	}
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

// allColumns is the SELECT column list for the unified tools table.
const allColumns = `name, type, command, install_cmd, upgrade_cmd, version_cmd,
	transport, url, args, env, slash_cmds, mcp_servers, config,
	health_status, last_checked, builtin, enabled, created_at`

// Get returns a tool by name. Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, name string) (*Tool, error) {
	if s.pg != nil {
		return s.pg.Get(ctx, name)
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+allColumns+` FROM tools WHERE name = ?`, name)
	return scanTool(row)
}

func scanTool(row *sql.Row) (*Tool, error) {
	var t Tool
	var toolType, installCmd, upgradeCmd, versionCmd sql.NullString
	var transport, url, argsJSON, envJSON sql.NullString
	var slashCmds, mcpServers, config sql.NullString
	var healthStatus, lastChecked sql.NullString
	if err := row.Scan(
		&t.Name, &toolType, &t.Command,
		&installCmd, &upgradeCmd, &versionCmd,
		&transport, &url, &argsJSON, &envJSON,
		&slashCmds, &mcpServers, &config,
		&healthStatus, &lastChecked,
		&t.Builtin, &t.Enabled, &t.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.Type = toolType.String
	if t.Type == "" {
		t.Type = ToolTypeProvider
	}
	t.InstallCmd = installCmd.String
	t.UpgradeCmd = upgradeCmd.String
	t.VersionCmd = versionCmd.String
	t.Transport = transport.String
	t.URL = url.String
	t.Args = unmarshalStrings(argsJSON.String)
	t.Env = unmarshalStringMap(envJSON.String)
	t.SlashCmds = unmarshalStrings(slashCmds.String)
	t.MCPServers = unmarshalStrings(mcpServers.String)
	t.Config = unmarshalMap(config.String)
	t.HealthStatus = healthStatus.String
	t.LastChecked = lastChecked.String
	return &t, nil
}

// ListOptions controls tool listing behavior.
type ListOptions struct {
	Types []string // filter by type (e.g., ["cli", "mcp"])
}

// List returns all tools, optionally filtered by type.
func (s *Store) List(ctx context.Context) ([]*Tool, error) {
	return s.ListWithOptions(ctx, ListOptions{})
}

// ListWithOptions returns tools filtered by the given options.
func (s *Store) ListWithOptions(ctx context.Context, opts ListOptions) ([]*Tool, error) {
	if s.pg != nil {
		return s.pg.List(ctx)
	}

	query := `SELECT ` + allColumns + ` FROM tools`
	var args []any
	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			placeholders[i] = "?"
			args = append(args, t)
		}
		query += ` WHERE type IN (` + strings.Join(placeholders, ",") + `)`
	}
	query += ` ORDER BY builtin DESC, type ASC, name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var tools []*Tool
	for rows.Next() {
		var t Tool
		var toolType, installCmd, upgradeCmd, versionCmd sql.NullString
		var transport, url, argsJSON, envJSON sql.NullString
		var slashCmds, mcpServers, config sql.NullString
		var healthStatus, lastChecked sql.NullString
		if err := rows.Scan(
			&t.Name, &toolType, &t.Command,
			&installCmd, &upgradeCmd, &versionCmd,
			&transport, &url, &argsJSON, &envJSON,
			&slashCmds, &mcpServers, &config,
			&healthStatus, &lastChecked,
			&t.Builtin, &t.Enabled, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.Type = toolType.String
		if t.Type == "" {
			t.Type = ToolTypeProvider
		}
		t.InstallCmd = installCmd.String
		t.UpgradeCmd = upgradeCmd.String
		t.VersionCmd = versionCmd.String
		t.Transport = transport.String
		t.URL = url.String
		t.Args = unmarshalStrings(argsJSON.String)
		t.Env = unmarshalStringMap(envJSON.String)
		t.SlashCmds = unmarshalStrings(slashCmds.String)
		t.MCPServers = unmarshalStrings(mcpServers.String)
		t.Config = unmarshalMap(config.String)
		t.HealthStatus = healthStatus.String
		t.LastChecked = lastChecked.String
		tools = append(tools, &t)
	}
	return tools, rows.Err()
}

// Update replaces a tool's mutable fields (command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, enabled).
func (s *Store) Update(ctx context.Context, t *Tool) error {
	if s.pg != nil {
		return s.pg.Update(ctx, t)
	}
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
	if s.pg != nil {
		return s.pg.Delete(ctx, name)
	}
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
	if s.pg != nil {
		return s.pg.SetEnabled(ctx, name, enabled)
	}
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
