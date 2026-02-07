package cmd

import (
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

// --- isValidTeamName Tests ---

func TestIsValidTeamName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"alphanumeric", "platform", true},
		{"with numbers", "team123", true},
		{"with hyphen", "core-team", true},
		{"with underscore", "core_team", true},
		{"mixed", "Platform-Team_01", true},
		{"uppercase", "PLATFORM", true},
		{"empty", "", false},
		{"with space", "core team", false},
		{"with special chars", "team@123", false},
		{"with dot", "team.name", false},
		{"with slash", "team/name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTeamName(tt.input); got != tt.want {
				t.Errorf("isValidTeamName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Agent Create Flags Tests ---

func TestAgentCreateHasParentFlag(t *testing.T) {
	flags := agentCreateCmd.Flags()
	if flags.Lookup("parent") == nil {
		t.Error("expected --parent flag on agent create")
	}
}

func TestAgentCreateHasTeamFlag(t *testing.T) {
	flags := agentCreateCmd.Flags()
	if flags.Lookup("team") == nil {
		t.Error("expected --team flag on agent create")
	}
}

// --- Agent Role Hierarchy Tests ---

func TestCanCreateRole_TechLeadCanCreateEngineer(t *testing.T) {
	if !agent.CanCreateRole(agent.RoleTechLead, agent.RoleEngineer) {
		t.Error("tech-lead should be able to create engineer")
	}
}

func TestCanCreateRole_EngineerCannotCreateEngineer(t *testing.T) {
	if agent.CanCreateRole(agent.RoleEngineer, agent.RoleEngineer) {
		t.Error("engineer should not be able to create engineer")
	}
}

func TestCanCreateRole_ManagerCanCreateEngineer(t *testing.T) {
	if !agent.CanCreateRole(agent.RoleManager, agent.RoleEngineer) {
		t.Error("manager should be able to create engineer")
	}
}

func TestCanCreateRole_ManagerCanCreateQA(t *testing.T) {
	if !agent.CanCreateRole(agent.RoleManager, agent.RoleQA) {
		t.Error("manager should be able to create qa")
	}
}
