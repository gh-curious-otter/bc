package config

import (
	"testing"
	"time"
)

// --- Scalar defaults ---

// Note: Name and Version are now in Roster as NameLegacy and VersionLegacy

// --- WorkspaceLegacyConfig ---

func TestWorkspaceLegacyDefaults(t *testing.T) {
	if WorkspaceLegacy.StateDir != ".bc" {
		t.Errorf("WorkspaceLegacy.StateDir = %q, want %q", WorkspaceLegacy.StateDir, ".bc")
	}
	if WorkspaceLegacy.MaxWorkers != 3 {
		t.Errorf("WorkspaceLegacy.MaxWorkers = %d, want %d", WorkspaceLegacy.MaxWorkers, 3)
	}
}

// --- AgentLegacyConfig ---

func TestAgentLegacyDefaults(t *testing.T) {
	if AgentLegacy.Command == "" {
		t.Error("AgentLegacy.Command should not be empty")
	}
	if AgentLegacy.CoordinatorName != "root" {
		t.Errorf("AgentLegacy.CoordinatorName = %q, want %q", AgentLegacy.CoordinatorName, "root")
	}
	if AgentLegacy.WorkerPrefix != "worker" {
		t.Errorf("AgentLegacy.WorkerPrefix = %q, want %q", AgentLegacy.WorkerPrefix, "worker")
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

func TestAgentLegacyConfigZeroValue(t *testing.T) {
	var ac AgentLegacyConfig
	if ac.Command != "" {
		t.Error("zero-value AgentLegacyConfig.Command should be empty")
	}
	if ac.CoordinatorName != "" {
		t.Error("zero-value AgentLegacyConfig.CoordinatorName should be empty")
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
