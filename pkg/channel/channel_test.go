package channel

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// newTestStore creates a Store backed by a temp directory, returning the store
// and a cleanup function.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}
	return NewStore(dir)
}

// --- Create ---

func TestCreate(t *testing.T) {
	s := newTestStore(t)

	ch, err := s.Create("general")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if ch.Name != "general" {
		t.Errorf("Name = %q, want %q", ch.Name, "general")
	}
	if len(ch.Members) != 0 {
		t.Errorf("Members = %v, want empty slice", ch.Members)
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("general"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Create("general")
	if err == nil {
		t.Fatal("Create duplicate: expected error, got nil")
	}
}

func TestCreateMultiple(t *testing.T) {
	s := newTestStore(t)

	names := []string{"general", "engineering", "qa"}
	for _, n := range names {
		if _, err := s.Create(n); err != nil {
			t.Fatalf("Create(%q): %v", n, err)
		}
	}
	if got := len(s.List()); got != 3 {
		t.Fatalf("List len = %d, want 3", got)
	}
}

// --- Get ---

func TestGet(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("general"); err != nil {
		t.Fatal(err)
	}

	ch, ok := s.Get("general")
	if !ok {
		t.Fatal("Get: expected channel to exist")
	}
	if ch.Name != "general" {
		t.Errorf("Name = %q, want %q", ch.Name, "general")
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)

	_, ok := s.Get("nonexistent")
	if ok {
		t.Fatal("Get nonexistent: expected ok=false")
	}
}

// --- List ---

func TestListEmpty(t *testing.T) {
	s := newTestStore(t)

	channels := s.List()
	if len(channels) != 0 {
		t.Fatalf("List empty store: got %d channels, want 0", len(channels))
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("beta"); err != nil {
		t.Fatal(err)
	}

	channels := s.List()
	if len(channels) != 2 {
		t.Fatalf("List: got %d channels, want 2", len(channels))
	}

	names := map[string]bool{}
	for _, ch := range channels {
		names[ch.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("List: missing expected channels, got %v", names)
	}
}

// --- Delete ---

func TestDelete(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("general"); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete("general"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("general"); ok {
		t.Fatal("Get after Delete: expected channel to be gone")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.Delete("nonexistent")
	if err == nil {
		t.Fatal("Delete nonexistent: expected error, got nil")
	}
}

// --- AddMember ---

func TestAddMember(t *testing.T) {
	tests := []struct {
		name    string
		members []string
		wantLen int
	}{
		{"single member", []string{"alice"}, 1},
		{"two members", []string{"alice", "bob"}, 2},
		{"three members", []string{"alice", "bob", "charlie"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			if _, err := s.Create("ch"); err != nil {
				t.Fatal(err)
			}

			for _, m := range tt.members {
				if err := s.AddMember("ch", m); err != nil {
					t.Fatalf("AddMember(%q): %v", m, err)
				}
			}

			members, err := s.GetMembers("ch")
			if err != nil {
				t.Fatal(err)
			}
			if len(members) != tt.wantLen {
				t.Errorf("members len = %d, want %d", len(members), tt.wantLen)
			}
		})
	}
}

func TestAddMemberDuplicate(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember("ch", "alice"); err != nil {
		t.Fatal(err)
	}

	err := s.AddMember("ch", "alice")
	if err == nil {
		t.Fatal("AddMember duplicate: expected error, got nil")
	}
}

func TestAddMemberChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.AddMember("nonexistent", "alice")
	if err == nil {
		t.Fatal("AddMember to nonexistent channel: expected error, got nil")
	}
}

// --- RemoveMember ---

func TestRemoveMember(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember("ch", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember("ch", "bob"); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveMember("ch", "alice"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members, err := s.GetMembers("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("members len = %d, want 1", len(members))
	}
	if members[0] != "bob" {
		t.Errorf("remaining member = %q, want %q", members[0], "bob")
	}
}

func TestRemoveMemberNotAMember(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	err := s.RemoveMember("ch", "alice")
	if err == nil {
		t.Fatal("RemoveMember non-member: expected error, got nil")
	}
}

func TestRemoveMemberChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.RemoveMember("nonexistent", "alice")
	if err == nil {
		t.Fatal("RemoveMember from nonexistent channel: expected error, got nil")
	}
}

// --- GetMembers ---

func TestGetMembersReturnsCopy(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember("ch", "alice"); err != nil {
		t.Fatal(err)
	}

	members, err := s.GetMembers("ch")
	if err != nil {
		t.Fatal(err)
	}
	members[0] = "MUTATED"

	original, err := s.GetMembers("ch")
	if err != nil {
		t.Fatal(err)
	}
	if original[0] != "alice" {
		t.Error("GetMembers did not return a copy; mutation leaked")
	}
}

func TestGetMembersChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetMembers("nonexistent")
	if err == nil {
		t.Fatal("GetMembers nonexistent: expected error, got nil")
	}
}

// --- AddHistory ---

func TestAddHistory(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	if err := s.AddHistory("ch", "test-user", "hello world"); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].Message != "hello world" {
		t.Errorf("Message = %q, want %q", history[0].Message, "hello world")
	}
	if history[0].Time.IsZero() {
		t.Error("Time should not be zero")
	}
}

func TestAddHistoryChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.AddHistory("nonexistent", "test-user", "msg")
	if err == nil {
		t.Fatal("AddHistory nonexistent: expected error, got nil")
	}
}

func TestAddHistoryTruncatesAt100(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 110; i++ {
		if err := s.AddHistory("ch", "test-user", fmt.Sprintf("msg-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 100 {
		t.Fatalf("history len = %d, want 100", len(history))
	}
	// Oldest kept message should be msg-10 (first 10 trimmed)
	if history[0].Message != "msg-10" {
		t.Errorf("oldest message = %q, want %q", history[0].Message, "msg-10")
	}
	if history[99].Message != "msg-109" {
		t.Errorf("newest message = %q, want %q", history[99].Message, "msg-109")
	}
}

// --- GetHistory ---

func TestGetHistoryReturnsCopy(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "test-user", "original"); err != nil {
		t.Fatal(err)
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	history[0].Message = "MUTATED"

	original, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if original[0].Message != "original" {
		t.Error("GetHistory did not return a copy; mutation leaked")
	}
}

func TestGetHistoryChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetHistory("nonexistent")
	if err == nil {
		t.Fatal("GetHistory nonexistent: expected error, got nil")
	}
}

func TestGetHistoryEmpty(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Errorf("history len = %d, want 0", len(history))
	}
}

func TestGetHistoryIncludesReactions(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "hello"); err != nil {
		t.Fatal(err)
	}

	// Add a reaction to the message
	if err := s.AddReaction("ch", 0, "👍", "bob"); err != nil {
		t.Fatal(err)
	}

	// Get history and verify reaction is included
	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}

	// Check reactions are populated
	if history[0].Reactions == nil {
		t.Fatal("expected Reactions to be populated, got nil")
	}
	if users, ok := history[0].Reactions["👍"]; !ok {
		t.Error("expected 👍 reaction to be present")
	} else if len(users) != 1 || users[0] != "bob" {
		t.Errorf("expected [bob] for 👍, got %v", users)
	}
}

// --- Load / Save round-trip ---

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	// Populate and save
	s1 := NewStore(dir)
	if _, err := s1.Create("general"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddMember("general", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddMember("general", "bob"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddHistory("general", "alice", "hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Create("engineering"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddMember("engineering", "charlie"); err != nil {
		t.Fatal(err)
	}

	if err := s1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into a fresh store
	s2 := NewStore(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify general channel
	ch, ok := s2.Get("general")
	if !ok {
		t.Fatal("Load: general channel not found")
	}
	if len(ch.Members) != 2 {
		t.Errorf("general members = %d, want 2", len(ch.Members))
	}
	if len(ch.History) != 1 {
		t.Errorf("general history = %d, want 1", len(ch.History))
	}
	if ch.History[0].Message != "hello" {
		t.Errorf("history message = %q, want %q", ch.History[0].Message, "hello")
	}

	// Verify engineering channel
	ch2, ok := s2.Get("engineering")
	if !ok {
		t.Fatal("Load: engineering channel not found")
	}
	if len(ch2.Members) != 1 || ch2.Members[0] != "charlie" {
		t.Errorf("engineering members = %v, want [charlie]", ch2.Members)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Load with no file on disk should succeed with empty state
	if err := s.Load(); err != nil {
		t.Fatalf("Load nonexistent: %v", err)
	}
	if len(s.List()) != 0 {
		t.Errorf("List after load nonexistent = %d, want 0", len(s.List()))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bcDir, "channels.json"), []byte("{bad json"), 0600); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	err := s.Load()
	if err == nil {
		t.Fatal("Load invalid JSON: expected error, got nil")
	}
}

// --- Concurrent access ---

func TestConcurrentCreateAndList(t *testing.T) {
	s := newTestStore(t)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := s.Create(fmt.Sprintf("ch-%d", i)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()

	if got := len(s.List()); got != 20 {
		t.Errorf("List after concurrent creates = %d, want 20", got)
	}
}

func TestConcurrentAddMember(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.AddMember("ch", fmt.Sprintf("agent-%d", i)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()

	members, err := s.GetMembers("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 20 {
		t.Errorf("members after concurrent adds = %d, want 20", len(members))
	}
}

func TestConcurrentAddHistory(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.AddHistory("ch", "test-user", fmt.Sprintf("msg-%d", i)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 50 {
		t.Errorf("history after concurrent adds = %d, want 50", len(history))
	}
}

// --- List Stable Ordering ---

func TestListStableOrdering(t *testing.T) {
	s := newTestStore(t)

	// Create channels in random order
	names := []string{"zebra", "alpha", "middle", "beta"}
	for _, name := range names {
		if _, err := s.Create(name); err != nil {
			t.Fatal(err)
		}
	}

	// List should return channels sorted alphabetically
	channels := s.List()
	if len(channels) != 4 {
		t.Fatalf("List() returned %d channels, want 4", len(channels))
	}

	expected := []string{"alpha", "beta", "middle", "zebra"}
	for i, ch := range channels {
		if ch.Name != expected[i] {
			t.Errorf("List()[%d].Name = %q, want %q", i, ch.Name, expected[i])
		}
	}

	// Call List multiple times - order should be stable
	for iter := 0; iter < 10; iter++ {
		channels := s.List()
		for i, ch := range channels {
			if ch.Name != expected[i] {
				t.Errorf("iter %d: List()[%d].Name = %q, want %q", iter, i, ch.Name, expected[i])
			}
		}
	}
}

// --- Description ---

func TestSetDescription(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("general"); err != nil {
		t.Fatal(err)
	}

	if err := s.SetDescription("general", "Main discussion channel"); err != nil {
		t.Fatalf("SetDescription: unexpected error: %v", err)
	}

	ch, ok := s.Get("general")
	if !ok {
		t.Fatal("Get: channel not found")
	}
	if ch.Description != "Main discussion channel" {
		t.Errorf("Description = %q, want %q", ch.Description, "Main discussion channel")
	}
}

func TestSetDescriptionNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.SetDescription("nonexistent", "test")
	if err == nil {
		t.Fatal("SetDescription: expected error for nonexistent channel")
	}
}

func TestGetDescription(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("general"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDescription("general", "Team channel"); err != nil {
		t.Fatal(err)
	}

	desc, err := s.GetDescription("general")
	if err != nil {
		t.Fatalf("GetDescription: unexpected error: %v", err)
	}
	if desc != "Team channel" {
		t.Errorf("GetDescription = %q, want %q", desc, "Team channel")
	}
}

func TestGetDescriptionNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetDescription("nonexistent")
	if err == nil {
		t.Fatal("GetDescription: expected error for nonexistent channel")
	}
}

func TestDescriptionPersistence(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	// Create store, set description, save
	s1 := NewStore(dir)
	if _, err := s1.Create("general"); err != nil {
		t.Fatal(err)
	}
	if err := s1.SetDescription("general", "Persisted description"); err != nil {
		t.Fatal(err)
	}
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Create new store, load, verify description persisted
	s2 := NewStore(dir)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	ch, ok := s2.Get("general")
	if !ok {
		t.Fatal("Get: channel not found after reload")
	}
	if ch.Description != "Persisted description" {
		t.Errorf("Description after reload = %q, want %q", ch.Description, "Persisted description")
	}
}

// TestOpenStoreUsesSQLiteWhenDbExists verifies that OpenStore uses SQLite when .bc/channels.db exists,
// so CLI/TUI see the same channels as bc up (part of #341/#340).
func TestOpenStoreUsesSQLiteWhenDbExists(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Create channels.db via SQLiteStore (as bc up does)
	sqlStore := NewSQLiteStore(dir)
	if err := sqlStore.Open(); err != nil {
		t.Fatalf("Open SQLite: %v", err)
	}
	if _, err := sqlStore.CreateChannel("standup", ChannelTypeGroup, "Daily standup"); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	_ = sqlStore.AddMember("standup", "engineer-01")
	_ = sqlStore.Close()

	// OpenStore should use SQLite and see the channel
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	list := store.List()
	if len(list) == 0 {
		t.Fatal("OpenStore: expected at least one channel (standup), got none")
	}
	var found bool
	for _, ch := range list {
		if ch.Name == "standup" {
			found = true
			if ch.Description != "Daily standup" {
				t.Errorf("standup Description = %q, want Daily standup", ch.Description)
			}
			if len(ch.Members) < 1 {
				t.Error("standup should have engineer-01 as member")
			}
			break
		}
	}
	if !found {
		t.Errorf("OpenStore: channel list %v missing standup", list)
	}
}

// --- Additional coverage tests (#1236) ---

// TestStoreClose tests the Close function
func TestStoreClose(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Create a channel first
	_, err := store.Create("test-close")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Close should be no-op for JSON backend (no error)
	if err := store.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Store should still be usable after close (JSON backend)
	list := store.List()
	if len(list) != 1 {
		t.Errorf("List after close: expected 1, got %d", len(list))
	}
}

// TestStoreGetNotFound tests Get for non-existent channel
func TestStoreGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	ch, found := store.Get("nonexistent")
	if found {
		t.Error("Get nonexistent: expected not found")
	}
	if ch != nil {
		t.Error("Get nonexistent: expected nil channel")
	}
}

// TestStoreGetExisting tests Get for existing channel
func TestStoreGetExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Create channel
	created, err := store.Create("test-get")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get should find it
	ch, found := store.Get("test-get")
	if !found {
		t.Fatal("Get: expected to find channel")
	}
	if ch.Name != created.Name {
		t.Errorf("Get Name = %q, want %q", ch.Name, created.Name)
	}
}

// TestStoreCreateDuplicate tests Create with duplicate name
func TestStoreCreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Create first channel
	_, err := store.Create("dup-test")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Create duplicate should fail
	_, err = store.Create("dup-test")
	if err == nil {
		t.Error("Create duplicate: expected error")
	}
}

// TestStoreSaveAndLoad tests Save persists to disk
func TestStoreSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	// Create and save
	store1 := NewStore(path)
	if err := store1.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err := store1.Create("persist-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load in new store
	store2 := NewStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load store2: %v", err)
	}

	ch, found := store2.Get("persist-test")
	if !found {
		t.Error("Persisted channel not found after reload")
	}
	if ch == nil || ch.Name != "persist-test" {
		t.Error("Persisted channel has wrong name")
	}
}

// TestStoreSaveCreatesDirectory tests Save creates parent directory
func TestStoreSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "nested", "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err := store.Create("nested-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Save should create nested directories
	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		t.Error("Save did not create file in nested directory")
	}
}

// TestStoreListEmpty tests List on empty store
func TestStoreListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := store.List()
	if len(list) != 0 {
		t.Errorf("List empty store: expected 0, got %d", len(list))
	}
}

// TestStoreDeleteNonExistent tests Delete for non-existent channel
func TestStoreDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "channels.json")

	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Delete nonexistent: expected error")
	}
}
