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

// executeCmd runs a cobra command with the given args.
func executeCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset flags to prevent leaking state
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

	tmpDir := t.TempDir()
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatalf("failed to create .bc/agents: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(bcDir, "demons"), 0750); err != nil {
		t.Fatalf("failed to create .bc/demons: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(bcDir, "memory"), 0750); err != nil {
		t.Fatalf("failed to create .bc/memory: %v", err)
	}

	// Create minimal config.toml for v2 workspace detection
	configPath := filepath.Join(bcDir, "config.toml")
	configContent := `[workspace]
name = "test"
version = 2

[tools]
default = "claude"

[tools.claude]
command = "claude"
enabled = true

[memory]
backend = "file"
path = ".bc/memory"

[roster]
engineers = 4
tech_leads = 2
qa = 2
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

// TestColorState_NoANSIWhenNotTTY ensures that when stdout is not a TTY (e.g. piped to cat or in logs),
// colorState returns plain text with no ANSI escape codes (fixes bc-siq).
func TestColorState_NoANSIWhenNotTTY(t *testing.T) {
	old := isStdoutTerminal
	isStdoutTerminal = func() bool { return false }
	t.Cleanup(func() { isStdoutTerminal = old })

	states := []agent.State{
		agent.StateIdle, agent.StateWorking, agent.StateDone,
		agent.StateStuck, agent.StateError, agent.StateStopped,
		agent.State("unknown"),
	}
	for _, s := range states {
		result := colorState(s)
		if strings.Contains(result, "\033[") {
			t.Errorf("colorState(%q) should not contain ANSI codes when not TTY, got: %q", s, result)
		}
		if !strings.Contains(result, string(s)) {
			t.Errorf("colorState(%q) should contain state name, got: %q", s, result)
		}
	}
}

// --- stateIcon tests ---

func TestStateIcon(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string
	}{
		{agent.StateIdle, "o"},
		{agent.StateWorking, ">"},
		{agent.StateDone, "+"},
		{agent.StateStuck, "!"},
		{agent.StateError, "x"},
		{agent.StateStarting, "~"},
		{agent.StateStopped, "-"},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := stateIcon(tt.state)
			if got != tt.want {
				t.Errorf("stateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

// --- parseRole tests ---

func TestParseRole(t *testing.T) {
	tests := []struct {
		input   string
		want    agent.Role
		wantErr bool
	}{
		{"worker", agent.RoleWorker, false},
		{"engineer", agent.RoleEngineer, false},
		{"manager", agent.RoleManager, false},
		{"product-manager", agent.RoleProductManager, false},
		{"pm", agent.RoleProductManager, false},
		{"coordinator", agent.RoleCoordinator, false},
		{"coord", agent.RoleCoordinator, false},
		{"qa", agent.RoleQA, false},
		{"WORKER", agent.RoleWorker, false},     // case insensitive
		{"Engineer", agent.RoleEngineer, false}, // case insensitive
		{"invalid", "", true},
		{"", "", true},
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
