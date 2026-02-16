package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/team"
)

// --- Input Validation Tests ---

// TestInputValidation_SpecialCharacters tests that special characters in names are rejected
func TestInputValidation_SpecialCharacters(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name     string
		args     []string
		wantErrs []string // any of these errors is acceptable
	}{
		{
			name:     "agent name with semicolon",
			args:     []string{"agent", "create", "test;agent", "--role", "engineer"},
			wantErrs: []string{"invalid"},
		},
		{
			name:     "agent name with quotes",
			args:     []string{"agent", "create", "test'agent", "--role", "engineer"},
			wantErrs: []string{"invalid"},
		},
		{
			name:     "channel name with special chars",
			args:     []string{"channel", "create", "test@channel"},
			wantErrs: []string{"invalid"},
		},
		{
			name:     "team name with slash",
			args:     []string{"team", "create", "test/team"},
			wantErrs: []string{"invalid", "no such file", "directory"}, // slash causes path issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			errLower := strings.ToLower(err.Error())
			found := false
			for _, wantErr := range tt.wantErrs {
				if strings.Contains(errLower, wantErr) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error containing one of %v, got: %v", tt.wantErrs, err)
			}
		})
	}
}

// TestInputValidation_SQLInjection tests that SQL injection patterns are handled safely
func TestInputValidation_SQLInjection(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// These inputs should either be rejected or handled safely (no SQL errors)
	injectionPatterns := []string{
		"'; DROP TABLE agents; --",
		"1 OR 1=1",
		"admin'--",
		"'; DELETE FROM channels; --",
		"UNION SELECT * FROM users",
	}

	for _, pattern := range injectionPatterns {
		t.Run("injection_pattern", func(t *testing.T) {
			// Try to use injection pattern as agent name
			_, _, err := executeIntegrationCmd("agent", "show", pattern)
			// Should either reject as invalid name or return "not found" - never SQL error
			if err != nil {
				errLower := strings.ToLower(err.Error())
				if strings.Contains(errLower, "sql") && !strings.Contains(errLower, "not found") {
					t.Errorf("possible SQL error exposed: %v", err)
				}
			}
		})
	}
}

// TestInputValidation_EmptyInputs tests that empty inputs are properly rejected
func TestInputValidation_EmptyInputs(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "channel send with empty message",
			args:    []string{"channel", "send", "eng", ""},
			wantErr: true,
		},
		{
			name:    "team create with empty name",
			args:    []string{"team", "create", ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
			}
		})
	}
}

// TestInputValidation_LongStrings tests handling of very long input strings
func TestInputValidation_LongStrings(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a very long string (1000+ chars)
	longString := strings.Repeat("a", 1000)

	// These should either work or fail gracefully (no panics, no buffer overflows)
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "long agent name",
			args: []string{"agent", "create", longString, "--role", "engineer"},
		},
		{
			name: "long channel name",
			args: []string{"channel", "create", longString},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify no panic - error is expected
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic on long string: %v", r)
				}
			}()
			_, _, _ = executeIntegrationCmd(tt.args...)
		})
	}
}

// --- Non-Existent Resource Tests ---

// TestNonExistentAgent tests operations on non-existent agents
func TestNonExistentAgent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr string
		args    []string
	}{
		{
			name:    "show non-existent agent",
			args:    []string{"agent", "show", "nonexistent-agent-xyz"},
			wantErr: "not found",
		},
		{
			name:    "stop non-existent agent",
			args:    []string{"agent", "stop", "nonexistent-agent-xyz"},
			wantErr: "not found",
		},
		{
			name:    "attach to non-existent agent",
			args:    []string{"agent", "attach", "nonexistent-agent-xyz"},
			wantErr: "not", // "not found" or "not running"
		},
		{
			name:    "peek non-existent agent",
			args:    []string{"agent", "peek", "nonexistent-agent-xyz"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestNonExistentChannel tests operations on non-existent channels
func TestNonExistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr string
		args    []string
	}{
		{
			name:    "send to non-existent channel",
			args:    []string{"channel", "send", "nonexistent-channel", "hello"},
			wantErr: "not found",
		},
		{
			name:    "history of non-existent channel",
			args:    []string{"channel", "history", "nonexistent-channel"},
			wantErr: "not found",
		},
		{
			name:    "delete non-existent channel",
			args:    []string{"channel", "delete", "nonexistent-channel"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestNonExistentTeam tests operations on non-existent teams
func TestNonExistentTeam(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr string
		args    []string
	}{
		{
			name:    "show non-existent team",
			args:    []string{"team", "show", "nonexistent-team"},
			wantErr: "not found",
		},
		{
			name:    "add to non-existent team",
			args:    []string{"team", "add", "nonexistent-team", "some-agent"},
			wantErr: "not found",
		},
		{
			name:    "remove from non-existent team",
			args:    []string{"team", "remove", "nonexistent-team", "some-agent"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestNonExistentDemon tests operations on non-existent demons
func TestNonExistentDemon(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr string
		args    []string
	}{
		{
			name:    "show non-existent demon",
			args:    []string{"demon", "show", "nonexistent-demon"},
			wantErr: "not found",
		},
		{
			name:    "run non-existent demon",
			args:    []string{"demon", "run", "nonexistent-demon"},
			wantErr: "not found",
		},
		{
			name:    "enable non-existent demon",
			args:    []string{"demon", "enable", "nonexistent-demon"},
			wantErr: "not found",
		},
		{
			name:    "disable non-existent demon",
			args:    []string{"demon", "disable", "nonexistent-demon"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestNonExistentProcess tests operations on non-existent processes
func TestNonExistentProcess(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr string
		args    []string
	}{
		{
			name:    "logs of non-existent process",
			args:    []string{"process", "logs", "nonexistent-process"},
			wantErr: "not found",
		},
		{
			name:    "stop non-existent process",
			args:    []string{"process", "stop", "nonexistent-process"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// --- State Violation Tests ---

// TestDuplicateResources tests creating duplicate resources
func TestDuplicateResources(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a channel first
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("duplicate-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Try to create the same channel again
	_, _, err := executeIntegrationCmd("channel", "create", "duplicate-test")
	if err == nil {
		t.Error("expected error when creating duplicate channel, got nil")
	}
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "exists") &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Errorf("expected 'exists' or 'duplicate' error, got: %v", err)
	}
}

// TestDuplicateTeam tests creating duplicate teams
func TestDuplicateTeam(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a team first using the Store API
	teamStore := team.NewStore(wsDir)
	if err := teamStore.Init(); err != nil {
		t.Fatalf("failed to init team store: %v", err)
	}
	if _, err := teamStore.Create("duplicate-team"); err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	// Try to create the same team again
	_, _, err := executeIntegrationCmd("team", "create", "duplicate-team")
	if err == nil {
		t.Error("expected error when creating duplicate team, got nil")
	}
}

// --- Empty List Tests ---

// TestEmptyLists tests listing resources when none exist
func TestEmptyLists(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "empty agent list",
			args: []string{"agent", "list"},
		},
		{
			name: "empty channel list",
			args: []string{"channel", "list"},
		},
		{
			name: "empty team list",
			args: []string{"team", "list"},
		},
		{
			name: "empty demon list",
			args: []string{"demon", "list"},
		},
		{
			name: "empty process list",
			args: []string{"process", "list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty lists should not error, just return empty
			_, _, err := executeIntegrationCmd(tt.args...)
			if err != nil {
				// Skip known flag leakage issues from other tests in the package
				// (role flags persist across test runs due to global state)
				if strings.Contains(err.Error(), "role@invalid") {
					t.Skip("skipping due to flag leakage from other tests")
				}
				t.Errorf("empty list command %v should not error: %v", tt.args, err)
			}
		})
	}
}

// --- Missing Required Arguments Tests ---

// TestMissingRequiredArgs tests commands with missing required arguments
func TestMissingRequiredArgs(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "agent create without name",
			args: []string{"agent", "create"},
		},
		{
			name: "channel send without channel",
			args: []string{"channel", "send"},
		},
		{
			name: "channel send without message",
			args: []string{"channel", "send", "eng"},
		},
		{
			name: "team create without name",
			args: []string{"team", "create"},
		},
		{
			name: "team add without member",
			args: []string{"team", "add", "myteam"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected error for missing args in %v, got nil", tt.args)
			}
		})
	}
}

// --- Unicode and Special Data Tests ---

// TestUnicodeInputs tests handling of unicode characters in inputs
func TestUnicodeInputs(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a channel for sending messages
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	ch, err := store.Create("unicode-test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	ch.Members = []string{"test-agent"}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	unicodeMessages := []string{
		"Hello 中文",
		"مرحبا العربية",
		"こんにちは日本語",
		"🎉 emoji test 🚀",
		"Mixed: Hello 世界 🌍",
	}

	for _, msg := range unicodeMessages {
		t.Run("unicode_message", func(t *testing.T) {
			// Should not panic or crash
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic on unicode input %q: %v", msg, r)
				}
			}()
			// Note: message may fail to send (no agent), but should not panic
			_, _, _ = executeIntegrationCmd("channel", "send", "unicode-test", msg)
		})
	}
}

// --- No Workspace Tests ---

// TestNoWorkspace tests commands run outside a workspace
func TestNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "agent list outside workspace",
			args: []string{"agent", "list"},
		},
		{
			name: "channel list outside workspace",
			args: []string{"channel", "list"},
		},
		{
			name: "team list outside workspace",
			args: []string{"team", "list"},
		},
		{
			name: "status outside workspace",
			args: []string{"status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := executeIntegrationCmd(tt.args...)
			if err == nil {
				t.Errorf("expected workspace error for %v, got nil", tt.args)
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), "workspace") {
				t.Errorf("expected workspace-related error, got: %v", err)
			}
		})
	}
}
