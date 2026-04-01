package tool

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	bcdb "github.com/gh-curious-otter/bc/pkg/db"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// PostgresStore provides Postgres-backed tool management.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a PostgresStore from an existing *sql.DB connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// InitSchema creates the tools table in Postgres if it doesn't exist.
func (p *PostgresStore) InitSchema() error {
	ctx := context.Background()

	stmt := `CREATE TABLE IF NOT EXISTS tools (
		name        TEXT PRIMARY KEY,
		command     TEXT NOT NULL,
		install_cmd TEXT,
		upgrade_cmd TEXT,
		slash_cmds  TEXT,
		mcp_servers TEXT,
		config      TEXT,
		builtin     BOOLEAN DEFAULT FALSE,
		enabled     BOOLEAN DEFAULT TRUE,
		created_at  TIMESTAMPTZ DEFAULT NOW()
	)`

	if _, err := p.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("postgres tools schema: %w", err)
	}
	return nil
}

// Close is a no-op — the shared DB is owned by the caller.
func (p *PostgresStore) Close() error {
	return nil
}

// SeedBuiltins seeds built-in tools and MCP servers if they don't exist.
func (p *PostgresStore) SeedBuiltins(ctx context.Context) error {
	for _, t := range builtinTools {
		t := t
		existing, err := p.Get(ctx, t.Name)
		if err != nil {
			return fmt.Errorf("failed to check %s: %w", t.Name, err)
		}
		if existing != nil {
			continue
		}
		if err := p.add(ctx, &t); err != nil {
			return fmt.Errorf("failed to seed %s: %w", t.Name, err)
		}
	}
	for _, t := range builtinMCPServers {
		t := t
		existing, err := p.Get(ctx, t.Name)
		if err != nil {
			return fmt.Errorf("failed to check MCP %s: %w", t.Name, err)
		}
		if existing != nil {
			continue
		}
		if err := p.add(ctx, &t); err != nil {
			return fmt.Errorf("failed to seed MCP %s: %w", t.Name, err)
		}
	}
	return nil
}

func (p *PostgresStore) add(ctx context.Context, t *Tool) error {
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

	_, err = p.db.ExecContext(ctx,
		`INSERT INTO tools (name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.Name, t.Command, t.InstallCmd, t.UpgradeCmd,
		slashCmds, mcpServers, config, t.Builtin, t.Enabled,
	)
	return err
}

// Add inserts a new tool.
func (p *PostgresStore) Add(ctx context.Context, t *Tool) error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if t.Command == "" {
		return fmt.Errorf("tool command is required")
	}
	existing, err := p.Get(ctx, t.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("tool %q already exists", t.Name)
	}
	return p.add(ctx, t)
}

// Get returns a tool by name. Returns nil, nil if not found.
func (p *PostgresStore) Get(ctx context.Context, name string) (*Tool, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled, created_at
		 FROM tools WHERE name = $1`, name)
	return pgScanTool(row)
}

func pgScanTool(row *sql.Row) (*Tool, error) {
	var t Tool
	var installCmd, upgradeCmd, slashCmds, mcpServers, config sql.NullString
	var createdAt time.Time
	if err := row.Scan(
		&t.Name, &t.Command,
		&installCmd, &upgradeCmd,
		&slashCmds, &mcpServers, &config,
		&t.Builtin, &t.Enabled, &createdAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.CreatedAt = createdAt
	t.InstallCmd = installCmd.String
	t.UpgradeCmd = upgradeCmd.String
	t.SlashCmds = unmarshalStrings(slashCmds.String)
	t.MCPServers = unmarshalStrings(mcpServers.String)
	t.Config = unmarshalMap(config.String)
	return &t, nil
}

// List returns all tools.
func (p *PostgresStore) List(ctx context.Context) ([]*Tool, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT name, command, install_cmd, upgrade_cmd, slash_cmds, mcp_servers, config, builtin, enabled, created_at
		 FROM tools ORDER BY builtin DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tools []*Tool
	for rows.Next() {
		var t Tool
		var installCmd, upgradeCmd, slashCmds, mcpServers, config sql.NullString
		var createdAt time.Time
		if scanErr := rows.Scan(
			&t.Name, &t.Command,
			&installCmd, &upgradeCmd,
			&slashCmds, &mcpServers, &config,
			&t.Builtin, &t.Enabled, &createdAt,
		); scanErr != nil {
			return nil, scanErr
		}
		t.CreatedAt = createdAt
		t.InstallCmd = installCmd.String
		t.UpgradeCmd = upgradeCmd.String
		t.SlashCmds = unmarshalStrings(slashCmds.String)
		t.MCPServers = unmarshalStrings(mcpServers.String)
		t.Config = unmarshalMap(config.String)
		tools = append(tools, &t)
	}
	return tools, rows.Err()
}

// Update replaces a tool's mutable fields.
func (p *PostgresStore) Update(ctx context.Context, t *Tool) error {
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

	res, err := p.db.ExecContext(ctx,
		`UPDATE tools SET command=$1, install_cmd=$2, upgrade_cmd=$3, slash_cmds=$4, mcp_servers=$5, config=$6, enabled=$7
		 WHERE name=$8`,
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
func (p *PostgresStore) Delete(ctx context.Context, name string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM tools WHERE name = $1`, name)
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
func (p *PostgresStore) SetEnabled(ctx context.Context, name string, enabled bool) error {
	res, err := p.db.ExecContext(ctx, `UPDATE tools SET enabled=$1 WHERE name=$2`, enabled, name)
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

// OpenStore opens the tool store using the shared workspace database.
// Uses the shared driver type to determine the backend (timescale or sqlite).
func OpenStore(stateDir string) (*Store, error) {
	driver := bcdb.SharedDriver()
	if driver == "timescale" {
		shared := bcdb.Shared()
		if shared == nil {
			return nil, fmt.Errorf("tool store: shared timescale connection is nil")
		}
		pg := NewPostgresStore(shared)
		if schemaErr := pg.InitSchema(); schemaErr != nil {
			return nil, fmt.Errorf("tool store: init timescale schema: %w", schemaErr)
		}
		if seedErr := pg.SeedBuiltins(context.Background()); seedErr != nil {
			return nil, fmt.Errorf("tool store: seed builtins: %w", seedErr)
		}
		log.Debug("tool store: using TimescaleDB backend")
		return &Store{pg: pg}, nil
	}

	// SQLite via shared DB
	s := NewStore(stateDir)
	if err := s.Open(); err != nil {
		return nil, err
	}
	return s, nil
}
