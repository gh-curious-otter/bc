package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/plugin"
)

// resetPluginFlags resets plugin command flags between tests
func resetPluginFlags() {
	pluginInitType = "tool"
}

func TestPluginListNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Clear BC_WORKSPACE to ensure tests use the temp workspace, not outer workspace
	origBCWorkspace := os.Getenv("BC_WORKSPACE")
	_ = os.Unsetenv("BC_WORKSPACE")
	defer func() {
		if origBCWorkspace != "" {
			_ = os.Setenv("BC_WORKSPACE", origBCWorkspace)
		}
	}()

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("plugin", "list")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestPluginListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("plugin", "list")
	if err != nil {
		t.Fatalf("plugin list returned error: %v", err)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Errorf("expected 'No plugins installed', got: %s", stdout)
	}
}

func TestPluginInstallNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "install", "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
	if !strings.Contains(err.Error(), "installation failed") {
		t.Errorf("expected installation error, got: %v", err)
	}
}

func TestPluginUninstallNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "uninstall", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing plugin, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestPluginEnableNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "enable", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing plugin, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestPluginDisableNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "disable", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing plugin, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestPluginInfoNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "info", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing plugin, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestPluginSearchNotImplemented(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("plugin", "search", "test")
	if err == nil {
		t.Fatal("expected error for unimplemented search, got nil")
	}
	if !strings.Contains(err.Error(), "plugin registry coming soon") {
		t.Errorf("expected 'plugin registry coming soon' error, got: %v", err)
	}
}

// setupTestPlugin creates a temporary plugin directory with manifest
func setupTestPlugin(t *testing.T, name, pluginType string) string {
	t.Helper()

	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, name)
	if err := os.MkdirAll(pluginDir, 0750); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifest := `name = "` + name + `"
version = "1.0.0"
description = "Test plugin"
type = "` + pluginType + `"
entrypoint = "main.go"
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.toml"), []byte(manifest), 0600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	return pluginDir
}

func TestPluginInstallLocal(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "test-plugin", "tool")

	stdout, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("plugin install returned error: %v", err)
	}
	if !strings.Contains(stdout, "Installed test-plugin") {
		t.Errorf("expected installation confirmation, got: %s", stdout)
	}
	if !strings.Contains(stdout, "v1.0.0") {
		t.Errorf("expected version in output, got: %s", stdout)
	}
}

func TestPluginInstallDuplicate(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "dup-plugin", "tool")

	// Install first time
	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("first install returned error: %v", err)
	}

	// Try to install again
	_, _, err = executeIntegrationCmd("plugin", "install", pluginDir)
	if err == nil {
		t.Fatal("expected error for duplicate install, got nil")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("expected 'already installed' error, got: %v", err)
	}
}

func TestPluginEnableDisable(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "toggle-plugin", "tool")

	// Install
	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	// Disable
	stdout, _, err := executeIntegrationCmd("plugin", "disable", "toggle-plugin")
	if err != nil {
		t.Fatalf("disable returned error: %v", err)
	}
	if !strings.Contains(stdout, "Disabled toggle-plugin") {
		t.Errorf("expected disable confirmation, got: %s", stdout)
	}

	// Enable
	stdout, _, err = executeIntegrationCmd("plugin", "enable", "toggle-plugin")
	if err != nil {
		t.Fatalf("enable returned error: %v", err)
	}
	if !strings.Contains(stdout, "Enabled toggle-plugin") {
		t.Errorf("expected enable confirmation, got: %s", stdout)
	}
}

func TestPluginUninstall(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "remove-plugin", "tool")

	// Install
	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	// Uninstall
	stdout, _, err := executeIntegrationCmd("plugin", "uninstall", "remove-plugin")
	if err != nil {
		t.Fatalf("uninstall returned error: %v", err)
	}
	if !strings.Contains(stdout, "Uninstalled remove-plugin") {
		t.Errorf("expected uninstall confirmation, got: %s", stdout)
	}

	// Verify it's gone
	_, _, err = executeIntegrationCmd("plugin", "info", "remove-plugin")
	if err == nil {
		t.Error("expected error for uninstalled plugin, got nil")
	}
}

func TestPluginInfo(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "info-plugin", "agent")

	// Install
	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	// Get info
	stdout, _, err := executeIntegrationCmd("plugin", "info", "info-plugin")
	if err != nil {
		t.Fatalf("info returned error: %v", err)
	}
	if !strings.Contains(stdout, "info-plugin") {
		t.Errorf("expected plugin name in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "1.0.0") {
		t.Errorf("expected version in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "agent") {
		t.Errorf("expected type in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "enabled") {
		t.Errorf("expected state in output, got: %s", stdout)
	}
}

func TestPluginListWithPlugins(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Install multiple plugins
	plugin1 := setupTestPlugin(t, "plugin-one", "tool")
	plugin2 := setupTestPlugin(t, "plugin-two", "agent")

	_, _, err := executeIntegrationCmd("plugin", "install", plugin1)
	if err != nil {
		t.Fatalf("install plugin1 returned error: %v", err)
	}

	_, _, err = executeIntegrationCmd("plugin", "install", plugin2)
	if err != nil {
		t.Fatalf("install plugin2 returned error: %v", err)
	}

	// List all
	stdout, _, err := executeIntegrationCmd("plugin", "list")
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if !strings.Contains(stdout, "plugin-one") {
		t.Errorf("expected plugin-one in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "plugin-two") {
		t.Errorf("expected plugin-two in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Installed Plugins") {
		t.Errorf("expected header in output, got: %s", stdout)
	}
}

func TestPluginListJSON(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "json-plugin", "tool")

	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("plugin", "list", "--json")
	if err != nil {
		t.Fatalf("list --json returned error: %v", err)
	}

	var plugins []*plugin.Plugin
	if err := json.Unmarshal([]byte(stdout), &plugins); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.Name != "json-plugin" {
		t.Errorf("expected plugin name 'json-plugin', got %q", plugins[0].Manifest.Name)
	}
}

func TestPluginInfoJSON(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	pluginDir := setupTestPlugin(t, "info-json-plugin", "role")

	_, _, err := executeIntegrationCmd("plugin", "install", pluginDir)
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("plugin", "info", "info-json-plugin", "--json")
	if err != nil {
		t.Fatalf("info --json returned error: %v", err)
	}

	var p plugin.Plugin
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if p.Manifest.Name != "info-json-plugin" {
		t.Errorf("expected plugin name 'info-json-plugin', got %q", p.Manifest.Name)
	}
	if p.Manifest.Type != "role" {
		t.Errorf("expected plugin type 'role', got %q", p.Manifest.Type)
	}
}

func TestPluginInit(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	resetPluginFlags()
	defer resetPluginFlags()

	stdout, _, err := executeIntegrationCmd("plugin", "init", "my-plugin")
	if err != nil {
		t.Fatalf("plugin init returned error: %v", err)
	}
	if !strings.Contains(stdout, "Created plugin scaffold") {
		t.Errorf("expected creation message, got: %s", stdout)
	}

	// Verify files were created
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	if _, err := os.Stat(filepath.Join(pluginDir, "plugin.toml")); os.IsNotExist(err) {
		t.Error("plugin.toml was not created")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "src", "main.go")); os.IsNotExist(err) {
		t.Error("src/main.go was not created")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md was not created")
	}
}

func TestPluginInitWithType(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	resetPluginFlags()
	pluginInitType = "agent"
	defer resetPluginFlags()

	stdout, _, err := executeIntegrationCmd("plugin", "init", "my-agent", "--type", "agent")
	if err != nil {
		t.Fatalf("plugin init --type agent returned error: %v", err)
	}
	if !strings.Contains(stdout, "Created plugin scaffold") {
		t.Errorf("expected creation message, got: %s", stdout)
	}

	// Verify manifest contains agent type
	manifest, err := os.ReadFile(filepath.Join(tmpDir, "my-agent", "plugin.toml")) //nolint:gosec // test code, path is safe
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if !strings.Contains(string(manifest), `type = "agent"`) {
		t.Errorf("expected type = 'agent' in manifest, got: %s", manifest)
	}
}

func TestPluginInitInvalidType(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	resetPluginFlags()
	pluginInitType = "invalid"
	defer resetPluginFlags()

	_, _, err = executeIntegrationCmd("plugin", "init", "bad-plugin", "--type", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid type") {
		t.Errorf("expected 'invalid type' error, got: %v", err)
	}
}

func TestPluginFlagDefaults(t *testing.T) {
	typeFlag := pluginInitCmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("type flag not found")
	}
	if typeFlag.DefValue != "tool" {
		t.Errorf("type default: got %q, want %q", typeFlag.DefValue, "tool")
	}
}
