package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserRCConfigPath(t *testing.T) {
	path := UserRCConfigPath()
	if path == "" {
		t.Skip("no home directory available")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}

	if filepath.Base(path) != ".bcrc" {
		t.Errorf("expected .bcrc filename, got: %s", filepath.Base(path))
	}
}

func TestDefaultUserRCConfig(t *testing.T) {
	cfg := DefaultUserRCConfig()

	if cfg.User.Nickname != DefaultNickname {
		t.Errorf("expected default nickname %s, got: %s", DefaultNickname, cfg.User.Nickname)
	}

	if cfg.Defaults.DefaultRole != "engineer" {
		t.Errorf("expected default role 'engineer', got: %s", cfg.Defaults.DefaultRole)
	}

	if !cfg.Defaults.AutoStartRoot {
		t.Error("expected auto_start_root to be true by default")
	}

	if len(cfg.Tools.Preferred) == 0 {
		t.Error("expected preferred tools to be set")
	}
}

func TestParseUserRCConfig(t *testing.T) {
	data := []byte(`
[user]
nickname = "@alice"

[defaults]
default_role = "manager"
auto_start_root = false

[tools]
preferred = ["cursor", "claude-code"]
`)

	cfg, err := ParseUserRCConfig(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if cfg.User.Nickname != "@alice" {
		t.Errorf("expected nickname '@alice', got: %s", cfg.User.Nickname)
	}

	if cfg.Defaults.DefaultRole != "manager" {
		t.Errorf("expected default role 'manager', got: %s", cfg.Defaults.DefaultRole)
	}

	if cfg.Defaults.AutoStartRoot {
		t.Error("expected auto_start_root to be false")
	}

	if len(cfg.Tools.Preferred) != 2 {
		t.Errorf("expected 2 preferred tools, got: %d", len(cfg.Tools.Preferred))
	}
}

func TestUserRCConfigSaveAndLoad(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create and save a config
	cfg := DefaultUserRCConfig()
	cfg.User.Nickname = "@testuser"

	err := cfg.Save()
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, ".bcrc")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("config file not created: %v", statErr)
	}

	// Load and verify
	loaded, err := LoadUserRCConfig()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("loaded config is nil")
	}

	if loaded.User.Nickname != "@testuser" {
		t.Errorf("expected nickname '@testuser', got: %s", loaded.User.Nickname)
	}
}

func TestMergeWithUserRC(t *testing.T) {
	// Create a workspace config with default nickname
	wsCfg := DefaultV2Config("test")

	// Create a user config with custom nickname
	rcCfg := &UserRCConfig{
		User: UserRCUserConfig{
			Nickname: "@custom",
		},
	}

	// Merge
	wsCfg.MergeWithUserRC(rcCfg)

	// User RC nickname should be used since workspace has default
	if wsCfg.User.Nickname != "@custom" {
		t.Errorf("expected merged nickname '@custom', got: %s", wsCfg.User.Nickname)
	}
}

func TestHasTool(t *testing.T) {
	cfg := DefaultV2Config("test")

	if !cfg.HasTool("gemini") {
		t.Error("expected gemini to be available")
	}

	if !cfg.HasTool("claude") {
		t.Error("expected claude to be available")
	}

	if cfg.HasTool("unknown-tool") {
		t.Error("expected unknown-tool to not be available")
	}
}

func TestHasToolAllTypes(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		cfg      V2Config
		want     bool
	}{
		{
			name:     "claude enabled",
			toolName: "claude",
			cfg: V2Config{
				Tools: ToolsConfig{Claude: &ToolConfig{Enabled: true}},
			},
			want: true,
		},
		{
			name:     "claude-code enabled",
			toolName: "claude-code",
			cfg: V2Config{
				Tools: ToolsConfig{Claude: &ToolConfig{Enabled: true}},
			},
			want: true,
		},
		{
			name:     "claude disabled",
			toolName: "claude",
			cfg: V2Config{
				Tools: ToolsConfig{Claude: &ToolConfig{Enabled: false}},
			},
			want: false,
		},
		{
			name:     "cursor enabled",
			toolName: "cursor",
			cfg: V2Config{
				Tools: ToolsConfig{Cursor: &ToolConfig{Enabled: true}},
			},
			want: true,
		},
		{
			name:     "cursor disabled",
			toolName: "cursor",
			cfg: V2Config{
				Tools: ToolsConfig{Cursor: &ToolConfig{Enabled: false}},
			},
			want: false,
		},
		{
			name:     "codex enabled",
			toolName: "codex",
			cfg: V2Config{
				Tools: ToolsConfig{Codex: &ToolConfig{Enabled: true}},
			},
			want: true,
		},
		{
			name:     "gemini enabled",
			toolName: "gemini",
			cfg: V2Config{
				Tools: ToolsConfig{Gemini: &ToolConfig{Enabled: true}},
			},
			want: true,
		},
		{
			name:     "custom tool enabled",
			toolName: "my-tool",
			cfg: V2Config{
				Tools: ToolsConfig{
					Custom: map[string]ToolConfig{
						"my-tool": {Enabled: true},
					},
				},
			},
			want: true,
		},
		{
			name:     "custom tool disabled still exists",
			toolName: "my-tool",
			cfg: V2Config{
				Tools: ToolsConfig{
					Custom: map[string]ToolConfig{
						"my-tool": {Enabled: false},
					},
				},
			},
			want: true, // custom tools check existence only, not Enabled
		},
		{
			name:     "custom tool not in map",
			toolName: "my-tool",
			cfg: V2Config{
				Tools: ToolsConfig{Custom: map[string]ToolConfig{}},
			},
			want: false,
		},
		{
			name:     "nil tools",
			toolName: "claude",
			cfg:      V2Config{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasTool(tt.toolName)
			if got != tt.want {
				t.Errorf("HasTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestUserRCExists(t *testing.T) {
	// Test with temp directory
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Initially should not exist
	if UserRCExists() {
		t.Error("UserRCExists should return false when .bcrc doesn't exist")
	}

	// Create .bcrc file
	rcPath := filepath.Join(tmpDir, ".bcrc")
	if err := os.WriteFile(rcPath, []byte("[user]\nnickname = \"@test\""), 0600); err != nil {
		t.Fatalf("failed to create .bcrc: %v", err)
	}

	// Now should exist
	if !UserRCExists() {
		t.Error("UserRCExists should return true when .bcrc exists")
	}
}

func TestGetPreferredTool(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment not critical
		name     string
		cfg      V2Config
		rc       *UserRCConfig
		expected string
	}{
		{
			name: "nil rc returns default",
			cfg: V2Config{
				Tools: ToolsConfig{
					Default: "claude",
					Claude:  &ToolConfig{Enabled: true},
				},
			},
			rc:       nil,
			expected: "claude",
		},
		{
			name: "empty preferred list returns default",
			cfg: V2Config{
				Tools: ToolsConfig{
					Default: "claude",
					Claude:  &ToolConfig{Enabled: true},
				},
			},
			rc:       &UserRCConfig{},
			expected: "claude",
		},
		{
			name: "first preferred tool available",
			cfg: V2Config{
				Tools: ToolsConfig{
					Default: "claude",
					Claude:  &ToolConfig{Enabled: true},
					Cursor:  &ToolConfig{Enabled: true},
				},
			},
			rc: &UserRCConfig{
				Tools: UserRCToolsConfig{
					Preferred: []string{"cursor", "claude"},
				},
			},
			expected: "cursor",
		},
		{
			name: "skip unavailable tool",
			cfg: V2Config{
				Tools: ToolsConfig{
					Default: "claude",
					Claude:  &ToolConfig{Enabled: true},
				},
			},
			rc: &UserRCConfig{
				Tools: UserRCToolsConfig{
					Preferred: []string{"cursor", "claude"},
				},
			},
			expected: "claude",
		},
		{
			name: "no preferred tools available",
			cfg: V2Config{
				Tools: ToolsConfig{
					Default: "gemini",
					Gemini:  &ToolConfig{Enabled: true},
				},
			},
			rc: &UserRCConfig{
				Tools: UserRCToolsConfig{
					Preferred: []string{"cursor", "claude"},
				},
			},
			expected: "gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetPreferredTool(tt.rc)
			if got != tt.expected {
				t.Errorf("GetPreferredTool() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadUserRCConfigNotFound(t *testing.T) {
	// Test with temp directory that has no .bcrc
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	cfg, err := LoadUserRCConfig()
	if err != nil {
		t.Errorf("LoadUserRCConfig should not error for missing file: %v", err)
	}
	if cfg != nil {
		t.Error("LoadUserRCConfig should return nil for missing file")
	}
}

func TestMergeWithUserRCNil(t *testing.T) {
	cfg := DefaultV2Config("test")
	originalNickname := cfg.User.Nickname

	// Merge with nil should not change anything
	cfg.MergeWithUserRC(nil)

	if cfg.User.Nickname != originalNickname {
		t.Errorf("MergeWithUserRC(nil) changed nickname from %q to %q", originalNickname, cfg.User.Nickname)
	}
}

func TestMergeWithUserRCPreserveWorkspace(t *testing.T) {
	cfg := DefaultV2Config("test")
	cfg.User.Nickname = "@workspace-user"

	rc := &UserRCConfig{
		User: UserRCUserConfig{
			Nickname: "@rc-user",
		},
	}

	cfg.MergeWithUserRC(rc)

	// Workspace nickname should be preserved
	if cfg.User.Nickname != "@workspace-user" {
		t.Errorf("MergeWithUserRC changed workspace nickname to %q", cfg.User.Nickname)
	}
}

func TestParseUserRCConfigInvalidTOML(t *testing.T) {
	_, err := ParseUserRCConfig([]byte("{invalid toml!!!"))
	if err == nil {
		t.Error("ParseUserRCConfig should fail on invalid TOML")
	}
}

func TestHasToolGitHubGitLabJira(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		cfg      V2Config
		want     bool
	}{
		{
			name:     "github enabled",
			toolName: "github",
			cfg:      V2Config{Tools: ToolsConfig{GitHub: &ToolConfig{Enabled: true}}},
			want:     true,
		},
		{
			name:     "github nil",
			toolName: "github",
			cfg:      V2Config{},
			want:     false,
		},
		{
			name:     "gitlab enabled",
			toolName: "gitlab",
			cfg:      V2Config{Tools: ToolsConfig{GitLab: &ToolConfig{Enabled: true}}},
			want:     true,
		},
		{
			name:     "gitlab nil",
			toolName: "gitlab",
			cfg:      V2Config{},
			want:     false,
		},
		{
			name:     "jira enabled",
			toolName: "jira",
			cfg:      V2Config{Tools: ToolsConfig{Jira: &ToolConfig{Enabled: true}}},
			want:     true,
		},
		{
			name:     "jira nil",
			toolName: "jira",
			cfg:      V2Config{},
			want:     false,
		},
		{
			name:     "custom tool nil map",
			toolName: "unknown",
			cfg:      V2Config{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasTool(tt.toolName)
			if got != tt.want {
				t.Errorf("HasTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestUserRCConfigSavePathEmpty(t *testing.T) {
	// Unset HOME to simulate path failure
	t.Setenv("HOME", "")

	cfg := DefaultUserRCConfig()
	err := cfg.Save()
	if err == nil {
		t.Error("Save should fail when home directory is empty")
	}
}
