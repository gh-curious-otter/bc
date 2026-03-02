// Package workspace provides workspace/project management.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigVersion is the current config schema version.
const ConfigVersion = 2

// V2Config represents the TOML-based workspace configuration for bc v2.
// Field order is optimized by fieldalignment for minimal struct padding.
type V2Config struct {
	Services    ServicesConfig    `toml:"services"`
	Providers   ProvidersConfig   `toml:"providers"`
	Tools       ToolsConfig       `toml:"tools"`
	Memory      MemoryConfig      `toml:"memory"`
	TUI         TUIConfig         `toml:"tui"`
	User        UserConfig        `toml:"user"`
	Workspace   WorkspaceConfig   `toml:"workspace"`
	Worktrees   WorktreesConfig   `toml:"worktrees"`
	Channels    ChannelsConfig    `toml:"channels"`
	Logs        LogsConfig        `toml:"logs"`
	Performance PerformanceConfig `toml:"performance"`
	Roster      RosterConfig      `toml:"roster"`
}

// LogsConfig configures persistent session log streaming.
type LogsConfig struct {
	Path         string `toml:"path"`
	MaxBytes     int64  `toml:"max_bytes"`
	PreserveAnsi bool   `toml:"preserve_ansi"`
}

// UserConfig holds user identity settings.
type UserConfig struct {
	Nickname string `toml:"nickname"` // User's display name for channel messages (e.g., "@puneet")
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
// DEPRECATED: Use ProvidersConfig and ServicesConfig instead.
// This is kept for backward compatibility with existing configs.
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

// ProvidersConfig configures AI agent providers (Claude, Gemini, etc.).
// Issue #1771: Separate AI providers from external service integrations.
type ProvidersConfig struct {
	Custom   map[string]ProviderConfig `toml:"-"`                  // Custom providers
	Claude   *ProviderConfig           `toml:"claude,omitempty"`   // Anthropic Claude Code
	Gemini   *ProviderConfig           `toml:"gemini,omitempty"`   // Google Gemini
	Cursor   *ProviderConfig           `toml:"cursor,omitempty"`   // Cursor Agent
	Codex    *ProviderConfig           `toml:"codex,omitempty"`    // OpenAI Codex
	OpenCode *ProviderConfig           `toml:"opencode,omitempty"` // OpenCode/Crush
	OpenClaw *ProviderConfig           `toml:"openclaw,omitempty"` // OpenClaw
	Aider    *ProviderConfig           `toml:"aider,omitempty"`    // Aider
	Default  string                    `toml:"default"`            // Default provider for new agents
}

// ProviderConfig defines an AI provider's configuration.
type ProviderConfig struct {
	Command string `toml:"command"`         // Command to launch the provider
	Model   string `toml:"model,omitempty"` // Default model (for API providers)
	Enabled bool   `toml:"enabled"`         // Whether the provider is enabled
}

// ServicesConfig configures external service integrations (GitHub, GitLab, etc.).
// Issue #1771: Separate external services from AI providers.
type ServicesConfig struct {
	GitHub *ServiceConfig `toml:"github,omitempty"` // GitHub CLI integration
	GitLab *ServiceConfig `toml:"gitlab,omitempty"` // GitLab CLI integration
	Jira   *ServiceConfig `toml:"jira,omitempty"`   // Jira CLI integration
}

// ServiceConfig defines an external service integration.
type ServiceConfig struct {
	Command   string `toml:"command"`              // Command to execute (e.g., "gh")
	TokenEnv  string `toml:"token_env,omitempty"`  // Environment variable for auth token
	RateLimit int    `toml:"rate_limit,omitempty"` // Requests per hour (0 = unlimited)
	Enabled   bool   `toml:"enabled"`              // Whether the service is enabled
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

// TUIConfig configures TUI appearance and theming.
type TUIConfig struct {
	Theme string `toml:"theme"` // Theme name: "dark", "light", "matrix", "synthwave", "high-contrast"
	Mode  string `toml:"mode"`  // Color mode: "auto", "dark", "light"
}

// Valid theme names.
var ValidThemes = []string{"dark", "light", "matrix", "synthwave", "high-contrast"}

// Valid theme modes.
var ValidModes = []string{"auto", "dark", "light"}

// User nickname limits.
const (
	NicknameMaxLength = 15 // Maximum nickname length including @ prefix
)

// DefaultNickname is the default user nickname.
const DefaultNickname = "@bc"

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
	ErrInvalidTheme              = errors.New("tui.theme must be one of: dark, light, matrix, synthwave, high-contrast")
	ErrInvalidThemeMode          = errors.New("tui.mode must be one of: auto, dark, light")
	ErrNicknameTooLong           = errors.New("user.nickname must be 15 characters or less")
	ErrNicknameMissingPrefix     = errors.New("user.nickname must start with @")
	ErrNicknameInvalidChars      = errors.New("user.nickname must contain only letters, numbers, and underscores")
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
		Providers: ProvidersConfig{
			Default: "gemini",
			Claude: &ProviderConfig{
				Command: "claude --dangerously-skip-permissions",
				Enabled: true,
			},
			Gemini: &ProviderConfig{
				Command: "gemini --yolo",
				Enabled: true,
			},
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
		Logs: LogsConfig{
			Path:         ".bc/logs",
			MaxBytes:     1048576, // 1MB
			PreserveAnsi: true,
		},
		Memory: MemoryConfig{
			Backend: "file",
			Path:    ".bc/memory",
		},
		Channels: ChannelsConfig{
			Default: []string{"general", "engineering"},
		},
		User: UserConfig{
			Nickname: DefaultNickname,
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
		TUI: TUIConfig{
			Theme: "dark",
			Mode:  "auto",
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

	// TUI validation (only validate if non-empty values are set)
	if err := c.validateTUI(); err != nil {
		return err
	}

	// User validation (only validate if nickname is set)
	if err := c.validateUser(); err != nil {
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

// validateTUI validates TUI config values.
// Empty values are acceptable (will use defaults).
func (c *V2Config) validateTUI() error {
	t := c.TUI

	// Validate theme (must be one of valid themes if set)
	if t.Theme != "" && !isValidTheme(t.Theme) {
		return ErrInvalidTheme
	}

	// Validate mode (must be one of valid modes if set)
	if t.Mode != "" && !isValidMode(t.Mode) {
		return ErrInvalidThemeMode
	}

	return nil
}

// isValidTheme checks if a theme name is valid.
func isValidTheme(theme string) bool {
	for _, valid := range ValidThemes {
		if theme == valid {
			return true
		}
	}
	return false
}

// isValidMode checks if a theme mode is valid.
func isValidMode(mode string) bool {
	for _, valid := range ValidModes {
		if mode == valid {
			return true
		}
	}
	return false
}

// validateUser validates user config values.
// Empty values are acceptable (will use default @bc).
func (c *V2Config) validateUser() error {
	u := c.User

	// Empty nickname is ok (uses default)
	if u.Nickname == "" {
		return nil
	}

	// Validate nickname format
	return ValidateNickname(u.Nickname)
}

// nicknameRegex matches valid nickname characters (after @ prefix).
var nicknameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// ValidateNickname validates a nickname and returns an error if invalid.
func ValidateNickname(nickname string) error {
	// Must start with @
	if !strings.HasPrefix(nickname, "@") {
		return ErrNicknameMissingPrefix
	}

	// Check length
	if len(nickname) > NicknameMaxLength {
		return ErrNicknameTooLong
	}

	// Check characters after @
	body := nickname[1:]
	if body == "" || !nicknameRegex.MatchString(body) {
		return ErrNicknameInvalidChars
	}

	return nil
}

// NormalizeNickname ensures a nickname has the @ prefix and is valid.
// Returns the normalized nickname or an error.
func NormalizeNickname(nickname string) (string, error) {
	// Trim whitespace
	nickname = strings.TrimSpace(nickname)

	// Empty means use default
	if nickname == "" {
		return DefaultNickname, nil
	}

	// Add @ prefix if missing
	if !strings.HasPrefix(nickname, "@") {
		nickname = "@" + nickname
	}

	// Validate
	if err := ValidateNickname(nickname); err != nil {
		return "", err
	}

	return nickname, nil
}

// hasToolDefined checks if a tool is configured (either in new or legacy config).
// Issue #1771: Updated to check both Providers/Services and legacy Tools.
func (c *V2Config) hasToolDefined(name string) bool {
	// Check if it's an AI provider
	if c.HasProviderDefined(name) {
		return true
	}
	// Check if it's an external service
	if c.HasServiceDefined(name) {
		return true
	}
	// Fall back to legacy Tools check
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
// DEPRECATED: Use GetProvider or GetService instead.
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

// GetProvider returns an AI provider's configuration by name.
// Falls back to legacy Tools config if new Providers section is not defined.
// Issue #1771: New method for cleaner provider access.
func (c *V2Config) GetProvider(name string) *ProviderConfig {
	// Try new Providers config first
	switch name {
	case "claude":
		if c.Providers.Claude != nil {
			return c.Providers.Claude
		}
		// Fall back to legacy Tools config
		if c.Tools.Claude != nil {
			return &ProviderConfig{Command: c.Tools.Claude.Command, Enabled: c.Tools.Claude.Enabled}
		}
	case "gemini":
		if c.Providers.Gemini != nil {
			return c.Providers.Gemini
		}
		if c.Tools.Gemini != nil {
			return &ProviderConfig{Command: c.Tools.Gemini.Command, Enabled: c.Tools.Gemini.Enabled}
		}
	case "cursor":
		if c.Providers.Cursor != nil {
			return c.Providers.Cursor
		}
		if c.Tools.Cursor != nil {
			return &ProviderConfig{Command: c.Tools.Cursor.Command, Enabled: c.Tools.Cursor.Enabled}
		}
	case "codex":
		if c.Providers.Codex != nil {
			return c.Providers.Codex
		}
		if c.Tools.Codex != nil {
			return &ProviderConfig{Command: c.Tools.Codex.Command, Enabled: c.Tools.Codex.Enabled}
		}
	case "opencode":
		if c.Providers.OpenCode != nil {
			return c.Providers.OpenCode
		}
	case "openclaw":
		if c.Providers.OpenClaw != nil {
			return c.Providers.OpenClaw
		}
	case "aider":
		if c.Providers.Aider != nil {
			return c.Providers.Aider
		}
	default:
		if cfg, ok := c.Providers.Custom[name]; ok {
			return &cfg
		}
	}
	return nil
}

// GetService returns an external service's configuration by name.
// Falls back to legacy Tools config if new Services section is not defined.
// Issue #1771: New method for cleaner service access.
func (c *V2Config) GetService(name string) *ServiceConfig {
	// Try new Services config first
	switch name {
	case "github":
		if c.Services.GitHub != nil {
			return c.Services.GitHub
		}
		// Fall back to legacy Tools config
		if c.Tools.GitHub != nil {
			return &ServiceConfig{Command: c.Tools.GitHub.Command, Enabled: c.Tools.GitHub.Enabled}
		}
	case "gitlab":
		if c.Services.GitLab != nil {
			return c.Services.GitLab
		}
		if c.Tools.GitLab != nil {
			return &ServiceConfig{Command: c.Tools.GitLab.Command, Enabled: c.Tools.GitLab.Enabled}
		}
	case "jira":
		if c.Services.Jira != nil {
			return c.Services.Jira
		}
		if c.Tools.Jira != nil {
			return &ServiceConfig{Command: c.Tools.Jira.Command, Enabled: c.Tools.Jira.Enabled}
		}
	}
	return nil
}

// GetDefaultProvider returns the default AI provider name.
// Falls back to legacy Tools.Default if new Providers.Default is not set.
func (c *V2Config) GetDefaultProvider() string {
	if c.Providers.Default != "" {
		return c.Providers.Default
	}
	return c.Tools.Default
}

// HasProviderDefined checks if an AI provider is configured.
func (c *V2Config) HasProviderDefined(name string) bool {
	return c.GetProvider(name) != nil
}

// HasServiceDefined checks if an external service is configured.
func (c *V2Config) HasServiceDefined(name string) bool {
	return c.GetService(name) != nil
}

// ListProviders returns the names of all enabled AI providers.
// Checks ProvidersConfig first, then falls back to legacy ToolsConfig.
func (c *V2Config) ListProviders() []string {
	seen := make(map[string]bool)
	var names []string

	// Check new ProvidersConfig
	providerFields := []struct {
		cfg  *ProviderConfig
		name string
	}{
		{c.Providers.Claude, "claude"},
		{c.Providers.Gemini, "gemini"},
		{c.Providers.Cursor, "cursor"},
		{c.Providers.Codex, "codex"},
		{c.Providers.OpenCode, "opencode"},
		{c.Providers.OpenClaw, "openclaw"},
		{c.Providers.Aider, "aider"},
	}
	for _, pf := range providerFields {
		if pf.cfg != nil && pf.cfg.Enabled {
			names = append(names, pf.name)
			seen[pf.name] = true
		}
	}
	for name, cfg := range c.Providers.Custom {
		if cfg.Enabled && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}

	// Fall back to legacy ToolsConfig for providers not already found
	legacyProviders := []struct {
		cfg  *ToolConfig
		name string
	}{
		{c.Tools.Claude, "claude"},
		{c.Tools.Gemini, "gemini"},
		{c.Tools.Cursor, "cursor"},
		{c.Tools.Codex, "codex"},
	}
	for _, lp := range legacyProviders {
		if lp.cfg != nil && lp.cfg.Enabled && !seen[lp.name] {
			names = append(names, lp.name)
			seen[lp.name] = true
		}
	}

	return names
}

// ListServices returns the names of all enabled external services.
// Checks ServicesConfig first, then falls back to legacy ToolsConfig.
func (c *V2Config) ListServices() []string {
	seen := make(map[string]bool)
	var names []string

	// Check new ServicesConfig
	serviceFields := []struct {
		cfg  *ServiceConfig
		name string
	}{
		{c.Services.GitHub, "github"},
		{c.Services.GitLab, "gitlab"},
		{c.Services.Jira, "jira"},
	}
	for _, sf := range serviceFields {
		if sf.cfg != nil && sf.cfg.Enabled {
			names = append(names, sf.name)
			seen[sf.name] = true
		}
	}

	// Fall back to legacy ToolsConfig for services not already found
	legacyServices := []struct {
		cfg  *ToolConfig
		name string
	}{
		{c.Tools.GitHub, "github"},
		{c.Tools.GitLab, "gitlab"},
		{c.Tools.Jira, "jira"},
	}
	for _, ls := range legacyServices {
		if ls.cfg != nil && ls.cfg.Enabled && !seen[ls.name] {
			names = append(names, ls.name)
		}
	}

	return names
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

// UserDefaultsConfig represents user-level defaults from ~/.bcrc.
// These settings are merged with workspace config, with workspace taking precedence.
type UserDefaultsConfig struct {
	User     UserDefaultsUser     `toml:"user"`
	Defaults UserDefaultsDefaults `toml:"defaults"`
	Tools    UserDefaultsTools    `toml:"tools"`
}

// UserDefaultsUser holds user identity in .bcrc.
type UserDefaultsUser struct {
	Nickname string `toml:"nickname"` // Default nickname (e.g., "@alice")
}

// UserDefaultsDefaults holds behavior defaults in .bcrc.
type UserDefaultsDefaults struct {
	DefaultRole   string `toml:"default_role"`    // Default role for new agents
	AutoStartRoot bool   `toml:"auto_start_root"` // Auto-start root agent with bc up
}

// UserDefaultsTools holds tool preferences in .bcrc.
type UserDefaultsTools struct {
	Preferred []string `toml:"preferred"` // Preferred tools in order
}

// UserDefaultsPath returns the path to the user's .bcrc file.
func UserDefaultsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bcrc")
}

// LoadUserDefaults loads the user's ~/.bcrc file if it exists.
// Returns nil if the file doesn't exist (not an error).
func LoadUserDefaults() (*UserDefaultsConfig, error) {
	path := UserDefaultsPath()
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // user home directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg UserDefaultsConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &cfg, nil
}

// MergeUserDefaults merges user defaults into a V2Config.
// Workspace config values take precedence over user defaults.
func MergeUserDefaults(cfg *V2Config, defaults *UserDefaultsConfig) {
	if defaults == nil {
		return
	}

	// Merge user nickname (only if workspace hasn't set one)
	if cfg.User.Nickname == "" && defaults.User.Nickname != "" {
		cfg.User.Nickname = defaults.User.Nickname
	}

	// Merge tool preference (only if workspace hasn't set a default)
	if cfg.Tools.Default == "" && len(defaults.Tools.Preferred) > 0 {
		cfg.Tools.Default = defaults.Tools.Preferred[0]
	}
}

// SaveUserDefaults saves user defaults to ~/.bcrc.
func SaveUserDefaults(defaults *UserDefaultsConfig) error {
	path := UserDefaultsPath()
	if path == "" {
		return errors.New("unable to determine home directory")
	}

	f, err := os.Create(path) //nolint:gosec // user home directory
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}

	encoder := toml.NewEncoder(f)
	encodeErr := encoder.Encode(defaults)
	closeErr := f.Close()

	if encodeErr != nil {
		return fmt.Errorf("failed to encode %s: %w", path, encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close %s: %w", path, closeErr)
	}

	return nil
}
