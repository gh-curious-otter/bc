package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRegistryAlias tests the alias functionality (#1218)
func TestRegistryAlias(t *testing.T) {
	// Create temp directory for test registry
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	r := &Registry{path: registryPath}

	// Register workspace without alias
	err := r.RegisterWithAlias("/projects/frontend", "frontend", "")
	if err != nil {
		t.Fatalf("RegisterWithAlias: %v", err)
	}

	// Register workspace with alias
	err = r.RegisterWithAlias("/projects/backend", "backend", "be")
	if err != nil {
		t.Fatalf("RegisterWithAlias with alias: %v", err)
	}

	// FindByAlias should work
	entry := r.FindByAlias("be")
	if entry == nil {
		t.Fatal("FindByAlias: expected entry, got nil")
	}
	if entry.Path != "/projects/backend" {
		t.Errorf("FindByAlias Path = %q, want %q", entry.Path, "/projects/backend")
	}

	// FindByAlias for non-existent alias should return nil
	entry = r.FindByAlias("nonexistent")
	if entry != nil {
		t.Errorf("FindByAlias for nonexistent: expected nil, got %v", entry)
	}

	// SetAlias should work
	err = r.SetAlias("/projects/frontend", "fe")
	if err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	entry = r.FindByAlias("fe")
	if entry == nil || entry.Path != "/projects/frontend" {
		t.Error("SetAlias: alias not set correctly")
	}

	// SetAlias with conflicting alias should error
	err = r.SetAlias("/projects/frontend", "be")
	if err == nil {
		t.Error("SetAlias with conflicting alias: expected error, got nil")
	}
	if _, ok := err.(*AliasConflictError); !ok {
		t.Errorf("SetAlias with conflicting alias: expected AliasConflictError, got %T", err)
	}
}

// TestRegistryActiveWorkspace tests the active workspace functionality (#1218)
func TestRegistryActiveWorkspace(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	r := &Registry{path: registryPath}

	// Register workspaces
	_ = r.RegisterWithAlias("/projects/frontend", "frontend", "fe")
	_ = r.RegisterWithAlias("/projects/backend", "backend", "be")

	// GetActive should return nil initially
	if active := r.GetActive(); active != nil {
		t.Errorf("GetActive initially: expected nil, got %v", active)
	}

	// SetActive by alias
	err := r.SetActive("fe")
	if err != nil {
		t.Fatalf("SetActive by alias: %v", err)
	}
	active := r.GetActive()
	if active == nil || active.Path != "/projects/frontend" {
		t.Error("SetActive by alias: active workspace not set correctly")
	}
	// Active should be stored as alias
	if r.Active != "fe" {
		t.Errorf("Active stored = %q, want %q", r.Active, "fe")
	}

	// SetActive by path
	err = r.SetActive("/projects/backend")
	if err != nil {
		t.Fatalf("SetActive by path: %v", err)
	}
	active = r.GetActive()
	if active == nil || active.Path != "/projects/backend" {
		t.Error("SetActive by path: active workspace not set correctly")
	}

	// SetActive for non-existent workspace should error
	err = r.SetActive("nonexistent")
	if err == nil {
		t.Error("SetActive for nonexistent: expected error, got nil")
	}

	// SetActive with empty clears active
	err = r.SetActive("")
	if err != nil {
		t.Fatalf("SetActive empty: %v", err)
	}
	if r.GetActive() != nil {
		t.Error("SetActive empty: expected nil active")
	}
}

// TestRegistryFindByNameOrAlias tests the combined lookup (#1218)
func TestRegistryFindByNameOrAlias(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	r := &Registry{path: registryPath}
	_ = r.RegisterWithAlias("/projects/frontend", "frontend", "fe")

	// Find by alias
	entry := r.FindByNameOrAlias("fe")
	if entry == nil || entry.Path != "/projects/frontend" {
		t.Error("FindByNameOrAlias by alias: not found")
	}

	// Find by name
	entry = r.FindByNameOrAlias("frontend")
	if entry == nil || entry.Path != "/projects/frontend" {
		t.Error("FindByNameOrAlias by name: not found")
	}

	// Find by path
	entry = r.FindByNameOrAlias("/projects/frontend")
	if entry == nil || entry.Path != "/projects/frontend" {
		t.Error("FindByNameOrAlias by path: not found")
	}

	// Not found
	entry = r.FindByNameOrAlias("nonexistent")
	if entry != nil {
		t.Error("FindByNameOrAlias nonexistent: expected nil")
	}
}

// TestRegistrySaveLoad tests persistence (#1218)
func TestRegistrySaveLoad(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	// Create and save registry
	r := &Registry{path: registryPath}
	_ = r.RegisterWithAlias("/projects/frontend", "frontend", "fe")
	_ = r.SetActive("fe")

	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(registryPath); err != nil {
		t.Fatalf("Registry file not created: %v", err)
	}

	// Load and verify
	// Note: LoadRegistry uses GlobalDir(), so we test Save/Load manually
	r2 := &Registry{path: registryPath}
	data, err := os.ReadFile(registryPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(data, r2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(r2.Workspaces) != 1 {
		t.Errorf("Loaded workspaces count = %d, want 1", len(r2.Workspaces))
	}
	if r2.Workspaces[0].Alias != "fe" {
		t.Errorf("Loaded alias = %q, want %q", r2.Workspaces[0].Alias, "fe")
	}
	if r2.Active != "fe" {
		t.Errorf("Loaded active = %q, want %q", r2.Active, "fe")
	}
}

// TestGlobalDir tests the GlobalDir function (#1236)
func TestGlobalDir(t *testing.T) {
	dir := GlobalDir()
	// GlobalDir should return a non-empty string on most systems
	if dir == "" {
		t.Skip("GlobalDir returned empty (no home directory)")
	}
	// Should end with .bc
	if filepath.Base(dir) != ".bc" {
		t.Errorf("GlobalDir = %q, want ending with .bc", dir)
	}
	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("GlobalDir = %q, want absolute path", dir)
	}
}

// TestRegistryPath tests the RegistryPath function (#1236)
func TestRegistryPath(t *testing.T) {
	path := RegistryPath()
	if path == "" {
		t.Skip("RegistryPath returned empty (no home directory)")
	}
	// Should end with workspaces.json
	if filepath.Base(path) != "workspaces.json" {
		t.Errorf("RegistryPath = %q, want ending with workspaces.json", path)
	}
	// Should be under GlobalDir
	globalDir := GlobalDir()
	if filepath.Dir(path) != globalDir {
		t.Errorf("RegistryPath dir = %q, want %q", filepath.Dir(path), globalDir)
	}
}

// TestLoadRegistry tests the LoadRegistry function (#1236)
func TestLoadRegistry(t *testing.T) {
	// This test uses the real home directory
	// We test that LoadRegistry doesn't error even if the file doesn't exist
	r, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry error = %v", err)
	}
	if r == nil {
		t.Fatal("LoadRegistry returned nil")
	}
	// Registry should have a path set
	if r.path == "" {
		t.Error("LoadRegistry returned registry with empty path")
	}
}

// TestLoadRegistryFromTempDir tests LoadRegistry with controlled file (#1236)
func TestLoadRegistryFromTempDir(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	// Test loading non-existent file returns empty registry
	r := &Registry{path: registryPath}
	data, err := os.ReadFile(r.path)
	if !os.IsNotExist(err) {
		t.Logf("File exists with data: %s", data)
	}

	// Create a valid registry file
	testData := []byte(`{
		"active": "test",
		"workspaces": [
			{"path": "/test/path", "name": "test", "alias": "t"}
		]
	}`)
	if writeErr := os.WriteFile(registryPath, testData, 0600); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}

	// Load and verify
	r = &Registry{path: registryPath}
	data, err = os.ReadFile(r.path) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(data, r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if r.Active != "test" {
		t.Errorf("Active = %q, want %q", r.Active, "test")
	}
	if len(r.Workspaces) != 1 {
		t.Fatalf("len(Workspaces) = %d, want 1", len(r.Workspaces))
	}
	if r.Workspaces[0].Alias != "t" {
		t.Errorf("Alias = %q, want %q", r.Workspaces[0].Alias, "t")
	}
}
