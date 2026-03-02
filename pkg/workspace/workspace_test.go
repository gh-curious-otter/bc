package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- DefaultConfig ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/tmp/myproject")

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.Name != "myproject" {
		t.Errorf("Name = %q, want %q", cfg.Name, "myproject")
	}
	if cfg.RootDir != "/tmp/myproject" {
		t.Errorf("RootDir = %q, want %q", cfg.RootDir, "/tmp/myproject")
	}
	if cfg.MaxWorkers != 3 {
		t.Errorf("MaxWorkers = %d, want 3", cfg.MaxWorkers)
	}
	wantState := filepath.Join("/tmp/myproject", ".bc")
	if cfg.StateDir != wantState {
		t.Errorf("StateDir = %q, want %q", cfg.StateDir, wantState)
	}
}

// --- Init ---

func TestInit(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Workspace struct is populated
	if ws.RootDir == "" {
		t.Error("RootDir is empty")
	}
	if ws.Config.Name != filepath.Base(dir) {
		t.Errorf("Config.Name = %q, want %q", ws.Config.Name, filepath.Base(dir))
	}

	// .bc directory was created
	stateDir := filepath.Join(dir, ".bc")
	if _, statErr := os.Stat(stateDir); statErr != nil {
		t.Errorf(".bc directory not created: %v", statErr)
	}

	// config.json was written
	configPath := filepath.Join(stateDir, "config.json")
	data, err := os.ReadFile(configPath) //nolint:gosec // test file read
	if err != nil {
		t.Fatalf("config.json not written: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("persisted Version = %d, want 1", cfg.Version)
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()

	ws1, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	ws2, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	if ws1.Config.Name != ws2.Config.Name {
		t.Errorf("second Init changed Name: %q vs %q", ws1.Config.Name, ws2.Config.Name)
	}
}

// --- Load ---

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ws.Config.Version != 1 {
		t.Errorf("Version = %d, want 1", ws.Config.Version)
	}
}

func TestLoadNotAWorkspace(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load non-workspace: expected error, got nil")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte("{bad"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load invalid JSON: expected error, got nil")
	}
}

func TestLoadUpdatesPathsIfMoved(t *testing.T) {
	// Init in one location, then copy .bc to a new location and Load
	orig := t.TempDir()
	if _, err := Init(orig); err != nil {
		t.Fatal(err)
	}

	moved := t.TempDir()
	// Copy .bc directory
	srcCfg := filepath.Join(orig, ".bc", "config.json")
	dstDir := filepath.Join(moved, ".bc")
	if err := os.MkdirAll(dstDir, 0750); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(srcCfg) //nolint:gosec // test file read
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(filepath.Join(dstDir, "config.json"), data, 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	ws, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(moved)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Config.RootDir != absDir {
		t.Errorf("RootDir = %q, want %q", ws.Config.RootDir, absDir)
	}
	wantState := filepath.Join(absDir, ".bc")
	if ws.Config.StateDir != wantState {
		t.Errorf("StateDir = %q, want %q", ws.Config.StateDir, wantState)
	}
}

// --- Find (upward search) ---

func TestFindInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir); err != nil {
		t.Fatal(err)
	}

	ws, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ws.RootDir != absDir {
		t.Errorf("RootDir = %q, want %q", ws.RootDir, absDir)
	}
}

func TestFindInParentDir(t *testing.T) {
	parent := t.TempDir()
	if _, err := Init(parent); err != nil {
		t.Fatal(err)
	}

	// Create a child directory (no workspace of its own)
	child := filepath.Join(parent, "subdir", "deep")
	if err := os.MkdirAll(child, 0750); err != nil {
		t.Fatal(err)
	}

	ws, err := Find(child)
	if err != nil {
		t.Fatalf("Find from child: %v", err)
	}

	absParent, err := filepath.Abs(parent)
	if err != nil {
		t.Fatal(err)
	}
	if ws.RootDir != absParent {
		t.Errorf("RootDir = %q, want %q (parent)", ws.RootDir, absParent)
	}
}

func TestFindNestedWorkspaces(t *testing.T) {
	// Outer workspace
	outer := t.TempDir()
	if _, err := Init(outer); err != nil {
		t.Fatal(err)
	}

	// Inner workspace inside outer
	inner := filepath.Join(outer, "projects", "sub")
	if err := os.MkdirAll(inner, 0750); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(inner); err != nil {
		t.Fatal(err)
	}

	// Find from inner should find the inner workspace, not outer
	ws, err := Find(inner)
	if err != nil {
		t.Fatal(err)
	}
	absInner, err := filepath.Abs(inner)
	if err != nil {
		t.Fatal(err)
	}
	if ws.RootDir != absInner {
		t.Errorf("RootDir = %q, want inner %q", ws.RootDir, absInner)
	}

	// Find from a child of inner should still find inner
	deepChild := filepath.Join(inner, "src", "pkg")
	if mkdirErr := os.MkdirAll(deepChild, 0750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	ws2, err := Find(deepChild)
	if err != nil {
		t.Fatal(err)
	}
	if ws2.RootDir != absInner {
		t.Errorf("RootDir = %q, want inner %q", ws2.RootDir, absInner)
	}
}

func TestFindNoWorkspace(t *testing.T) {
	dir := t.TempDir()

	_, err := Find(dir)
	if err == nil {
		t.Fatal("Find in non-workspace tree: expected error, got nil")
	}
}

// --- Save ---

func TestSave(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Modify config
	ws.Config.MaxWorkers = 10
	ws.Config.Name = "renamed"

	if saveErr := ws.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	// Reload and verify
	ws2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ws2.Config.MaxWorkers != 10 {
		t.Errorf("MaxWorkers = %d, want 10", ws2.Config.MaxWorkers)
	}
	if ws2.Config.Name != "renamed" {
		t.Errorf("Name = %q, want %q", ws2.Config.Name, "renamed")
	}
}

// --- Path helpers ---

func TestPathHelpers(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	bcDir := filepath.Join(absDir, ".bc")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"StateDir", ws.StateDir(), bcDir},
		{"AgentsDir", ws.AgentsDir(), filepath.Join(bcDir, "agents")},
		{"LogsDir", ws.LogsDir(), filepath.Join(bcDir, "logs")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// --- LogsDir ---

func TestLogsDirV2CustomPath(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Set a custom logs path in V2Config
	ws.V2Config = &V2Config{
		Logs: LogsConfig{Path: "custom/logs"},
	}

	got := ws.LogsDir()
	want := filepath.Join(absDir, "custom/logs")
	if got != want {
		t.Errorf("LogsDir() = %q, want %q", got, want)
	}
}

func TestLogsDirV2EmptyPath(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	// V2Config exists but Logs.Path is empty — should fall back to StateDir/logs
	ws.V2Config = &V2Config{
		Logs: LogsConfig{Path: ""},
	}

	got := ws.LogsDir()
	want := filepath.Join(absDir, ".bc", "logs")
	if got != want {
		t.Errorf("LogsDir() = %q, want %q", got, want)
	}
}

func TestLogsDirV1Fallback(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	// No V2Config — should use StateDir/logs
	ws.V2Config = nil

	got := ws.LogsDir()
	want := filepath.Join(absDir, ".bc", "logs")
	if got != want {
		t.Errorf("LogsDir() = %q, want %q", got, want)
	}
}

// --- EnsureDirs ---

func TestEnsureDirs(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	for _, d := range []string{ws.StateDir(), ws.AgentsDir(), ws.LogsDir()} {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("directory %q not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", d)
		}
	}
}

func TestEnsureDirsV2(t *testing.T) {
	dir := t.TempDir()

	// InitV2 creates a v2 workspace
	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs V2: %v", err)
	}

	// V2 creates additional dirs: roles, memory, worktrees, channels
	v2Dirs := []string{
		ws.RolesDir(),
		ws.MemoryDir(),
		ws.WorktreesDir(),
		ws.ChannelsDir(),
	}
	for _, d := range v2Dirs {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("V2 directory %q not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", d)
		}
	}
}

func TestEnsureDirsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Call twice — should not error
	if err := ws.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("second EnsureDirs: %v", err)
	}
}

// --- IsWorkspace ---

func TestIsWorkspace(t *testing.T) {
	tests := []struct {
		setup func(t *testing.T) string
		name  string
		want  bool
	}{
		{
			func(t *testing.T) string {
				dir := t.TempDir()
				if _, err := Init(dir); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			"initialized workspace",
			true,
		},
		{
			func(t *testing.T) string {
				return t.TempDir()
			},
			"empty directory",
			false,
		},
		{
			func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			"nonexistent directory",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			if got := IsWorkspace(dir); got != tt.want {
				t.Errorf("IsWorkspace = %v, want %v", got, tt.want)
			}
		})
	}
}

// =====================
// Registry tests
// =====================

// newTestRegistry creates a Registry backed by a temp file.
func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	dir := t.TempDir()
	return &Registry{
		path: filepath.Join(dir, "workspaces.json"),
	}
}

// --- Register ---

func TestRegistryRegister(t *testing.T) {
	r := newTestRegistry(t)

	r.Register("/projects/foo", "foo")

	if len(r.Workspaces) != 1 {
		t.Fatalf("Workspaces len = %d, want 1", len(r.Workspaces))
	}
	if r.Workspaces[0].Path != "/projects/foo" {
		t.Errorf("Path = %q, want %q", r.Workspaces[0].Path, "/projects/foo")
	}
	if r.Workspaces[0].Name != "foo" {
		t.Errorf("Name = %q, want %q", r.Workspaces[0].Name, "foo")
	}
	if r.Workspaces[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestRegistryRegisterUpdatesExisting(t *testing.T) {
	r := newTestRegistry(t)

	r.Register("/projects/foo", "foo")
	time.Sleep(time.Millisecond) // ensure time difference
	r.Register("/projects/foo", "foo-renamed")

	if len(r.Workspaces) != 1 {
		t.Fatalf("Workspaces len = %d, want 1 (should update, not duplicate)", len(r.Workspaces))
	}
	if r.Workspaces[0].Name != "foo-renamed" {
		t.Errorf("Name = %q, want %q", r.Workspaces[0].Name, "foo-renamed")
	}
}

func TestRegistryRegisterMultiple(t *testing.T) {
	r := newTestRegistry(t)

	r.Register("/a", "a")
	r.Register("/b", "b")
	r.Register("/c", "c")

	if len(r.Workspaces) != 3 {
		t.Fatalf("Workspaces len = %d, want 3", len(r.Workspaces))
	}
}

// --- Unregister ---

func TestRegistryUnregister(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/a", "a")
	r.Register("/b", "b")

	r.Unregister("/a")

	if len(r.Workspaces) != 1 {
		t.Fatalf("Workspaces len = %d, want 1", len(r.Workspaces))
	}
	if r.Workspaces[0].Path != "/b" {
		t.Errorf("remaining Path = %q, want %q", r.Workspaces[0].Path, "/b")
	}
}

func TestRegistryUnregisterNotFound(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/a", "a")

	// Should be a no-op, not panic
	r.Unregister("/nonexistent")

	if len(r.Workspaces) != 1 {
		t.Errorf("Workspaces len = %d, want 1", len(r.Workspaces))
	}
}

// --- Touch ---

func TestRegistryTouch(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/a", "a")
	originalTime := r.Workspaces[0].LastAccessed

	time.Sleep(time.Millisecond)
	r.Touch("/a")

	if !r.Workspaces[0].LastAccessed.After(originalTime) {
		t.Error("Touch did not update LastAccessed")
	}
}

func TestRegistryTouchNotFound(t *testing.T) {
	r := newTestRegistry(t)

	// Should be a no-op, not panic
	r.Touch("/nonexistent")
}

// --- Find ---

func TestRegistryFind(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/a", "a")
	r.Register("/b", "b")

	entry := r.Find("/b")
	if entry == nil {
		t.Fatal("Find: expected entry, got nil")
	}
	if entry.Name != "b" {
		t.Errorf("Name = %q, want %q", entry.Name, "b")
	}
}

func TestRegistryFindNotFound(t *testing.T) {
	r := newTestRegistry(t)

	if entry := r.Find("/nonexistent"); entry != nil {
		t.Errorf("Find nonexistent: expected nil, got %+v", entry)
	}
}

// --- List ---

func TestRegistryListEmpty(t *testing.T) {
	r := newTestRegistry(t)

	list := r.List()
	if len(list) != 0 {
		t.Errorf("List empty = %d, want 0", len(list))
	}
}

func TestRegistryList(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/a", "a")
	r.Register("/b", "b")

	if len(r.List()) != 2 {
		t.Errorf("List = %d, want 2", len(r.List()))
	}
}

// --- Prune ---

func TestRegistryPrune(t *testing.T) {
	r := newTestRegistry(t)

	// Create a real workspace
	realDir := t.TempDir()
	if _, err := Init(realDir); err != nil {
		t.Fatal(err)
	}
	absReal, err := filepath.Abs(realDir)
	if err != nil {
		t.Fatal(err)
	}

	// Register it plus a fake one
	r.Register(absReal, "real")
	r.Register("/nonexistent/fake/workspace", "fake")

	pruned := r.Prune()

	if pruned != 1 {
		t.Errorf("pruned = %d, want 1", pruned)
	}
	if len(r.Workspaces) != 1 {
		t.Fatalf("Workspaces len = %d, want 1", len(r.Workspaces))
	}
	if r.Workspaces[0].Path != absReal {
		t.Errorf("remaining Path = %q, want %q", r.Workspaces[0].Path, absReal)
	}
}

func TestRegistryPruneAllGone(t *testing.T) {
	r := newTestRegistry(t)
	r.Register("/gone/a", "a")
	r.Register("/gone/b", "b")

	pruned := r.Prune()
	if pruned != 2 {
		t.Errorf("pruned = %d, want 2", pruned)
	}
	if len(r.Workspaces) != 0 {
		t.Errorf("Workspaces len = %d, want 0", len(r.Workspaces))
	}
}

func TestRegistryPruneNothingToPrune(t *testing.T) {
	r := newTestRegistry(t)

	realDir := t.TempDir()
	if _, err := Init(realDir); err != nil {
		t.Fatal(err)
	}
	absReal, err := filepath.Abs(realDir)
	if err != nil {
		t.Fatal(err)
	}
	r.Register(absReal, "real")

	pruned := r.Prune()
	if pruned != 0 {
		t.Errorf("pruned = %d, want 0", pruned)
	}
}

// --- Registry Save / Load round-trip ---

func TestRegistrySaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaces.json")

	r1 := &Registry{path: path}
	r1.Register("/projects/alpha", "alpha")
	r1.Register("/projects/beta", "beta")

	if err := r1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into a fresh registry
	r2 := &Registry{path: path}
	data, err := os.ReadFile(path) //nolint:gosec // test file read
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, r2); err != nil {
		t.Fatal(err)
	}

	if len(r2.Workspaces) != 2 {
		t.Fatalf("loaded Workspaces len = %d, want 2", len(r2.Workspaces))
	}

	// Verify entries
	names := map[string]bool{}
	for _, w := range r2.Workspaces {
		names[w.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("loaded names = %v, want alpha and beta", names)
	}
}

func TestRegistrySaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "workspaces.json")

	r := &Registry{path: path}
	r.Register("/test", "test")

	if err := r.Save(); err != nil {
		t.Fatalf("Save with nested dir: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

// =====================
// V2 Workspace Tests
// =====================

func TestInitV2(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatalf("InitV2: %v", err)
	}

	// Check version
	if ws.ConfigVersion() != 2 {
		t.Errorf("ConfigVersion = %d, want 2", ws.ConfigVersion())
	}
	if !ws.IsV2() {
		t.Error("IsV2 should return true")
	}

	// Check V2Config is set
	if ws.V2Config == nil {
		t.Fatal("V2Config is nil")
	}
	if ws.V2Config.Workspace.Name != filepath.Base(dir) {
		t.Errorf("V2Config.Workspace.Name = %q, want %q", ws.V2Config.Workspace.Name, filepath.Base(dir))
	}

	// Check config.toml was created
	tomlPath := filepath.Join(dir, ".bc", "config.toml")
	if _, err := os.Stat(tomlPath); err != nil {
		t.Errorf("config.toml not created: %v", err)
	}

	// Check roles directory and default root.md
	rolesDir := filepath.Join(dir, ".bc", "roles")
	if _, err := os.Stat(rolesDir); err != nil {
		t.Errorf("roles directory not created: %v", err)
	}
	rootRole := filepath.Join(rolesDir, "root.md")
	if _, err := os.Stat(rootRole); err != nil {
		t.Errorf("root.md not created: %v", err)
	}

	// Check RoleManager is initialized
	if ws.RoleManager == nil {
		t.Error("RoleManager is nil")
	}
}

func TestLoadV2Workspace(t *testing.T) {
	dir := t.TempDir()

	// Initialize v2 workspace
	_, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Load it back
	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if ws.ConfigVersion() != 2 {
		t.Errorf("ConfigVersion = %d, want 2", ws.ConfigVersion())
	}
	if ws.V2Config == nil {
		t.Fatal("V2Config is nil after load")
	}
	if ws.RoleManager == nil {
		t.Error("RoleManager is nil after load")
	}

	// Check that root role was loaded
	role, ok := ws.RoleManager.GetRole("root")
	if !ok {
		t.Error("root role should be loaded")
	}
	if role.Metadata.Name != "root" {
		t.Errorf("root role name = %q, want %q", role.Metadata.Name, "root")
	}
}

func TestLoadV1WorkspaceFallback(t *testing.T) {
	dir := t.TempDir()

	// Initialize v1 workspace
	_, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Load it - should work with deprecation warning
	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load v1: %v", err)
	}

	if ws.ConfigVersion() != 1 {
		t.Errorf("ConfigVersion = %d, want 1", ws.ConfigVersion())
	}
	if ws.IsV2() {
		t.Error("IsV2 should return false for v1 workspace")
	}
	if ws.V2Config != nil {
		t.Error("V2Config should be nil for v1 workspace")
	}
	if ws.RoleManager != nil {
		t.Error("RoleManager should be nil for v1 workspace")
	}
}

func TestLoadPrefersTOMLOverJSON(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create both config.json (v1) and config.toml (v2)
	jsonPath := filepath.Join(stateDir, "config.json")
	jsonCfg := Config{Version: 1, Name: "v1-name", RootDir: dir, StateDir: stateDir}
	jsonData, _ := json.Marshal(jsonCfg)
	if err := os.WriteFile(jsonPath, jsonData, 0600); err != nil {
		t.Fatal(err)
	}

	tomlCfg := DefaultV2Config("v2-name")
	if err := tomlCfg.Save(filepath.Join(stateDir, "config.toml")); err != nil {
		t.Fatal(err)
	}

	// Create roles dir for v2
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rolesDir, "root.md"), []byte(DefaultRootRole), 0600); err != nil {
		t.Fatal(err)
	}

	// Load should prefer TOML
	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if ws.ConfigVersion() != 2 {
		t.Errorf("should load v2, got version %d", ws.ConfigVersion())
	}
	if ws.Config.Name != "v2-name" {
		t.Errorf("Name = %q, want %q", ws.Config.Name, "v2-name")
	}
}

func TestWorkspaceV2Directories(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Check all v2 directories exist
	dirs := map[string]string{
		"RolesDir":     ws.RolesDir(),
		"MemoryDir":    ws.MemoryDir(),
		"WorktreesDir": ws.WorktreesDir(),
		"ChannelsDir":  ws.ChannelsDir(),
	}

	for name, path := range dirs {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s (%s) not created: %v", name, path, err)
		}
	}
}

func TestWorkspaceGetRole(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Get default root role
	role, err := ws.GetRole("root")
	if err != nil {
		t.Fatalf("GetRole(root): %v", err)
	}
	if !role.Metadata.IsSingleton {
		t.Error("root role should be singleton")
	}

	// Get nonexistent role
	_, err = ws.GetRole("nonexistent")
	if err == nil {
		t.Error("GetRole should fail for nonexistent role")
	}
}

func TestWorkspaceGetRolePrompt(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	prompt := ws.GetRolePrompt("root")
	if prompt == "" {
		t.Error("GetRolePrompt(root) should not be empty")
	}

	// Nonexistent role returns empty
	prompt = ws.GetRolePrompt("nonexistent")
	if prompt != "" {
		t.Error("GetRolePrompt(nonexistent) should be empty")
	}
}

func TestWorkspaceDefaultTool(t *testing.T) {
	dir := t.TempDir()

	// v2 workspace - default tool is gemini (minimal root-only startup)
	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	if ws.DefaultTool() != "gemini" {
		t.Errorf("DefaultTool = %q, want %q", ws.DefaultTool(), "gemini")
	}

	cmd := ws.DefaultToolCommand()
	if cmd != "gemini --yolo" {
		t.Errorf("DefaultToolCommand = %q, want %q", cmd, "gemini --yolo")
	}
}

func TestWorkspaceSaveV2(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Modify config
	ws.V2Config.Workspace.Name = "modified-name"

	// Save
	if saveErr := ws.Save(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	// Reload and verify
	ws2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}

	if ws2.Config.Name != "modified-name" {
		t.Errorf("Name after reload = %q, want %q", ws2.Config.Name, "modified-name")
	}
}

func TestWorkspaceV1GetRoleError(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// v1 workspace should error on GetRole
	_, err = ws.GetRole("root")
	if err == nil {
		t.Error("GetRole should fail for v1 workspace")
	}
}

func TestWorkspaceDefaultChannels(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	channels := ws.DefaultChannels()
	if len(channels) != 2 {
		t.Errorf("DefaultChannels len = %d, want 2", len(channels))
	}
}

func TestWorkspaceDefaultToolV1(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// V1 default tool is claude
	if ws.DefaultTool() != "claude" {
		t.Errorf("DefaultTool v1 = %q, want claude", ws.DefaultTool())
	}

	// V1 default command
	if ws.DefaultToolCommand() != "claude --dangerously-skip-permissions" {
		t.Errorf("DefaultToolCommand v1 = %q, want default claude command", ws.DefaultToolCommand())
	}
}

func TestWorkspaceDefaultToolV1WithConfig(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Set custom tool in v1 config
	ws.Config.Tool = "cursor"
	ws.Config.AgentCommand = "cursor --wait"

	if ws.DefaultTool() != "cursor" {
		t.Errorf("DefaultTool v1 custom = %q, want cursor", ws.DefaultTool())
	}

	if ws.DefaultToolCommand() != "cursor --wait" {
		t.Errorf("DefaultToolCommand v1 custom = %q, want cursor --wait", ws.DefaultToolCommand())
	}
}

func TestWorkspaceDefaultChannelsV1(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	channels := ws.DefaultChannels()
	if len(channels) != 2 {
		t.Errorf("DefaultChannels v1 len = %d, want 2", len(channels))
	}
	if channels[0] != "general" {
		t.Errorf("First channel = %q, want general", channels[0])
	}
	if channels[1] != "engineering" {
		t.Errorf("Second channel = %q, want engineering", channels[1])
	}
}

func TestWorkspaceMemoryDir(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	memDir := ws.MemoryDir()
	if memDir == "" {
		t.Error("MemoryDir should not be empty")
	}

	// Should contain .bc/memory
	expected := filepath.Join(dir, ".bc", "memory")
	if memDir != expected {
		t.Errorf("MemoryDir = %q, want %q", memDir, expected)
	}
}

func TestWorkspaceWorktreesDir(t *testing.T) {
	dir := t.TempDir()

	ws, err := InitV2(dir)
	if err != nil {
		t.Fatal(err)
	}

	wtDir := ws.WorktreesDir()
	if wtDir == "" {
		t.Error("WorktreesDir should not be empty")
	}

	// Should contain .bc/worktrees
	expected := filepath.Join(dir, ".bc", "worktrees")
	if wtDir != expected {
		t.Errorf("WorktreesDir = %q, want %q", wtDir, expected)
	}
}

func TestWorkspaceMemoryDirV1(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	memDir := ws.MemoryDir()
	// V1 should still return a memory dir path
	if memDir == "" {
		t.Error("MemoryDir should not be empty for v1")
	}
}

func TestWorkspaceWorktreesDirV1(t *testing.T) {
	dir := t.TempDir()

	ws, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	wtDir := ws.WorktreesDir()
	// V1 should still return a worktrees dir path
	if wtDir == "" {
		t.Error("WorktreesDir should not be empty for v1")
	}
}

func TestCopyDefaultPrompts(t *testing.T) {
	// Create source directory with prompts
	rootDir := t.TempDir()
	sourceDir := filepath.Join(rootDir, "prompts")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a test prompt file
	testPrompt := "This is a test prompt."
	if err := os.WriteFile(filepath.Join(sourceDir, "test.md"), []byte(testPrompt), 0600); err != nil {
		t.Fatal(err)
	}

	// Create state directory and prompts subdirectory
	stateDir := filepath.Join(rootDir, ".bc")
	destDir := filepath.Join(stateDir, "prompts")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Copy prompts
	if err := copyDefaultPrompts(rootDir, stateDir); err != nil {
		t.Fatalf("copyDefaultPrompts: %v", err)
	}

	// Verify prompt was copied
	destPath := filepath.Join(stateDir, "prompts", "test.md")
	data, err := os.ReadFile(destPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("copied file not found: %v", err)
	}
	if string(data) != testPrompt {
		t.Errorf("copied content = %q, want %q", string(data), testPrompt)
	}
}

func TestCopyDefaultPromptsNoSource(t *testing.T) {
	// When no prompts directory exists, should silently succeed
	rootDir := t.TempDir()
	stateDir := filepath.Join(rootDir, ".bc")
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Should not error
	if err := copyDefaultPrompts(rootDir, stateDir); err != nil {
		t.Errorf("copyDefaultPrompts without source should not error: %v", err)
	}
}

func TestCopyDefaultPromptsExistingDest(t *testing.T) {
	// Create source directory with prompts
	rootDir := t.TempDir()
	sourceDir := filepath.Join(rootDir, "prompts")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.md"), []byte("source content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create state directory with existing prompts
	stateDir := filepath.Join(rootDir, ".bc")
	destDir := filepath.Join(stateDir, "prompts")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Create existing file with different content
	if err := os.WriteFile(filepath.Join(destDir, "test.md"), []byte("existing content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Copy should skip existing files (not overwrite)
	if err := copyDefaultPrompts(rootDir, stateDir); err != nil {
		t.Fatalf("copyDefaultPrompts: %v", err)
	}

	// Verify existing content was preserved
	data, err := os.ReadFile(filepath.Join(destDir, "test.md")) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing content" {
		t.Errorf("existing file was overwritten, got %q", string(data))
	}
}
