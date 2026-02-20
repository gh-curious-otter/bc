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
