package config

import (
	"testing"
)

// --- AgentLegacyConfig ---

func TestAgentLegacyDefaults(t *testing.T) {
	if AgentLegacy.Command == "" {
		t.Error("AgentLegacy.Command should not be empty")
	}
}

// --- TmuxConfig ---

func TestTmuxDefaults(t *testing.T) {
	if Tmux.SessionPrefix != "bc-" {
		t.Errorf("Tmux.SessionPrefix = %q, want %q", Tmux.SessionPrefix, "bc-")
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

// --- Struct zero-value / mutability tests ---

func TestAgentLegacyConfigZeroValue(t *testing.T) {
	var ac AgentLegacyConfig
	if ac.Command != "" {
		t.Error("zero-value AgentLegacyConfig.Command should be empty")
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
