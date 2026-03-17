package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/rpuneet/bc/pkg/agent"
)

// --- Test helpers ---

// clearWorkspaceEnv clears BC_WORKSPACE env var and returns a cleanup function.
// Use this in tests that expect no workspace to be found.
func clearWorkspaceEnv(t *testing.T) func() {
	t.Helper()
	origBCWorkspace := os.Getenv("BC_WORKSPACE")
	_ = os.Unsetenv("BC_WORKSPACE")
	return func() {
		if origBCWorkspace != "" {
			_ = os.Setenv("BC_WORKSPACE", origBCWorkspace)
		}
	}
}

// executeCmd runs a cobra command with the given args.
func executeCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset flags to prevent leaking state
	// Reset root persistent flags first
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		// Also reset value to default for boolean flags
		if f.Value.Type() == "bool" {
			_ = f.Value.Set("false")
		}
	})
	// Reset subcommand flags
	for _, sub := range rootCmd.Commands() {
		sub.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}

	err := rootCmd.Execute()
	return buf.String(), err
}

// setupTestWorkspace creates a temporary bc workspace and changes into it.
// Returns the workspace root directory path (for use with demon.NewStore, etc.).
func setupTestWorkspace(t *testing.T) string {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Clear BC_WORKSPACE to ensure tests use the temp workspace, not outer workspace
	origBCWorkspace := os.Getenv("BC_WORKSPACE")
	_ = os.Unsetenv("BC_WORKSPACE")
	t.Cleanup(func() {
		if origBCWorkspace != "" {
			_ = os.Setenv("BC_WORKSPACE", origBCWorkspace)
		}
	})

	tmpDir := t.TempDir()
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatalf("failed to create .bc/agents: %v", err)
	}
	// demons directory removed in CLI restructure (#1916)
	// Create minimal config.toml for v2 workspace detection
	configPath := filepath.Join(bcDir, "config.toml")
	configContent := `[workspace]
name = "test"
version = 2

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config.toml: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	return tmpDir // Return workspace root, not .bc directory
}

// --- formatDuration tests ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		want  string
		input time.Duration
	}{
		{"0s", 0},
		{"30s", 30 * time.Second},
		{"1m 30s", 90 * time.Second},
		{"1h 1m", 3661 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.input)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- colorState tests ---

func TestColorState(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string
	}{
		{agent.StateIdle, "idle"},
		{agent.StateWorking, "working"},
		{agent.StateDone, "done"},
		{agent.StateStuck, "stuck"},
		{agent.StateError, "error"},
		{agent.StateStopped, "stopped"},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			result := colorState(tt.state)
			if !strings.Contains(result, tt.want) {
				t.Errorf("colorState(%q) = %q, should contain %q", tt.state, result, tt.want)
			}
		})
	}
}

// --- stateIcon tests ---

// --- parseRole tests ---

func TestParseRole(t *testing.T) {
	// All roles are custom now - parseRole accepts any valid alphanumeric name
	// No alias expansion (pm, coord, tl are returned as-is)
	// Empty defaults to root
	tests := []struct {
		input   string
		want    agent.Role
		wantErr bool
	}{
		{"worker", agent.Role("worker"), false},
		{"engineer", agent.Role("engineer"), false},
		{"manager", agent.Role("manager"), false},
		{"product-manager", agent.Role("product-manager"), false},
		{"pm", agent.Role("pm"), false}, // No expansion, returned as-is
		{"coordinator", agent.Role("coordinator"), false},
		{"coord", agent.Role("coord"), false}, // No expansion, returned as-is
		{"qa", agent.Role("qa"), false},
		{"WORKER", agent.Role("worker"), false},           // case insensitive (lowercased)
		{"Engineer", agent.Role("engineer"), false},       // case insensitive (lowercased)
		{"custom-role", agent.Role("custom-role"), false}, // Custom roles accepted
		{"", agent.RoleRoot, false},                       // Empty defaults to root
		{"role@invalid", "", true},                        // Format error (contains @)
		{"role with space", "", true},                     // Format error (contains space)
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRole(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- SetVersionInfo tests ---

func TestSetVersionInfo(t *testing.T) {
	// Test that SetVersionInfo doesn't panic
	SetVersionInfo("1.2.3", "abc123", "2025-01-15")
	// Reset
	SetVersionInfo("dev", "none", "unknown")
}

// --- Channel command tests ---

func TestChannelCommand_NoWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("channel", "list")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
}
