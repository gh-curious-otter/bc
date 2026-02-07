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
	if cfg.Tools.Default != "claude" {
		t.Errorf("expected default tool 'claude', got %q", cfg.Tools.Default)
	}
	if cfg.Tools.Claude == nil {
		t.Error("expected claude tool to be configured")
	}
	if cfg.Memory.Backend != "file" {
		t.Errorf("expected memory backend 'file', got %q", cfg.Memory.Backend)
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

[beads]
enabled = true
issues_dir = ".beads/issues"

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
	if !cfg.Beads.Enabled {
		t.Error("expected beads to be enabled")
	}
	if cfg.Beads.IssuesDir != ".beads/issues" {
		t.Errorf("expected beads issues_dir '.beads/issues', got %q", cfg.Beads.IssuesDir)
	}
	if len(cfg.Channels.Default) != 2 {
		t.Errorf("expected 2 default channels, got %d", len(cfg.Channels.Default))
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
	cfg.Beads.Enabled = false
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
	if loaded.Beads.Enabled {
		t.Error("expected beads to be disabled")
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
