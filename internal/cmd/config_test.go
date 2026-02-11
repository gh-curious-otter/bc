package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

func TestConfigShow(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "show")
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	// Check that output contains expected sections
	expectedSections := []string{
		"[workspace]",
		"[worktrees]",
		"[tools]",
		"[memory]",
		"[beads]",
		"[channels]",
		"[roster]",
	}

	for _, section := range expectedSections {
		if !strings.Contains(stdout, section) {
			t.Errorf("expected output to contain %s, got:\n%s", section, stdout)
		}
	}
}

func TestConfigShowSection(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "show", "roster")
	if err != nil {
		t.Fatalf("config show roster failed: %v", err)
	}

	if !strings.Contains(stdout, "engineers") {
		t.Errorf("expected output to contain 'engineers', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "tech_leads") {
		t.Errorf("expected output to contain 'tech_leads', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "qa") {
		t.Errorf("expected output to contain 'qa', got:\n%s", stdout)
	}
}

func TestConfigShowJSON(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "show", "roster", "--json")
	if err != nil {
		t.Fatalf("config show --json failed: %v", err)
	}

	// Parse JSON output
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Check expected fields
	if _, ok := data["Engineers"]; !ok {
		t.Error("expected 'Engineers' field in JSON output")
	}
	if _, ok := data["TechLeads"]; !ok {
		t.Error("expected 'TechLeads' field in JSON output")
	}
	if _, ok := data["QA"]; !ok {
		t.Error("expected 'QA' field in JSON output")
	}
}

func TestConfigGet(t *testing.T) {
	_ = setupTestWorkspace(t)

	tests := []struct {
		key      string
		expected string
	}{
		{"roster.engineers", "4"},
		{"roster.tech_leads", "2"},
		{"roster.qa", "2"},
		{"tools.default", "claude"},
		{"memory.backend", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			stdout, _, err := executeIntegrationCmd("config", "get", tt.key)
			if err != nil {
				t.Fatalf("config get %s failed: %v", tt.key, err)
			}

			stdout = strings.TrimSpace(stdout)
			if stdout != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, stdout)
			}
		})
	}
}

func TestConfigGetInvalidKey(t *testing.T) {
	_ = setupTestWorkspace(t)

	_, _, err := executeIntegrationCmd("config", "get", "invalid.key")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}

	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("expected 'unknown config key' error, got: %v", err)
	}
}

func TestConfigSet(t *testing.T) {
	_ = setupTestWorkspace(t)

	tests := []struct {
		key      string
		value    string
		checkKey string
		expected string
	}{
		{"roster.engineers", "6", "roster.engineers", "6"},
		{"roster.tech_leads", "3", "roster.tech_leads", "3"},
		{"tools.claude.command", "claude --force", "tools.claude.command", "claude --force"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// Set the value
			stdout, _, err := executeIntegrationCmd("config", "set", tt.key, tt.value)
			if err != nil {
				t.Fatalf("config set %s=%s failed: %v", tt.key, tt.value, err)
			}

			if !strings.Contains(stdout, "Set "+tt.key) {
				t.Errorf("expected confirmation message, got: %s", stdout)
			}

			// Verify the value was set
			stdout, _, err = executeIntegrationCmd("config", "get", tt.checkKey)
			if err != nil {
				t.Fatalf("config get %s failed: %v", tt.checkKey, err)
			}

			stdout = strings.TrimSpace(stdout)
			if stdout != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, stdout)
			}
		})
	}
}

func TestConfigSetInvalidValue(t *testing.T) {
	_ = setupTestWorkspace(t)

	tests := []struct {
		key   string
		value string
		desc  string
	}{
		{"roster.engineers", "not-a-number", "invalid integer"},
		{"worktrees.auto_cleanup", "not-a-bool", "invalid boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, _, err := executeIntegrationCmd("config", "set", tt.key, tt.value)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.desc)
			}
		})
	}
}

func TestConfigList(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "list")
	if err != nil {
		t.Fatalf("config list failed: %v", err)
	}

	expectedKeys := []string{
		"workspace.name",
		"workspace.version",
		"worktrees.path",
		"tools.default",
		"roster.engineers",
		"roster.tech_leads",
		"roster.qa",
		"memory.backend",
	}

	for _, key := range expectedKeys {
		if !strings.Contains(stdout, key) {
			t.Errorf("expected output to contain key %s", key)
		}
	}
}

func TestConfigListJSON(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "list", "--json")
	if err != nil {
		t.Fatalf("config list --json failed: %v", err)
	}

	// Parse JSON output
	var keys []string
	if err := json.Unmarshal([]byte(stdout), &keys); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(keys) == 0 {
		t.Error("expected at least one config key")
	}

	// Check for expected keys
	hasWorkspaceName := false
	for _, key := range keys {
		if key == "workspace.name" {
			hasWorkspaceName = true
			break
		}
	}
	if !hasWorkspaceName {
		t.Error("expected 'workspace.name' in keys list")
	}
}

func TestConfigValidate(t *testing.T) {
	_ = setupTestWorkspace(t)

	stdout, _, err := executeIntegrationCmd("config", "validate")
	if err != nil {
		t.Fatalf("config validate failed: %v", err)
	}

	if !strings.Contains(stdout, "Config is valid") {
		t.Errorf("expected 'Config is valid' message, got: %s", stdout)
	}
}

func TestConfigValidateInvalid(t *testing.T) {
	projectDir := setupTestWorkspace(t)

	// Break the config by setting invalid version
	configPath := workspace.ConfigPath(projectDir)
	if err := os.WriteFile(configPath, []byte("[workspace]\nname=\"test\"\nversion=99\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, _, err := executeIntegrationCmd("config", "validate")
	if err == nil {
		t.Fatal("expected validation error for invalid config")
	}

	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version validation error, got: %v", err)
	}
}

func TestConfigNoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	_, _, err := executeIntegrationCmd("config", "show")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}

	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected 'not in a bc workspace' error, got: %v", err)
	}
}

func TestConfigCommandStructure(t *testing.T) {
	subcommands := configCmd.Commands()

	expectedCmds := map[string]bool{
		"show":     false,
		"get":      false,
		"set":      false,
		"list":     false,
		"edit":     false,
		"validate": false,
		"reset":    false,
	}

	for _, cmd := range subcommands {
		if _, ok := expectedCmds[cmd.Name()]; ok {
			expectedCmds[cmd.Name()] = true
		}
	}

	for name, found := range expectedCmds {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}
