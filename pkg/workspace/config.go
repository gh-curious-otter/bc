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
	Workspace   WorkspaceConfig   `toml:"workspace"`
	Worktrees   WorktreesConfig   `toml:"worktrees"`
	Tools       ToolsConfig       `toml:"tools"`
	Memory      MemoryConfig      `toml:"memory"`
	Channels    ChannelsConfig    `toml:"channels"`
	Roster      RosterConfig      `toml:"roster"`
	Performance PerformanceConfig `toml:"performance"`
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

// ToolsConfig configures available AI tools and integrations.
type ToolsConfig struct {
	Custom  map[string]ToolConfig `toml:"-"`
	Claude  *ToolConfig           `toml:"claude,omitempty"`
	Cursor  *ToolConfig           `toml:"cursor,omitempty"`
	Codex   *ToolConfig           `toml:"codex,omitempty"`
	Gemini  *ToolConfig           `toml:"gemini,omitempty"`
	GitHub  *ToolConfig           `toml:"github,omitempty"`
	GitLab  *ToolConfig           `toml:"gitlab,omitempty"`
	Jira    *ToolConfig           `toml:"jira,omitempty"`
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

// ChannelsConfig configures communication channels.
type ChannelsConfig struct {
	Default []string `toml:"default"`
}

// RosterConfig configures the default agent roster for bc up.
type RosterConfig struct {
	ProductManager int `toml:"product_manager"` // Number of product-manager agents (default: 1)
	Manager        int `toml:"manager"`         // Number of manager agents (default: 1)
	Engineers      int `toml:"engineers"`       // Number of engineer agents (default: 4)
	TechLeads      int `toml:"tech_leads"`      // Number of tech-lead agents (default: 2)
	QA             int `toml:"qa"`              // Number of QA agents (default: 2)
}

// PerformanceConfig configures TUI polling intervals and cache TTLs.
// All values are in milliseconds. Minimum poll interval is 500ms.
type PerformanceConfig struct {
	// TUI polling intervals (min: 500ms)
	PollIntervalAgents   int64 `toml:"poll_interval_agents"`   // Agent status updates (default: 2000)
	PollIntervalChannels int64 `toml:"poll_interval_channels"` // Channel message polling (default: 3000)
	PollIntervalCosts    int64 `toml:"poll_interval_costs"`    // Cost data refresh (default: 5000)
	PollIntervalStatus   int64 `toml:"poll_interval_status"`   // Dashboard status (default: 2000)
	PollIntervalLogs     int64 `toml:"poll_interval_logs"`     // Log viewer refresh (default: 3000)
	PollIntervalTeams    int64 `toml:"poll_interval_teams"`    // Team data refresh (default: 10000)
	PollIntervalDemons   int64 `toml:"poll_interval_demons"`   // Scheduled tasks refresh (default: 5000)

	// Cache TTLs
	CacheTTLTmux     int64 `toml:"cache_ttl_tmux"`     // Tmux session state cache (default: 2000)
	CacheTTLCommands int64 `toml:"cache_ttl_commands"` // CLI command result cache (default: 5000)

	// Adaptive polling thresholds (for useAdaptivePolling hook)
	AdaptiveFastInterval   int64 `toml:"adaptive_fast_interval"`   // When agents are actively working (default: 1000)
	AdaptiveNormalInterval int64 `toml:"adaptive_normal_interval"` // Normal operation (default: 2000)
	AdaptiveSlowInterval   int64 `toml:"adaptive_slow_interval"`   // Low activity period (default: 4000)
	AdaptiveMaxInterval    int64 `toml:"adaptive_max_interval"`    // Maximum backoff interval (default: 8000)
}

// Roster limits.
const (
	RosterMinPerRole = 0  // Minimum agents per role
	RosterMaxPerRole = 10 // Maximum agents per role
)

// Performance limits.
const (
	PollIntervalMin = 500   // Minimum poll interval in ms
	CacheTTLMin     = 100   // Minimum cache TTL in ms
	CacheTTLMax     = 60000 // Maximum cache TTL in ms (1 minute)
)

// Validation errors.
var (
	ErrMissingWorkspaceName      = errors.New("workspace.name is required")
	ErrInvalidVersion            = errors.New("workspace.version must be 2")
	ErrMissingDefaultTool        = errors.New("tools.default is required")
	ErrDefaultToolNotFound       = errors.New("tools.default references undefined tool")
	ErrMissingMemoryBackend      = errors.New("memory.backend is required")
	ErrMissingMemoryPath         = errors.New("memory.path is required")
	ErrRosterProductManagerRange = errors.New("roster.product_manager must be between 0 and 10")
	ErrRosterManagerRange        = errors.New("roster.manager must be between 0 and 10")
	ErrRosterEngineersRange      = errors.New("roster.engineers must be between 0 and 10")
	ErrRosterTechLeadsRange      = errors.New("roster.tech_leads must be between 0 and 10")
	ErrRosterQARange             = errors.New("roster.qa must be between 0 and 10")
	ErrPollIntervalTooLow        = errors.New("poll intervals must be at least 500ms")
	ErrCacheTTLRange             = errors.New("cache TTL must be between 100ms and 60000ms")
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
			Default: "gemini",
			Claude: &ToolConfig{
				Command: "claude --dangerously-skip-permissions",
				Enabled: true,
			},
			Gemini: &ToolConfig{
				Command: "gemini --yolo",
				Enabled: true,
			},
		},
		Memory: MemoryConfig{
			Backend: "file",
			Path:    ".bc/memory",
		},
		Channels: ChannelsConfig{
			Default: []string{"general", "engineering"},
		},
		Roster: RosterConfig{
			ProductManager: 0,
			Manager:        0,
			Engineers:      0,
			TechLeads:      0,
			QA:             0,
		},
		Performance: PerformanceConfig{
			PollIntervalAgents:     2000,
			PollIntervalChannels:   3000,
			PollIntervalCosts:      5000,
			PollIntervalStatus:     2000,
			PollIntervalLogs:       3000,
			PollIntervalTeams:      10000,
			PollIntervalDemons:     5000,
			CacheTTLTmux:           2000,
			CacheTTLCommands:       5000,
			AdaptiveFastInterval:   1000,
			AdaptiveNormalInterval: 2000,
			AdaptiveSlowInterval:   4000,
			AdaptiveMaxInterval:    8000,
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

	// Roster validation
	if c.Roster.ProductManager < RosterMinPerRole || c.Roster.ProductManager > RosterMaxPerRole {
		return ErrRosterProductManagerRange
	}
	if c.Roster.Manager < RosterMinPerRole || c.Roster.Manager > RosterMaxPerRole {
		return ErrRosterManagerRange
	}
	if c.Roster.Engineers < RosterMinPerRole || c.Roster.Engineers > RosterMaxPerRole {
		return ErrRosterEngineersRange
	}
	if c.Roster.TechLeads < RosterMinPerRole || c.Roster.TechLeads > RosterMaxPerRole {
		return ErrRosterTechLeadsRange
	}
	if c.Roster.QA < RosterMinPerRole || c.Roster.QA > RosterMaxPerRole {
		return ErrRosterQARange
	}

	// Performance validation (only validate if non-zero values are set)
	if err := c.validatePerformance(); err != nil {
		return err
	}

	return nil
}

// validatePerformance validates performance config values.
// Zero values are acceptable (will use defaults).
func (c *V2Config) validatePerformance() error {
	p := c.Performance

	// Validate poll intervals (must be >= 500ms if set)
	pollIntervals := []int64{
		p.PollIntervalAgents, p.PollIntervalChannels, p.PollIntervalCosts,
		p.PollIntervalStatus, p.PollIntervalLogs, p.PollIntervalTeams,
		p.PollIntervalDemons,
	}
	for _, interval := range pollIntervals {
		if interval > 0 && interval < PollIntervalMin {
			return ErrPollIntervalTooLow
		}
	}

	// Validate adaptive intervals (must be >= 500ms if set)
	adaptiveIntervals := []int64{
		p.AdaptiveFastInterval, p.AdaptiveNormalInterval,
		p.AdaptiveSlowInterval, p.AdaptiveMaxInterval,
	}
	for _, interval := range adaptiveIntervals {
		if interval > 0 && interval < PollIntervalMin {
			return ErrPollIntervalTooLow
		}
	}

	// Validate cache TTLs
	cacheTTLs := []int64{p.CacheTTLTmux, p.CacheTTLCommands}
	for _, ttl := range cacheTTLs {
		if ttl > 0 && (ttl < CacheTTLMin || ttl > CacheTTLMax) {
			return ErrCacheTTLRange
		}
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
	case "gemini":
		return c.Tools.Gemini != nil
	case "github":
		return c.Tools.GitHub != nil
	case "gitlab":
		return c.Tools.GitLab != nil
	case "jira":
		return c.Tools.Jira != nil
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
	case "gemini":
		return c.Tools.Gemini
	case "github":
		return c.Tools.GitHub
	case "gitlab":
		return c.Tools.GitLab
	case "jira":
		return c.Tools.Jira
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
