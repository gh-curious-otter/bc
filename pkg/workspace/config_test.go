package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultV2Config(t *testing.T) {
	cfg := DefaultV2Config("test-project")

	if cfg.Workspace.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", cfg.Workspace.Name)
	}
	if cfg.Workspace.Version != ConfigVersion {
		t.Errorf("expected version %d, got %d", ConfigVersion, cfg.Workspace.Version)
	}
	// Default tool is gemini (minimal root-only startup)
	if cfg.Tools.Default != "gemini" {
		t.Errorf("expected default tool 'gemini', got %q", cfg.Tools.Default)
	}
	if cfg.Tools.Gemini == nil {
		t.Error("expected gemini tool to be configured")
	}
	if cfg.Memory.Backend != "file" {
		t.Errorf("expected memory backend 'file', got %q", cfg.Memory.Backend)
	}
	// Minimal startup: roster values default to 0
	if cfg.Roster.Engineers != 0 {
		t.Errorf("expected roster.engineers = 0, got %d", cfg.Roster.Engineers)
	}
	if cfg.Roster.TechLeads != 0 {
		t.Errorf("expected roster.tech_leads = 0, got %d", cfg.Roster.TechLeads)
	}
	if cfg.Roster.QA != 0 {
		t.Errorf("expected roster.qa = 0, got %d", cfg.Roster.QA)
	}
}

func TestParseV2Config(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "my-project"
version = 2

[worktrees]
path = ".bc/worktrees"
auto_cleanup = true

[tools]
default = "claude"

[tools.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[memory]
backend = "file"
path = ".bc/memory"

[channels]
default = ["general", "engineering"]
`)

	cfg, err := ParseV2Config(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Workspace.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", cfg.Workspace.Name)
	}
	if cfg.Workspace.Version != 2 {
		t.Errorf("expected version 2, got %d", cfg.Workspace.Version)
	}
	if cfg.Worktrees.Path != ".bc/worktrees" {
		t.Errorf("expected worktrees path '.bc/worktrees', got %q", cfg.Worktrees.Path)
	}
	if !cfg.Worktrees.AutoCleanup {
		t.Error("expected auto_cleanup to be true")
	}
	if cfg.Tools.Default != "claude" {
		t.Errorf("expected default tool 'claude', got %q", cfg.Tools.Default)
	}
	if cfg.Tools.Claude == nil {
		t.Fatal("expected claude tool config")
	}
	if cfg.Tools.Claude.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("unexpected claude command: %q", cfg.Tools.Claude.Command)
	}
	if !cfg.Tools.Claude.Enabled {
		t.Error("expected claude to be enabled")
	}
	if cfg.Memory.Backend != "file" {
		t.Errorf("expected memory backend 'file', got %q", cfg.Memory.Backend)
	}
	if cfg.Memory.Path != ".bc/memory" {
		t.Errorf("expected memory path '.bc/memory', got %q", cfg.Memory.Path)
	}
	if len(cfg.Channels.Default) != 2 {
		t.Errorf("expected 2 default channels, got %d", len(cfg.Channels.Default))
	}
}

func TestParseV2ConfigWithRoster(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "roster-project"
version = 2

[tools]
default = "claude"

[tools.claude]
command = "claude"
enabled = true

[memory]
backend = "file"
path = ".bc/memory"

[roster]
engineers = 5
tech_leads = 3
qa = 1
`)

	cfg, err := ParseV2Config(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Roster.Engineers != 5 {
		t.Errorf("expected roster.engineers = 5, got %d", cfg.Roster.Engineers)
	}
	if cfg.Roster.TechLeads != 3 {
		t.Errorf("expected roster.tech_leads = 3, got %d", cfg.Roster.TechLeads)
	}
	if cfg.Roster.QA != 1 {
		t.Errorf("expected roster.qa = 1, got %d", cfg.Roster.QA)
	}
}

func TestV2ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		cfg     V2Config
	}{
		{
			name:    "missing workspace name",
			wantErr: ErrMissingWorkspaceName,
			cfg:     V2Config{Workspace: WorkspaceConfig{Version: 2}},
		},
		{
			name:    "invalid version",
			wantErr: ErrInvalidVersion,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 1},
			},
		},
		{
			name:    "missing default tool",
			wantErr: ErrMissingDefaultTool,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
			},
		},
		{
			name:    "default tool not defined",
			wantErr: ErrDefaultToolNotFound,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "nonexistent"},
			},
		},
		{
			name:    "missing memory backend",
			wantErr: ErrMissingMemoryBackend,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "claude", Claude: &ToolConfig{Enabled: true}},
			},
		},
		{
			name:    "missing memory path",
			wantErr: ErrMissingMemoryPath,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "claude", Claude: &ToolConfig{Enabled: true}},
				Memory:    MemoryConfig{Backend: "file"},
			},
		},
		{
			name:    "valid config",
			wantErr: nil,
			cfg:     DefaultV2Config("test"),
		},
		{
			name:    "roster product_manager too high",
			wantErr: ErrRosterProductManagerRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.ProductManager = 11
				return cfg
			}(),
		},
		{
			name:    "roster product_manager negative",
			wantErr: ErrRosterProductManagerRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.ProductManager = -1
				return cfg
			}(),
		},
		{
			name:    "roster manager too high",
			wantErr: ErrRosterManagerRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.Manager = 11
				return cfg
			}(),
		},
		{
			name:    "roster manager negative",
			wantErr: ErrRosterManagerRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.Manager = -1
				return cfg
			}(),
		},
		{
			name:    "roster engineers too high",
			wantErr: ErrRosterEngineersRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.Engineers = 11
				return cfg
			}(),
		},
		{
			name:    "roster engineers negative",
			wantErr: ErrRosterEngineersRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.Engineers = -1
				return cfg
			}(),
		},
		{
			name:    "roster tech_leads too high",
			wantErr: ErrRosterTechLeadsRange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.TechLeads = 11
				return cfg
			}(),
		},
		{
			name:    "roster qa too high",
			wantErr: ErrRosterQARange,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.QA = 11
				return cfg
			}(),
		},
		{
			name:    "roster zero values valid",
			wantErr: nil,
			cfg: func() V2Config {
				cfg := DefaultV2Config("test")
				cfg.Roster.ProductManager = 0
				cfg.Roster.Manager = 0
				cfg.Roster.Engineers = 0
				cfg.Roster.TechLeads = 0
				cfg.Roster.QA = 0
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

func TestV2ConfigGetTool(t *testing.T) {
	cfg := DefaultV2Config("test")

	// Test getting claude (default)
	tool := cfg.GetTool("claude")
	if tool == nil {
		t.Fatal("expected claude tool config")
	}
	if tool.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("unexpected command: %q", tool.Command)
	}

	// Test getting non-existent tool
	tool = cfg.GetTool("nonexistent")
	if tool != nil {
		t.Error("expected nil for nonexistent tool")
	}

	// Test GetDefaultTool
	tool = cfg.GetDefaultTool()
	if tool == nil {
		t.Fatal("expected default tool config")
	}
}

func TestV2ConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".bc", "config.toml")

	// Create and save config
	cfg := DefaultV2Config("save-test")
	cfg.Channels.Default = []string{"custom-channel"}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadV2Config(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Workspace.Name != "save-test" {
		t.Errorf("expected name 'save-test', got %q", loaded.Workspace.Name)
	}
	if len(loaded.Channels.Default) != 1 || loaded.Channels.Default[0] != "custom-channel" {
		t.Errorf("unexpected channels: %v", loaded.Channels.Default)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath("/home/user/project")
	expected := "/home/user/project/.bc/config.toml"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLoadV2ConfigNotFound(t *testing.T) {
	_, err := LoadV2Config("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseV2ConfigInvalid(t *testing.T) {
	_, err := ParseV2Config([]byte("invalid toml {{{"))
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestV2ConfigGetTool_Cursor(t *testing.T) {
	cfg := V2Config{
		Workspace: WorkspaceConfig{Name: "test", Version: 2},
		Tools: ToolsConfig{
			Default: "cursor",
			Cursor: &ToolConfig{
				Command: "cursor --wait",
				Enabled: true,
			},
		},
		Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
	}

	// Validate should pass with cursor as default
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// GetTool should return cursor config
	tool := cfg.GetTool("cursor")
	if tool == nil {
		t.Fatal("expected cursor tool config")
	}
	if tool.Command != "cursor --wait" {
		t.Errorf("expected command 'cursor --wait', got %q", tool.Command)
	}
	if !tool.Enabled {
		t.Error("expected cursor to be enabled")
	}

	// Claude should be nil when not configured
	if cfg.GetTool("claude") != nil {
		t.Error("expected nil for unconfigured claude")
	}
}

func TestV2ConfigGetTool_Codex(t *testing.T) {
	cfg := V2Config{
		Workspace: WorkspaceConfig{Name: "test", Version: 2},
		Tools: ToolsConfig{
			Default: "codex",
			Codex: &ToolConfig{
				Command: "codex --full-auto",
				Enabled: true,
			},
		},
		Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
	}

	// Validate should pass with codex as default
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// GetTool should return codex config
	tool := cfg.GetTool("codex")
	if tool == nil {
		t.Fatal("expected codex tool config")
	}
	if tool.Command != "codex --full-auto" {
		t.Errorf("expected command 'codex --full-auto', got %q", tool.Command)
	}
	if !tool.Enabled {
		t.Error("expected codex to be enabled")
	}

	// Cursor should be nil when not configured
	if cfg.GetTool("cursor") != nil {
		t.Error("expected nil for unconfigured cursor")
	}
}

func TestV2ConfigGetTool_Gemini(t *testing.T) {
	cfg := V2Config{
		Workspace: WorkspaceConfig{Name: "test", Version: 2},
		Tools: ToolsConfig{
			Default: "gemini",
			Gemini: &ToolConfig{
				Command: "gemini --yolo",
				Enabled: true,
			},
		},
		Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
	}

	// Validate should pass with gemini as default
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// GetTool should return gemini config
	tool := cfg.GetTool("gemini")
	if tool == nil {
		t.Fatal("expected gemini tool config")
	}
	if tool.Command != "gemini --yolo" {
		t.Errorf("expected command 'gemini --yolo', got %q", tool.Command)
	}
	if !tool.Enabled {
		t.Error("expected gemini to be enabled")
	}

	// Cursor should be nil when not configured
	if cfg.GetTool("cursor") != nil {
		t.Error("expected nil for unconfigured cursor")
	}
}

func TestV2ConfigCustomTools(t *testing.T) {
	cfg := V2Config{
		Workspace: WorkspaceConfig{Name: "test", Version: 2},
		Tools: ToolsConfig{
			Default: "my-custom-agent",
			Custom: map[string]ToolConfig{
				"my-custom-agent": {
					Command: "/usr/local/bin/my-agent --special-flag",
					Enabled: true,
				},
				"another-tool": {
					Command: "another-tool run",
					Enabled: false,
				},
			},
		},
		Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
	}

	// Validate should pass with custom tool as default
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// GetTool should return custom tool config
	tool := cfg.GetTool("my-custom-agent")
	if tool == nil {
		t.Fatal("expected custom tool config")
	}
	if tool.Command != "/usr/local/bin/my-agent --special-flag" {
		t.Errorf("unexpected command: %q", tool.Command)
	}
	if !tool.Enabled {
		t.Error("expected custom tool to be enabled")
	}

	// GetTool should return second custom tool
	tool2 := cfg.GetTool("another-tool")
	if tool2 == nil {
		t.Fatal("expected another-tool config")
	}
	if tool2.Command != "another-tool run" {
		t.Errorf("unexpected command: %q", tool2.Command)
	}
	if tool2.Enabled {
		t.Error("expected another-tool to be disabled")
	}

	// GetDefaultTool should return the custom default
	defaultTool := cfg.GetDefaultTool()
	if defaultTool == nil {
		t.Fatal("expected default tool config")
	}
	if defaultTool.Command != "/usr/local/bin/my-agent --special-flag" {
		t.Errorf("unexpected default tool command: %q", defaultTool.Command)
	}

	// Non-existent custom tool should return nil
	if cfg.GetTool("undefined-tool") != nil {
		t.Error("expected nil for undefined custom tool")
	}
}

func TestV2ConfigHasToolDefined(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		cfg      V2Config
		want     bool
	}{
		{
			name:     "claude defined",
			toolName: "claude",
			cfg: V2Config{
				Tools: ToolsConfig{Claude: &ToolConfig{Command: "claude"}},
			},
			want: true,
		},
		{
			name:     "claude not defined",
			toolName: "claude",
			cfg:      V2Config{Tools: ToolsConfig{}},
			want:     false,
		},
		{
			name:     "cursor defined",
			toolName: "cursor",
			cfg: V2Config{
				Tools: ToolsConfig{Cursor: &ToolConfig{Command: "cursor"}},
			},
			want: true,
		},
		{
			name:     "cursor not defined",
			toolName: "cursor",
			cfg:      V2Config{Tools: ToolsConfig{}},
			want:     false,
		},
		{
			name:     "codex defined",
			toolName: "codex",
			cfg: V2Config{
				Tools: ToolsConfig{Codex: &ToolConfig{Command: "codex"}},
			},
			want: true,
		},
		{
			name:     "codex not defined",
			toolName: "codex",
			cfg:      V2Config{Tools: ToolsConfig{}},
			want:     false,
		},
		{
			name:     "custom tool defined",
			toolName: "my-agent",
			cfg: V2Config{
				Tools: ToolsConfig{
					Custom: map[string]ToolConfig{
						"my-agent": {Command: "my-agent"},
					},
				},
			},
			want: true,
		},
		{
			name:     "custom tool not defined",
			toolName: "my-agent",
			cfg: V2Config{
				Tools: ToolsConfig{Custom: map[string]ToolConfig{}},
			},
			want: false,
		},
		{
			name:     "custom tool with nil map",
			toolName: "my-agent",
			cfg:      V2Config{Tools: ToolsConfig{}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.hasToolDefined(tt.toolName)
			if got != tt.want {
				t.Errorf("hasToolDefined(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestV2ConfigValidation_ToolVariants(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		cfg     V2Config
	}{
		{
			name:    "valid with cursor default",
			wantErr: nil,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools: ToolsConfig{
					Default: "cursor",
					Cursor:  &ToolConfig{Command: "cursor", Enabled: true},
				},
				Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
			},
		},
		{
			name:    "valid with codex default",
			wantErr: nil,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools: ToolsConfig{
					Default: "codex",
					Codex:   &ToolConfig{Command: "codex", Enabled: true},
				},
				Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
			},
		},
		{
			name:    "valid with custom tool default",
			wantErr: nil,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools: ToolsConfig{
					Default: "my-tool",
					Custom: map[string]ToolConfig{
						"my-tool": {Command: "my-tool", Enabled: true},
					},
				},
				Memory: MemoryConfig{Backend: "file", Path: ".bc/memory"},
			},
		},
		{
			name:    "cursor default but not defined",
			wantErr: ErrDefaultToolNotFound,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "cursor"},
				Memory:    MemoryConfig{Backend: "file", Path: ".bc/memory"},
			},
		},
		{
			name:    "codex default but not defined",
			wantErr: ErrDefaultToolNotFound,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "codex"},
				Memory:    MemoryConfig{Backend: "file", Path: ".bc/memory"},
			},
		},
		{
			name:    "custom default but not defined",
			wantErr: ErrDefaultToolNotFound,
			cfg: V2Config{
				Workspace: WorkspaceConfig{Name: "test", Version: 2},
				Tools:     ToolsConfig{Default: "undefined-custom"},
				Memory:    MemoryConfig{Backend: "file", Path: ".bc/memory"},
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

func TestParseV2Config_MultipleTools(t *testing.T) {
	tomlData := []byte(`
[workspace]
name = "multi-tool-project"
version = 2

[tools]
default = "claude"

[tools.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[tools.cursor]
command = "cursor --wait"
enabled = true

[tools.codex]
command = "codex --full-auto"
enabled = false

[memory]
backend = "file"
path = ".bc/memory"
`)

	cfg, err := ParseV2Config(tomlData)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify all tools are parsed
	if cfg.Tools.Claude == nil {
		t.Error("expected claude to be configured")
	}
	if cfg.Tools.Cursor == nil {
		t.Error("expected cursor to be configured")
	}
	if cfg.Tools.Codex == nil {
		t.Error("expected codex to be configured")
	}

	// Verify tool properties
	if cfg.Tools.Cursor.Command != "cursor --wait" {
		t.Errorf("unexpected cursor command: %q", cfg.Tools.Cursor.Command)
	}
	if !cfg.Tools.Cursor.Enabled {
		t.Error("expected cursor to be enabled")
	}

	if cfg.Tools.Codex.Command != "codex --full-auto" {
		t.Errorf("unexpected codex command: %q", cfg.Tools.Codex.Command)
	}
	if cfg.Tools.Codex.Enabled {
		t.Error("expected codex to be disabled")
	}

	// Validation should pass
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}
