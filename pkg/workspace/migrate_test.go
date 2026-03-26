package workspace_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeV1Workspace(t *testing.T, cfg workspace.V1Config) string {
	t.Helper()
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "config.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// ─── LoadV1Config ──────────────────────────────────────────────────────────────

func TestLoadV1Config_Basic(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "my-project",
		Provider: "claude",
		Command:  "claude --dangerously-skip-permissions",
	})

	cfg, err := workspace.LoadV1Config(dir)
	if err != nil {
		t.Fatalf("LoadV1Config: %v", err)
	}
	if cfg.Name != "my-project" {
		t.Errorf("Name = %q, want my-project", cfg.Name)
	}
	if cfg.Provider != "claude" {
		t.Errorf("Provider = %q, want claude", cfg.Provider)
	}
}

func TestLoadV1Config_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.LoadV1Config(dir)
	if !errors.Is(err, workspace.ErrNotV1Workspace) {
		t.Errorf("want ErrNotV1Workspace, got %v", err)
	}
}

func TestLoadV1Config_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := workspace.LoadV1Config(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ─── MigrateV1ToV2 ────────────────────────────────────────────────────────────

func TestMigrateV1ToV2_Basic(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "test-project",
		Provider: "claude",
		Command:  "claude --dangerously-skip-permissions",
	})

	result, err := workspace.MigrateV1ToV2(dir)
	if err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	if !result.ConfigMigrated {
		t.Error("ConfigMigrated should be true")
	}

	// Backup must exist
	if result.BackupPath == "" {
		t.Error("BackupPath should not be empty")
	}
	if _, statErr := os.Stat(result.BackupPath); statErr != nil {
		t.Errorf("backup file not found: %v", statErr)
	}

	// settings.json must exist and be loadable
	tomlPath := filepath.Join(dir, ".bc", "settings.json")
	if _, statErr := os.Stat(tomlPath); statErr != nil {
		t.Fatalf("settings.json not written: %v", statErr)
	}
}

func TestMigrateV1ToV2_ProducesValidConfig(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "test-project",
		Provider: "gemini",
		Command:  "gemini --yolo",
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	// The written settings.json must be loadable by workspace.Load.
	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("Load after migration: %v", err)
	}
	if ws.Name() != "test-project" {
		t.Errorf("Name = %q, want test-project", ws.Name())
	}
}

func TestMigrateV1ToV2_DefaultProvider_Claude(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "proj",
		Provider: "claude",
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	cfg, err := workspace.LoadConfig(filepath.Join(dir, ".bc", "settings.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Providers.Default != "claude" {
		t.Errorf("providers.default = %q, want claude", cfg.Providers.Default)
	}
	if cfg.Providers.Claude == nil {
		t.Error("providers.claude should be set")
	}
}

func TestMigrateV1ToV2_ProvidersMap(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:      "proj",
		Provider:  "claude",
		Providers: map[string]string{"claude": "claude --dangerously-skip-permissions", "gemini": "gemini --yolo"},
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	cfg, err := workspace.LoadConfig(filepath.Join(dir, ".bc", "settings.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Providers.Claude == nil {
		t.Error("providers.claude should be set from Providers map")
	}
	if cfg.Providers.Gemini == nil {
		t.Error("providers.gemini should be set from Providers map")
	}
}

func TestMigrateV1ToV2_NicknamePreserved(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "proj",
		Provider: "claude",
		Nickname: "@alice",
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	cfg, err := workspace.LoadConfig(filepath.Join(dir, ".bc", "settings.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.User.Name != "@alice" {
		t.Errorf("user.nickname = %q, want @alice", cfg.User.Name)
	}
}

func TestMigrateV1ToV2_RuntimePreserved(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "proj",
		Provider: "claude",
		Runtime:  "docker",
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}

	cfg, err := workspace.LoadConfig(filepath.Join(dir, ".bc", "settings.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Runtime.Default != "docker" {
		t.Errorf("runtime.backend = %q, want docker", cfg.Runtime.Default)
	}
}

func TestMigrateV1ToV2_NotV1Workspace(t *testing.T) {
	dir := t.TempDir()

	_, err := workspace.MigrateV1ToV2(dir)
	if !errors.Is(err, workspace.ErrNotV1Workspace) {
		t.Errorf("want ErrNotV1Workspace, got %v", err)
	}
}

func TestMigrateV1ToV2_AgentFilesCount(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{Name: "proj", Provider: "claude"})
	stateDir := filepath.Join(dir, ".bc")

	// Seed some legacy agent JSON files
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"alice.json", "bob.json"} {
		if err := os.WriteFile(filepath.Join(agentsDir, f), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Also the top-level agents.json
	if err := os.WriteFile(filepath.Join(stateDir, "agents.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := workspace.MigrateV1ToV2(dir)
	if err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}
	if result.AgentFiles != 3 {
		t.Errorf("AgentFiles = %d, want 3", result.AgentFiles)
	}
}

func TestMigrateV1ToV2_ChannelJSONNotMigrated(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{Name: "proj", Provider: "claude"})
	stateDir := filepath.Join(dir, ".bc")
	if err := os.WriteFile(filepath.Join(stateDir, "channels.json"), []byte(`[]`), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := workspace.MigrateV1ToV2(dir)
	if err != nil {
		t.Fatalf("MigrateV1ToV2: %v", err)
	}
	// Channel JSON migration was removed; the flag should be false.
	if result.ChannelJSON {
		t.Error("ChannelJSON should be false — channel JSON migration is no longer performed during v1→v2 migration")
	}
}

// ─── workspace.Load backward compatibility ────────────────────────────────────

func TestLoad_V1WorkspaceReturnsErrNotV1Workspace(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{Name: "proj", Provider: "claude"})

	_, err := workspace.Load(dir)
	if err == nil {
		t.Fatal("expected error loading v1 workspace without settings.json")
	}
	if !errors.Is(err, workspace.ErrNotV1Workspace) {
		t.Errorf("want error wrapping ErrNotV1Workspace, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bc workspace migrate") {
		t.Errorf("error should mention 'bc workspace migrate', got: %v", err)
	}
}

func TestLoad_AfterMigration_Succeeds(t *testing.T) {
	dir := makeV1Workspace(t, workspace.V1Config{
		Name:     "migrated-ws",
		Provider: "gemini",
		Command:  "gemini --yolo",
	})

	if _, err := workspace.MigrateV1ToV2(dir); err != nil {
		t.Fatalf("migration: %v", err)
	}

	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("Load after migration: %v", err)
	}
	if ws.Name() != "migrated-ws" {
		t.Errorf("Name = %q, want migrated-ws", ws.Name())
	}
}

// ─── CountLegacyAgentFiles ─────────────────────────────────────────────────────

func TestCountLegacyAgentFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	n := workspace.CountLegacyAgentFiles(dir)
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestCountLegacyAgentFiles_WithFiles(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Two per-agent JSON files + one non-JSON
	for _, f := range []string{"alice.json", "bob.json", "alice.log"} {
		if err := os.WriteFile(filepath.Join(agentsDir, f), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Top-level agents.json
	if err := os.WriteFile(filepath.Join(stateDir, "agents.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	n := workspace.CountLegacyAgentFiles(stateDir)
	if n != 3 { // alice.json + bob.json (in agents/) + agents.json (top-level)
		t.Errorf("expected 3, got %d", n)
	}
}
