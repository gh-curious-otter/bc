package cmd

import (
	"bytes"
	"os"
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
