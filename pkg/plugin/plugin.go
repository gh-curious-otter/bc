// Package plugin implements the bc plugin system.
//
// Plugins extend bc with custom agents, tools, and capabilities.
// They can be installed from a registry or local paths.
//
// Plugin types:
//   - Agent: Custom AI agent implementations
//   - Tool: Additional tool capabilities
//   - Role: Custom role definitions
//
// Issue #1213: Plugin system for Phase 4 Ecosystem
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Plugin types
const (
	TypeAgent   = "agent"
	TypeTool    = "tool"
	TypeRole    = "role"
	TypeHook    = "hook"    // RFC 001: Intercept bc events
	TypeCommand = "command" // RFC 001: Add CLI commands
	TypeView    = "view"    // RFC 001: Custom TUI views
)

// Plugin states
const (
	StateInstalled = "installed"
	StateEnabled   = "enabled"
	StateDisabled  = "disabled"
	StateError     = "error"
)

// DefaultRegistry is the default plugin registry URL
const DefaultRegistry = "https://plugins.bc.dev"

// DefaultDirectory is the default plugin installation directory
// Note: This is relative to the state directory (.bc/), not workspace root
const DefaultDirectory = "plugins"

// Manifest describes a plugin's metadata and capabilities
//
//nolint:govet // fieldalignment: logical field grouping preferred over memory optimization
type Manifest struct {
	Name         string       `toml:"name" json:"name"`
	Version      string       `toml:"version" json:"version"`
	Description  string       `toml:"description" json:"description"`
	Author       string       `toml:"author" json:"author"`
	License      string       `toml:"license" json:"license"`
	Homepage     string       `toml:"homepage,omitempty" json:"homepage,omitempty"`
	Repository   string       `toml:"repository,omitempty" json:"repository,omitempty"`
	Type         string       `toml:"type" json:"type"`
	Entrypoint   string       `toml:"entrypoint" json:"entrypoint"`
	BCVersion    string       `toml:"bc_version,omitempty" json:"bc_version,omitempty"`
	Capabilities []string     `toml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Dependencies []Dependency `toml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// RFC 001: Plugin extensibility features
	Hooks       map[string]HookDef    `toml:"hooks,omitempty" json:"hooks,omitempty"`
	Commands    map[string]CommandDef `toml:"commands,omitempty" json:"commands,omitempty"`
	Tools       map[string]ToolDef    `toml:"tools,omitempty" json:"tools,omitempty"`
	Permissions *Permissions          `toml:"permissions,omitempty" json:"permissions,omitempty"`
}

// Dependency describes a plugin dependency
type Dependency struct {
	Name    string `toml:"name" json:"name"`
	Version string `toml:"version" json:"version"`
}

// RFC 001: Hook, Command, and Tool definitions for plugin extensibility

// HookDef defines a hook that intercepts bc events
type HookDef struct {
	Script      string `toml:"script" json:"script"`
	Description string `toml:"description,omitempty" json:"description,omitempty"`
}

// CommandDef defines a plugin command accessible via `bc <plugin> <command>`
type CommandDef struct {
	Script      string `toml:"script" json:"script"`
	Description string `toml:"description,omitempty" json:"description,omitempty"`
}

// ToolDef defines a tool that agents can invoke
type ToolDef struct {
	Script      string `toml:"script" json:"script"`
	Description string `toml:"description,omitempty" json:"description,omitempty"`
}

// Permissions defines what the plugin can access (RFC 001)
//
//nolint:govet // fieldalignment: logical field grouping preferred
type Permissions struct {
	EnvVars    []string `toml:"env_vars,omitempty" json:"env_vars,omitempty"`
	Filesystem string   `toml:"filesystem" json:"filesystem"` // none, workspace, home, all
	Network    bool     `toml:"network" json:"network"`
}

// Plugin represents an installed plugin
//
//nolint:govet // fieldalignment: logical field grouping preferred
type Plugin struct {
	InstalledAt time.Time  `json:"installedAt"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	State       string     `json:"state"`
	Path        string     `json:"path"`
	Error       string     `json:"error,omitempty"`
	Manifest    Manifest   `json:"manifest"`
}

// Registry represents a plugin registry
type Registry struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// SearchResult represents a plugin search result
type SearchResult struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Type        string   `json:"type"`
	Tags        []string `json:"tags,omitempty"`
	Downloads   int      `json:"downloads"`
	Stars       int      `json:"stars"`
}

// Manager manages plugin lifecycle
type Manager struct {
	plugins    map[string]*Plugin
	pluginsDir string
	registries []Registry
}

// NewManager creates a new plugin manager
func NewManager(workspaceDir string) *Manager {
	return &Manager{
		pluginsDir: filepath.Join(workspaceDir, DefaultDirectory),
		registries: []Registry{
			{URL: DefaultRegistry, Name: "default", Enabled: true},
		},
		plugins: make(map[string]*Plugin),
	}
}

// Load loads installed plugins from disk
func (m *Manager) Load(_ context.Context) error {
	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(m.pluginsDir, 0750); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Read plugins state file
	statePath := filepath.Join(m.pluginsDir, "plugins.json")
	data, err := os.ReadFile(statePath) //nolint:gosec // plugins.json is internal state
	if os.IsNotExist(err) {
		return nil // No plugins installed yet
	}
	if err != nil {
		return fmt.Errorf("failed to read plugins state: %w", err)
	}

	var plugins []*Plugin
	if err := json.Unmarshal(data, &plugins); err != nil {
		return fmt.Errorf("failed to parse plugins state: %w", err)
	}

	for _, p := range plugins {
		m.plugins[p.Manifest.Name] = p
	}

	return nil
}

// Save saves plugin state to disk
func (m *Manager) Save() error {
	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}

	data, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugins state: %w", err)
	}

	statePath := filepath.Join(m.pluginsDir, "plugins.json")
	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write plugins state: %w", err)
	}

	return nil
}

// List returns all installed plugins
func (m *Manager) List() []*Plugin {
	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Get returns a plugin by name
func (m *Manager) Get(name string) (*Plugin, bool) {
	p, ok := m.plugins[name]
	return p, ok
}

// Install installs a plugin from a path or URL
func (m *Manager) Install(_ context.Context, source string) (*Plugin, error) {
	// Determine if source is a local path or URL
	var manifestPath string
	var pluginDir string

	if info, err := os.Stat(source); err == nil && info.IsDir() {
		// Local directory
		pluginDir = source
		manifestPath = filepath.Join(source, "plugin.toml")
	} else {
		// TODO: Download from URL or registry
		return nil, fmt.Errorf("remote installation not yet implemented: %s", source)
	}

	// Parse manifest
	var manifest Manifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest: %w", err)
	}

	// Validate manifest
	if err := validateManifest(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	// Check if already installed
	if existing, ok := m.plugins[manifest.Name]; ok {
		return nil, fmt.Errorf("plugin %q already installed (version %s)", manifest.Name, existing.Manifest.Version)
	}

	// Create plugin entry
	now := time.Now()
	plugin := &Plugin{
		Manifest:    manifest,
		State:       StateEnabled,
		Path:        pluginDir,
		InstalledAt: now,
	}

	m.plugins[manifest.Name] = plugin

	if err := m.Save(); err != nil {
		return nil, fmt.Errorf("failed to save plugin state: %w", err)
	}

	return plugin, nil
}

// Uninstall removes a plugin
func (m *Manager) Uninstall(_ context.Context, name string) error {
	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	// Remove from registry
	delete(m.plugins, name)

	if err := m.Save(); err != nil {
		return fmt.Errorf("failed to save plugin state: %w", err)
	}

	// Clean up plugin directory if it's in our plugins dir
	cleanPath := filepath.Clean(plugin.Path)
	cleanPluginsDir := filepath.Clean(m.pluginsDir)
	if strings.HasPrefix(cleanPath, cleanPluginsDir+string(filepath.Separator)) {
		if err := os.RemoveAll(cleanPath); err != nil {
			return fmt.Errorf("failed to remove plugin directory: %w", err)
		}
	}

	return nil
}

// Enable enables a plugin
func (m *Manager) Enable(name string) error {
	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	plugin.State = StateEnabled
	return m.Save()
}

// Disable disables a plugin
func (m *Manager) Disable(name string) error {
	plugin, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	plugin.State = StateDisabled
	return m.Save()
}

// Search searches for plugins in registries
func (m *Manager) Search(_ context.Context, _ string) ([]SearchResult, error) {
	// TODO: Implement registry search
	return nil, fmt.Errorf("registry search not yet implemented")
}

// validateManifest validates a plugin manifest
func validateManifest(m *Manifest) error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Type == "" {
		return fmt.Errorf("type is required")
	}

	switch m.Type {
	case TypeAgent, TypeTool, TypeRole, TypeHook, TypeCommand, TypeView:
		// Valid type
	default:
		return fmt.Errorf("invalid type %q (must be agent, tool, role, hook, command, or view)", m.Type)
	}

	// RFC 001: Validate permissions if present
	if m.Permissions != nil {
		switch m.Permissions.Filesystem {
		case "", "none", "workspace", "home", "all":
			// Valid filesystem permission
		default:
			return fmt.Errorf("invalid permissions.filesystem %q (must be none, workspace, home, or all)", m.Permissions.Filesystem)
		}
	}

	return nil
}

// Info returns detailed information about a plugin
func (m *Manager) Info(name string) (*Plugin, error) {
	plugin, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", name)
	}
	return plugin, nil
}

// Enabled returns all enabled plugins of a given type
func (m *Manager) Enabled(pluginType string) []*Plugin {
	var plugins []*Plugin
	for _, p := range m.plugins {
		if p.State == StateEnabled && (pluginType == "" || p.Manifest.Type == pluginType) {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// RFC 001: Hook execution support

// HookEvent represents an event that can trigger hooks
type HookEvent struct {
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Name      string                 `json:"name"` // e.g., "agent.start", "channel.send"
}

// HookResult represents the result of a hook execution
type HookResult struct {
	Plugin   string `json:"plugin"`
	Hook     string `json:"hook"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// ExecuteHooks runs all registered hooks for an event
// Returns results for each hook. Exit code 0=success, 1=error (log warning), 2=abort
func (m *Manager) ExecuteHooks(ctx context.Context, event HookEvent) ([]HookResult, error) {
	var results []HookResult

	for _, plugin := range m.plugins {
		if plugin.State != StateEnabled {
			continue
		}

		// Check if plugin has hooks for this event
		hookDef, ok := plugin.Manifest.Hooks[event.Name]
		if !ok {
			continue
		}

		result := m.executeHook(ctx, plugin, event.Name, hookDef, event)
		results = append(results, result)

		// Exit code 2 means abort the operation
		if result.ExitCode == 2 {
			return results, fmt.Errorf("hook %s/%s aborted operation: %s", plugin.Manifest.Name, event.Name, result.Output)
		}
	}

	return results, nil
}

// executeHook runs a single hook script
func (m *Manager) executeHook(ctx context.Context, plugin *Plugin, hookName string, hookDef HookDef, event HookEvent) HookResult {
	result := HookResult{
		Plugin: plugin.Manifest.Name,
		Hook:   hookName,
	}

	// Build script path
	scriptPath := filepath.Join(plugin.Path, hookDef.Script)
	if _, err := os.Stat(scriptPath); err != nil {
		result.ExitCode = 1
		result.Error = fmt.Sprintf("hook script not found: %s", scriptPath)
		return result
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("BC_PLUGIN_NAME=%s", plugin.Manifest.Name))
	env = append(env, fmt.Sprintf("BC_EVENT=%s", event.Name))

	// Add payload as individual env vars
	for k, v := range event.Payload {
		env = append(env, fmt.Sprintf("BC_%s=%v", strings.ToUpper(k), v))
	}

	// Prepare payload as JSON for stdin
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		result.ExitCode = 1
		result.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		return result
	}

	// Execute the script
	output, exitCode, err := runScript(ctx, scriptPath, plugin.Path, env, string(payloadJSON))
	result.Output = output
	result.ExitCode = exitCode
	if err != nil {
		result.Error = err.Error()
	}

	return result
}

// runScript executes a script and returns output and exit code
func runScript(ctx context.Context, scriptPath, workDir string, env []string, stdin string) (string, int, error) {
	cmd := exec.CommandContext(ctx, scriptPath) //nolint:gosec // Script path validated before call
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), exitErr.ExitCode(), nil
		}
		return string(output), 1, err
	}
	return string(output), 0, nil
}
