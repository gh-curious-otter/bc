package cmd

import (
	"strings"
	"testing"
)

// --- Channel Lifecycle E2E Tests ---

func TestChannelLifecycle_ListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	// Channel list should succeed even with no channels
	_, err := executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list error: %v", err)
	}
}

func TestChannelLifecycle_ListJSON(t *testing.T) {
	setupTestWorkspace(t)

	// Channel list --json should succeed even with no channels
	_, err := executeCmd("channel", "list", "--json")
	if err != nil {
		t.Fatalf("channel list --json error: %v", err)
	}
}

func TestChannelLifecycle_CreateAndList(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "test-channel")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// List channels should now show the new channel
	_, err = executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list error: %v", err)
	}
}

func TestChannelLifecycle_CreateDuplicate(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "dup-channel")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// Create duplicate should fail
	_, err = executeCmd("channel", "create", "dup-channel")
	if err == nil {
		t.Error("expected error for duplicate channel")
	}
	if err != nil && !strings.Contains(err.Error(), "exists") {
		t.Errorf("expected 'exists' error, got: %v", err)
	}
}

func TestChannelLifecycle_CreateEmptyName(t *testing.T) {
	setupTestWorkspace(t)

	// Create with empty name should fail
	_, err := executeCmd("channel", "create", "")
	if err == nil {
		t.Error("expected error for empty channel name")
	}
}

func TestChannelLifecycle_AddMember(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "members-channel")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// Add a member
	_, err = executeCmd("channel", "add", "members-channel", "agent-01")
	if err != nil {
		t.Fatalf("channel add error: %v", err)
	}
}

func TestChannelLifecycle_AddMemberToNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	// Add member to non-existent channel prints a warning but doesn't error
	// Command returns success but with 0 members added
	_, err := executeCmd("channel", "add", "nonexistent-channel", "agent-01")
	if err != nil {
		t.Fatalf("channel add should not error (shows warning instead): %v", err)
	}
}

func TestChannelLifecycle_RemoveMember(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel and add a member
	_, err := executeCmd("channel", "create", "remove-test")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	_, err = executeCmd("channel", "add", "remove-test", "agent-01")
	if err != nil {
		t.Fatalf("channel add error: %v", err)
	}

	// Remove the member
	_, err = executeCmd("channel", "remove", "remove-test", "agent-01")
	if err != nil {
		t.Fatalf("channel remove error: %v", err)
	}
}

func TestChannelLifecycle_RemoveMemberNotInChannel(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "no-member")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// Remove a member that's not in the channel should fail
	_, err = executeCmd("channel", "remove", "no-member", "agent-01")
	if err == nil {
		t.Error("expected error for removing non-member")
	}
}

func TestChannelLifecycle_Delete(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "delete-test")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// Delete the channel
	_, err = executeCmd("channel", "delete", "delete-test")
	if err != nil {
		t.Fatalf("channel delete error: %v", err)
	}
}

func TestChannelLifecycle_DeleteNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	// Delete non-existent channel should fail
	_, err := executeCmd("channel", "delete", "nonexistent")
	if err == nil {
		t.Error("expected error for deleting non-existent channel")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestChannelLifecycle_History(t *testing.T) {
	setupTestWorkspace(t)

	// Create a channel
	_, err := executeCmd("channel", "create", "history-test")
	if err != nil {
		t.Fatalf("channel create error: %v", err)
	}

	// History should work (even if empty)
	_, err = executeCmd("channel", "history", "history-test")
	if err != nil {
		t.Fatalf("channel history error: %v", err)
	}
}

func TestChannelLifecycle_HistoryNonexistent(t *testing.T) {
	setupTestWorkspace(t)

	// History of non-existent channel should fail
	_, err := executeCmd("channel", "history", "nonexistent")
	if err == nil {
		t.Error("expected error for history of non-existent channel")
	}
}

// --- Channel Command Structure Tests ---

func TestChannelCommandStructure(t *testing.T) {
	// Verify channelCmd has expected subcommands
	subcommands := channelCmd.Commands()

	expectedCmds := map[string]bool{
		"create":  false,
		"list":    false,
		"add":     false,
		"remove":  false,
		"send":    false,
		"delete":  false,
		"join":    false,
		"leave":   false,
		"history": false,
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

func TestChannelListFlags(t *testing.T) {
	flags := channelListCmd.Flags()

	if flags.Lookup("json") == nil {
		t.Error("expected --json flag on channel list")
	}
}
