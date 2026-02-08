package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
)

// resetAgentFlags resets agent command flags between tests
func resetAgentFlags() {
	agentCreateTool = ""
	agentCreateRole = "worker"
	agentCreateParent = ""
	agentCreateTeam = ""
	agentListRole = ""
	agentListJSON = false
	agentPeekLines = 50
	agentStopForce = false
}

// --- Agent Lifecycle E2E Tests ---

func TestAgentLifecycle_ListEmpty(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	// Agent list uses fmt.Printf, so output not captured by cobra buffer
	// Just verify the command succeeds
	_, err := executeCmd("agent", "list")
	if err != nil {
		t.Fatalf("agent list error: %v", err)
	}
}

func TestAgentLifecycle_ListWithRoleFilter(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	// Filter by role should work even with no agents
	// Agent list uses fmt.Printf, so output not captured
	_, err := executeCmd("agent", "list", "--role", "engineer")
	if err != nil {
		t.Fatalf("agent list --role error: %v", err)
	}
}

func TestAgentLifecycle_ListInvalidRole(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "list", "--role", "invalid-role")
	if err == nil {
		t.Error("expected error for invalid role filter")
	}
}

func TestAgentLifecycle_CreateNoWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "create", "test-agent")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestAgentLifecycle_StopNotFound(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "stop", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestAgentLifecycle_PeekNotFound(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "peek", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestAgentLifecycle_SendNotFound(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "send", "nonexistent-agent", "hello")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestAgentLifecycle_AttachNotRunning(t *testing.T) {
	setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	_, err := executeCmd("agent", "attach", "nonexistent-agent")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("expected 'not running' error, got: %v", err)
	}
}

// --- Agent State Workflow Tests ---

func TestAgentStateWorkflow_ManagerOperations(t *testing.T) {
	wsDir := setupTestWorkspace(t)
	resetAgentFlags()
	defer resetAgentFlags()

	// Create workspace manager
	mgr := agent.NewWorkspaceManager(filepath.Join(wsDir, ".bc", "agents"), wsDir)
	if err := mgr.LoadState(); err != nil {
		t.Logf("initial load (expected to be empty): %v", err)
	}

	// List agents - should be empty
	agents := mgr.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected no agents initially, got %d", len(agents))
	}

	// Running count should be 0
	if count := mgr.RunningCount(); count != 0 {
		t.Errorf("expected running count 0, got %d", count)
	}

	// Agent count should be 0
	if count := mgr.AgentCount(); count != 0 {
		t.Errorf("expected agent count 0, got %d", count)
	}
}

func TestAgentStateWorkflow_GetNonExistentAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	mgr := agent.NewWorkspaceManager(filepath.Join(wsDir, ".bc", "agents"), wsDir)
	_ = mgr.LoadState()

	// GetAgent should return nil for non-existent
	a := mgr.GetAgent("does-not-exist")
	if a != nil {
		t.Error("expected nil for non-existent agent")
	}
}

// --- Channel Communication Workflow Tests ---
// Note: Channel commands use fmt.Printf which doesn't go through cobra's test buffer
// So we test that commands succeed/fail appropriately, not output content

func TestChannelWorkflow_CreateAndList(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel (should succeed without error)
	_, err := executeCmd("channel", "create", "test-channel")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// List channels (should succeed)
	_, err = executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list error: %v", err)
	}
}

func TestChannelWorkflow_AddRemoveMembers(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel using store directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load channel store: %v", err)
	}
	if _, err := store.Create("members-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save channel: %v", err)
	}

	// Add member (should succeed)
	_, err := executeCmd("channel", "add", "members-test", "agent-01")
	if err != nil {
		t.Fatalf("channel add error: %v", err)
	}

	// Verify membership via store
	if loadErr := store.Load(); loadErr != nil {
		t.Fatalf("failed to reload store: %v", loadErr)
	}
	members, membersErr := store.GetMembers("members-test")
	if membersErr != nil {
		t.Fatalf("failed to get members: %v", membersErr)
	}
	found := false
	for _, m := range members {
		if m == "agent-01" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected agent-01 to be a member")
	}

	// Remove member (should succeed)
	_, err = executeCmd("channel", "remove", "members-test", "agent-01")
	if err != nil {
		t.Fatalf("channel remove error: %v", err)
	}

	// Verify removal via store
	if loadErr := store.Load(); loadErr != nil {
		t.Fatalf("failed to reload store: %v", loadErr)
	}
	members, membersErr = store.GetMembers("members-test")
	if membersErr != nil {
		t.Fatalf("failed to get members: %v", membersErr)
	}
	for _, m := range members {
		if m == "agent-01" {
			t.Error("agent-01 should have been removed")
		}
	}
}

func TestChannelWorkflow_JoinWithoutAgentID(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("join-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Use t.Setenv with empty string to unset (t.Setenv handles cleanup)
	t.Setenv("BC_AGENT_ID", "")

	// Try to join - should fail without BC_AGENT_ID
	_, err := executeCmd("channel", "join", "join-test")
	if err == nil {
		t.Error("expected error for join without BC_AGENT_ID")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
	}
}

func TestChannelWorkflow_JoinWithAgentID(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("join-with-id"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Set BC_AGENT_ID using t.Setenv (handles cleanup automatically)
	t.Setenv("BC_AGENT_ID", "test-agent-01")

	// Join channel (should succeed)
	_, err := executeCmd("channel", "join", "join-with-id")
	if err != nil {
		t.Fatalf("channel join error: %v", err)
	}

	// Verify membership via store
	if loadErr := store.Load(); loadErr != nil {
		t.Fatalf("failed to reload store: %v", loadErr)
	}
	members, membersErr := store.GetMembers("join-with-id")
	if membersErr != nil {
		t.Fatalf("failed to get members: %v", membersErr)
	}
	found := false
	for _, m := range members {
		if m == "test-agent-01" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected test-agent-01 to be a member after join")
	}
}

func TestChannelWorkflow_LeaveChannel(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel and add member directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("leave-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddMember("leave-test", "leaving-agent"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Set BC_AGENT_ID using t.Setenv (handles cleanup automatically)
	t.Setenv("BC_AGENT_ID", "leaving-agent")

	// Leave channel (should succeed)
	_, err := executeCmd("channel", "leave", "leave-test")
	if err != nil {
		t.Fatalf("channel leave error: %v", err)
	}

	// Verify removal via store
	if loadErr := store.Load(); loadErr != nil {
		t.Fatalf("failed to reload store: %v", loadErr)
	}
	members, membersErr := store.GetMembers("leave-test")
	if membersErr != nil {
		t.Fatalf("failed to get members: %v", membersErr)
	}
	for _, m := range members {
		if m == "leaving-agent" {
			t.Error("leaving-agent should have been removed")
		}
	}
}

func TestChannelWorkflow_SendToEmptyChannel(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create empty channel directly
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

	// Send to empty channel (should succeed but do nothing)
	_, err := executeCmd("channel", "send", "empty-channel", "hello world")
	if err != nil {
		t.Fatalf("channel send error: %v", err)
	}
}

func TestChannelWorkflow_DeleteChannel(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("delete-me"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Delete channel (should succeed)
	_, err := executeCmd("channel", "delete", "delete-me")
	if err != nil {
		t.Fatalf("channel delete error: %v", err)
	}

	// Verify deleted via store
	if err := store.Load(); err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	channels := store.List()
	for _, ch := range channels {
		if ch.Name == "delete-me" {
			t.Error("channel should have been deleted")
		}
	}
}

func TestChannelWorkflow_DeleteNonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "delete", "nonexistent-channel")
	if err == nil {
		t.Error("expected error deleting non-existent channel")
	}
}

func TestChannelWorkflow_HistoryEmpty(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create channel directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("history-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Check empty history (should succeed)
	_, err := executeCmd("channel", "history", "history-test")
	if err != nil {
		t.Fatalf("channel history error: %v", err)
	}
}

func TestChannelWorkflow_HistoryNonExistent(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "history", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent channel history")
	}
}

// --- Agent Team Workflow Tests ---

func TestAgentTeamWorkflow_InvalidTeamName(t *testing.T) {
	invalidNames := []string{
		"team with spaces",
		"team@special",
		"team.dot",
		"",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			if isValidTeamName(name) {
				t.Errorf("expected %q to be invalid team name", name)
			}
		})
	}
}

func TestAgentTeamWorkflow_ValidTeamName(t *testing.T) {
	validNames := []string{
		"engineering",
		"qa-team",
		"team_01",
		"Platform-2024",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if !isValidTeamName(name) {
				t.Errorf("expected %q to be valid team name", name)
			}
		})
	}
}

// --- Role Hierarchy Workflow Tests ---

func TestRoleHierarchy_CanCreate(t *testing.T) {
	// Test role hierarchy based on actual RoleHierarchy map:
	// Manager: TechLead, Engineer, QA
	// TechLead: Engineer only
	// Engineer, QA, Worker: cannot create children
	tests := []struct {
		name      string
		parent    agent.Role
		child     agent.Role
		canCreate bool
	}{
		{"manager creates engineer", agent.RoleManager, agent.RoleEngineer, true},
		{"manager creates qa", agent.RoleManager, agent.RoleQA, true},
		{"manager creates tech-lead", agent.RoleManager, agent.RoleTechLead, true},
		{"manager cannot create worker", agent.RoleManager, agent.RoleWorker, false},
		{"tech-lead creates engineer", agent.RoleTechLead, agent.RoleEngineer, true},
		{"tech-lead cannot create qa", agent.RoleTechLead, agent.RoleQA, false},
		{"engineer cannot create engineer", agent.RoleEngineer, agent.RoleEngineer, false},
		{"worker cannot create anything", agent.RoleWorker, agent.RoleWorker, false},
		{"qa cannot create engineer", agent.RoleQA, agent.RoleEngineer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.CanCreateRole(tt.parent, tt.child)
			if result != tt.canCreate {
				t.Errorf("CanCreateRole(%v, %v) = %v, want %v",
					tt.parent, tt.child, result, tt.canCreate)
			}
		})
	}
}

// --- Command Argument Validation Tests ---

func TestAgentCmdArgs_CreateAcceptsOptionalName(t *testing.T) {
	// create accepts 0 or 1 args
	err := agentCreateCmd.Args(agentCreateCmd, []string{})
	if err != nil {
		t.Errorf("expected create to accept 0 args, got: %v", err)
	}

	err = agentCreateCmd.Args(agentCreateCmd, []string{"my-agent"})
	if err != nil {
		t.Errorf("expected create to accept 1 arg, got: %v", err)
	}

	err = agentCreateCmd.Args(agentCreateCmd, []string{"a", "b"})
	if err == nil {
		t.Error("expected create to reject 2 args")
	}
}

func TestAgentCmdArgs_StopRequiresName(t *testing.T) {
	err := agentStopCmd.Args(agentStopCmd, []string{})
	if err == nil {
		t.Error("expected stop to require agent name")
	}

	err = agentStopCmd.Args(agentStopCmd, []string{"my-agent"})
	if err != nil {
		t.Errorf("expected stop to accept 1 arg, got: %v", err)
	}
}

func TestAgentCmdArgs_SendRequiresNameAndMessage(t *testing.T) {
	err := agentSendCmd.Args(agentSendCmd, []string{})
	if err == nil {
		t.Error("expected send to require args")
	}

	err = agentSendCmd.Args(agentSendCmd, []string{"agent"})
	if err == nil {
		t.Error("expected send to require message")
	}

	err = agentSendCmd.Args(agentSendCmd, []string{"agent", "message"})
	if err != nil {
		t.Errorf("expected send to accept name and message, got: %v", err)
	}

	// Multiple words in message should work
	err = agentSendCmd.Args(agentSendCmd, []string{"agent", "hello", "world"})
	if err != nil {
		t.Errorf("expected send to accept multi-word message, got: %v", err)
	}
}

// --- No Workspace Error Tests ---

func TestNoWorkspace_AgentList(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("agent", "list")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestNoWorkspace_AgentStop(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("agent", "stop", "any-agent")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestNoWorkspace_AgentPeek(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("agent", "peek", "any-agent")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestNoWorkspace_AgentSend(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("agent", "send", "any-agent", "message")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestNoWorkspace_ChannelCreate(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("channel", "create", "test")
	if err == nil {
		t.Error("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}
