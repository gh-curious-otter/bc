package config

import (
	"testing"
	"time"
)

// --- Scalar defaults ---

func TestName(t *testing.T) {
	if Name != "bc" {
		t.Errorf("Name = %q, want %q", Name, "bc")
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", Version, "0.1.0")
	}
}

// --- WorkspaceConfig ---

func TestWorkspaceDefaults(t *testing.T) {
	if Workspace.StateDir != ".bc" {
		t.Errorf("Workspace.StateDir = %q, want %q", Workspace.StateDir, ".bc")
	}
	if Workspace.MaxWorkers != 3 {
		t.Errorf("Workspace.MaxWorkers = %d, want %d", Workspace.MaxWorkers, 3)
	}
}

// --- AgentConfig ---

func TestAgentDefaults(t *testing.T) {
	if Agent.Command == "" {
		t.Error("Agent.Command should not be empty")
	}
	if Agent.CoordinatorName != "coordinator" {
		t.Errorf("Agent.CoordinatorName = %q, want %q", Agent.CoordinatorName, "coordinator")
	}
	if Agent.WorkerPrefix != "worker" {
		t.Errorf("Agent.WorkerPrefix = %q, want %q", Agent.WorkerPrefix, "worker")
	}
}

// --- TmuxConfig ---

func TestTmuxDefaults(t *testing.T) {
	if Tmux.SessionPrefix != "bc-" {
		t.Errorf("Tmux.SessionPrefix = %q, want %q", Tmux.SessionPrefix, "bc-")
	}
}

// --- TuiConfig ---

func TestTuiDefaults(t *testing.T) {
	if Tui.RefreshInterval != 2*time.Second {
		t.Errorf("Tui.RefreshInterval = %v, want %v", Tui.RefreshInterval, 2*time.Second)
	}
	if Tui.Theme != "ayu-dark" {
		t.Errorf("Tui.Theme = %q, want %q", Tui.Theme, "ayu-dark")
	}
}

// --- CostsConfig ---

func TestCostsDefaults(t *testing.T) {
	if !Costs.Enabled {
		t.Error("Costs.Enabled should be true by default")
	}
	if Costs.Limit != 100 {
		t.Errorf("Costs.Limit = %f, want %f", Costs.Limit, 100.0)
	}
	if Costs.WarnThreshold != 10 {
		t.Errorf("Costs.WarnThreshold = %f, want %f", Costs.WarnThreshold, 10.0)
	}
}

// --- Agents list ---

func TestAgentsNotEmpty(t *testing.T) {
	if len(Agents) == 0 {
		t.Fatal("Agents list should not be empty")
	}
}

func TestAgentsHaveRequiredFields(t *testing.T) {
	for i, a := range Agents {
		if a.Name == "" {
			t.Errorf("Agents[%d].Name is empty", i)
		}
		if a.Command == "" {
			t.Errorf("Agents[%d].Command is empty (name=%q)", i, a.Name)
		}
	}
}

func TestAgentsContainsClaude(t *testing.T) {
	found := false
	for _, a := range Agents {
		if a.Name == "claude" {
			found = true
			if a.Command == "" {
				t.Error("claude agent command should not be empty")
			}
			break
		}
	}
	if !found {
		t.Error("Agents list should contain a 'claude' entry")
	}
}

func TestAgentsNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, a := range Agents {
		if seen[a.Name] {
			t.Errorf("duplicate agent name: %q", a.Name)
		}
		seen[a.Name] = true
	}
}

// --- Roles list ---

func TestRolesNotEmpty(t *testing.T) {
	if len(Roles) == 0 {
		t.Fatal("Roles list should not be empty")
	}
}

func TestRolesHaveRequiredFields(t *testing.T) {
	for i, r := range Roles {
		if r.Name == "" {
			t.Errorf("Roles[%d].Name is empty", i)
		}
		if r.Description == "" {
			t.Errorf("Roles[%d].Description is empty (name=%q)", i, r.Name)
		}
		if len(r.Permissions) == 0 {
			t.Errorf("Roles[%d].Permissions is empty (name=%q)", i, r.Name)
		}
	}
}

func TestRolesContainsExpectedRoles(t *testing.T) {
	expected := map[string]bool{
		"product_manager": false,
		"manager":         false,
		"engineer":        false,
	}
	for _, r := range Roles {
		if _, ok := expected[r.Name]; ok {
			expected[r.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected role %q not found in Roles", name)
		}
	}
}

func TestRolesNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, r := range Roles {
		if seen[r.Name] {
			t.Errorf("duplicate role name: %q", r.Name)
		}
		seen[r.Name] = true
	}
}

func TestManagerRoleHasSpawnPermission(t *testing.T) {
	for _, r := range Roles {
		if r.Name == "manager" {
			for _, p := range r.Permissions {
				if p == "spawn" {
					return
				}
			}
			t.Error("manager role should have 'spawn' permission")
		}
	}
}

func TestEngineerRoleHasGitPermissions(t *testing.T) {
	for _, r := range Roles {
		if r.Name == "engineer" {
			perms := make(map[string]bool)
			for _, p := range r.Permissions {
				perms[p] = true
			}
			if !perms["git.branch"] {
				t.Error("engineer role should have 'git.branch' permission")
			}
			if !perms["git.commit"] {
				t.Error("engineer role should have 'git.commit' permission")
			}
			return
		}
	}
}

// --- Struct zero-value / mutability tests ---

func TestAgentConfigZeroValue(t *testing.T) {
	var ac AgentConfig
	if ac.Command != "" {
		t.Error("zero-value AgentConfig.Command should be empty")
	}
	if ac.CoordinatorName != "" {
		t.Error("zero-value AgentConfig.CoordinatorName should be empty")
	}
}

func TestCostsConfigZeroValue(t *testing.T) {
	var cc CostsConfig
	if cc.Enabled {
		t.Error("zero-value CostsConfig.Enabled should be false")
	}
	if cc.Limit != 0 {
		t.Error("zero-value CostsConfig.Limit should be 0")
	}
}

func TestAgentsSliceMutability(t *testing.T) {
	// Verify that package-level Agents slice can be modified
	// (used by tests that override config, e.g. agent_test.go)
	original := make([]AgentsItem, len(Agents))
	copy(original, Agents)
	defer func() { Agents = original }()

	Agents = append(Agents, AgentsItem{
		Name:    "test-tool",
		Command: "test-cmd",
	})

	found := false
	for _, a := range Agents {
		if a.Name == "test-tool" {
			found = true
		}
	}
	if !found {
		t.Error("should be able to append to Agents slice")
	}
}
