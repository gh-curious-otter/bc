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
	if _, err := os.Stat(stateDir); err != nil {
		t.Errorf(".bc directory not created: %v", err)
	}

	// config.json was written
	configPath := filepath.Join(stateDir, "config.json")
	data, err := os.ReadFile(configPath)
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
	Init(dir)

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
	os.MkdirAll(bcDir, 0755)
	os.WriteFile(filepath.Join(bcDir, "config.json"), []byte("{bad"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load invalid JSON: expected error, got nil")
	}
}

func TestLoadUpdatesPathsIfMoved(t *testing.T) {
	// Init in one location, then copy .bc to a new location and Load
	orig := t.TempDir()
	Init(orig)

	moved := t.TempDir()
	// Copy .bc directory
	srcCfg := filepath.Join(orig, ".bc", "config.json")
	dstDir := filepath.Join(moved, ".bc")
	os.MkdirAll(dstDir, 0755)
	data, _ := os.ReadFile(srcCfg)
	os.WriteFile(filepath.Join(dstDir, "config.json"), data, 0644)

	ws, err := Load(moved)
	if err != nil {
		t.Fatal(err)
	}

	absDir, _ := filepath.Abs(moved)
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
	Init(dir)

	ws, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	absDir, _ := filepath.Abs(dir)
	if ws.RootDir != absDir {
		t.Errorf("RootDir = %q, want %q", ws.RootDir, absDir)
	}
}

func TestFindInParentDir(t *testing.T) {
	parent := t.TempDir()
	Init(parent)

	// Create a child directory (no workspace of its own)
	child := filepath.Join(parent, "subdir", "deep")
	os.MkdirAll(child, 0755)

	ws, err := Find(child)
	if err != nil {
		t.Fatalf("Find from child: %v", err)
	}

	absParent, _ := filepath.Abs(parent)
	if ws.RootDir != absParent {
		t.Errorf("RootDir = %q, want %q (parent)", ws.RootDir, absParent)
	}
}

func TestFindNestedWorkspaces(t *testing.T) {
	// Outer workspace
	outer := t.TempDir()
	Init(outer)

	// Inner workspace inside outer
	inner := filepath.Join(outer, "projects", "sub")
	os.MkdirAll(inner, 0755)
	Init(inner)

	// Find from inner should find the inner workspace, not outer
	ws, err := Find(inner)
	if err != nil {
		t.Fatal(err)
	}
	absInner, _ := filepath.Abs(inner)
	if ws.RootDir != absInner {
		t.Errorf("RootDir = %q, want inner %q", ws.RootDir, absInner)
	}

	// Find from a child of inner should still find inner
	deepChild := filepath.Join(inner, "src", "pkg")
	os.MkdirAll(deepChild, 0755)
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
	ws, _ := Init(dir)

	// Modify config
	ws.Config.MaxWorkers = 10
	ws.Config.Name = "renamed"

	if err := ws.Save(); err != nil {
		t.Fatalf("Save: %v", err)
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
	ws, _ := Init(dir)

	absDir, _ := filepath.Abs(dir)
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

// --- EnsureDirs ---

func TestEnsureDirs(t *testing.T) {
	dir := t.TempDir()
	ws, _ := Init(dir)

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

func TestEnsureDirsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ws, _ := Init(dir)

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
		name string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			"initialized workspace",
			func(t *testing.T) string {
				dir := t.TempDir()
				Init(dir)
				return dir
			},
			true,
		},
		{
			"empty directory",
			func(t *testing.T) string {
				return t.TempDir()
			},
			false,
		},
		{
			"nonexistent directory",
			func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
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
	Init(realDir)
	absReal, _ := filepath.Abs(realDir)

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
	Init(realDir)
	absReal, _ := filepath.Abs(realDir)
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
	data, err := os.ReadFile(path)
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
