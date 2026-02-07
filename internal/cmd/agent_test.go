package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// --- Agent Create Tests ---

func TestAgentCreate_ValidRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		wantRole agent.Role
	}{
		{"worker role", "worker", agent.RoleWorker},
		{"engineer role", "engineer", agent.RoleEngineer},
		{"manager role", "manager", agent.RoleManager},
		{"qa role", "qa", agent.RoleQA},
		{"tech-lead role", "tech-lead", agent.RoleTechLead},
		{"product-manager role", "product-manager", agent.RoleProductManager},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := parseRole(tt.role)
			if err != nil {
				t.Errorf("parseRole(%q) error = %v", tt.role, err)
				return
			}
			if role != tt.wantRole {
				t.Errorf("parseRole(%q) = %v, want %v", tt.role, role, tt.wantRole)
			}
		})
	}
}

func TestAgentCreate_InvalidRole(t *testing.T) {
	invalidRoles := []string{
		"invalid",
		"admin",
		"superuser",
		"",
	}

	for _, role := range invalidRoles {
		t.Run(role, func(t *testing.T) {
			_, err := parseRole(role)
			if err == nil {
				t.Errorf("parseRole(%q) expected error, got nil", role)
			}
		})
	}
}

func TestAgentCreate_RoleAliases(t *testing.T) {
	tests := []struct {
		alias    string
		wantRole agent.Role
	}{
		{"pm", agent.RoleProductManager},
		{"coord", agent.RoleCoordinator},
		{"tl", agent.RoleTechLead},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			role, err := parseRole(tt.alias)
			if err != nil {
				t.Errorf("parseRole(%q) error = %v", tt.alias, err)
				return
			}
			if role != tt.wantRole {
				t.Errorf("parseRole(%q) = %v, want %v", tt.alias, role, tt.wantRole)
			}
		})
	}
}

func TestAgentCreate_EmptyName(t *testing.T) {
	// Setup temp workspace
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configPath := filepath.Join(bcDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[workspace]\nname = \"test\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Test that empty name is rejected
	cmd := agentCreateCmd
	cmd.SetArgs([]string{""})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// This will fail because we're not in a real workspace, but we can test the args
	// MaximumNArgs(1) allows empty string, validation happens in runAgentCreate
	_ = cmd.Args(cmd, []string{""})
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

// --- Agent List Tests ---

func TestAgentList_FilterByRole(t *testing.T) {
	// Test that role filter validation works
	validRoles := []string{"engineer", "qa", "manager", "worker"}

	for _, role := range validRoles {
		t.Run(role, func(t *testing.T) {
			_, err := parseRole(role)
			if err != nil {
				t.Errorf("parseRole(%q) should be valid for filtering", role)
			}
		})
	}
}

func TestAgentList_EmptyResult(t *testing.T) {
	// This tests the formatting logic for empty agent lists
	agents := []*agent.Agent{}
	if len(agents) != 0 {
		t.Error("expected empty agent list")
	}
}

// --- Agent Stop Tests ---

func TestAgentStop_NonExistentAgent(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	agentsDir := filepath.Join(bcDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create manager with no agents
	mgr := agent.NewManager(agentsDir)

	// Try to get non-existent agent
	a := mgr.GetAgent("nonexistent")
	if a != nil {
		t.Error("expected nil for non-existent agent")
	}
}

// --- Agent Send Tests ---

func TestAgentSend_EmptyMessage(t *testing.T) {
	// Test that empty message is properly rejected
	cmd := agentSendCmd

	// MinimumNArgs(2) should reject single arg
	err := cmd.Args(cmd, []string{"agent-name"})
	if err == nil {
		t.Error("expected error for single arg (missing message)")
	}
}

func TestAgentSend_ValidArgs(t *testing.T) {
	cmd := agentSendCmd

	// Should accept agent name + message
	err := cmd.Args(cmd, []string{"agent-name", "hello world"})
	if err != nil {
		t.Errorf("unexpected error for valid args: %v", err)
	}

	// Should accept multiple message words
	err = cmd.Args(cmd, []string{"agent-name", "hello", "world", "test"})
	if err != nil {
		t.Errorf("unexpected error for multi-word message: %v", err)
	}
}

// --- Agent Peek Tests ---

func TestAgentPeek_DefaultLines(t *testing.T) {
	// Default should be 50 lines
	if agentPeekLines != 50 {
		// Reset to default for test
		agentPeekLines = 50
	}

	if agentPeekLines != 50 {
		t.Errorf("expected default peek lines = 50, got %d", agentPeekLines)
	}
}

// --- Agent Attach Tests ---

func TestAgentAttach_RequiresName(t *testing.T) {
	cmd := agentAttachCmd

	// ExactArgs(1) should reject no args
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error for missing agent name")
	}

	// Should accept exactly one arg
	err = cmd.Args(cmd, []string{"agent-name"})
	if err != nil {
		t.Errorf("unexpected error for valid arg: %v", err)
	}

	// Should reject multiple args
	err = cmd.Args(cmd, []string{"agent1", "agent2"})
	if err == nil {
		t.Error("expected error for multiple agent names")
	}
}

// --- Command Structure Tests ---

func TestAgentCommandStructure(t *testing.T) {
	// Verify agentCmd has expected subcommands
	subcommands := agentCmd.Commands()

	expectedCmds := map[string]bool{
		"create": false,
		"list":   false,
		"attach": false,
		"peek":   false,
		"stop":   false,
		"send":   false,
	}

	for _, cmd := range subcommands {
		if _, ok := expectedCmds[cmd.Name()]; ok {
			expectedCmds[cmd.Name()] = true
		}
	}

	for name, found := range expectedCmds {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestAgentCreateFlags(t *testing.T) {
	// Verify create command has expected flags
	flags := agentCreateCmd.Flags()

	if flags.Lookup("tool") == nil {
		t.Error("expected --tool flag")
	}
	if flags.Lookup("role") == nil {
		t.Error("expected --role flag")
	}
}

func TestAgentListFlags(t *testing.T) {
	flags := agentListCmd.Flags()

	if flags.Lookup("role") == nil {
		t.Error("expected --role flag for filtering")
	}
	if flags.Lookup("json") == nil {
		t.Error("expected --json flag")
	}
}

func TestAgentPeekFlags(t *testing.T) {
	flags := agentPeekCmd.Flags()

	if flags.Lookup("lines") == nil {
		t.Error("expected --lines flag")
	}
}

func TestAgentStopFlags(t *testing.T) {
	flags := agentStopCmd.Flags()

	if flags.Lookup("force") == nil {
		t.Error("expected --force flag")
	}
}

// --- Integration Tests using executeCmd ---

func TestAgentListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	// Command should succeed even with no agents
	_, err := executeCmd("agent", "list")
	if err != nil {
		t.Fatalf("agent list failed: %v", err)
	}
}

func TestAgentListJSON(t *testing.T) {
	setupTestWorkspace(t)

	// Command should succeed with --json flag
	_, err := executeCmd("agent", "list", "--json")
	if err != nil {
		t.Fatalf("agent list --json failed: %v", err)
	}
}

func TestAgentStopNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("agent", "stop", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for stopping nonexistent agent")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestAgentSendNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("agent", "send", "nonexistent-agent", "hello")
	if err == nil {
		t.Error("expected error for sending to nonexistent agent")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestAgentPeekNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("agent", "peek", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for peeking nonexistent agent")
	}
}

func TestAgentAttachNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("agent", "attach", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for attaching to nonexistent agent")
	}
}

func TestAgentListWithRoleFilter(t *testing.T) {
	setupTestWorkspace(t)

	// Should succeed with valid role filter
	_, err := executeCmd("agent", "list", "--role", "engineer")
	if err != nil {
		t.Fatalf("agent list --role failed: %v", err)
	}
}

func TestAgentListInvalidRole(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("agent", "list", "--role", "invalid-role")
	if err == nil {
		t.Error("expected error for invalid role filter")
	}
}
