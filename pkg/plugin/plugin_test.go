package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// NewManager is called with the state directory (.bc), not workspace root
	mgr := NewManager("/tmp/test-workspace/.bc")
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	// Should create plugins dir inside state dir
	expectedDir := "/tmp/test-workspace/.bc/plugins"
	if mgr.pluginsDir != expectedDir {
		t.Errorf("pluginsDir = %q, want %q", mgr.pluginsDir, expectedDir)
	}

	if len(mgr.registries) != 1 {
		t.Errorf("len(registries) = %d, want 1", len(mgr.registries))
	}

	if mgr.registries[0].URL != DefaultRegistry {
		t.Errorf("registry URL = %q, want %q", mgr.registries[0].URL, DefaultRegistry)
	}
}

func TestManagerList(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace/.bc")

	// Empty list initially
	plugins := mgr.List()
	if len(plugins) != 0 {
		t.Errorf("len(plugins) = %d, want 0", len(plugins))
	}
}

func TestManagerGet(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace/.bc")

	// Plugin not found
	_, ok := mgr.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent plugin")
	}
}

func TestValidateManifest(t *testing.T) {
	//nolint:govet // fieldalignment: test struct ordering for readability
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{
			name: "valid manifest",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    TypeTool,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			m: Manifest{
				Version: "1.0.0",
				Type:    TypeTool,
			},
			wantErr: true,
		},
		{
			name: "missing version",
			m: Manifest{
				Name: "test-plugin",
				Type: TypeTool,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "agent type",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    TypeAgent,
			},
			wantErr: false,
		},
		{
			name: "role type",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    TypeRole,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManifest(&tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// setupTestPlugin creates a temporary directory with a plugin for testing.
// Returns the temp dir, plugin dir, and a cleanup function.
func setupTestPlugin(t *testing.T, manifest string) (string, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pluginDir := filepath.Join(tmpDir, "test-plugin")
	if err = os.MkdirAll(pluginDir, 0750); err != nil {
		os.RemoveAll(tmpDir) //nolint:errcheck // cleanup on error
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	if err = os.WriteFile(filepath.Join(pluginDir, "plugin.toml"), []byte(manifest), 0600); err != nil {
		os.RemoveAll(tmpDir) //nolint:errcheck // cleanup on error
		t.Fatalf("failed to write manifest: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir) //nolint:errcheck // cleanup in test
	}

	return tmpDir, pluginDir, cleanup
}

func TestManagerInstallLocal(t *testing.T) {
	manifest := `name = "test-plugin"
version = "1.0.0"
description = "A test plugin"
type = "tool"
entrypoint = "main.go"
`
	tmpDir, pluginDir, cleanup := setupTestPlugin(t, manifest)
	defer cleanup()

	// Create manager with temp workspace
	mgr := NewManager(tmpDir)
	if err := mgr.Load(context.Background()); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Install plugin
	p, err := mgr.Install(context.Background(), pluginDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if p.Manifest.Name != "test-plugin" {
		t.Errorf("plugin name = %q, want %q", p.Manifest.Name, "test-plugin")
	}

	if p.State != StateEnabled {
		t.Errorf("plugin state = %q, want %q", p.State, StateEnabled)
	}

	// Verify plugin is listed
	plugins := mgr.List()
	if len(plugins) != 1 {
		t.Errorf("len(plugins) = %d, want 1", len(plugins))
	}

	// Try to install again (should fail)
	_, err = mgr.Install(context.Background(), pluginDir)
	if err == nil {
		t.Error("Install() should fail for already installed plugin")
	}
}

func TestManagerEnableDisable(t *testing.T) {
	manifest := `name = "test-plugin"
version = "1.0.0"
type = "tool"
entrypoint = "main.go"
`
	tmpDir, pluginDir, cleanup := setupTestPlugin(t, manifest)
	defer cleanup()

	// Create manager and install plugin
	mgr := NewManager(tmpDir)
	if err := mgr.Load(context.Background()); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if _, err := mgr.Install(context.Background(), pluginDir); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	// Disable plugin
	if err := mgr.Disable("test-plugin"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	p, ok := mgr.Get("test-plugin")
	if !ok {
		t.Fatal("plugin not found after disable")
	}
	if p.State != StateDisabled {
		t.Errorf("state = %q, want %q", p.State, StateDisabled)
	}

	// Enable plugin
	if err := mgr.Enable("test-plugin"); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	p, _ = mgr.Get("test-plugin")
	if p.State != StateEnabled {
		t.Errorf("state = %q, want %q", p.State, StateEnabled)
	}

	// Enable/Disable nonexistent plugin
	if err := mgr.Enable("nonexistent"); err == nil {
		t.Error("Enable() should fail for nonexistent plugin")
	}
	if err := mgr.Disable("nonexistent"); err == nil {
		t.Error("Disable() should fail for nonexistent plugin")
	}
}

func TestManagerUninstall(t *testing.T) {
	manifest := `name = "test-plugin"
version = "1.0.0"
type = "tool"
entrypoint = "main.go"
`
	tmpDir, pluginDir, cleanup := setupTestPlugin(t, manifest)
	defer cleanup()

	// Create manager and install plugin
	mgr := NewManager(tmpDir)
	if err := mgr.Load(context.Background()); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if _, err := mgr.Install(context.Background(), pluginDir); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	// Uninstall plugin
	if err := mgr.Uninstall(context.Background(), "test-plugin"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	// Verify plugin is gone
	_, ok := mgr.Get("test-plugin")
	if ok {
		t.Error("plugin should not exist after uninstall")
	}

	// Uninstall nonexistent plugin
	if err := mgr.Uninstall(context.Background(), "nonexistent"); err == nil {
		t.Error("Uninstall() should fail for nonexistent plugin")
	}
}

func TestManagerEnabled(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace/.bc")

	// Empty list initially
	enabled := mgr.Enabled("")
	if len(enabled) != 0 {
		t.Errorf("len(enabled) = %d, want 0", len(enabled))
	}

	enabled = mgr.Enabled(TypeTool)
	if len(enabled) != 0 {
		t.Errorf("len(enabled) = %d, want 0", len(enabled))
	}
}

func TestTypeConstants(t *testing.T) {
	if TypeAgent != "agent" {
		t.Errorf("TypeAgent = %q, want %q", TypeAgent, "agent")
	}
	if TypeTool != "tool" {
		t.Errorf("TypeTool = %q, want %q", TypeTool, "tool")
	}
	if TypeRole != "role" {
		t.Errorf("TypeRole = %q, want %q", TypeRole, "role")
	}
}

func TestStateConstants(t *testing.T) {
	if StateInstalled != "installed" {
		t.Errorf("StateInstalled = %q, want %q", StateInstalled, "installed")
	}
	if StateEnabled != "enabled" {
		t.Errorf("StateEnabled = %q, want %q", StateEnabled, "enabled")
	}
	if StateDisabled != "disabled" {
		t.Errorf("StateDisabled = %q, want %q", StateDisabled, "disabled")
	}
	if StateError != "error" {
		t.Errorf("StateError = %q, want %q", StateError, "error")
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultRegistry != "https://plugins.bc.dev" {
		t.Errorf("DefaultRegistry = %q, want %q", DefaultRegistry, "https://plugins.bc.dev")
	}
	if DefaultDirectory != "plugins" {
		t.Errorf("DefaultDirectory = %q, want %q", DefaultDirectory, "plugins")
	}
}

func TestManagerSearch(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace")

	// Search returns not implemented error
	results, err := mgr.Search(context.Background(), "test")
	if err == nil {
		t.Error("Search() should return error (not implemented)")
	}
	if results != nil {
		t.Errorf("Search() results should be nil, got %v", results)
	}
	if err != nil && err.Error() != "registry search not yet implemented" {
		t.Errorf("Search() error = %q, want 'registry search not yet implemented'", err.Error())
	}
}

func TestManagerInfo(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace")

	// Info for nonexistent plugin
	_, err := mgr.Info("nonexistent")
	if err == nil {
		t.Error("Info() should fail for nonexistent plugin")
	}

	// Add a plugin manually for testing
	mgr.plugins["test-plugin"] = &Plugin{
		Manifest: Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Type:    TypeTool,
		},
		State: StateEnabled,
		Path:  "/tmp/test-plugin",
	}

	// Info for existing plugin
	info, err := mgr.Info("test-plugin")
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}
	if info == nil {
		t.Fatal("Info() returned nil")
	}
	if info.Manifest.Name != "test-plugin" {
		t.Errorf("Info().Name = %q, want %q", info.Manifest.Name, "test-plugin")
	}
}

func TestManagerLoadWithState(t *testing.T) {
	// Create temp directory with plugins state
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	// Create plugins.json with test data
	stateFile := filepath.Join(pluginsDir, "plugins.json")
	stateData := `[{"manifest":{"name":"test","version":"1.0.0","type":"tool"},"state":"enabled","path":"/tmp/test"}]`
	if err := os.WriteFile(stateFile, []byte(stateData), 0600); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	mgr := NewManager(tmpDir)
	if err := mgr.Load(context.Background()); err != nil {
		t.Errorf("Load() error = %v", err)
	}

	// Verify plugin was loaded
	plugins := mgr.List()
	if len(plugins) != 1 {
		t.Errorf("len(plugins) = %d, want 1", len(plugins))
	}
}

func TestManagerLoadInvalidJSON(t *testing.T) {
	// Create temp directory with invalid plugins state
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	// Create invalid plugins.json
	stateFile := filepath.Join(pluginsDir, "plugins.json")
	if err := os.WriteFile(stateFile, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	mgr := NewManager(tmpDir)
	err := mgr.Load(context.Background())
	if err == nil {
		t.Error("Load() should fail with invalid JSON")
	}
}

func TestManagerEnabledWithPlugins(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace")

	// Add plugins with different states and types
	mgr.plugins["tool1"] = &Plugin{
		Manifest: Manifest{Name: "tool1", Type: TypeTool},
		State:    StateEnabled,
	}
	mgr.plugins["tool2"] = &Plugin{
		Manifest: Manifest{Name: "tool2", Type: TypeTool},
		State:    StateDisabled,
	}
	mgr.plugins["agent1"] = &Plugin{
		Manifest: Manifest{Name: "agent1", Type: TypeAgent},
		State:    StateEnabled,
	}

	// Get all enabled
	enabled := mgr.Enabled("")
	if len(enabled) != 2 {
		t.Errorf("len(Enabled('')) = %d, want 2", len(enabled))
	}

	// Get enabled tools only
	enabledTools := mgr.Enabled(TypeTool)
	if len(enabledTools) != 1 {
		t.Errorf("len(Enabled(TypeTool)) = %d, want 1", len(enabledTools))
	}

	// Get enabled agents only
	enabledAgents := mgr.Enabled(TypeAgent)
	if len(enabledAgents) != 1 {
		t.Errorf("len(Enabled(TypeAgent)) = %d, want 1", len(enabledAgents))
	}

	// Get enabled roles (none)
	enabledRoles := mgr.Enabled(TypeRole)
	if len(enabledRoles) != 0 {
		t.Errorf("len(Enabled(TypeRole)) = %d, want 0", len(enabledRoles))
	}
}
