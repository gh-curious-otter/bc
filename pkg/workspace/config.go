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

// Config represents the TOML-based workspace configuration for bc v2.
// Field order is optimized by fieldalignment for minimal struct padding.
type Config struct {
	Services    ServicesConfig    `toml:"services"`
	Providers   ProvidersConfig   `toml:"providers"`
	TUI         TUIConfig         `toml:"tui"`
	User        UserConfig        `toml:"user"`
	Workspace   WorkspaceConfig   `toml:"workspace"`
	Channels    ChannelsConfig    `toml:"channels"`
	Logs        LogsConfig        `toml:"logs"`
	Runtime     RuntimeConfig     `toml:"runtime"`
	Performance PerformanceConfig `toml:"performance"`
}

// RuntimeConfig configures the agent session backend.
type RuntimeConfig struct {
	Backend string              `toml:"backend"` // "tmux" or "docker" (default)
	Docker  DockerRuntimeConfig `toml:"docker"`
}

// DockerRuntimeConfig configures Docker container settings for agents.
type DockerRuntimeConfig struct {
	Image       string   `toml:"image"`
	Network     string   `toml:"network"`
	ExtraMounts []string `toml:"extra_mounts"`
	CPUs        float64  `toml:"cpus"`
	MemoryMB    int64    `toml:"memory_mb"`
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
	Path    string `toml:"path"`
	Version int    `toml:"version"`
}

// ProvidersConfig configures AI agent providers (Claude, Gemini, etc.).
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

// ChannelsConfig configures communication channels.
type ChannelsConfig struct{}

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

// Performance limits.
const (
	PollIntervalMin = 500   // Minimum poll interval in ms
	CacheTTLMin     = 100   // Minimum cache TTL in ms
	CacheTTLMax     = 60000 // Maximum cache TTL in ms (1 minute)
)

// Validation errors.
var (
	ErrMissingWorkspaceName  = errors.New("workspace.name is required")
	ErrInvalidVersion        = errors.New("workspace.version must be 2")
	ErrMissingDefaultProvider = errors.New("providers.default is required")
	ErrDefaultProviderNotFound = errors.New("providers.default references undefined provider")
	ErrPollIntervalTooLow    = errors.New("poll intervals must be at least 500ms")
	ErrCacheTTLRange         = errors.New("cache TTL must be between 100ms and 60000ms")
	ErrInvalidTheme          = errors.New("tui.theme must be one of: dark, light, matrix, synthwave, high-contrast")
	ErrInvalidThemeMode      = errors.New("tui.mode must be one of: auto, dark, light")
	ErrNicknameTooLong       = errors.New("user.nickname must be 15 characters or less")
	ErrNicknameMissingPrefix = errors.New("user.nickname must start with @")
	ErrNicknameInvalidChars  = errors.New("user.nickname must contain only letters, numbers, and underscores")
)

// DefaultConfig returns sensible defaults for a new v2 workspace.
func DefaultConfig(name string) Config {
	return Config{
		Workspace: WorkspaceConfig{
			Name:    name,
			Version: ConfigVersion,
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
		Logs: LogsConfig{
			Path:         ".bc/logs",
			MaxBytes:     1048576, // 1MB
			PreserveAnsi: true,
		},
		User: UserConfig{
			Nickname: DefaultNickname,
		},
		Runtime: RuntimeConfig{
			Backend: "docker",
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

// LoadConfig reads and parses a TOML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path provided by caller
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return ParseConfig(data)
}

// ParseConfig parses TOML data into a Config.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the config for required fields and consistency.
func (c *Config) Validate() error {
	// Workspace validation
	if c.Workspace.Name == "" {
		return ErrMissingWorkspaceName
	}
	if c.Workspace.Version != ConfigVersion {
		return ErrInvalidVersion
	}

	// Providers validation
	if c.Providers.Default == "" {
		return ErrMissingDefaultProvider
	}
	if !c.HasProviderDefined(c.Providers.Default) && !c.HasServiceDefined(c.Providers.Default) {
		return ErrDefaultProviderNotFound
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
func (c *Config) validatePerformance() error {
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
func (c *Config) validateTUI() error {
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
func (c *Config) validateUser() error {
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

// GetProvider returns an AI provider's configuration by name.
func (c *Config) GetProvider(name string) *ProviderConfig {
	switch name {
	case "claude":
		return c.Providers.Claude
	case "gemini":
		return c.Providers.Gemini
	case "cursor":
		return c.Providers.Cursor
	case "codex":
		return c.Providers.Codex
	case "opencode":
		return c.Providers.OpenCode
	case "openclaw":
		return c.Providers.OpenClaw
	case "aider":
		return c.Providers.Aider
	default:
		if cfg, ok := c.Providers.Custom[name]; ok {
			return &cfg
		}
	}
	return nil
}

// GetService returns an external service's configuration by name.
func (c *Config) GetService(name string) *ServiceConfig {
	switch name {
	case "github":
		return c.Services.GitHub
	case "gitlab":
		return c.Services.GitLab
	case "jira":
		return c.Services.Jira
	}
	return nil
}

// GetDefaultProvider returns the default AI provider name.
func (c *Config) GetDefaultProvider() string {
	return c.Providers.Default
}

// HasProviderDefined checks if an AI provider is configured.
func (c *Config) HasProviderDefined(name string) bool {
	return c.GetProvider(name) != nil
}

// HasServiceDefined checks if an external service is configured.
func (c *Config) HasServiceDefined(name string) bool {
	return c.GetService(name) != nil
}

// ListProviders returns the names of all enabled AI providers.
func (c *Config) ListProviders() []string {
	var names []string
	seen := make(map[string]bool)

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
		}
	}

	return names
}

// ListServices returns the names of all enabled external services.
func (c *Config) ListServices() []string {
	var names []string

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
		}
	}

	return names
}

// Save writes the config to a TOML file.
func (c *Config) Save(path string) error {
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

// MergeUserDefaults merges user defaults into a Config.
// Workspace config values take precedence over user defaults.
func MergeUserDefaults(cfg *Config, defaults *UserDefaultsConfig) {
	if defaults == nil {
		return
	}

	// Merge user nickname (only if workspace hasn't set one)
	if cfg.User.Nickname == "" && defaults.User.Nickname != "" {
		cfg.User.Nickname = defaults.User.Nickname
	}

	// Merge provider preference (only if workspace hasn't set a default)
	if cfg.Providers.Default == "" && len(defaults.Tools.Preferred) > 0 {
		cfg.Providers.Default = defaults.Tools.Preferred[0]
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
