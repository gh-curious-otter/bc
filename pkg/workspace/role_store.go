package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/db"
)

// RoleStore provides SQLite-backed persistence for roles.
// It replaces the filesystem-based .bc/roles/*.md storage with a single
// bc.db table, enabling CRUD from the web UI and transactional updates.
type RoleStore struct {
	db *db.DB
}

// NewRoleStore opens (or creates) the database at dbPath and ensures
// the roles table exists.
func NewRoleStore(dbPath string) (*RoleStore, error) {
	d, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open roles db: %w", err)
	}

	if err := createRolesTable(d); err != nil {
		_ = d.Close()
		return nil, err
	}

	return &RoleStore{db: d}, nil
}

func createRolesTable(d *db.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS roles (
			name          TEXT PRIMARY KEY,
			description   TEXT NOT NULL DEFAULT '',
			prompt        TEXT NOT NULL DEFAULT '',
			mcp_servers   TEXT NOT NULL DEFAULT '[]',
			parent_roles  TEXT NOT NULL DEFAULT '[]',
			secrets       TEXT NOT NULL DEFAULT '[]',
			plugins       TEXT NOT NULL DEFAULT '[]',
			settings      TEXT NOT NULL DEFAULT '{}',
			rules         TEXT NOT NULL DEFAULT '{}',
			agents        TEXT NOT NULL DEFAULT '{}',
			skills        TEXT NOT NULL DEFAULT '{}',
			commands      TEXT NOT NULL DEFAULT '{}',
			prompt_create TEXT NOT NULL DEFAULT '',
			prompt_start  TEXT NOT NULL DEFAULT '',
			prompt_stop   TEXT NOT NULL DEFAULT '',
			prompt_delete TEXT NOT NULL DEFAULT '',
			review        TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		);
	`
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("create roles table: %w", err)
	}
	return nil
}

// Save persists a single role (INSERT OR REPLACE).
func (s *RoleStore) Save(role *Role) error {
	if role.Metadata.Name == "" {
		return fmt.Errorf("role name is required")
	}

	mcpServers, err := json.Marshal(role.Metadata.MCPServers)
	if err != nil {
		return fmt.Errorf("marshal mcp_servers: %w", err)
	}

	parentRoles, err := json.Marshal(role.Metadata.ParentRoles)
	if err != nil {
		return fmt.Errorf("marshal parent_roles: %w", err)
	}

	secretsJSON, err := json.Marshal(role.Metadata.Secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	pluginsJSON, err := json.Marshal(role.Metadata.Plugins)
	if err != nil {
		return fmt.Errorf("marshal plugins: %w", err)
	}

	settingsJSON, err := json.Marshal(role.Metadata.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	rulesJSON, err := json.Marshal(role.Metadata.Rules)
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}

	agentsJSON, err := json.Marshal(role.Metadata.Agents)
	if err != nil {
		return fmt.Errorf("marshal agents: %w", err)
	}

	skillsJSON, err := json.Marshal(role.Metadata.Skills)
	if err != nil {
		return fmt.Errorf("marshal skills: %w", err)
	}

	commandsJSON, err := json.Marshal(role.Metadata.Commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}

	now := time.Now().Format(time.RFC3339)

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO roles
		(name, description, prompt, mcp_servers, parent_roles, secrets, plugins,
		 settings, rules, agents, skills, commands,
		 prompt_create, prompt_start, prompt_stop, prompt_delete, review,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		 COALESCE((SELECT created_at FROM roles WHERE name = ?), ?), ?)`,
		role.Metadata.Name, role.Metadata.Description, role.Prompt,
		string(mcpServers), string(parentRoles), string(secretsJSON), string(pluginsJSON),
		string(settingsJSON), string(rulesJSON), string(agentsJSON), string(skillsJSON), string(commandsJSON),
		role.Metadata.PromptCreate, role.Metadata.PromptStart,
		role.Metadata.PromptStop, role.Metadata.PromptDelete, role.Metadata.Review,
		role.Metadata.Name, now, now,
	)
	return err
}

// Load reads a single role by name. Returns an error if not found.
func (s *RoleStore) Load(name string) (*Role, error) {
	row := s.db.QueryRow(`
		SELECT name, description, prompt, mcp_servers, parent_roles, secrets, plugins,
		       settings, rules, agents, skills, commands,
		       prompt_create, prompt_start, prompt_stop, prompt_delete, review,
		       created_at, updated_at
		FROM roles WHERE name = ?`, name)

	return scanRoleRow(row)
}

// LoadAll reads every role into a map keyed by name.
func (s *RoleStore) LoadAll() (map[string]*Role, error) {
	rows, err := s.db.Query(`
		SELECT name, description, prompt, mcp_servers, parent_roles, secrets, plugins,
		       settings, rules, agents, skills, commands,
		       prompt_create, prompt_start, prompt_stop, prompt_delete, review,
		       created_at, updated_at
		FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	roles := make(map[string]*Role)
	for rows.Next() {
		role, err := scanRoleRow(rows)
		if err != nil {
			return nil, err
		}
		roles[role.Metadata.Name] = role
	}
	return roles, rows.Err()
}

// Delete removes a single role by name.
func (s *RoleStore) Delete(name string) error {
	res, err := s.db.Exec("DELETE FROM roles WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("role %q not found", name)
	}
	return nil
}

// Has checks if a role exists in the database.
func (s *RoleStore) Has(name string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM roles WHERE name = ?", name).Scan(&count)
	return err == nil && count > 0
}

// Close closes the database.
func (s *RoleStore) Close() error {
	return s.db.Close()
}

// MigrateFromFiles scans a roles directory for .md files and inserts
// them into the database. Existing roles in the DB are not overwritten.
// The source files are NOT deleted (kept as backup).
func (s *RoleStore) MigrateFromFiles(rolesDir string) (int, error) {
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read roles directory: %w", err)
	}

	var migrated int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")

		// Skip if already in DB
		if s.Has(name) {
			continue
		}

		filePath := filepath.Join(rolesDir, entry.Name())
		data, readErr := os.ReadFile(filePath) //nolint:gosec // path constructed from known roles dir
		if readErr != nil {
			return migrated, fmt.Errorf("read role file %s: %w", entry.Name(), readErr)
		}

		role, parseErr := ParseRoleFile(data)
		if parseErr != nil {
			return migrated, fmt.Errorf("parse role file %s: %w", entry.Name(), parseErr)
		}

		// Ensure name is set
		if role.Metadata.Name == "" {
			role.Metadata.Name = name
		}
		role.FilePath = filePath

		if saveErr := s.Save(role); saveErr != nil {
			return migrated, fmt.Errorf("save role %s: %w", name, saveErr)
		}
		migrated++
	}

	return migrated, nil
}

// MigrateDefaults inserts the built-in default roles (base, root, and
// DefaultRoles map) into the database if they don't already exist.
func (s *RoleStore) MigrateDefaults() error {
	// base role
	if !s.Has("base") {
		role, err := ParseRoleFile([]byte(DefaultBaseRole))
		if err != nil {
			return fmt.Errorf("parse default base role: %w", err)
		}
		if saveErr := s.Save(role); saveErr != nil {
			return fmt.Errorf("save default base role: %w", saveErr)
		}
	}

	// root role
	if !s.Has("root") {
		role, err := ParseRoleFile([]byte(DefaultRootRole))
		if err != nil {
			return fmt.Errorf("parse default root role: %w", err)
		}
		if saveErr := s.Save(role); saveErr != nil {
			return fmt.Errorf("save default root role: %w", saveErr)
		}
	}

	// other default roles
	for name, content := range DefaultRoles {
		if s.Has(name) {
			continue
		}
		role, err := ParseRoleFile([]byte(content))
		if err != nil {
			return fmt.Errorf("parse default role %s: %w", name, err)
		}
		if role.Metadata.Name == "" {
			role.Metadata.Name = name
		}
		if saveErr := s.Save(role); saveErr != nil {
			return fmt.Errorf("save default role %s: %w", name, saveErr)
		}
	}

	return nil
}

// scanRoleRow scans a role from a database row/rows scanner.
func scanRoleRow(s interface{ Scan(...any) error }) (*Role, error) {
	var (
		name, description, prompt                           string
		mcpServersJSON, parentRolesJSON                     string
		secretsJSON, pluginsJSON                            string
		settingsJSON, rulesJSON, agentsJSON                 string
		skillsJSON, commandsJSON                            string
		promptCreate, promptStart, promptStop, promptDelete string
		review                                              string
		createdAt, updatedAt                                string //nolint:unused // reserved for future use
	)

	err := s.Scan(
		&name, &description, &prompt,
		&mcpServersJSON, &parentRolesJSON, &secretsJSON, &pluginsJSON,
		&settingsJSON, &rulesJSON, &agentsJSON, &skillsJSON, &commandsJSON,
		&promptCreate, &promptStart, &promptStop, &promptDelete, &review,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}

	role := &Role{
		Prompt: prompt,
		Metadata: RoleMetadata{
			Name:         name,
			Description:  description,
			PromptCreate: promptCreate,
			PromptStart:  promptStart,
			PromptStop:   promptStop,
			PromptDelete: promptDelete,
			Review:       review,
		},
	}

	// Unmarshal JSON array/object fields
	if mcpServersJSON != "" && mcpServersJSON != "[]" {
		_ = json.Unmarshal([]byte(mcpServersJSON), &role.Metadata.MCPServers) //nolint:errcheck // best-effort
	}
	if parentRolesJSON != "" && parentRolesJSON != "[]" {
		_ = json.Unmarshal([]byte(parentRolesJSON), &role.Metadata.ParentRoles) //nolint:errcheck
	}
	if secretsJSON != "" && secretsJSON != "[]" {
		_ = json.Unmarshal([]byte(secretsJSON), &role.Metadata.Secrets) //nolint:errcheck
	}
	if pluginsJSON != "" && pluginsJSON != "[]" {
		_ = json.Unmarshal([]byte(pluginsJSON), &role.Metadata.Plugins) //nolint:errcheck
	}
	if settingsJSON != "" && settingsJSON != "{}" {
		_ = json.Unmarshal([]byte(settingsJSON), &role.Metadata.Settings) //nolint:errcheck
	}
	if rulesJSON != "" && rulesJSON != "{}" {
		_ = json.Unmarshal([]byte(rulesJSON), &role.Metadata.Rules) //nolint:errcheck
	}
	if agentsJSON != "" && agentsJSON != "{}" {
		_ = json.Unmarshal([]byte(agentsJSON), &role.Metadata.Agents) //nolint:errcheck
	}
	if skillsJSON != "" && skillsJSON != "{}" {
		_ = json.Unmarshal([]byte(skillsJSON), &role.Metadata.Skills) //nolint:errcheck
	}
	if commandsJSON != "" && commandsJSON != "{}" {
		_ = json.Unmarshal([]byte(commandsJSON), &role.Metadata.Commands) //nolint:errcheck
	}

	return role, nil
}
