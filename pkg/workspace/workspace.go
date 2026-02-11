// Package workspace provides workspace/project management.
package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/log"
)

// Config represents workspace configuration (v1 format, deprecated).
type Config struct {
	Name     string `json:"name"`
	RootDir  string `json:"root_dir"`
	StateDir string `json:"state_dir"`

	// Agent settings
	AgentCommand string `json:"agent_command,omitempty"` // Custom command (overrides Tool)
	Tool         string `json:"tool,omitempty"`          // Tool type: claude, cursor, codex, server
	Version      int    `json:"version"`
	MaxWorkers   int    `json:"max_workers"`
}

// Workspace represents an active workspace.
type Workspace struct {
	V2Config    *V2Config    // v2 TOML config (nil if v1 workspace)
	RoleManager *RoleManager // Role file manager
	RootDir     string
	Config      Config // v1 config (deprecated, for backward compat)
	version     int    // Detected config version (1 or 2)
}

// DefaultConfig returns default workspace configuration (v1 format).
func DefaultConfig(rootDir string) Config {
	return Config{
		Version:    1,
		Name:       filepath.Base(rootDir),
		RootDir:    rootDir,
		StateDir:   filepath.Join(rootDir, config.Workspace.StateDir),
		MaxWorkers: int(config.Workspace.MaxWorkers),
	}
}

// Init initializes a new workspace in the given directory.
// For v2 workspaces, use InitV2 instead.
func Init(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	// Create state directory
	stateDir := filepath.Join(absRoot, ".bc")
	if err = os.MkdirAll(stateDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create default config
	cfg := DefaultConfig(absRoot)

	// Save config
	configPath := filepath.Join(stateDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return nil, err
	}

	return &Workspace{
		Config:  cfg,
		RootDir: absRoot,
		version: 1,
	}, nil
}

// InitV2 initializes a new v2 workspace with TOML config.
func InitV2(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")

	// Create required directories
	dirs := []string{
		stateDir,
		filepath.Join(stateDir, "agents"),
		filepath.Join(stateDir, "roles"),
		filepath.Join(stateDir, "memory"),
		filepath.Join(stateDir, "worktrees"),
		filepath.Join(stateDir, "channels"),
		filepath.Join(stateDir, "prompts"),
	}
	for _, dir := range dirs {
		if err = os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Copy default prompts from root prompts/ to .bc/prompts/
	if err := copyDefaultPrompts(absRoot, stateDir); err != nil {
		log.Warn("failed to copy default prompts", "error", err)
		// Non-fatal - workspace can still function
	}

	// Create default v2 config
	v2cfg := DefaultV2Config(filepath.Base(absRoot))

	// Save config.toml
	configPath := filepath.Join(stateDir, "config.toml")
	if err := v2cfg.Save(configPath); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Initialize role manager and create default root.md
	rm := NewRoleManager(stateDir)
	if _, err := rm.EnsureDefaultRoot(); err != nil {
		return nil, fmt.Errorf("failed to create default role: %w", err)
	}

	// Create legacy config for backward compat
	legacyCfg := Config{
		Version:  2,
		Name:     v2cfg.Workspace.Name,
		RootDir:  absRoot,
		StateDir: stateDir,
		Tool:     v2cfg.Tools.Default,
	}

	return &Workspace{
		RootDir:     absRoot,
		Config:      legacyCfg,
		V2Config:    &v2cfg,
		RoleManager: rm,
		version:     2,
	}, nil
}

// Load loads a workspace from a directory.
// Prefers config.toml (v2) over config.json (v1).
// Falls back to config.json with deprecation warning if config.toml not found.
func Load(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")

	// Check for v2 config.toml first
	tomlPath := filepath.Join(stateDir, "config.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		return loadV2Workspace(absRoot, stateDir, tomlPath)
	}

	// Fall back to v1 config.json
	jsonPath := filepath.Join(stateDir, "config.json")
	if _, err := os.Stat(jsonPath); err == nil {
		log.Warn("deprecated v1 workspace detected, consider migrating to v2",
			"path", absRoot,
			"hint", "backup .bc/ and run 'bc init' to create v2 workspace")
		return loadV1Workspace(absRoot, stateDir, jsonPath)
	}

	return nil, fmt.Errorf("not a bc workspace (no .bc/config.toml or .bc/config.json found)")
}

// loadV2Workspace loads a v2 workspace with TOML config.
func loadV2Workspace(absRoot, stateDir, configPath string) (*Workspace, error) {
	v2cfg, err := LoadV2Config(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config.toml: %w", err)
	}

	if err := v2cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config.toml: %w", err)
	}

	// Initialize role manager
	rm := NewRoleManager(stateDir)
	if _, err := rm.LoadAllRoles(); err != nil {
		return nil, fmt.Errorf("failed to load roles: %w", err)
	}

	// Create legacy config for backward compat
	legacyCfg := Config{
		Version:  2,
		Name:     v2cfg.Workspace.Name,
		RootDir:  absRoot,
		StateDir: stateDir,
		Tool:     v2cfg.Tools.Default,
	}
	if tool := v2cfg.GetDefaultTool(); tool != nil {
		legacyCfg.AgentCommand = tool.Command
	}

	return &Workspace{
		RootDir:     absRoot,
		Config:      legacyCfg,
		V2Config:    v2cfg,
		RoleManager: rm,
		version:     2,
	}, nil
}

// loadV1Workspace loads a legacy v1 workspace with JSON config.
func loadV1Workspace(absRoot, stateDir, configPath string) (*Workspace, error) {
	data, err := os.ReadFile(configPath) //nolint:gosec // path constructed from known state dir
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}

	// Update paths if directory was moved
	cfg.RootDir = absRoot
	cfg.StateDir = stateDir

	return &Workspace{
		Config:  cfg,
		RootDir: absRoot,
		version: 1,
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
		// Check for .bc directory
		stateDir := filepath.Join(current, ".bc")
		if _, err := os.Stat(stateDir); err == nil {
			return Load(current)
		}

		// Go up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return nil, fmt.Errorf("no workspace found (searched from %s to root)", absDir)
		}
		current = parent
	}
}

// Save saves the workspace configuration.
// For v2 workspaces, saves config.toml. For v1, saves config.json.
func (w *Workspace) Save() error {
	if w.version == 2 && w.V2Config != nil {
		configPath := filepath.Join(w.Config.StateDir, "config.toml")
		return w.V2Config.Save(configPath)
	}

	// v1 fallback
	configPath := filepath.Join(w.Config.StateDir, "config.json")
	data, err := json.MarshalIndent(w.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

// StateDir returns the state directory path.
func (w *Workspace) StateDir() string {
	return w.Config.StateDir
}

// AgentsDir returns the agents state directory.
func (w *Workspace) AgentsDir() string {
	return filepath.Join(w.Config.StateDir, "agents")
}

// LogsDir returns the logs directory.
func (w *Workspace) LogsDir() string {
	return filepath.Join(w.Config.StateDir, "logs")
}

// RolesDir returns the roles directory path.
func (w *Workspace) RolesDir() string {
	return filepath.Join(w.Config.StateDir, "roles")
}

// MemoryDir returns the memory directory path.
func (w *Workspace) MemoryDir() string {
	if w.V2Config != nil {
		return filepath.Join(w.RootDir, w.V2Config.Memory.Path)
	}
	return filepath.Join(w.Config.StateDir, "memory")
}

// WorktreesDir returns the worktrees directory path.
func (w *Workspace) WorktreesDir() string {
	if w.V2Config != nil {
		return filepath.Join(w.RootDir, w.V2Config.Worktrees.Path)
	}
	return filepath.Join(w.Config.StateDir, "worktrees")
}

// ChannelsDir returns the channels directory path.
func (w *Workspace) ChannelsDir() string {
	return filepath.Join(w.Config.StateDir, "channels")
}

// EnsureDirs creates all required directories.
func (w *Workspace) EnsureDirs() error {
	dirs := []string{
		w.Config.StateDir,
		w.AgentsDir(),
		w.LogsDir(),
	}

	if w.version == 2 {
		dirs = append(dirs,
			w.RolesDir(),
			w.MemoryDir(),
			w.WorktreesDir(),
			w.ChannelsDir(),
		)
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
	stateDir := filepath.Join(dir, config.Workspace.StateDir)
	_, err := os.Stat(stateDir)
	return err == nil
}

// ConfigVersion returns the detected config version (1 or 2).
func (w *Workspace) ConfigVersion() int {
	return w.version
}

// IsV2 returns true if this is a v2 workspace.
func (w *Workspace) IsV2() bool {
	return w.version == 2
}

// GetRole returns a role by name, loading it if necessary.
// Returns nil if no role manager is available (v1 workspace).
func (w *Workspace) GetRole(name string) (*Role, error) {
	if w.RoleManager == nil {
		return nil, fmt.Errorf("role management not available in v1 workspace")
	}

	// Check if already loaded
	if role, ok := w.RoleManager.GetRole(name); ok {
		return role, nil
	}

	// Try to load
	return w.RoleManager.LoadRole(name)
}

// GetRolePrompt returns the prompt content for a role.
// Returns empty string if role not found or v1 workspace.
func (w *Workspace) GetRolePrompt(name string) string {
	role, err := w.GetRole(name)
	if err != nil {
		return ""
	}
	return role.Prompt
}

// DefaultTool returns the default tool name for this workspace.
func (w *Workspace) DefaultTool() string {
	if w.V2Config != nil {
		return w.V2Config.Tools.Default
	}
	if w.Config.Tool != "" {
		return w.Config.Tool
	}
	return "claude"
}

// DefaultToolCommand returns the command for the default tool.
func (w *Workspace) DefaultToolCommand() string {
	if w.V2Config != nil {
		if tool := w.V2Config.GetDefaultTool(); tool != nil {
			return tool.Command
		}
	}
	if w.Config.AgentCommand != "" {
		return w.Config.AgentCommand
	}
	return "claude --dangerously-skip-permissions"
}

// BeadsEnabled returns whether beads integration is enabled.
func (w *Workspace) BeadsEnabled() bool {
	if w.V2Config != nil {
		return w.V2Config.Beads.Enabled
	}
	return true // Default to enabled for v1
}

// DefaultChannels returns the default channel names.
func (w *Workspace) DefaultChannels() []string {
	if w.V2Config != nil {
		return w.V2Config.Channels.Default
	}
	return []string{"general", "engineering"}
}

// copyDefaultPrompts copies default prompt files from root prompts/ to .bc/prompts/.
// This allows users to customize prompts per workspace.
func copyDefaultPrompts(rootDir, stateDir string) error {
	sourceDir := filepath.Join(rootDir, "prompts")
	destDir := filepath.Join(stateDir, "prompts")

	// Check if source prompts directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		// No prompts directory at root, skip silently
		return nil
	}

	// Read all files in source prompts directory
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read prompts directory: %w", err)
	}

	// Copy each .md file
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

		// Skip if destination already exists (don't overwrite customizations)
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		// Copy file
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

	// Copy file permissions
	if info, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, info.Mode())
	}

	return nil
}
