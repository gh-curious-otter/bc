// Package workspace provides workspace and project management for bc.
//
// A workspace represents a project directory containing bc configuration
// and agent state in .bc/config.toml.
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

	"github.com/rpuneet/bc/pkg/log"
)

// Workspace represents an active workspace.
type Workspace struct {
	Config      *Config      // TOML config
	RoleManager *RoleManager // Role file manager
	RootDir     string
}

// Init initializes a new workspace with TOML config.
func Init(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")

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

	cfg := DefaultConfig(filepath.Base(absRoot))

	configPath := filepath.Join(stateDir, "config.toml")
	if saveErr := cfg.Save(configPath); saveErr != nil {
		return nil, fmt.Errorf("failed to save config: %w", saveErr)
	}

	rm, closeStore, err := initRoleManager(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init role manager: %w", err)
	}
	_ = closeStore // store stays open for workspace lifetime

	return &Workspace{
		RootDir:     absRoot,
		Config:      &cfg,
		RoleManager: rm,
	}, nil
}

// Load loads a workspace from a directory.
// If only a v1 config.json exists (no config.toml), Load returns an error
// wrapping ErrNotV1Workspace so callers can suggest migration.
func Load(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")

	tomlPath := filepath.Join(stateDir, "config.toml")
	if _, statErr := os.Stat(tomlPath); statErr != nil {
		// No config.toml — check for v1 workspace to give an actionable error.
		if _, v1Err := os.Stat(filepath.Join(stateDir, "config.json")); v1Err == nil {
			return nil, fmt.Errorf("%w: run 'bc workspace migrate' to upgrade", ErrNotV1Workspace)
		}
		return nil, fmt.Errorf("not a bc workspace (no .bc/config.toml found in %s)", absRoot)
	}

	cfg, loadErr := LoadConfig(tomlPath)
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load config.toml: %w", loadErr)
	}

	// Backward-compatible version handling: if a config.toml has version < 2
	// (written by an older bc release), bump and rewrite it automatically.
	if cfg.Workspace.Version > 0 && cfg.Workspace.Version < ConfigVersion {
		cfg.Workspace.Version = ConfigVersion
		_ = cfg.Save(tomlPath) // best-effort; don't block Load on write error
	}

	if valErr := cfg.Validate(); valErr != nil {
		return nil, fmt.Errorf("invalid config.toml: %w", valErr)
	}

	rm, closeStore, err := loadRoleManager(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load roles: %w", err)
	}
	_ = closeStore // store stays open for workspace lifetime

	return &Workspace{
		RootDir:     absRoot,
		Config:      cfg,
		RoleManager: rm,
	}, nil
}

// Find searches for a workspace starting from dir and going up.
func Find(dir string) (*Workspace, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

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
	configPath := filepath.Join(w.StateDir(), "config.toml")
	return w.Config.Save(configPath)
}

// StateDir returns the state directory path (.bc/).
func (w *Workspace) StateDir() string {
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
func IsWorkspace(dir string) bool {
	stateDir := filepath.Join(dir, ".bc")
	_, err := os.Stat(stateDir)
	return err == nil
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

// initRoleManager creates a role manager with SQLite store for workspace Init.
// It creates the store, migrates defaults, and ensures filesystem defaults for
// backward compatibility. Returns the manager plus a close function for the store.
func initRoleManager(stateDir string) (*RoleManager, func() error, error) {
	dbPath := filepath.Join(stateDir, "bc.db")
	store, err := NewRoleStore(dbPath)
	if err != nil {
		// Fall back to filesystem-only if SQLite fails
		log.Warn("failed to open role store, using filesystem", "error", err)
		rm := NewRoleManager(stateDir)
		if _, ensureErr := rm.EnsureDefaultRoot(); ensureErr != nil {
			return nil, nil, ensureErr
		}
		if _, ensureErr := rm.EnsureDefaultRoles(); ensureErr != nil {
			log.Warn("failed to create default roles", "error", ensureErr)
		}
		return rm, func() error { return nil }, nil
	}

	// Migrate defaults into SQLite
	if migrateErr := store.MigrateDefaults(); migrateErr != nil {
		log.Warn("failed to migrate default roles to store", "error", migrateErr)
	}

	// Also migrate any existing files
	rolesDir := filepath.Join(stateDir, "roles")
	if _, migrateErr := store.MigrateFromFiles(rolesDir); migrateErr != nil {
		log.Warn("failed to migrate role files to store", "error", migrateErr)
	}

	rm := NewRoleManagerWithStore(stateDir, store)

	// Still ensure filesystem defaults for backward compatibility
	if _, ensureErr := rm.EnsureDefaultRoot(); ensureErr != nil {
		log.Warn("failed to create default root role file", "error", ensureErr)
	}
	if _, ensureErr := rm.EnsureDefaultRoles(); ensureErr != nil {
		log.Warn("failed to create default role files", "error", ensureErr)
	}

	return rm, store.Close, nil
}

// loadRoleManager creates a role manager with SQLite store for workspace Load.
// It opens the store, migrates any new files, and loads all roles into the cache.
func loadRoleManager(stateDir string) (*RoleManager, func() error, error) {
	dbPath := filepath.Join(stateDir, "bc.db")
	store, err := NewRoleStore(dbPath)
	if err != nil {
		// Fall back to filesystem-only if SQLite fails
		log.Warn("failed to open role store, using filesystem", "error", err)
		rm := NewRoleManager(stateDir)
		if _, loadErr := rm.LoadAllRoles(); loadErr != nil {
			return nil, nil, loadErr
		}
		return rm, func() error { return nil }, nil
	}

	// Migrate any filesystem roles that aren't in SQLite yet
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

// Name returns the workspace name.
func (w *Workspace) Name() string {
	if w.Config != nil {
		return w.Config.Workspace.Name
	}
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
