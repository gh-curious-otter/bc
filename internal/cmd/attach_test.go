package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestAttachNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("attach", "coordinator")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestAttachAgentNotRunning(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create agents dir and seed a stopped agent
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"coordinator": {
			Name:      "coordinator",
			Role:      agent.RoleCoordinator,
			State:     agent.StateStopped,
			Session:   "bc-coordinator",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	})

	_, _, err := executeIntegrationCmd("attach", "coordinator")
	if err == nil {
		t.Fatal("expected error for non-running agent, got nil")
	}
	// Should fail because tmux session doesn't exist
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}

func TestAttachNonexistentAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create agents dir
	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	_, _, err := executeIntegrationCmd("attach", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent, got nil")
	}
	if !strings.Contains(err.Error(), "not running") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not running/not found error, got: %v", err)
	}
}

func TestAttachRequiresArg(t *testing.T) {
	_, _, err := executeIntegrationCmd("attach")
	if err == nil {
		t.Fatal("expected error for missing arg, got nil")
	}
	// Cobra should complain about missing argument
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}
