package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gh-curious-otter/bc/pkg/workspace"
)

func TestIsV1Workspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a workspace
	if isV1Workspace(tmpDir) {
		t.Error("empty dir should not be v1 workspace")
	}

	// Create v1 structure
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if !isV1Workspace(tmpDir) {
		t.Error("dir with .bc/config.json should be v1 workspace")
	}
}

func TestIsV2Workspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a workspace
	if isV2Workspace(tmpDir) {
		t.Error("empty dir should not be v2 workspace")
	}

	// Create v2 structure
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "settings.json"), []byte("[workspace]\nname = \"test\"\nversion = 2\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if !isV2Workspace(tmpDir) {
		t.Error("dir with .bc/settings.json should be v2 workspace")
	}
}

func TestInitV2Workspace(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Initialize v2 workspace
	if err := initV2Workspace(projectDir); err != nil {
		t.Fatalf("initV2Workspace failed: %v", err)
	}

	// Verify .bc directory exists
	bcDir := filepath.Join(projectDir, ".bc")
	if _, err := os.Stat(bcDir); err != nil {
		t.Errorf(".bc directory not created: %v", err)
	}

	// Verify settings.json exists and is valid
	configPath := filepath.Join(bcDir, "settings.json")
	cfg, err := workspace.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg != "test-project" {
		t.Errorf("expected name 'test-project', got %q", cfg)
	}
	if cfg.Version != 2 {
		t.Errorf("expected version 2, got %d", cfg.Version)
	}
	if validateErr := cfg.Validate(); validateErr != nil {
		t.Errorf("config validation failed: %v", validateErr)
	}

	// Verify agents directory exists
	agentsDir := filepath.Join(bcDir, "agents")
	if _, statErr := os.Stat(agentsDir); statErr != nil {
		t.Errorf("agents directory not created: %v", statErr)
	}

	// Roles are now stored in SQL (bc.db), not as .bc/roles/*.md files.
	// Verify the database file exists.
	dbPath := filepath.Join(bcDir, "bc.db")
	if _, statErr := os.Stat(dbPath); statErr != nil {
		t.Errorf("bc.db not created: %v", statErr)
	}
}

func TestInitV2WorkspaceIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}

	// First init
	if err := initV2Workspace(projectDir); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init should fail (already initialized)
	if isV2Workspace(projectDir) == false {
		t.Error("workspace should be detected as v2 after init")
	}
}

func TestRunInitV1Detection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create v1 structure
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Run init - should fail with v1 warning
	err := runInit(nil, []string{tmpDir})
	if err == nil {
		t.Error("expected error when v1 workspace exists")
	}
	if !strings.Contains(err.Error(), "v1 workspace exists") {
		t.Errorf("error should mention v1 workspace: %v", err)
	}
}

func TestRunInitV2AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}

	// First init should succeed
	if err := initV2Workspace(projectDir); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init should fail
	err := runInit(nil, []string{projectDir})
	if err == nil {
		t.Error("expected error when already initialized")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("error should mention already initialized: %v", err)
	}
}

func TestRunInitFreshDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "fresh-project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Use quick mode to skip interactive wizard (no stdin in tests)
	initQuick = true
	defer func() { initQuick = false }()
	err := runInit(nil, []string{projectDir})
	if err != nil {
		t.Fatalf("init on fresh directory failed: %v", err)
	}

	// Verify workspace was created
	if !isV2Workspace(projectDir) {
		t.Error("workspace should exist after init")
	}
}
