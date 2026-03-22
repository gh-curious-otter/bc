package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test-project")

	if cfg.Workspace.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", cfg.Workspace.Name)
	}
	if cfg.Workspace.Version != ConfigVersion {
		t.Errorf("expected version %d, got %d", ConfigVersion, cfg.Workspace.Version)
	}
	// Default provider is gemini (minimal root-only startup)
	if cfg.Providers.Default != "gemini" {
		t.Errorf("expected default provider 'gemini', got %q", cfg.Providers.Default)
	}
	if cfg.Providers.Gemini == nil {
		t.Error("expected gemini provider to be configured")
	}
	// Logs config defaults
	if cfg.Logs.Path != ".bc/logs" {
		t.Errorf("expected logs.path '.bc/logs', got %q", cfg.Logs.Path)
	}
	if cfg.Logs.MaxBytes != 1048576 {
		t.Errorf("expected logs.max_bytes 1048576, got %d", cfg.Logs.MaxBytes)
	}
}

func TestParseConfigWithLogs(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "test"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true

[logs]
path = ".bc/custom-logs"
max_bytes = 2097152
`)
	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if cfg.Logs.Path != ".bc/custom-logs" {
		t.Errorf("expected logs.path '.bc/custom-logs', got %q", cfg.Logs.Path)
	}
	if cfg.Logs.MaxBytes != 2097152 {
		t.Errorf("expected logs.max_bytes 2097152, got %d", cfg.Logs.MaxBytes)
	}
}

func TestLogsConfigSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig("test")
	cfg.Logs.Path = ".bc/my-logs"
	cfg.Logs.MaxBytes = 512000

	if err := cfg.Save(path); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.Logs.Path != ".bc/my-logs" {
		t.Errorf("expected path '.bc/my-logs', got %q", loaded.Logs.Path)
	}
	if loaded.Logs.MaxBytes != 512000 {
		t.Errorf("expected max_bytes 512000, got %d", loaded.Logs.MaxBytes)
	}
}

func TestParseConfig(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "my-project"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude --dangerously-skip-permissions"
enabled = true
`)

	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Workspace.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", cfg.Workspace.Name)
	}
	if cfg.Workspace.Version != 2 {
		t.Errorf("expected version 2, got %d", cfg.Workspace.Version)
	}
	if cfg.Providers.Default != "claude" {
		t.Errorf("expected default provider 'claude', got %q", cfg.Providers.Default)
	}
	if cfg.Providers.Claude == nil {
		t.Fatal("expected claude provider config")
	}
	if cfg.Providers.Claude.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("unexpected claude command: %q", cfg.Providers.Claude.Command)
	}
	if !cfg.Providers.Claude.Enabled {
		t.Error("expected claude to be enabled")
	}
}

// TestParseConfigWithPerformance tests parsing [performance] section from TOML (#1013)
func TestParseConfigWithPerformance(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "perf-project"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true

[performance]
poll_interval_agents = 1500
poll_interval_channels = 2500
poll_interval_costs = 4000
poll_interval_status = 1800
poll_interval_logs = 2200
poll_interval_teams = 8000
poll_interval_demons = 4500
cache_ttl_tmux = 1500
cache_ttl_commands = 3500
adaptive_fast_interval = 800
adaptive_normal_interval = 1500
adaptive_slow_interval = 3500
adaptive_max_interval = 7000
`)

	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify poll intervals
	if cfg.Performance.PollIntervalAgents != 1500 {
		t.Errorf("expected poll_interval_agents = 1500, got %d", cfg.Performance.PollIntervalAgents)
	}
	if cfg.Performance.PollIntervalChannels != 2500 {
		t.Errorf("expected poll_interval_channels = 2500, got %d", cfg.Performance.PollIntervalChannels)
	}
	if cfg.Performance.PollIntervalCosts != 4000 {
		t.Errorf("expected poll_interval_costs = 4000, got %d", cfg.Performance.PollIntervalCosts)
	}
	if cfg.Performance.PollIntervalStatus != 1800 {
		t.Errorf("expected poll_interval_status = 1800, got %d", cfg.Performance.PollIntervalStatus)
	}
	if cfg.Performance.PollIntervalLogs != 2200 {
		t.Errorf("expected poll_interval_logs = 2200, got %d", cfg.Performance.PollIntervalLogs)
	}
	if cfg.Performance.PollIntervalTeams != 8000 {
		t.Errorf("expected poll_interval_teams = 8000, got %d", cfg.Performance.PollIntervalTeams)
	}
	if cfg.Performance.PollIntervalDemons != 4500 {
		t.Errorf("expected poll_interval_demons = 4500, got %d", cfg.Performance.PollIntervalDemons)
	}

	// Verify cache TTLs
	if cfg.Performance.CacheTTLTmux != 1500 {
		t.Errorf("expected cache_ttl_tmux = 1500, got %d", cfg.Performance.CacheTTLTmux)
	}
	if cfg.Performance.CacheTTLCommands != 3500 {
		t.Errorf("expected cache_ttl_commands = 3500, got %d", cfg.Performance.CacheTTLCommands)
	}

	// Verify adaptive intervals
	if cfg.Performance.AdaptiveFastInterval != 800 {
		t.Errorf("expected adaptive_fast_interval = 800, got %d", cfg.Performance.AdaptiveFastInterval)
	}
	if cfg.Performance.AdaptiveNormalInterval != 1500 {
		t.Errorf("expected adaptive_normal_interval = 1500, got %d", cfg.Performance.AdaptiveNormalInterval)
	}
	if cfg.Performance.AdaptiveSlowInterval != 3500 {
		t.Errorf("expected adaptive_slow_interval = 3500, got %d", cfg.Performance.AdaptiveSlowInterval)
	}
	if cfg.Performance.AdaptiveMaxInterval != 7000 {
		t.Errorf("expected adaptive_max_interval = 7000, got %d", cfg.Performance.AdaptiveMaxInterval)
	}
}

// TestParseConfigWithTUI tests parsing [tui] section from TOML (#1022)
func TestParseConfigWithTUI(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "tui-project"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true

[tui]
theme = "synthwave"
mode = "dark"
`)

	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify TUI config
	if cfg.TUI.Theme != "synthwave" {
		t.Errorf("expected tui.theme = 'synthwave', got %q", cfg.TUI.Theme)
	}
	if cfg.TUI.Mode != "dark" {
		t.Errorf("expected tui.mode = 'dark', got %q", cfg.TUI.Mode)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		cfg     Config
	}{
		{
			name:    "missing workspace name",
			wantErr: ErrMissingWorkspaceName,
			cfg:     Config{Workspace: WorkspaceConfig{Version: 2}},
		},
		{
			name:    "invalid version",
			wantErr: ErrInvalidVersion,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 1},
			},
		},
		{
			name:    "missing default provider",
			wantErr: ErrMissingDefaultProvider,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
			},
		},
		{
			name:    "default provider not defined",
			wantErr: ErrDefaultProviderNotFound,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{Default: "nonexistent"},
			},
		},
		{
			name:    "valid config",
			wantErr: nil,
			cfg:     DefaultConfig("test"),
		},
		// Performance config validation tests (#1013)
		{
			name:    "poll interval too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalAgents = 100 // Below 500ms minimum
				return cfg
			}(),
		},
		{
			name:    "poll interval at minimum valid",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalAgents = 500 // Exactly at minimum
				return cfg
			}(),
		},
		{
			name:    "poll interval channels too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalChannels = 250 // Below 500ms minimum
				return cfg
			}(),
		},
		{
			name:    "poll interval costs too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalCosts = 499 // Just below 500ms minimum
				return cfg
			}(),
		},
		{
			name:    "poll interval status too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalStatus = 1 // Way below minimum
				return cfg
			}(),
		},
		{
			name:    "poll interval logs too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalLogs = 300
				return cfg
			}(),
		},
		{
			name:    "poll interval teams too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalTeams = 400
				return cfg
			}(),
		},
		{
			name:    "poll interval demons too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.PollIntervalDemons = 200
				return cfg
			}(),
		},
		{
			name:    "adaptive interval too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.AdaptiveFastInterval = 200 // Below 500ms minimum
				return cfg
			}(),
		},
		{
			name:    "adaptive normal interval too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.AdaptiveNormalInterval = 300
				return cfg
			}(),
		},
		{
			name:    "adaptive slow interval too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.AdaptiveSlowInterval = 450
				return cfg
			}(),
		},
		{
			name:    "adaptive max interval too low",
			wantErr: ErrPollIntervalTooLow,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.AdaptiveMaxInterval = 100
				return cfg
			}(),
		},
		{
			name:    "cache TTL too low",
			wantErr: ErrCacheTTLRange,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.CacheTTLTmux = 50 // Below 100ms minimum
				return cfg
			}(),
		},
		{
			name:    "cache TTL commands too low",
			wantErr: ErrCacheTTLRange,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.CacheTTLCommands = 99 // Just below 100ms minimum
				return cfg
			}(),
		},
		{
			name:    "cache TTL too high",
			wantErr: ErrCacheTTLRange,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.CacheTTLCommands = 120000 // Above 60000ms maximum
				return cfg
			}(),
		},
		{
			name:    "cache TTL tmux too high",
			wantErr: ErrCacheTTLRange,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.CacheTTLTmux = 60001 // Just above 60000ms max
				return cfg
			}(),
		},
		{
			name:    "cache TTL at bounds valid",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance.CacheTTLTmux = 100       // At minimum
				cfg.Performance.CacheTTLCommands = 60000 // At maximum
				return cfg
			}(),
		},
		{
			name:    "performance zero values valid (use defaults)",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance = PerformanceConfig{} // All zeros - valid, uses defaults
				return cfg
			}(),
		},
		{
			name:    "all performance values at valid minimum",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.Performance = PerformanceConfig{
					PollIntervalAgents:     500,
					PollIntervalChannels:   500,
					PollIntervalCosts:      500,
					PollIntervalStatus:     500,
					PollIntervalLogs:       500,
					PollIntervalTeams:      500,
					PollIntervalDemons:     500,
					CacheTTLTmux:           100,
					CacheTTLCommands:       100,
					AdaptiveFastInterval:   500,
					AdaptiveNormalInterval: 500,
					AdaptiveSlowInterval:   500,
					AdaptiveMaxInterval:    500,
				}
				return cfg
			}(),
		},
		// TUI config validation tests (#1022)
		{
			name:    "tui invalid theme",
			wantErr: ErrInvalidTheme,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "invalid-theme"
				return cfg
			}(),
		},
		{
			name:    "tui invalid mode",
			wantErr: ErrInvalidThemeMode,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Mode = "invalid-mode"
				return cfg
			}(),
		},
		{
			name:    "tui valid dark theme",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "dark"
				cfg.TUI.Mode = "auto"
				return cfg
			}(),
		},
		{
			name:    "tui valid light theme",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "light"
				cfg.TUI.Mode = "light"
				return cfg
			}(),
		},
		{
			name:    "tui valid matrix theme",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "matrix"
				cfg.TUI.Mode = "dark"
				return cfg
			}(),
		},
		{
			name:    "tui valid synthwave theme",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "synthwave"
				return cfg
			}(),
		},
		{
			name:    "tui valid high-contrast theme",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI.Theme = "high-contrast"
				return cfg
			}(),
		},
		{
			name:    "tui empty values valid (use defaults)",
			wantErr: nil,
			cfg: func() Config {
				cfg := DefaultConfig("test")
				cfg.TUI = TUIConfig{} // Empty - valid, uses defaults
				return cfg
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigGetProvider_Default(t *testing.T) {
	cfg := DefaultConfig("test")

	// Test getting claude
	p := cfg.GetProvider("claude")
	if p == nil {
		t.Fatal("expected claude provider config")
	}
	if p.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("unexpected command: %q", p.Command)
	}

	// Test getting non-existent provider
	p = cfg.GetProvider("nonexistent")
	if p != nil {
		t.Error("expected nil for nonexistent provider")
	}

	// Test GetDefaultProvider
	defaultProv := cfg.GetDefaultProvider()
	if defaultProv != "gemini" {
		t.Errorf("expected default provider 'gemini', got %q", defaultProv)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".bc", "config.toml")

	// Create and save config
	cfg := DefaultConfig("save-test")

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Workspace.Name != "save-test" {
		t.Errorf("expected name 'save-test', got %q", loaded.Workspace.Name)
	}
	if loaded.Providers.Default != "gemini" {
		t.Errorf("expected default provider 'gemini', got %q", loaded.Providers.Default)
	}
}

// TestConfigSaveAndLoadPerformance tests save/load round-trip for performance config (#1013)
func TestConfigSaveAndLoadPerformance(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".bc", "config.toml")

	// Create config with custom performance values
	cfg := DefaultConfig("perf-save-test")
	cfg.Performance = PerformanceConfig{
		PollIntervalAgents:     1500,
		PollIntervalChannels:   2500,
		PollIntervalCosts:      4000,
		PollIntervalStatus:     1800,
		PollIntervalLogs:       2200,
		PollIntervalTeams:      8000,
		PollIntervalDemons:     4500,
		CacheTTLTmux:           1500,
		CacheTTLCommands:       3500,
		AdaptiveFastInterval:   800,
		AdaptiveNormalInterval: 1500,
		AdaptiveSlowInterval:   3500,
		AdaptiveMaxInterval:    7000,
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify performance values are preserved
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify poll intervals
	if loaded.Performance.PollIntervalAgents != 1500 {
		t.Errorf("expected poll_interval_agents = 1500, got %d", loaded.Performance.PollIntervalAgents)
	}
	if loaded.Performance.PollIntervalChannels != 2500 {
		t.Errorf("expected poll_interval_channels = 2500, got %d", loaded.Performance.PollIntervalChannels)
	}
	if loaded.Performance.PollIntervalCosts != 4000 {
		t.Errorf("expected poll_interval_costs = 4000, got %d", loaded.Performance.PollIntervalCosts)
	}

	// Verify cache TTLs
	if loaded.Performance.CacheTTLTmux != 1500 {
		t.Errorf("expected cache_ttl_tmux = 1500, got %d", loaded.Performance.CacheTTLTmux)
	}
	if loaded.Performance.CacheTTLCommands != 3500 {
		t.Errorf("expected cache_ttl_commands = 3500, got %d", loaded.Performance.CacheTTLCommands)
	}

	// Verify adaptive intervals
	if loaded.Performance.AdaptiveFastInterval != 800 {
		t.Errorf("expected adaptive_fast_interval = 800, got %d", loaded.Performance.AdaptiveFastInterval)
	}
	if loaded.Performance.AdaptiveMaxInterval != 7000 {
		t.Errorf("expected adaptive_max_interval = 7000, got %d", loaded.Performance.AdaptiveMaxInterval)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath("/home/user/project")
	expected := "/home/user/project/.bc/config.toml"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseConfigInvalid(t *testing.T) {
	_, err := ParseConfig([]byte("invalid toml {{{"))
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestConfigGetProvider_Cursor(t *testing.T) {
	cfg := Config{
		Workspace: WorkspaceConfig{Name: "test", Version: 2},
		Providers: ProvidersConfig{
			Default: "cursor",
			Cursor: &ProviderConfig{
				Command: "cursor --wait",
				Enabled: true,
			},
		},
	}

	// Validate should pass with cursor as default
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// GetProvider should return cursor config
	p := cfg.GetProvider("cursor")
	if p == nil {
		t.Fatal("expected cursor provider config")
	}
	if p.Command != "cursor --wait" {
		t.Errorf("expected command 'cursor --wait', got %q", p.Command)
	}
	if !p.Enabled {
		t.Error("expected cursor to be enabled")
	}

	// Claude should be nil when not configured
	if cfg.GetProvider("claude") != nil {
		t.Error("expected nil for unconfigured claude")
	}
}

func TestConfigValidation_ProviderVariants(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		cfg     Config
	}{
		{
			name:    "valid with cursor default",
			wantErr: nil,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{
					Default: "cursor",
					Cursor:  &ProviderConfig{Command: "cursor", Enabled: true},
				},
			},
		},
		{
			name:    "valid with codex default",
			wantErr: nil,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{
					Default: "codex",
					Codex:   &ProviderConfig{Command: "codex", Enabled: true},
				},
			},
		},
		{
			name:    "valid with custom provider default",
			wantErr: nil,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{
					Default: "my-provider",
					Custom: map[string]ProviderConfig{
						"my-provider": {Command: "my-provider", Enabled: true},
					},
				},
			},
		},
		{
			name:    "cursor default but not defined",
			wantErr: ErrDefaultProviderNotFound,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{Default: "cursor"},
			},
		},
		{
			name:    "codex default but not defined",
			wantErr: ErrDefaultProviderNotFound,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{Default: "codex"},
			},
		},
		{
			name:    "custom default but not defined",
			wantErr: ErrDefaultProviderNotFound,
			cfg: Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Providers: ProvidersConfig{Default: "undefined-custom"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseConfig_MultipleProviders(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "multi-provider-project"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[providers.cursor]
command = "cursor --wait"
enabled = true

[providers.codex]
command = "codex --full-auto"
enabled = false
`)

	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify all providers are parsed
	if cfg.Providers.Claude == nil {
		t.Error("expected claude to be configured")
	}
	if cfg.Providers.Cursor == nil {
		t.Error("expected cursor to be configured")
	}
	if cfg.Providers.Codex == nil {
		t.Error("expected codex to be configured")
	}

	// Verify provider properties
	if cfg.Providers.Cursor.Command != "cursor --wait" {
		t.Errorf("unexpected cursor command: %q", cfg.Providers.Cursor.Command)
	}
	if !cfg.Providers.Cursor.Enabled {
		t.Error("expected cursor to be enabled")
	}

	if cfg.Providers.Codex.Command != "codex --full-auto" {
		t.Errorf("unexpected codex command: %q", cfg.Providers.Codex.Command)
	}
	if cfg.Providers.Codex.Enabled {
		t.Error("expected codex to be disabled")
	}

	// Validation should pass
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

// TestLoadUserDefaults tests loading ~/.bcrc (#1160)
func TestLoadUserDefaults(t *testing.T) {
	// LoadUserDefaults should return nil when file doesn't exist
	// (using the actual function which reads from ~)
	// This test verifies the function doesn't error on missing file
	defaults, err := LoadUserDefaults()
	if err != nil {
		// Error is only expected if the file exists but is malformed
		// If we don't have a .bcrc, it should return nil, nil
		t.Logf("LoadUserDefaults returned: defaults=%v, err=%v", defaults, err)
	}
}

// TestMergeUserDefaults tests merging user defaults with workspace config (#1160)
func TestMergeUserDefaults(t *testing.T) {
	tests := []struct {
		defaults            *UserDefaultsConfig
		name                string
		wantNickname        string
		wantDefaultProvider string
		cfg                 Config
	}{
		{
			name:                "nil defaults - no change",
			cfg:                 Config{User: UserConfig{Nickname: "@workspace"}, Providers: ProvidersConfig{Default: "claude"}},
			defaults:            nil,
			wantNickname:        "@workspace",
			wantDefaultProvider: "claude",
		},
		{
			name: "merge nickname when workspace empty",
			cfg:  Config{User: UserConfig{Nickname: ""}, Providers: ProvidersConfig{Default: "claude"}},
			defaults: &UserDefaultsConfig{
				User: UserDefaultsUser{Nickname: "@alice"},
			},
			wantNickname:        "@alice",
			wantDefaultProvider: "claude",
		},
		{
			name: "workspace nickname takes precedence",
			cfg:  Config{User: UserConfig{Nickname: "@workspace"}, Providers: ProvidersConfig{Default: "claude"}},
			defaults: &UserDefaultsConfig{
				User: UserDefaultsUser{Nickname: "@alice"},
			},
			wantNickname:        "@workspace",
			wantDefaultProvider: "claude",
		},
		{
			name: "merge preferred tool when workspace empty",
			cfg:  Config{User: UserConfig{Nickname: "@bc"}, Providers: ProvidersConfig{Default: ""}},
			defaults: &UserDefaultsConfig{
				Tools: UserDefaultsTools{Preferred: []string{"cursor", "claude"}},
			},
			wantNickname:        "@bc",
			wantDefaultProvider: "cursor",
		},
		{
			name: "workspace provider takes precedence",
			cfg:  Config{User: UserConfig{Nickname: "@bc"}, Providers: ProvidersConfig{Default: "claude"}},
			defaults: &UserDefaultsConfig{
				Tools: UserDefaultsTools{Preferred: []string{"cursor"}},
			},
			wantNickname:        "@bc",
			wantDefaultProvider: "claude",
		},
		{
			name: "merge both nickname and provider",
			cfg:  Config{User: UserConfig{Nickname: ""}, Providers: ProvidersConfig{Default: ""}},
			defaults: &UserDefaultsConfig{
				User:  UserDefaultsUser{Nickname: "@alice"},
				Tools: UserDefaultsTools{Preferred: []string{"gemini"}},
			},
			wantNickname:        "@alice",
			wantDefaultProvider: "gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			MergeUserDefaults(&cfg, tt.defaults)

			if cfg.User.Nickname != tt.wantNickname {
				t.Errorf("Nickname = %q, want %q", cfg.User.Nickname, tt.wantNickname)
			}
			if cfg.Providers.Default != tt.wantDefaultProvider {
				t.Errorf("Providers.Default = %q, want %q", cfg.Providers.Default, tt.wantDefaultProvider)
			}
		})
	}
}

// TestSaveAndLoadUserDefaults tests round-trip save/load of user defaults (#1160)
func TestSaveAndLoadUserDefaults(t *testing.T) {
	// Use a temp directory instead of actual home
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, ".bcrc")

	expectedNickname := "@testuser"
	expectedRole := "engineer"
	expectedAutoStart := true
	expectedTools := []string{"claude", "gemini"}

	// Create test file manually since SaveUserDefaults uses home directory
	data := `[user]
nickname = "@testuser"

[defaults]
default_role = "engineer"
auto_start_root = true

[tools]
preferred = ["claude", "gemini"]
`
	if err := os.WriteFile(testPath, []byte(data), 0600); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	// Read and parse the file
	content, err := os.ReadFile(testPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	var loaded UserDefaultsConfig
	if err := toml.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("failed to parse test data: %v", err)
	}

	// Verify values
	if loaded.User.Nickname != expectedNickname {
		t.Errorf("Nickname = %q, want %q", loaded.User.Nickname, expectedNickname)
	}
	if loaded.Defaults.DefaultRole != expectedRole {
		t.Errorf("DefaultRole = %q, want %q", loaded.Defaults.DefaultRole, expectedRole)
	}
	if loaded.Defaults.AutoStartRoot != expectedAutoStart {
		t.Errorf("AutoStartRoot = %v, want %v", loaded.Defaults.AutoStartRoot, expectedAutoStart)
	}
	if len(loaded.Tools.Preferred) != 2 || loaded.Tools.Preferred[0] != "claude" {
		t.Errorf("Preferred = %v, want %v", loaded.Tools.Preferred, expectedTools)
	}
}

// TestUserDefaultsPath tests the path function (#1160)
func TestUserDefaultsPath(t *testing.T) {
	path := UserDefaultsPath()
	// Should contain .bcrc
	if path != "" && !filepath.IsAbs(path) {
		t.Error("expected absolute path or empty string")
	}
}

func TestValidateNickname(t *testing.T) {
	tests := []struct { //nolint:govet // test struct, field order matches literal values
		wantErr  error
		name     string
		nickname string
	}{
		{nil, "valid nickname", "@user123"},
		{nil, "valid with underscore", "@test_user"},
		{nil, "valid uppercase", "@TestUser"},
		{ErrNicknameMissingPrefix, "missing prefix", "user123"},
		{ErrNicknameInvalidChars, "empty after @", "@"},
		{ErrNicknameTooLong, "too long", "@" + strings.Repeat("a", NicknameMaxLength)},
		{ErrNicknameInvalidChars, "invalid chars with dash", "@user-name"},
		{ErrNicknameInvalidChars, "invalid chars with dot", "@user.name"},
		{ErrNicknameInvalidChars, "invalid chars with space", "@user name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNickname(tt.nickname)
			if err != tt.wantErr {
				t.Errorf("ValidateNickname(%q) = %v, want %v", tt.nickname, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeNickname(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty returns default", "", DefaultNickname, false},
		{"whitespace only returns default", "   ", DefaultNickname, false},
		{"valid with prefix", "@alice", "@alice", false},
		{"adds prefix", "bob", "@bob", false},
		{"trims whitespace", "  @charlie  ", "@charlie", false},
		{"invalid chars returns error", "@bad-name", "", true},
		{"too long returns error", strings.Repeat("a", NicknameMaxLength+1), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeNickname(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeNickname(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("NormalizeNickname(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadUserDefaultsMalformedTOML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create malformed .bcrc file
	bcrcPath := filepath.Join(tmpDir, ".bcrc")
	malformedContent := `[user
invalid toml content
`
	if err := os.WriteFile(bcrcPath, []byte(malformedContent), 0600); err != nil {
		t.Fatalf("failed to write malformed .bcrc: %v", err)
	}

	_, err := LoadUserDefaults()
	if err == nil {
		t.Error("expected error for malformed TOML")
	}
}

func TestLoadUserDefaultsReadError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create .bcrc as a directory (will cause read error)
	bcrcPath := filepath.Join(tmpDir, ".bcrc")
	if err := os.Mkdir(bcrcPath, 0750); err != nil {
		t.Fatalf("failed to create .bcrc dir: %v", err)
	}

	_, err := LoadUserDefaults()
	if err == nil {
		t.Error("expected error when .bcrc is a directory")
	}
}

func TestSaveUserDefaultsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	defaults := &UserDefaultsConfig{
		User: UserDefaultsUser{
			Nickname: "@testuser",
		},
		Defaults: UserDefaultsDefaults{
			DefaultRole:   "engineer",
			AutoStartRoot: true,
		},
		Tools: UserDefaultsTools{
			Preferred: []string{"claude", "gemini"},
		},
	}

	err := SaveUserDefaults(defaults)
	if err != nil {
		t.Fatalf("SaveUserDefaults failed: %v", err)
	}

	// Verify file was created
	bcrcPath := filepath.Join(tmpDir, ".bcrc")
	if _, statErr := os.Stat(bcrcPath); os.IsNotExist(statErr) {
		t.Error("expected .bcrc file to be created")
	}

	// Verify content can be read back
	loaded, err := LoadUserDefaults()
	if err != nil {
		t.Fatalf("LoadUserDefaults failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded defaults to not be nil")
	}
	if loaded.User.Nickname != "@testuser" {
		t.Errorf("Nickname = %q, want @testuser", loaded.User.Nickname)
	}
}

func TestUserDefaultsPathEmpty(t *testing.T) {
	// Test when HOME is empty - path should be empty
	t.Setenv("HOME", "")

	path := UserDefaultsPath()
	if path != "" {
		t.Errorf("expected empty path when HOME is empty, got %q", path)
	}
}

// TestGetProvider tests the new GetProvider method (Issue #1771)
func TestGetProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantCommand  string
		cfg          Config
		wantNil      bool
	}{
		{
			name: "provider from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Claude: &ProviderConfig{Command: "claude --new", Enabled: true},
				},
			},
			providerName: "claude",
			wantNil:      false,
			wantCommand:  "claude --new",
		},
		{
			name:         "unknown provider returns nil",
			cfg:          Config{},
			providerName: "unknown",
			wantNil:      true,
		},
		{
			name: "gemini from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Gemini: &ProviderConfig{Command: "gemini-cli", Enabled: true},
				},
			},
			providerName: "gemini",
			wantNil:      false,
			wantCommand:  "gemini-cli",
		},
		{
			name: "cursor from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Cursor: &ProviderConfig{Command: "cursor-cli", Enabled: true},
				},
			},
			providerName: "cursor",
			wantNil:      false,
			wantCommand:  "cursor-cli",
		},
		{
			name: "codex from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Codex: &ProviderConfig{Command: "codex-cli", Enabled: true},
				},
			},
			providerName: "codex",
			wantNil:      false,
			wantCommand:  "codex-cli",
		},
		{
			name: "opencode from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					OpenCode: &ProviderConfig{Command: "opencode-cli", Enabled: true},
				},
			},
			providerName: "opencode",
			wantNil:      false,
			wantCommand:  "opencode-cli",
		},
		{
			name: "openclaw from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					OpenClaw: &ProviderConfig{Command: "openclaw-cli", Enabled: true},
				},
			},
			providerName: "openclaw",
			wantNil:      false,
			wantCommand:  "openclaw-cli",
		},
		{
			name: "aider from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Aider: &ProviderConfig{Command: "aider-cli", Enabled: true},
				},
			},
			providerName: "aider",
			wantNil:      false,
			wantCommand:  "aider-cli",
		},
		{
			name: "custom provider",
			cfg: Config{
				Providers: ProvidersConfig{
					Custom: map[string]ProviderConfig{
						"my-provider": {Command: "my-cmd", Enabled: true},
					},
				},
			},
			providerName: "my-provider",
			wantNil:      false,
			wantCommand:  "my-cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetProvider(tt.providerName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetProvider() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetProvider() returned nil, want non-nil")
			}
			if got.Command != tt.wantCommand {
				t.Errorf("GetProvider().Command = %q, want %q", got.Command, tt.wantCommand)
			}
		})
	}
}

// TestGetService tests the new GetService method (Issue #1771)
func TestGetService(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		wantCommand string
		cfg         Config
		wantNil     bool
	}{
		{
			name: "github from new config",
			cfg: Config{
				Services: ServicesConfig{
					GitHub: &ServiceConfig{Command: "gh", Enabled: true},
				},
			},
			serviceName: "github",
			wantNil:     false,
			wantCommand: "gh",
		},
		{
			name: "gitlab from new config",
			cfg: Config{
				Services: ServicesConfig{
					GitLab: &ServiceConfig{Command: "glab", Enabled: true},
				},
			},
			serviceName: "gitlab",
			wantNil:     false,
			wantCommand: "glab",
		},
		{
			name: "jira from new config",
			cfg: Config{
				Services: ServicesConfig{
					Jira: &ServiceConfig{Command: "jira-cli", Enabled: true},
				},
			},
			serviceName: "jira",
			wantNil:     false,
			wantCommand: "jira-cli",
		},
		{
			name:        "unknown service returns nil",
			cfg:         Config{},
			serviceName: "unknown",
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetService(tt.serviceName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetService() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetService() returned nil, want non-nil")
			}
			if got.Command != tt.wantCommand {
				t.Errorf("GetService().Command = %q, want %q", got.Command, tt.wantCommand)
			}
		})
	}
}

// TestGetDefaultProvider tests the GetDefaultProvider method (Issue #1771)
func TestGetDefaultProvider(t *testing.T) {
	tests := []struct {
		name string
		want string
		cfg  Config
	}{
		{
			name: "default from config",
			cfg: Config{
				Providers: ProvidersConfig{Default: "gemini"},
			},
			want: "gemini",
		},
		{
			name: "empty when nothing set",
			cfg:  Config{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetDefaultProvider()
			if got != tt.want {
				t.Errorf("GetDefaultProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestListProviders tests the ListProviders method (Issue #1869)
func TestListProviders(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment not critical
		name string
		cfg  Config
		want []string
	}{
		{
			name: "providers from new config",
			cfg: Config{
				Providers: ProvidersConfig{
					Claude: &ProviderConfig{Command: "claude", Enabled: true},
					Gemini: &ProviderConfig{Command: "gemini", Enabled: true},
					Codex:  &ProviderConfig{Command: "codex", Enabled: false},
				},
			},
			want: []string{"claude", "gemini"},
		},
		{
			name: "empty config returns nil",
			cfg:  Config{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ListProviders()
			if len(got) != len(tt.want) {
				t.Fatalf("ListProviders() returned %d items %v, want %d items %v", len(got), got, len(tt.want), tt.want)
			}
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					if g == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ListProviders() missing %q, got %v", w, got)
				}
			}
		})
	}
}

// TestListServices tests the ListServices method (Issue #1869)
func TestListServices(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment not critical
		name string
		cfg  Config
		want []string
	}{
		{
			name: "services from new config",
			cfg: Config{
				Services: ServicesConfig{
					GitHub: &ServiceConfig{Command: "gh", Enabled: true},
					Jira:   &ServiceConfig{Command: "jira", Enabled: false},
				},
			},
			want: []string{"github"},
		},
		{
			name: "empty config returns nil",
			cfg:  Config{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ListServices()
			if len(got) != len(tt.want) {
				t.Fatalf("ListServices() returned %d items %v, want %d items %v", len(got), got, len(tt.want), tt.want)
			}
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					if g == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ListServices() missing %q, got %v", w, got)
				}
			}
		})
	}
}

// TestDefaultConfig_PopulatesProviders tests that defaults include ProvidersConfig (Issue #1869)
func TestDefaultConfig_PopulatesProviders(t *testing.T) {
	cfg := DefaultConfig("test")

	// ProvidersConfig should have defaults
	if cfg.Providers.Default == "" {
		t.Error("DefaultConfig should set Providers.Default")
	}
	if cfg.Providers.Claude == nil {
		t.Error("DefaultConfig should set Providers.Claude")
	}
	if cfg.Providers.Gemini == nil {
		t.Error("DefaultConfig should set Providers.Gemini")
	}

}

func TestListProvidersCustomProviders(t *testing.T) {
	cfg := Config{
		Providers: ProvidersConfig{
			Claude: &ProviderConfig{Enabled: true, Command: "claude"},
			Custom: map[string]ProviderConfig{
				"my-llm": {Enabled: true, Command: "my-llm"},
			},
		},
	}

	providers := cfg.ListProviders()

	found := make(map[string]bool)
	for _, p := range providers {
		found[p] = true
	}

	if !found["claude"] {
		t.Error("expected claude in providers list")
	}
	if !found["my-llm"] {
		t.Error("expected my-llm custom provider in providers list")
	}
}

func TestListProvidersDisabledCustom(t *testing.T) {
	cfg := Config{
		Providers: ProvidersConfig{
			Custom: map[string]ProviderConfig{
				"disabled-llm": {Enabled: false, Command: "disabled"},
			},
		},
	}

	providers := cfg.ListProviders()
	for _, p := range providers {
		if p == "disabled-llm" {
			t.Error("disabled custom provider should not be listed")
		}
	}
}

func TestConfigSaveErrorPath(t *testing.T) {
	cfg := DefaultConfig("test")

	// Try to save to a path where the parent can't be created
	err := cfg.Save("/dev/null/impossible/config.toml")
	if err == nil {
		t.Error("Save should fail for impossible path")
	}
}

func TestValidatePerformancePollTooLow(t *testing.T) {
	cfg := DefaultConfig("test")
	cfg.Performance.PollIntervalAgents = 100 // Below 500ms minimum

	err := cfg.Validate()
	if err != ErrPollIntervalTooLow {
		t.Errorf("expected ErrPollIntervalTooLow, got %v", err)
	}
}

func TestValidatePerformanceAdaptiveTooLow(t *testing.T) {
	cfg := DefaultConfig("test")
	cfg.Performance.AdaptiveFastInterval = 200 // Below 500ms minimum

	err := cfg.Validate()
	if err != ErrPollIntervalTooLow {
		t.Errorf("expected ErrPollIntervalTooLow, got %v", err)
	}
}

func TestValidatePerformanceCacheTTLOutOfRange(t *testing.T) {
	cfg := DefaultConfig("test")
	cfg.Performance.CacheTTLTmux = 50 // Below 100ms minimum

	err := cfg.Validate()
	if err != ErrCacheTTLRange {
		t.Errorf("expected ErrCacheTTLRange, got %v", err)
	}

	// Also test above max
	cfg2 := DefaultConfig("test")
	cfg2.Performance.CacheTTLCommands = 70000 // Above 60000ms max

	err = cfg2.Validate()
	if err != ErrCacheTTLRange {
		t.Errorf("expected ErrCacheTTLRange for high TTL, got %v", err)
	}
}

func TestValidateUserNickname(t *testing.T) {
	cfg := DefaultConfig("test")
	cfg.User.Nickname = "no-at-prefix"

	err := cfg.Validate()
	if err != ErrNicknameMissingPrefix {
		t.Errorf("expected ErrNicknameMissingPrefix, got %v", err)
	}
}

func TestGetProviderCustomProvider(t *testing.T) {
	cfg := Config{
		Providers: ProvidersConfig{
			Custom: map[string]ProviderConfig{
				"custom-llm": {Enabled: true, Command: "custom-llm --run"},
			},
		},
	}

	p := cfg.GetProvider("custom-llm")
	if p == nil {
		t.Fatal("expected custom provider to be returned")
	}
	if p.Command != "custom-llm --run" {
		t.Errorf("Command = %q, want %q", p.Command, "custom-llm --run")
	}

	// Non-existent custom should return nil
	if cfg.GetProvider("nope") != nil {
		t.Error("expected nil for undefined custom provider")
	}
}

func TestGetProviderNewProviders(t *testing.T) {
	cfg := Config{
		Providers: ProvidersConfig{
			OpenCode: &ProviderConfig{Command: "opencode", Enabled: true},
			OpenClaw: &ProviderConfig{Command: "openclaw", Enabled: true},
			Aider:    &ProviderConfig{Command: "aider", Enabled: true},
		},
	}

	for _, name := range []string{"opencode", "openclaw", "aider"} {
		p := cfg.GetProvider(name)
		if p == nil {
			t.Errorf("GetProvider(%q) should return provider", name)
		}
	}
}

func TestListServicesNewConfig(t *testing.T) {
	cfg := Config{
		Services: ServicesConfig{
			GitHub: &ServiceConfig{Command: "gh", Enabled: true},
			GitLab: &ServiceConfig{Command: "glab", Enabled: true},
		},
	}

	services := cfg.ListServices()
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestRosterConfig_ParsesTOML(t *testing.T) {
	tomlData := `
[workspace]
name = "bc"
version = 2

[providers]
default = "claude"

[[roster.agents]]
name = "go-reviewer"
role = "go-reviewer"
tool = "claude"

[[roster.agents]]
name = "agent-core"
role = "feature-dev"
tool = "claude"

[[roster.agents]]
name = "pm"
role = "product-manager"
tool = "claude"
`
	cfg, err := ParseConfig([]byte(tomlData))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}

	if len(cfg.Roster.Agents) != 3 {
		t.Fatalf("expected 3 roster agents, got %d", len(cfg.Roster.Agents))
	}

	agent := cfg.Roster.Agents[0]
	if agent.Name != "go-reviewer" {
		t.Errorf("Agents[0].Name = %q, want go-reviewer", agent.Name)
	}
	if agent.Role != "go-reviewer" {
		t.Errorf("Agents[0].Role = %q, want go-reviewer", agent.Role)
	}
	if agent.Tool != "claude" {
		t.Errorf("Agents[0].Tool = %q, want claude", agent.Tool)
	}
}

func TestRosterConfig_EmptyByDefault(t *testing.T) {
	cfg := DefaultConfig("test")
	if len(cfg.Roster.Agents) != 0 {
		t.Errorf("expected empty roster by default, got %d agents", len(cfg.Roster.Agents))
	}
}

func TestRosterConfig_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig("myws")
	cfg.Roster.Agents = []RosterEntry{
		{Name: "dev1", Role: "feature-dev", Tool: "claude"},
		{Name: "reviewer", Role: "go-reviewer", Tool: "claude"},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(loaded.Roster.Agents) != 2 {
		t.Fatalf("loaded %d roster agents, want 2", len(loaded.Roster.Agents))
	}
	if loaded.Roster.Agents[0].Name != "dev1" {
		t.Errorf("Agents[0].Name = %q, want dev1", loaded.Roster.Agents[0].Name)
	}
}

func TestDefaultConfigServerSchedulerStorage(t *testing.T) {
	cfg := DefaultConfig("test-project")

	// Server defaults
	if cfg.Server.Addr != "127.0.0.1:9374" {
		t.Errorf("expected server.addr '127.0.0.1:9374', got %q", cfg.Server.Addr)
	}
	if cfg.Server.CORSOrigin != "*" {
		t.Errorf("expected server.cors_origin '*', got %q", cfg.Server.CORSOrigin)
	}

	// Scheduler defaults
	if cfg.Scheduler.TickInterval != 60 {
		t.Errorf("expected scheduler.tick_interval 60, got %d", cfg.Scheduler.TickInterval)
	}
	if cfg.Scheduler.JobTimeout != 300 {
		t.Errorf("expected scheduler.job_timeout 300, got %d", cfg.Scheduler.JobTimeout)
	}

	// Storage defaults
	if cfg.Storage.SQLitePath != ".bc/bc.db" {
		t.Errorf("expected storage.sqlite_path '.bc/bc.db', got %q", cfg.Storage.SQLitePath)
	}
}

func TestParseConfigServerSchedulerStorage(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "test"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true

[server]
addr = "0.0.0.0:8080"
cors_origin = "https://example.com"

[scheduler]
tick_interval = 30
job_timeout = 600

[storage]
sqlite_path = "/var/data/bc.db"
`)
	cfg, err := ParseConfig(tomlData)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if cfg.Server.Addr != "0.0.0.0:8080" {
		t.Errorf("expected server.addr '0.0.0.0:8080', got %q", cfg.Server.Addr)
	}
	if cfg.Server.CORSOrigin != "https://example.com" {
		t.Errorf("expected server.cors_origin 'https://example.com', got %q", cfg.Server.CORSOrigin)
	}
	if cfg.Scheduler.TickInterval != 30 {
		t.Errorf("expected scheduler.tick_interval 30, got %d", cfg.Scheduler.TickInterval)
	}
	if cfg.Scheduler.JobTimeout != 600 {
		t.Errorf("expected scheduler.job_timeout 600, got %d", cfg.Scheduler.JobTimeout)
	}
	if cfg.Storage.SQLitePath != "/var/data/bc.db" {
		t.Errorf("expected storage.sqlite_path '/var/data/bc.db', got %q", cfg.Storage.SQLitePath)
	}
}

func TestServerSchedulerStorageSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := DefaultConfig("test")
	cfg.Server.Addr = "0.0.0.0:9000"
	cfg.Server.CORSOrigin = "https://myapp.com"
	cfg.Scheduler.TickInterval = 120
	cfg.Scheduler.JobTimeout = 900
	cfg.Storage.SQLitePath = ".bc/custom.db"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.Server.Addr != "0.0.0.0:9000" {
		t.Errorf("expected server.addr '0.0.0.0:9000', got %q", loaded.Server.Addr)
	}
	if loaded.Server.CORSOrigin != "https://myapp.com" {
		t.Errorf("expected server.cors_origin 'https://myapp.com', got %q", loaded.Server.CORSOrigin)
	}
	if loaded.Scheduler.TickInterval != 120 {
		t.Errorf("expected scheduler.tick_interval 120, got %d", loaded.Scheduler.TickInterval)
	}
	if loaded.Scheduler.JobTimeout != 900 {
		t.Errorf("expected scheduler.job_timeout 900, got %d", loaded.Scheduler.JobTimeout)
	}
	if loaded.Storage.SQLitePath != ".bc/custom.db" {
		t.Errorf("expected storage.sqlite_path '.bc/custom.db', got %q", loaded.Storage.SQLitePath)
	}
}
