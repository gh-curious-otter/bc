package channel

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpDir := t.TempDir()
	store := &SQLiteStore{
		path: filepath.Join(tmpDir, ".bc", "channels.db"),
	}
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLiteStore_Open(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSQLiteStore(tmpDir)

	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	dbPath := filepath.Join(tmpDir, ".bc", "bc.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("database file not created: %v", err)
	}

	channels, err := store.ListChannels()
	if err != nil {
		t.Fatalf("failed to list channels: %v", err)
	}

	names := make(map[string]bool)
	for _, ch := range channels {
		names[ch.Name] = true
	}

	for _, expected := range []string{"general", "engineering", "all"} {
		if !names[expected] {
			t.Errorf("expected default channel %q not found", expected)
		}
	}
}

func TestSQLiteStore_CreateChannel(t *testing.T) {
	store := setupTestDB(t)

	ch, err := store.CreateChannel("test-channel", ChannelTypeGroup, "Test description")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if ch.Name != "test-channel" {
		t.Errorf("expected name 'test-channel', got %q", ch.Name)
	}
	if ch.Type != ChannelTypeGroup {
		t.Errorf("expected type 'group', got %q", ch.Type)
	}
	if ch.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %q", ch.Description)
	}

	_, err = store.CreateChannel("test-channel", ChannelTypeGroup, "")
	if err == nil {
		t.Error("expected error for duplicate channel")
	}
}

func TestSQLiteStore_GetChannel(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("my-channel", ChannelTypeDirect, "My channel"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	ch, err := store.GetChannel("my-channel")
	if err != nil {
		t.Fatalf("failed to get channel: %v", err)
	}
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
	if ch.Name != "my-channel" {
		t.Errorf("expected name 'my-channel', got %q", ch.Name)
	}

	ch, err = store.GetChannel("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch != nil {
		t.Error("expected nil for nonexistent channel")
	}
}

func TestSQLiteStore_DeleteChannel(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("to-delete", ChannelTypeGroup, ""); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if err := store.DeleteChannel("to-delete"); err != nil {
		t.Fatalf("failed to delete channel: %v", err)
	}

	ch, _ := store.GetChannel("to-delete")
	if ch != nil {
		t.Error("channel should be deleted")
	}

	err := store.DeleteChannel("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestSQLiteStore_Members(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("team", ChannelTypeGroup, ""); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if err := store.AddMember("team", "engineer-01"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.AddMember("team", "engineer-02"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	members, err := store.GetMembers("team")
	if err != nil {
		t.Fatalf("failed to get members: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	if addDupErr := store.AddMember("team", "engineer-01"); addDupErr != nil {
		t.Errorf("unexpected error adding duplicate: %v", addDupErr)
	}

	if removeErr := store.RemoveMember("team", "engineer-01"); removeErr != nil {
		t.Fatalf("failed to remove member: %v", removeErr)
	}

	members, _ = store.GetMembers("team")
	if len(members) != 1 {
		t.Errorf("expected 1 member after removal, got %d", len(members))
	}

	err = store.RemoveMember("team", "engineer-01")
	if err == nil {
		t.Error("expected error removing non-member")
	}
}

func TestSQLiteStore_GetChannelsForAgent(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("channel-a", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateChannel("channel-b", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateChannel("channel-c", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	if err := store.AddMember("channel-a", "agent-01"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMember("channel-b", "agent-01"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMember("channel-c", "agent-02"); err != nil {
		t.Fatal(err)
	}

	channels, err := store.GetChannelsForAgent("agent-01")
	if err != nil {
		t.Fatalf("failed to get channels: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

func TestSQLiteStore_Messages(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("dev", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	msg1, err := store.AddMessage("dev", "engineer-01", "Hello world", TypeText, "")
	if err != nil {
		t.Fatalf("failed to add message: %v", err)
	}
	if msg1.Sender != "engineer-01" {
		t.Errorf("expected sender 'engineer-01', got %q", msg1.Sender)
	}
	if msg1.Type != TypeText {
		t.Errorf("expected type 'text', got %q", msg1.Type)
	}

	msg2, err := store.AddMessage("dev", "manager", "@engineer-01 implement feature X", TypeTask, `{"pr":"#123"}`)
	if err != nil {
		t.Fatalf("failed to add task message: %v", err)
	}

	history, err := store.GetHistory("dev", 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}

	var foundHello, foundTask bool
	for _, m := range history {
		if m.Content == "Hello world" {
			foundHello = true
		}
		if m.ID == msg2.ID {
			foundTask = true
		}
	}
	if !foundHello {
		t.Error("expected to find 'Hello world' message")
	}
	if !foundTask {
		t.Error("expected to find task message")
	}
}

func TestSQLiteStore_GetMessagesByType(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("work", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddMessage("work", "user1", "text 1", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("work", "user1", "task 1", TypeTask, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("work", "user1", "text 2", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("work", "user1", "review 1", TypeReview, ""); err != nil {
		t.Fatal(err)
	}

	tasks, err := store.GetMessagesByType("work", TypeTask, 10)
	if err != nil {
		t.Fatalf("failed to get tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	texts, err := store.GetMessagesByType("work", TypeText, 10)
	if err != nil {
		t.Fatalf("failed to get texts: %v", err)
	}
	if len(texts) != 2 {
		t.Errorf("expected 2 texts, got %d", len(texts))
	}
}

func TestSQLiteStore_Mentions(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("mentions-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	msg, err := store.AddMessage("mentions-test", "manager", "@engineer-01 do this", TypeTask, "")
	if err != nil {
		t.Fatal(err)
	}

	if mentionErr := store.AddMention(msg.ID, "engineer-01"); mentionErr != nil {
		t.Fatalf("failed to add mention: %v", mentionErr)
	}

	mentions, err := store.GetUnreadMentions("engineer-01")
	if err != nil {
		t.Fatalf("failed to get unread mentions: %v", err)
	}
	if len(mentions) != 1 {
		t.Errorf("expected 1 unread mention, got %d", len(mentions))
	}

	if ackErr := store.AcknowledgeMentions("engineer-01"); ackErr != nil {
		t.Fatalf("failed to acknowledge: %v", ackErr)
	}

	mentions, _ = store.GetUnreadMentions("engineer-01")
	if len(mentions) != 0 {
		t.Errorf("expected 0 unread mentions after ack, got %d", len(mentions))
	}
}

func TestSQLiteStore_MigrateFromJSON(t *testing.T) {
	tmpDir := t.TempDir()

	jsonPath := filepath.Join(tmpDir, "channels.json")
	jsonData := `[
		{
			"name": "legacy-channel",
			"members": ["agent-a", "agent-b"],
			"history": [
				{"time": "2024-01-01T10:00:00Z", "sender": "agent-a", "message": "Hello from JSON"}
			]
		}
	]`
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0600); err != nil {
		t.Fatalf("failed to write JSON: %v", err)
	}

	store := NewSQLiteStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.MigrateFromJSON(jsonPath); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	ch, err := store.GetChannel("legacy-channel")
	if err != nil {
		t.Fatalf("failed to get channel: %v", err)
	}
	if ch == nil {
		t.Fatal("legacy channel not migrated")
	}

	members, _ := store.GetMembers("legacy-channel")
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	history, _ := store.GetHistory("legacy-channel", 10)
	if len(history) != 1 {
		t.Errorf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != "Hello from JSON" {
		t.Errorf("unexpected message content: %q", history[0].Content)
	}

	if _, err := os.Stat(jsonPath + ".migrated"); err != nil {
		t.Errorf("JSON file not renamed: %v", err)
	}
}

func TestSQLiteStore_SearchMessages(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("search-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddMessage("search-test", "user1", "The quick brown fox", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("search-test", "user2", "jumps over the lazy dog", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("search-test", "user1", "authentication bug in login", TypeTask, ""); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchMessages("fox", 10)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'fox', got %d", len(results))
	}

	results, _ = store.SearchMessages("bug", 10)
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'bug', got %d", len(results))
	}
}

func TestSQLiteStore_ChannelType(t *testing.T) {
	store := setupTestDB(t)

	direct, err := store.CreateChannel("engineer-01", ChannelTypeDirect, "DM for engineer-01")
	if err != nil {
		t.Fatalf("failed to create direct channel: %v", err)
	}
	if direct.Type != ChannelTypeDirect {
		t.Errorf("expected type 'direct', got %q", direct.Type)
	}

	group, err := store.CreateChannel("team-alpha", ChannelTypeGroup, "")
	if err != nil {
		t.Fatalf("failed to create group channel: %v", err)
	}
	if group.Type != ChannelTypeGroup {
		t.Errorf("expected type 'group', got %q", group.Type)
	}
}

func TestSQLiteStore_MessageMetadata(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("meta-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	metadata := `{"pr_number": 123, "branch": "feature/test"}`
	msg, err := store.AddMessage("meta-test", "bot", "Review PR #123", TypeReview, metadata)
	if err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	if msg.Metadata != metadata {
		t.Errorf("metadata not preserved: got %q", msg.Metadata)
	}

	retrieved, _ := store.GetMessage(msg.ID)
	if retrieved.Metadata != metadata {
		t.Errorf("metadata not retrieved: got %q", retrieved.Metadata)
	}
}

func TestSQLiteStore_Timestamps(t *testing.T) {
	store := setupTestDB(t)

	before := time.Now().UTC().Add(-1 * time.Second)
	ch, _ := store.CreateChannel("timestamp-test", ChannelTypeGroup, "")
	after := time.Now().UTC().Add(1 * time.Second)

	if ch.CreatedAt.Before(before) || ch.CreatedAt.After(after) {
		t.Errorf("unexpected created_at: %v (expected between %v and %v)", ch.CreatedAt, before, after)
	}

	if _, err := store.AddMessage("timestamp-test", "user", "test", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	history, _ := store.GetHistory("timestamp-test", 10)

	if len(history) > 0 {
		if history[0].CreatedAt.Before(before) {
			t.Errorf("message timestamp too early: %v (before %v)", history[0].CreatedAt, before)
		}
	}
}

func TestSQLiteStore_Reactions(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("reaction-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	msg, err := store.AddMessage("reaction-test", "user1", "Great work!", TypeText, "")
	if err != nil {
		t.Fatal(err)
	}

	// Add reactions
	err = store.AddReaction(msg.ID, "👍", "user2")
	if err != nil {
		t.Fatalf("failed to add reaction: %v", err)
	}
	err = store.AddReaction(msg.ID, "👍", "user3")
	if err != nil {
		t.Fatalf("failed to add second reaction: %v", err)
	}
	err = store.AddReaction(msg.ID, "🎉", "user2")
	if err != nil {
		t.Fatalf("failed to add different emoji: %v", err)
	}

	// Get reactions
	reactions, err := store.GetReactions(msg.ID)
	if err != nil {
		t.Fatalf("failed to get reactions: %v", err)
	}

	if len(reactions["👍"]) != 2 {
		t.Errorf("expected 2 thumbsup reactions, got %d", len(reactions["👍"]))
	}
	if len(reactions["🎉"]) != 1 {
		t.Errorf("expected 1 party reaction, got %d", len(reactions["🎉"]))
	}

	// Remove reaction
	err = store.RemoveReaction(msg.ID, "👍", "user2")
	if err != nil {
		t.Fatalf("failed to remove reaction: %v", err)
	}

	reactions, _ = store.GetReactions(msg.ID)
	if len(reactions["👍"]) != 1 {
		t.Errorf("expected 1 thumbsup after removal, got %d", len(reactions["👍"]))
	}

	// Toggle reaction (remove)
	added, err := store.ToggleReaction(msg.ID, "👍", "user3")
	if err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	if added {
		t.Error("expected toggle to remove, not add")
	}

	// Toggle reaction (add)
	added, err = store.ToggleReaction(msg.ID, "🚀", "user4")
	if err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	if !added {
		t.Error("expected toggle to add, not remove")
	}

	reactions, _ = store.GetReactions(msg.ID)
	if len(reactions["🚀"]) != 1 {
		t.Errorf("expected rocket reaction after toggle add, got %d", len(reactions["🚀"]))
	}
}

func TestSQLiteStore_Close(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSQLiteStore(tmpDir)

	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Close again should be safe (db is nil)
	if err := store.Close(); err != nil {
		t.Errorf("second close should not error: %v", err)
	}
}

func TestSQLiteStore_DB(t *testing.T) {
	store := setupTestDB(t)

	db := store.DB()
	if db == nil {
		t.Error("DB() should return non-nil connection")
	}

	// Verify the connection works
	ctx := context.Background()
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		t.Fatalf("DB connection not working: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %d", result)
	}
}

func TestSQLiteStore_SetChannelDescription(t *testing.T) {
	store := setupTestDB(t)

	// Create a channel
	if _, err := store.CreateChannel("desc-test", ChannelTypeGroup, "Original description"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// Update description
	if err := store.SetChannelDescription("desc-test", "Updated description"); err != nil {
		t.Fatalf("failed to set description: %v", err)
	}

	// Verify update
	ch, err := store.GetChannel("desc-test")
	if err != nil {
		t.Fatalf("failed to get channel: %v", err)
	}
	if ch.Description != "Updated description" {
		t.Errorf("expected 'Updated description', got %q", ch.Description)
	}

	// Test nonexistent channel
	err = store.SetChannelDescription("nonexistent", "desc")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestSQLiteStore_CloseNilDB(t *testing.T) {
	store := &SQLiteStore{}
	// Close on nil db should not error
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil db should not error: %v", err)
	}
}

// TestSQLiteStore_DeleteChannelWithMessages tests deletion of a channel
// that has messages, members, mentions, and reactions (issue #738).
func TestSQLiteStore_DeleteChannelWithMessages(t *testing.T) {
	store := setupTestDB(t)

	// Create channel with members
	if _, err := store.CreateChannel("to-delete-full", ChannelTypeGroup, "Test channel"); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if err := store.AddMember("to-delete-full", "engineer-01"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := store.AddMember("to-delete-full", "engineer-02"); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	// Add messages (which creates FTS entries)
	msg1, err := store.AddMessage("to-delete-full", "engineer-01", "Hello world", TypeText, "")
	if err != nil {
		t.Fatalf("failed to add message: %v", err)
	}
	msg2, err := store.AddMessage("to-delete-full", "engineer-02", "@engineer-01 check this", TypeTask, "")
	if err != nil {
		t.Fatalf("failed to add task message: %v", err)
	}

	// Add mention
	if err := store.AddMention(msg2.ID, "engineer-01"); err != nil {
		t.Fatalf("failed to add mention: %v", err)
	}

	// Add reactions
	if err := store.AddReaction(msg1.ID, "👍", "engineer-02"); err != nil {
		t.Fatalf("failed to add reaction: %v", err)
	}
	if err := store.AddReaction(msg2.ID, "✅", "engineer-01"); err != nil {
		t.Fatalf("failed to add reaction: %v", err)
	}

	// Verify data exists
	history, _ := store.GetHistory("to-delete-full", 10)
	if len(history) != 2 {
		t.Errorf("expected 2 messages before delete, got %d", len(history))
	}

	// This is the bug from issue #738 - deletion should work
	if err := store.DeleteChannel("to-delete-full"); err != nil {
		t.Fatalf("failed to delete channel with messages: %v", err)
	}

	// Verify channel is gone
	ch, _ := store.GetChannel("to-delete-full")
	if ch != nil {
		t.Error("channel should be deleted")
	}
}
