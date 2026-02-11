package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestDownNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("down")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestDownEmptyWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create agents dir
	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("down")
	if err != nil {
		t.Fatalf("down returned error: %v", err)
	}
	if !strings.Contains(stdout, "No agents running") {
		t.Errorf("expected 'No agents running', got: %s", stdout)
	}
}

func TestDownWithAgents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed some agents (already stopped, so StopAgent will just mark them)
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"coordinator": {
			Name:      "coordinator",
			Role:      agent.RoleRoot,
			State:     agent.StateStopped,
			Session:   "bc-coord",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
		"worker-01": {
			Name:      "worker-01",
			Role:      agent.Role("worker"),
			State:     agent.StateStopped,
			Session:   "bc-worker-01",
			StartedAt: time.Now().Add(-30 * time.Minute),
		},
	})

	stdout, _, err := executeIntegrationCmd("down")
	if err != nil {
		t.Fatalf("down returned error: %v", err)
	}
	if !strings.Contains(stdout, "Stopping") {
		t.Errorf("expected 'Stopping' message, got: %s", stdout)
	}
	if !strings.Contains(stdout, "All agents stopped") {
		t.Errorf("expected 'All agents stopped', got: %s", stdout)
	}
}

func TestDownForceFlag(t *testing.T) {
	// Reset flag before test
	downForce = false
	defer func() { downForce = false }()

	// Parse --force flag
	if err := downCmd.ParseFlags([]string{"--force"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if !downForce {
		t.Error("--force flag should set downForce to true")
	}
}

func TestDownFlagDefault(t *testing.T) {
	forceFlag := downCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("force flag not found")
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("force default: got %q, want %q", forceFlag.DefValue, "false")
	}
}
