package channel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddReaction(t *testing.T) {
	store := newTestStore(t)

	// Create a channel with a message
	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Add a reaction
	err = store.AddReaction("test", 0, "👍", "bob")
	if err != nil {
		t.Fatalf("failed to add reaction: %v", err)
	}

	// Verify reaction was added
	reactions, err := store.GetReactions("test", 0)
	if err != nil {
		t.Fatalf("failed to get reactions: %v", err)
	}

	if len(reactions) != 1 {
		t.Errorf("expected 1 reaction type, got %d", len(reactions))
	}

	users := reactions["👍"]
	if len(users) != 1 || users[0] != "bob" {
		t.Errorf("expected [bob], got %v", users)
	}
}

func TestAddReactionDuplicate(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Add same reaction twice
	_ = store.AddReaction("test", 0, "👍", "bob")
	_ = store.AddReaction("test", 0, "👍", "bob")

	reactions, _ := store.GetReactions("test", 0)
	users := reactions["👍"]
	if len(users) != 1 {
		t.Errorf("expected 1 user after duplicate add, got %d", len(users))
	}
}

func TestRemoveReaction(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Add and then remove a reaction
	_ = store.AddReaction("test", 0, "👍", "bob")
	err = store.RemoveReaction("test", 0, "👍", "bob")
	if err != nil {
		t.Fatalf("failed to remove reaction: %v", err)
	}

	reactions, _ := store.GetReactions("test", 0)
	if len(reactions) > 0 {
		t.Errorf("expected no reactions after removal, got %v", reactions)
	}
}

func TestToggleReaction(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Toggle on
	added, err := store.ToggleReaction("test", 0, "👍", "bob")
	if err != nil {
		t.Fatalf("failed to toggle reaction: %v", err)
	}
	if !added {
		t.Errorf("expected reaction to be added")
	}

	reactions, _ := store.GetReactions("test", 0)
	if len(reactions["👍"]) != 1 {
		t.Errorf("expected 1 user after toggle on")
	}

	// Toggle off
	added, err = store.ToggleReaction("test", 0, "👍", "bob")
	if err != nil {
		t.Fatalf("failed to toggle reaction: %v", err)
	}
	if added {
		t.Errorf("expected reaction to be removed")
	}

	reactions, _ = store.GetReactions("test", 0)
	if len(reactions) > 0 {
		t.Errorf("expected no reactions after toggle off")
	}
}

func TestMultipleReactions(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Great work!")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Multiple users react with different emojis
	_ = store.AddReaction("test", 0, "👍", "bob")
	_ = store.AddReaction("test", 0, "👍", "charlie")
	_ = store.AddReaction("test", 0, "🎉", "david")
	_ = store.AddReaction("test", 0, "❤️", "bob")

	reactions, err := store.GetReactions("test", 0)
	if err != nil {
		t.Fatalf("failed to get reactions: %v", err)
	}

	if len(reactions["👍"]) != 2 {
		t.Errorf("expected 2 users for 👍, got %d", len(reactions["👍"]))
	}
	if len(reactions["🎉"]) != 1 {
		t.Errorf("expected 1 user for 🎉, got %d", len(reactions["🎉"]))
	}
	if len(reactions["❤️"]) != 1 {
		t.Errorf("expected 1 user for ❤️, got %d", len(reactions["❤️"]))
	}
}

func TestReactionInvalidChannel(t *testing.T) {
	store := newTestStore(t)

	err := store.AddReaction("nonexistent", 0, "👍", "bob")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestReactionInvalidMessageIndex(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	err = store.AddReaction("test", 0, "👍", "bob")
	if err == nil {
		t.Error("expected error for invalid message index")
	}

	err = store.AddReaction("test", -1, "👍", "bob")
	if err == nil {
		t.Error("expected error for negative message index")
	}
}

func TestReactionsPersistence(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	store := NewStore(dir)
	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	_ = store.AddReaction("test", 0, "👍", "bob")
	_ = store.Close()

	// Reopen store and verify persistence
	store2 := NewStore(dir)
	defer func() { _ = store2.Close() }()

	reactions, err := store2.GetReactions("test", 0)
	if err != nil {
		t.Fatalf("failed to get reactions: %v", err)
	}

	if len(reactions["👍"]) != 1 || reactions["👍"][0] != "bob" {
		t.Errorf("reaction not persisted correctly: %v", reactions)
	}
}

func TestCommonReactions(t *testing.T) {
	expected := []string{"👍", "👎", "❤️", "🎉", "👀", "🚀"}
	if len(CommonReactions) != len(expected) {
		t.Errorf("expected %d common reactions, got %d", len(expected), len(CommonReactions))
	}

	for i, emoji := range expected {
		if CommonReactions[i] != emoji {
			t.Errorf("common reaction %d: expected %s, got %s", i, emoji, CommonReactions[i])
		}
	}
}

func TestGetReactionsReturnsEmptyForNoReactions(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Create("test")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	err = store.AddHistory("test", "alice", "Hello world")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	reactions, err := store.GetReactions("test", 0)
	if err != nil {
		t.Fatalf("failed to get reactions: %v", err)
	}

	if len(reactions) != 0 {
		t.Errorf("expected empty reactions, got %v", reactions)
	}
}

// TestStoreBackendNotNil verifies NewStore always creates a backend.
func TestStoreBackendNotNil(t *testing.T) {
	store := newTestStore(t)
	if store.backend == nil {
		t.Error("expected non-nil backend")
	}
}
