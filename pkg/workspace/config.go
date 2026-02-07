// Package workspace provides workspace/project management.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigVersion is the current config schema version.
const ConfigVersion = 2

// V2Config represents the TOML-based workspace configuration for bc v2.
type V2Config struct {
	Workspace WorkspaceConfig `toml:"workspace"`
	Worktrees WorktreesConfig `toml:"worktrees"`
	Tools     ToolsConfig     `toml:"tools"`
	Memory    MemoryConfig    `toml:"memory"`
	Beads     BeadsConfig     `toml:"beads"`
	Channels  ChannelsConfig  `toml:"channels"`
}

// WorkspaceConfig holds core workspace settings.
type WorkspaceConfig struct {
	Name    string `toml:"name"`
	Version int    `toml:"version"`
}

// WorktreesConfig configures git worktree management.
type WorktreesConfig struct {
	Path        string `toml:"path"`
	AutoCleanup bool   `toml:"auto_cleanup"`
}

// ToolsConfig configures available AI tools.
type ToolsConfig struct {
	Custom  map[string]ToolConfig `toml:"-"`
	Claude  *ToolConfig           `toml:"claude,omitempty"`
	Cursor  *ToolConfig           `toml:"cursor,omitempty"`
	Codex   *ToolConfig           `toml:"codex,omitempty"`
	Default string                `toml:"default"`
}

// ToolConfig defines a single tool's configuration.
type ToolConfig struct {
	Command string `toml:"command"`
	Enabled bool   `toml:"enabled"`
}

// MemoryConfig configures agent memory/context persistence.
type MemoryConfig struct {
	Backend string `toml:"backend"` // "file", "sqlite", etc.
	Path    string `toml:"path"`
}

// BeadsConfig configures beads issue tracker integration.
type BeadsConfig struct {
	IssuesDir string `toml:"issues_dir"`
	Enabled   bool   `toml:"enabled"`
}

// ChannelsConfig configures communication channels.
type ChannelsConfig struct {
	Default []string `toml:"default"`
}

// Validation errors.
var (
	ErrMissingWorkspaceName = errors.New("workspace.name is required")
	ErrInvalidVersion       = errors.New("workspace.version must be 2")
	ErrMissingDefaultTool   = errors.New("tools.default is required")
	ErrDefaultToolNotFound  = errors.New("tools.default references undefined tool")
	ErrMissingMemoryBackend = errors.New("memory.backend is required")
	ErrMissingMemoryPath    = errors.New("memory.path is required")
)

// DefaultV2Config returns sensible defaults for a new v2 workspace.
func DefaultV2Config(name string) V2Config {
	return V2Config{
		Workspace: WorkspaceConfig{
			Name:    name,
			Version: ConfigVersion,
		},
		Worktrees: WorktreesConfig{
			Path:        ".bc/worktrees",
			AutoCleanup: true,
		},
		Tools: ToolsConfig{
			Default: "claude",
			Claude: &ToolConfig{
				Command: "claude --dangerously-skip-permissions",
				Enabled: true,
			},
		},
		Memory: MemoryConfig{
			Backend: "file",
			Path:    ".bc/memory",
		},
		Beads: BeadsConfig{
			Enabled:   true,
			IssuesDir: ".beads/issues",
		},
		Channels: ChannelsConfig{
			Default: []string{"general", "engineering"},
		},
	}
}

// LoadV2Config reads and parses a TOML config file.
func LoadV2Config(path string) (*V2Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path provided by caller
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return ParseV2Config(data)
}

// ParseV2Config parses TOML data into a V2Config.
func ParseV2Config(data []byte) (*V2Config, error) {
	var cfg V2Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the config for required fields and consistency.
func (c *V2Config) Validate() error {
	// Workspace validation
	if c.Workspace.Name == "" {
		return ErrMissingWorkspaceName
	}
	if c.Workspace.Version != ConfigVersion {
		return ErrInvalidVersion
	}

	// Tools validation
	if c.Tools.Default == "" {
		return ErrMissingDefaultTool
	}
	if !c.hasToolDefined(c.Tools.Default) {
		return ErrDefaultToolNotFound
	}

	// Memory validation
	if c.Memory.Backend == "" {
		return ErrMissingMemoryBackend
	}
	if c.Memory.Path == "" {
		return ErrMissingMemoryPath
	}

	return nil
}

// hasToolDefined checks if a tool is configured.
func (c *V2Config) hasToolDefined(name string) bool {
	switch name {
	case "claude":
		return c.Tools.Claude != nil
	case "cursor":
		return c.Tools.Cursor != nil
	case "codex":
		return c.Tools.Codex != nil
	default:
		_, ok := c.Tools.Custom[name]
		return ok
	}
}

// GetTool returns the configuration for a named tool.
func (c *V2Config) GetTool(name string) *ToolConfig {
	switch name {
	case "claude":
		return c.Tools.Claude
	case "cursor":
		return c.Tools.Cursor
	case "codex":
		return c.Tools.Codex
	default:
		if cfg, ok := c.Tools.Custom[name]; ok {
			return &cfg
		}
		return nil
	}
}

// GetDefaultTool returns the default tool configuration.
func (c *V2Config) GetDefaultTool() *ToolConfig {
	return c.GetTool(c.Tools.Default)
}

// Save writes the config to a TOML file.
func (c *V2Config) Save(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(path) //nolint:gosec // path provided by caller
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	encoder := toml.NewEncoder(f)
	encodeErr := encoder.Encode(c)
	closeErr := f.Close()

	if encodeErr != nil {
		return fmt.Errorf("failed to encode config: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close config file: %w", closeErr)
	}

	return nil
}

// ConfigPath returns the standard config file path for a workspace root.
func ConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".bc", "config.toml")
}
