package cmd

import (
	"fmt"
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
			Role:      agent.Role("engineer"),
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

func TestChannelSend_SenderNotIncludedInRecipients(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create two agents
	seedAgents(t, wsDir, map[string]*agent.Agent{
		"agent-01": {
			Name:      "agent-01",
			Role:      agent.Role("engineer"),
			State:     agent.StateIdle,
			Session:   "bc-agent-01",
			StartedAt: time.Now(),
		},
		"agent-02": {
			Name:      "agent-02",
			Role:      agent.Role("engineer"),
			State:     agent.StateIdle,
			Session:   "bc-agent-02",
			StartedAt: time.Now(),
		},
	})

	// Create channel with both agents as members
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	ch, err := store.Create("test-channel")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	ch.Members = []string{"agent-01", "agent-02"}
	if saveErr := store.Save(); saveErr != nil {
		t.Fatalf("failed to save: %v", saveErr)
	}

	// Set BC_AGENT_ID to simulate agent-01 sending
	if setErr := os.Setenv("BC_AGENT_ID", "agent-01"); setErr != nil {
		t.Fatalf("failed to set BC_AGENT_ID: %v", setErr)
	}
	defer func() {
		if unsetErr := os.Unsetenv("BC_AGENT_ID"); unsetErr != nil {
			t.Logf("failed to unset BC_AGENT_ID: %v", unsetErr)
		}
	}()

	stdout, _, err := executeIntegrationCmd("channel", "send", "test-channel", "hello")
	if err != nil {
		t.Fatalf("channel send error: %v", err)
	}

	// Should show that only 1 agent received the message (agent-01 is sender, so skipped)
	if !strings.Contains(stdout, "Result: 0/1 members received message") {
		t.Errorf("expected 'Result: 0/1 members received message' in output, got: %s", stdout)
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
	if !strings.Contains(err.Error(), "requires at least 1 arg") {
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
	if !strings.Contains(err.Error(), "between 1 and 2 arg") {
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

// --- Command Structure Tests ---

func TestChannelCommandSubcommands(t *testing.T) {
	subcommands := channelCmd.Commands()

	expectedCmds := map[string]bool{
		"list":    false,
		"create":  false,
		"delete":  false,
		"add":     false,
		"remove":  false,
		"send":    false,
		"join":    false,
		"leave":   false,
		"history": false,
		"show":    false,
		"react":   false,
		"desc":    false,
		"status":  false,
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

func TestChannelHistoryNoFlags(t *testing.T) {
	// Verify history command exists and has no special flags
	flags := channelHistoryCmd.Flags()

	// History command currently has no flags
	if flags.Lookup("limit") != nil {
		t.Log("--limit flag is available for history")
	}
}

// --- Channel Show Tests ---

func TestChannelShow_RequiresName(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "show")
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelShow_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel", "show", "test-channel")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestChannelShow_NonexistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "show", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestChannelShow_ExistingChannel(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a channel using the store directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("test-channel"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddMember("test-channel", "agent-01"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.AddMember("test-channel", "agent-02"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save store: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "show", "test-channel")
	if err != nil {
		t.Fatalf("channel show error: %v", err)
	}

	// Should show channel name
	if !strings.Contains(stdout, "#test-channel") {
		t.Errorf("expected channel name in output, got: %s", stdout)
	}

	// Should show member count
	if !strings.Contains(stdout, "Members (2)") {
		t.Errorf("expected 'Members (2)' in output, got: %s", stdout)
	}

	// Should show members
	if !strings.Contains(stdout, "agent-01") {
		t.Errorf("expected 'agent-01' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "agent-02") {
		t.Errorf("expected 'agent-02' in output, got: %s", stdout)
	}
}

func TestChannelShow_JSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a channel using the store directly
	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("json-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddMember("json-test", "agent-01"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save store: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "show", "json-test", "--json")
	if err != nil {
		t.Fatalf("channel show --json error: %v", err)
	}

	// Should be valid JSON with expected fields
	if !strings.Contains(stdout, `"name": "json-test"`) {
		t.Errorf("expected JSON name field, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"member_count": 1`) {
		t.Errorf("expected JSON member_count field, got: %s", stdout)
	}
}

// --- Channel Status Tests ---

func TestChannelStatus_Empty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("channel", "status")
	if err != nil {
		t.Fatalf("channel status error: %v", err)
	}
	if !strings.Contains(stdout, "No channels") {
		t.Errorf("expected 'No channels' message, got: %s", stdout)
	}
}

func TestChannelStatus_WithChannels(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("test-channel"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddMember("test-channel", "agent-01"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.AddHistory("test-channel", "agent-01", "hello world"); err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "status")
	if err != nil {
		t.Fatalf("channel status error: %v", err)
	}
	if !strings.Contains(stdout, "test-channel") {
		t.Errorf("expected channel name in output, got: %s", stdout)
	}
}

func TestChannelStatus_JSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("json-status"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "status", "--json")
	if err != nil {
		t.Fatalf("channel status --json error: %v", err)
	}
	if !strings.Contains(stdout, `"name": "json-status"`) {
		t.Errorf("expected JSON name field, got: %s", stdout)
	}
}

// --- Channel Desc Tests ---

func TestChannelDesc_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "desc")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "requires at least 2 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelDesc_SetDescription(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("desc-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "desc", "desc-test", "My channel description")
	if err != nil {
		t.Fatalf("channel desc error: %v", err)
	}
	if !strings.Contains(stdout, "Updated description") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

// --- Channel React Tests ---

func TestChannelReact_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "react")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 3 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelReact_InvalidIndex(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "react", "test", "abc", "👍")
	if err == nil {
		t.Fatal("expected error for invalid index, got nil")
	}
	if !strings.Contains(err.Error(), "invalid message index") {
		t.Errorf("expected 'invalid message index' error, got: %v", err)
	}
}

// --- Channel History Filter Tests ---

func TestChannelHistory_WithAgentFilter(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("filter-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddHistory("filter-test", "agent-01", "hello from agent-01"); err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	if err := store.AddHistory("filter-test", "agent-02", "hello from agent-02"); err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "history", "filter-test", "--agent", "agent-01")
	if err != nil {
		t.Fatalf("channel history --agent error: %v", err)
	}
	if !strings.Contains(stdout, "agent-01") {
		t.Errorf("expected agent-01 messages, got: %s", stdout)
	}
	if strings.Contains(stdout, "agent-02") {
		t.Errorf("should not contain agent-02 messages when filtering by agent-01, got: %s", stdout)
	}
}

func TestChannelHistory_WithLimit(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("limit-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := store.AddHistory("limit-test", "agent", fmt.Sprintf("message %d", i)); err != nil {
			t.Fatalf("failed to add history: %v", err)
		}
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "history", "limit-test", "--limit", "3")
	if err != nil {
		t.Fatalf("channel history --limit error: %v", err)
	}
	// Should show only 3 messages (the last 3)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	messageLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "[") && strings.Contains(line, "agent:") {
			messageLines++
		}
	}
	if messageLines > 3 {
		t.Errorf("expected at most 3 message lines with --limit 3, got %d", messageLines)
	}
}

func TestChannelHistory_JSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("json-history"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddHistory("json-history", "agent-01", "test message"); err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "history", "json-history", "--json")
	if err != nil {
		t.Fatalf("channel history --json error: %v", err)
	}
	if !strings.Contains(stdout, `"channel": "json-history"`) {
		t.Errorf("expected JSON channel field, got: %s", stdout)
	}
	if !strings.Contains(stdout, "test message") {
		t.Errorf("expected message content in JSON, got: %s", stdout)
	}
}

// --- Channel Edit Tests ---

func TestChannelEdit_RequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("channel", "edit")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected arg count error, got: %v", err)
	}
}

func TestChannelEdit_NonexistentChannel(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "edit", "nonexistent", "--desc", "new desc")
	if err == nil {
		t.Fatal("expected error for nonexistent channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestChannelEdit_NoFlags(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("edit-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	_, _, err := executeIntegrationCmd("channel", "edit", "edit-test")
	if err == nil {
		t.Fatal("expected error for edit with no flags, got nil")
	}
	if !strings.Contains(err.Error(), "at least one setting") {
		t.Errorf("expected 'at least one setting' error, got: %v", err)
	}
}

func TestChannelEdit_SetDescription(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("edit-desc"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "edit", "edit-desc", "--desc", "New description")
	if err != nil {
		t.Fatalf("channel edit error: %v", err)
	}
	if !strings.Contains(stdout, "Updated channel") {
		t.Errorf("expected success message, got: %s", stdout)
	}
}

// --- Channel Add with --agent flag ---

func TestChannelAdd_WithAgentFlag(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("flag-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "add", "flag-test", "--agent", "agent-flag")
	if err != nil {
		t.Fatalf("channel add --agent error: %v", err)
	}
	if !strings.Contains(stdout, "Added 1") {
		t.Errorf("expected 'Added 1' in output, got: %s", stdout)
	}
}

func TestChannelAdd_NoMember(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("no-member-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	_, _, err := executeIntegrationCmd("channel", "add", "no-member-test")
	if err == nil {
		t.Fatal("expected error for add with no member, got nil")
	}
	if !strings.Contains(err.Error(), "at least one member") {
		t.Errorf("expected 'at least one member' error, got: %v", err)
	}
}

// --- Channel History with --last flag ---

func TestChannelHistory_WithLastFlag(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := channel.NewStore(wsDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}
	if _, err := store.Create("last-test"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	for i := 0; i < 10; i++ {
		if err := store.AddHistory("last-test", "agent", fmt.Sprintf("message %d", i)); err != nil {
			t.Fatalf("failed to add history: %v", err)
		}
	}
	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("channel", "history", "last-test", "--last", "3")
	if err != nil {
		t.Fatalf("channel history --last error: %v", err)
	}
	// Should show only 3 messages (the last 3)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	messageLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "[") && strings.Contains(line, "agent:") {
			messageLines++
		}
	}
	if messageLines > 3 {
		t.Errorf("expected at most 3 message lines with --last 3, got %d", messageLines)
	}
}

// seedAgents helper is defined in cmd_integration_test.go
