package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
)

// --- Channel Send Tests ---

func TestChannelSend_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel", "send", "test-channel", "hello")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestChannelSend_NonexistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "send", "nonexistent", "hello")
	if err == nil {
		t.Fatal("expected error for nonexistent channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestChannelSend_EmptyChannel(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create channel with no members
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("empty-channel"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "send", "empty-channel", "hello")
	if err != nil {
		t.Fatalf("channel send error: %v", err)
	}
	if !strings.Contains(stdout, "no members") {
		t.Errorf("expected 'no members' message, got: %s", stdout)
	}
}

func TestChannelSend_WithMemberNotFound(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create channel with a member that doesn't exist as an agent
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	ch, err := store.Create("test-channel")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	ch.Members = []string{"nonexistent-agent"}
	if saveErr := store.Save(); saveErr != nil {
		t.Fatalf("failed to save: %v", saveErr)
	}

	stdout, _, err := executeIntegrationCmd("channel", "send", "test-channel", "hello", "world")
	if err != nil {
		t.Fatalf("channel send error: %v", err)
	}
	if !strings.Contains(stdout, "agent not found") {
		t.Errorf("expected 'agent not found' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "0/1") {
		t.Errorf("expected 'Sent to 0/1' in output, got: %s", stdout)
	}
}

func TestChannelSend_WithStoppedAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a stopped agent
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"stopped-agent": {
			Name:      "stopped-agent",
			Role:      agent.RoleEngineer,
			State:     agent.StateStopped,
			Session:   "bc-stopped",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	})

	// Create channel with the stopped agent as member
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	ch, err := store.Create("test-channel")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	ch.Members = []string{"stopped-agent"}
	if saveErr := store.Save(); saveErr != nil {
		t.Fatalf("failed to save: %v", saveErr)
	}

	stdout, _, err := executeIntegrationCmd("channel", "send", "test-channel", "hello")
	if err != nil {
		t.Fatalf("channel send error: %v", err)
	}
	if !strings.Contains(stdout, "agent stopped") {
		t.Errorf("expected 'agent stopped' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "0/1") {
		t.Errorf("expected 'Sent to 0/1' in output, got: %s", stdout)
	}
}

func TestChannelSend_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "send")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	// Cobra should complain about missing arguments
	if !strings.Contains(err.Error(), "requires at least 2 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelSend_RequiresMessage(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "send", "test-channel")
	if err == nil {
		t.Fatal("expected error for missing message, got nil")
	}
	if !strings.Contains(err.Error(), "requires at least 2 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

// --- Channel Create Tests ---

func TestChannelCreate_RequiresName(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "create")
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelCreate_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel", "create", "test")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

// --- Channel Add/Remove Member Tests ---

func TestChannelAdd_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel", "add", "test-channel", "agent")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestChannelAdd_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "add")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "requires at least 2 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelAdd_NonexistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Adding to nonexistent channel prints warning but doesn't error
	stdout, _, err := executeIntegrationCmd("channel", "add", "nonexistent", "agent1")
	if err != nil {
		t.Fatalf("channel add error: %v", err)
	}
	// Should show warning about nonexistent channel and add 0 members
	if !strings.Contains(stdout, "Warning") {
		t.Errorf("expected warning in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Added 0") {
		t.Errorf("expected 'Added 0' in output, got: %s", stdout)
	}
}

func TestChannelRemove_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "remove")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelRemove_NonexistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "remove", "nonexistent", "agent1")
	if err == nil {
		t.Fatal("expected error for nonexistent channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- Channel Join/Leave Tests ---

func TestChannelJoin_RequiresArg(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "join")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelLeave_RequiresArg(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "leave")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

// --- Channel History Tests ---

func TestChannelHistory_RequiresArg(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "history")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelHistory_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel", "history", "test")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

// --- Channel Delete Tests ---

func TestChannelDelete_RequiresArg(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "delete")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

// Note: TestChannelCommandStructure and TestChannelListFlags are in channel_e2e_test.go

// seedAgents helper is defined in cmd_integration_test.go
