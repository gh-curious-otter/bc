// Package workspace provides workspace and project management for bc.
//
// A workspace represents a project directory containing bc configuration
// and agent state in .bc/settings.json.
//
// # Basic Usage
//
// Find the current workspace:
//
//	ws, err := workspace.Find(".")
//	if err != nil {
//	    log.Fatal("not in a bc workspace")
//	}
//	fmt.Println("Workspace:", ws.Name())
//
// Initialize a new workspace:
//
//	ws, err := workspace.Init("/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Load an existing workspace:
//
//	ws, err := workspace.Load("/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gh-curious-otter/bc/pkg/db"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// Workspace represents an active workspace.
type Workspace struct {
	Config      *Config      // JSON config
	RoleManager *RoleManager // Role file manager
	RootDir     string       // Project root directory
	stateDir    string       // State directory (~/.bc/workspaces/<id>/ or legacy .bc/)
}

// Init initializes a new workspace. State is stored under ~/.bc/workspaces/<id>/.
func Init(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	if err := EnsureBCHome(); err != nil {
		return nil, err
	}

	stateDir, err := GlobalStateDir(absRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot determine state directory: %w", err)
	}

	dirs := []string{
		stateDir,
		filepath.Join(stateDir, "agents"),
		filepath.Join(stateDir, "roles"),
		filepath.Join(stateDir, "channels"),
		filepath.Join(stateDir, "prompts"),
	}
	for _, dir := range dirs {
		if err = os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if cpErr := copyDefaultPrompts(absRoot, stateDir); cpErr != nil {
		log.Warn("failed to copy default prompts", "error", cpErr)
	}

	cfg := DefaultConfig()

	configPath := filepath.Join(stateDir, "settings.json")
	if saveErr := cfg.Save(configPath); saveErr != nil {
		return nil, fmt.Errorf("failed to save config: %w", saveErr)
	}

	rm, closeStore, err := initRoleManager(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init role manager: %w", err)
	}
	_ = closeStore // store stays open for workspace lifetime

	// Register in global registry so Find()/IsWorkspace() work without .bc/ marker.
	if reg, regErr := LoadRegistry(); regErr == nil {
		reg.Register(absRoot, cfg.User.Name)
		_ = reg.Save() //nolint:errcheck // best-effort
	}

	return &Workspace{
		RootDir:     absRoot,
		stateDir:    stateDir,
		Config:      &cfg,
		RoleManager: rm,
	}, nil
}

// Load loads a workspace from a directory.
// If only a v1 config.json exists (no settings.json), Load returns an error
// wrapping ErrNotV1Workspace so callers can suggest migration.
func Load(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	// Try global state dir first (~/.bc/workspaces/<id>/)
	stateDir, stateDirErr := GlobalStateDir(absRoot)
	if stateDirErr != nil {
		stateDir = filepath.Join(absRoot, ".bc") // fallback to legacy
	}

	// Load settings.json — check global dir first, then legacy .bc/
	jsonPath := filepath.Join(stateDir, "settings.json")
	if _, statErr := os.Stat(jsonPath); statErr != nil {
		// Try legacy path — auto-migrate if found
		legacyDir := filepath.Join(absRoot, ".bc")
		legacyPath := filepath.Join(legacyDir, "settings.json")
		if _, legacyErr := os.Stat(legacyPath); legacyErr == nil {
			// Auto-migrate legacy .bc/ to ~/.bc/workspaces/<id>/
			if NeedsMigration(absRoot) {
				log.Info("migrating workspace state to ~/.bc/", "from", legacyDir)
				newDir, migrateErr := MigrateToGlobalState(absRoot)
				if migrateErr == nil {
					stateDir = newDir
					jsonPath = filepath.Join(newDir, "settings.json")
				} else {
					log.Warn("migration failed, using legacy path", "error", migrateErr)
					stateDir = legacyDir
					jsonPath = legacyPath
				}
			} else {
				stateDir = legacyDir
				jsonPath = legacyPath
			}
		} else {
			// Check for v1 workspace
			if _, v1Err := os.Stat(filepath.Join(legacyDir, "config.json")); v1Err == nil {
				return nil, fmt.Errorf("%w: run 'bc workspace migrate' to upgrade", ErrNotV1Workspace)
			}
			return nil, fmt.Errorf("not a bc workspace (no settings.json found in %s or %s)", stateDir, legacyDir)
		}
	}

	cfg, loadErr := LoadConfig(jsonPath)
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load settings.json: %w", loadErr)
	}

	cfg.FillDefaults()

	if valErr := cfg.Validate(); valErr != nil {
		return nil, fmt.Errorf("invalid settings.json: %w", valErr)
	}

	rm, closeStore, err := loadRoleManager(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load roles: %w", err)
	}
	_ = closeStore // store stays open for workspace lifetime

	return &Workspace{
		RootDir:     absRoot,
		stateDir:    stateDir,
		Config:      cfg,
		RoleManager: rm,
	}, nil
}

// Find searches for a workspace starting from dir and going up.
// It checks the registry first (for .bc/-free workspaces), then
// falls back to the legacy .bc/ directory walk.
func Find(dir string) (*Workspace, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Registry-first: check if any registered workspace matches this dir or a parent.
	if reg, regErr := LoadRegistry(); regErr == nil {
		current := absDir
		for {
			if entry := reg.Find(current); entry != nil {
				return Load(current)
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	// Legacy fallback: walk up looking for .bc/ directory marker.
	current := absDir
	for {
		stateDir := filepath.Join(current, ".bc")
		if _, err := os.Stat(stateDir); err == nil {
			return Load(current)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil, fmt.Errorf("no workspace found (searched from %s to root)", absDir)
		}
		current = parent
	}
}

// Save saves the workspace configuration.
func (w *Workspace) Save() error {
	configPath := filepath.Join(w.StateDir(), "settings.json")
	return w.Config.Save(configPath)
}

// StateDir returns the state directory path.
// Uses the resolved state dir (global ~/.bc/workspaces/<id>/ or legacy .bc/).
func (w *Workspace) StateDir() string {
	if w.stateDir != "" {
		return w.stateDir
	}
	return filepath.Join(w.RootDir, ".bc")
}

// AgentsDir returns the agents state directory.
func (w *Workspace) AgentsDir() string {
	return filepath.Join(w.StateDir(), "agents")
}

// LogsDir returns the logs directory.
func (w *Workspace) LogsDir() string {
	if w.Config != nil && w.Config.Logs.Path != "" {
		return filepath.Join(w.RootDir, w.Config.Logs.Path)
	}
	return filepath.Join(w.StateDir(), "logs")
}

// RolesDir returns the roles directory path.
func (w *Workspace) RolesDir() string {
	return filepath.Join(w.StateDir(), "roles")
}

// ChannelsDir returns the channels directory path.
func (w *Workspace) ChannelsDir() string {
	return filepath.Join(w.StateDir(), "channels")
}

// EnsureDirs creates all required directories.
func (w *Workspace) EnsureDirs() error {
	dirs := []string{
		w.StateDir(),
		w.AgentsDir(),
		w.LogsDir(),
		w.RolesDir(),
		w.ChannelsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}

	return nil
}

// IsWorkspace checks if a directory is a workspace.
// Checks legacy .bc/ directory and global state dir (~/.bc/workspaces/<id>/).
func IsWorkspace(dir string) bool {
	// Check legacy .bc/ marker
	stateDir := filepath.Join(dir, ".bc")
	if _, err := os.Stat(stateDir); err == nil {
		return true
	}
	// Check global state dir exists on disk
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if globalDir, gErr := GlobalStateDir(absDir); gErr == nil {
		if _, statErr := os.Stat(globalDir); statErr == nil {
			return true
		}
	}
	return false
}

// GetRole returns a role by name, loading it if necessary.
func (w *Workspace) GetRole(name string) (*Role, error) {
	if w.RoleManager == nil {
		return nil, fmt.Errorf("role manager not initialized")
	}

	if role, ok := w.RoleManager.GetRole(name); ok {
		return role, nil
	}

	return w.RoleManager.LoadRole(name)
}

// GetRolePrompt returns the prompt content for a role.
func (w *Workspace) GetRolePrompt(name string) string {
	role, err := w.GetRole(name)
	if err != nil {
		return ""
	}
	return role.Prompt
}

// openRoleStore creates a RoleStore using the shared Postgres connection if
// DATABASE_URL is set, otherwise falls back to SQLite at .bc/bc.db.
func openRoleStore(stateDir string) (*RoleStore, error) {
	if db.IsPostgresEnabled() {
		pgDB, pgErr := db.TryOpenPostgres()
		if pgErr != nil {
			return nil, fmt.Errorf("open postgres for roles: %w", pgErr)
		}
		return NewRoleStoreFromDB(pgDB, "postgres")
	}

	dbPath := filepath.Join(stateDir, "bc.db")
	return NewRoleStore(dbPath)
}

// initRoleManager creates a role manager with SQL store for workspace Init.
// It creates the store, migrates defaults, and migrates any legacy filesystem
// roles. Returns the manager plus a close function for the store.
func initRoleManager(stateDir string) (*RoleManager, func() error, error) {
	store, err := openRoleStore(stateDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open role store: %w", err)
	}

	// Migrate defaults into store
	if migrateErr := store.MigrateDefaults(); migrateErr != nil {
		log.Warn("failed to migrate default roles to store", "error", migrateErr)
	}

	// Also migrate any existing filesystem files
	rolesDir := filepath.Join(stateDir, "roles")
	if _, migrateErr := store.MigrateFromFiles(rolesDir); migrateErr != nil {
		log.Warn("failed to migrate role files to store", "error", migrateErr)
	}

	rm := NewRoleManagerWithStore(stateDir, store)

	// Ensure base and root roles exist in the store
	if _, ensureErr := rm.EnsureDefaultRoot(); ensureErr != nil {
		log.Warn("failed to ensure default root role", "error", ensureErr)
	}
	if _, ensureErr := rm.EnsureDefaultRoles(); ensureErr != nil {
		log.Warn("failed to ensure default roles", "error", ensureErr)
	}

	return rm, store.Close, nil
}

// loadRoleManager creates a role manager with SQL store for workspace Load.
// It opens the store, migrates any new filesystem files, and loads all roles
// into the cache.
func loadRoleManager(stateDir string) (*RoleManager, func() error, error) {
	store, err := openRoleStore(stateDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open role store: %w", err)
	}

	// Seed defaults if store is empty (e.g. fresh Postgres)
	if migrateErr := store.MigrateDefaults(); migrateErr != nil {
		log.Warn("failed to seed default roles", "error", migrateErr)
	}

	// Migrate any filesystem roles that aren't in the store yet
	rolesDir := filepath.Join(stateDir, "roles")
	if _, migrateErr := store.MigrateFromFiles(rolesDir); migrateErr != nil {
		log.Warn("failed to migrate role files to store", "error", migrateErr)
	}

	rm := NewRoleManagerWithStore(stateDir, store)
	if _, loadErr := rm.LoadAllRoles(); loadErr != nil {
		_ = store.Close()
		return nil, nil, loadErr
	}

	return rm, store.Close, nil
}

// DefaultProvider returns the default provider name for this workspace.
func (w *Workspace) DefaultProvider() string {
	if w.Config != nil {
		return w.Config.GetDefaultProvider()
	}
	return "claude"
}

// DefaultProviderCommand returns the command for the default provider.
func (w *Workspace) DefaultProviderCommand() string {
	if w.Config != nil {
		if p := w.Config.GetProvider(w.Config.GetDefaultProvider()); p != nil {
			return p.Command
		}
	}
	return ""
}

// Name returns the workspace name (derived from directory).
func (w *Workspace) Name() string {
	return filepath.Base(w.RootDir)
}

// copyDefaultPrompts copies default prompt files from root prompts/ to .bc/prompts/.
func copyDefaultPrompts(rootDir, stateDir string) error {
	sourceDir := filepath.Join(rootDir, "prompts")
	destDir := filepath.Join(stateDir, "prompts")

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}

		sourcePath := filepath.Join(sourceDir, name)
		destPath := filepath.Join(destDir, name)

		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", name, err)
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	// #nosec G304 - src path is from internal prompts directory
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	// #nosec G304 - dst path is in workspace .bc/prompts directory
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	if info, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, info.Mode())
	}

	return nil
}
