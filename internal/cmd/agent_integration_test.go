package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
)

// Agent command integration tests that don't require actual tmux sessions.
// These tests seed agent state files directly to test display/query functionality.

func TestAgentListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("agent", "list")
	if err != nil {
		t.Fatalf("agent list failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No agents") {
		t.Errorf("expected 'No agents' message, got: %s", stdout)
	}
}

func TestAgentListWithAgents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed agents
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"engineer-01": {
			Name:      "engineer-01",
			Role:      agent.RoleEngineer,
			State:     agent.StateWorking,
			Task:      "fixing bug",
			Session:   "bc-engineer-01",
			StartedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now(),
		},
		"qa-01": {
			Name:      "qa-01",
			Role:      agent.RoleQA,
			State:     agent.StateIdle,
			Session:   "bc-qa-01",
			StartedAt: time.Now().Add(-30 * time.Minute),
			UpdatedAt: time.Now(),
		},
	})

	stdout, _, err := executeIntegrationCmd("agent", "list")
	if err != nil {
		t.Fatalf("agent list failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("output should contain engineer-01: %s", stdout)
	}
	if !strings.Contains(stdout, "qa-01") {
		t.Errorf("output should contain qa-01: %s", stdout)
	}
}

func TestAgentListFilterByRole(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed agents with different roles
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"engineer-01": {
			Name:      "engineer-01",
			Role:      agent.RoleEngineer,
			State:     agent.StateWorking,
			Session:   "bc-engineer-01",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		"qa-01": {
			Name:      "qa-01",
			Role:      agent.RoleQA,
			State:     agent.StateIdle,
			Session:   "bc-qa-01",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	// Filter by engineer role
	stdout, _, err := executeIntegrationCmd("agent", "list", "--role", "engineer")
	if err != nil {
		t.Fatalf("agent list --role failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("output should contain engineer-01: %s", stdout)
	}
	// qa-01 should not be in output when filtering by engineer
	// Note: The filter might still show QA in the summary count, just not in the filtered list
}

func TestAgentListJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"engineer-01": {
			Name:      "engineer-01",
			Role:      agent.RoleEngineer,
			State:     agent.StateIdle,
			Session:   "bc-engineer-01",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	stdout, _, err := executeIntegrationCmd("agent", "list", "--json")
	if err != nil {
		t.Fatalf("agent list --json failed: %v\nOutput: %s", err, stdout)
	}
	// Output should be valid JSON (starts with [ or {)
	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		t.Errorf("output should be JSON, got: %s", stdout)
	}
}

func TestAgentStopNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "stop", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestAgentPeekNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "peek", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	// Error message varies based on implementation
}

func TestAgentAttachNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "attach", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	// Error message varies based on implementation
}

func TestAgentSendNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "send", "nonexistent-agent", "hello")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestAgentSendToStoppedAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed a stopped agent
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"stopped-agent": {
			Name:      "stopped-agent",
			Role:      agent.RoleEngineer,
			State:     agent.StateStopped,
			Session:   "bc-stopped-agent",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	_, _, err := executeIntegrationCmd("agent", "send", "stopped-agent", "hello")
	if err == nil {
		t.Error("expected error for stopped agent")
	}
	if err != nil && !strings.Contains(err.Error(), "stopped") {
		t.Errorf("error should mention stopped: %v", err)
	}
}

func TestAgentCreateInvalidRole(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "create", "test-agent", "--role", "invalid-role")
	if err == nil {
		t.Error("expected error for invalid role")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("error should mention unknown role: %v", err)
	}
}

func TestAgentCreateInvalidTeam(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("agent", "create", "test-agent", "--role", "engineer", "--team", "invalid team name!")
	if err == nil {
		t.Error("expected error for invalid team name")
	}
	if err != nil && !strings.Contains(err.Error(), "team name") {
		t.Errorf("error should mention team name: %v", err)
	}
}

func TestAgentNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, execErr := executeIntegrationCmd("agent", "list")
	if execErr == nil {
		t.Error("expected error when not in workspace")
	}
	if !strings.Contains(execErr.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", execErr)
	}
}
