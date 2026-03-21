package channel

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// newTestStore creates a Store backed by SQLite in a temp directory.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}
	s := NewStore(dir)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// --- Create ---

func TestCreate(t *testing.T) {
	s := newTestStore(t)

	ch, err := s.Create("test-create")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if ch.Name != "test-create" {
		t.Errorf("Name = %q, want %q", ch.Name, "test-create")
	}
	if len(ch.Members) != 0 {
		t.Errorf("Members = %v, want empty slice", ch.Members)
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("dup-ch"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Create("dup-ch")
	if err == nil {
		t.Fatal("Create duplicate: expected error, got nil")
	}
}

func TestCreateMultiple(t *testing.T) {
	s := newTestStore(t)

	names := []string{"proj-a", "proj-b", "proj-c"}
	for _, n := range names {
		if _, err := s.Create(n); err != nil {
			t.Fatalf("Create(%q): %v", n, err)
		}
	}
	// 3 seeded + 3 created = 6
	if got := len(s.List()); got != 6 {
		t.Fatalf("List len = %d, want 6", got)
	}
}

// --- Get ---

func TestGet(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("test-get"); err != nil {
		t.Fatal(err)
	}

	ch, ok := s.Get("test-get")
	if !ok {
		t.Fatal("Get: expected channel to exist")
	}
	if ch.Name != "test-get" {
		t.Errorf("Name = %q, want %q", ch.Name, "test-get")
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

func TestListDefaultChannels(t *testing.T) {
	s := newTestStore(t)

	// SQLite schema seeds 3 default channels: all, engineering, general
	channels := s.List()
	if len(channels) != 3 {
		t.Fatalf("List default store: got %d channels, want 3", len(channels))
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

	// 3 seeded + 2 created = 5
	channels := s.List()
	if len(channels) != 5 {
		t.Fatalf("List: got %d channels, want 5", len(channels))
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

	if _, err := s.Create("to-delete"); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("to-delete"); ok {
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

	// SQLite uses INSERT OR IGNORE — duplicate adds are idempotent (no error)
	if err := s.AddMember("ch", "alice"); err != nil {
		t.Fatalf("AddMember duplicate: unexpected error: %v", err)
	}

	// Verify only one member exists
	members, err := s.GetMembers("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("members len = %d, want 1 (should be idempotent)", len(members))
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

// --- Persistence round-trip ---

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	// Populate via first store instance
	s1 := NewStore(dir)
	// "general" and "engineering" are seeded by SQLite schema
	if err := s1.AddMember("general", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddMember("general", "bob"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddHistory("general", "alice", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := s1.AddMember("engineering", "charlie"); err != nil {
		t.Fatal(err)
	}
	_ = s1.Close()

	// Open a fresh store on the same directory — data should persist
	s2 := NewStore(dir)
	defer func() { _ = s2.Close() }()

	// Verify general channel
	ch, ok := s2.Get("general")
	if !ok {
		t.Fatal("general channel not found after reopen")
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
		t.Fatal("engineering channel not found after reopen")
	}
	if len(ch2.Members) != 1 || ch2.Members[0] != "charlie" {
		t.Errorf("engineering members = %v, want [charlie]", ch2.Members)
	}
}

func TestLoadNoOp(t *testing.T) {
	s := newTestStore(t)

	// Load is a no-op, should always succeed
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Default seeded channels should be present
	if len(s.List()) != 3 {
		t.Errorf("List after Load = %d, want 3 (seeded defaults)", len(s.List()))
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

	// 3 seeded + 20 created = 23
	if got := len(s.List()); got != 23 {
		t.Errorf("List after concurrent creates = %d, want 23", got)
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
	// 3 seeded (all, engineering, general) + 4 created = 7
	channels := s.List()
	if len(channels) != 7 {
		t.Fatalf("List() returned %d channels, want 7", len(channels))
	}

	expected := []string{"all", "alpha", "beta", "engineering", "general", "middle", "zebra"}
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

	if _, err := s.Create("desc-ch"); err != nil {
		t.Fatal(err)
	}

	if err := s.SetDescription("desc-ch", "Main discussion channel"); err != nil {
		t.Fatalf("SetDescription: unexpected error: %v", err)
	}

	ch, ok := s.Get("desc-ch")
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

	if _, err := s.Create("getdesc-ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDescription("getdesc-ch", "Team channel"); err != nil {
		t.Fatal(err)
	}

	desc, err := s.GetDescription("getdesc-ch")
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

	// Create store, set description, close
	s1 := NewStore(dir)
	if _, err := s1.Create("persist-desc"); err != nil {
		t.Fatal(err)
	}
	if err := s1.SetDescription("persist-desc", "Persisted description"); err != nil {
		t.Fatal(err)
	}
	_ = s1.Close()

	// Reopen store, verify description persisted
	s2 := NewStore(dir)
	defer func() { _ = s2.Close() }()
	ch, ok := s2.Get("persist-desc")
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
	store := newTestStore(t)

	// Create a channel first
	_, err := store.Create("test-close")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Close should succeed
	if err := store.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestStoreGetNotFound tests Get for non-existent channel
func TestStoreGetNotFound(t *testing.T) {
	store := newTestStore(t)

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
	store := newTestStore(t)

	// Create channel
	created, err := store.Create("test-get-existing")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get should find it
	ch, found := store.Get("test-get-existing")
	if !found {
		t.Fatal("Get: expected to find channel")
	}
	if ch.Name != created.Name {
		t.Errorf("Get Name = %q, want %q", ch.Name, created.Name)
	}
}

// TestStoreCreateDuplicate tests Create with duplicate name
func TestStoreCreateDuplicate(t *testing.T) {
	store := newTestStore(t)

	// Create first channel
	_, err := store.Create("dup-test-2")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	// Create duplicate should fail
	_, err = store.Create("dup-test-2")
	if err == nil {
		t.Error("Create duplicate: expected error")
	}
}

// TestStorePersistence tests that data persists across store instances
func TestStorePersistence(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	// Create and close
	store1 := NewStore(dir)
	_, err := store1.Create("persist-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = store1.Close()

	// Reopen and verify
	store2 := NewStore(dir)
	defer func() { _ = store2.Close() }()

	ch, found := store2.Get("persist-test")
	if !found {
		t.Error("Persisted channel not found after reload")
	}
	if ch == nil || ch.Name != "persist-test" {
		t.Error("Persisted channel has wrong name")
	}
}

// TestStoreListDefault tests List returns seeded channels
func TestStoreListDefault(t *testing.T) {
	store := newTestStore(t)

	list := store.List()
	// SQLite schema seeds 3 default channels
	if len(list) != 3 {
		t.Errorf("List default store: expected 3, got %d", len(list))
	}
}

// TestStoreDeleteNonExistent tests Delete for non-existent channel
func TestStoreDeleteNonExistent(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Delete nonexistent: expected error")
	}
}

// --- SQLite-backed Store tests (#1309) ---

// newSQLiteTestStore creates a Store backed by SQLite for testing.
// This is now identical to newTestStore since all stores use SQLite.
func newSQLiteTestStore(t *testing.T) *Store {
	t.Helper()
	return newTestStore(t)
}

// TestGetSQLiteBackend tests Get with SQLite backend
func TestGetSQLiteBackend(t *testing.T) {
	store := newSQLiteTestStore(t)

	// Default channels exist (general, engineering, all)
	ch, exists := store.Get("general")
	if !exists {
		t.Fatal("Get general: expected to exist")
	}
	if ch == nil {
		t.Fatal("Get general: returned nil channel")
	}
	if ch.Name != "general" {
		t.Errorf("Name = %q, want general", ch.Name)
	}
}

// TestGetSQLiteBackendNotFound tests Get for non-existent channel with SQLite
func TestGetSQLiteBackendNotFound(t *testing.T) {
	store := newSQLiteTestStore(t)

	_, exists := store.Get("nonexistent-channel")
	if exists {
		t.Error("Get nonexistent: should not exist")
	}
}

// TestListSQLiteBackend tests List with SQLite backend
func TestListSQLiteBackend(t *testing.T) {
	store := newSQLiteTestStore(t)

	channels := store.List()
	if len(channels) < 3 {
		t.Errorf("List: expected at least 3 default channels, got %d", len(channels))
	}

	// Check for default channels
	names := make(map[string]bool)
	for _, ch := range channels {
		names[ch.Name] = true
	}
	for _, name := range []string{"general", "engineering", "all"} {
		if !names[name] {
			t.Errorf("List: missing default channel %q", name)
		}
	}
}

// TestCreateSQLiteBackend tests Create with SQLite backend
func TestCreateSQLiteBackend(t *testing.T) {
	store := newSQLiteTestStore(t)

	ch, err := store.Create("test-channel-sqlite")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ch.Name != "test-channel-sqlite" {
		t.Errorf("Name = %q, want test-channel-sqlite", ch.Name)
	}

	// Verify it can be retrieved
	got, exists := store.Get("test-channel-sqlite")
	if !exists {
		t.Fatal("Get after Create: should exist")
	}
	if got.Name != "test-channel-sqlite" {
		t.Errorf("Name after Get = %q, want test-channel-sqlite", got.Name)
	}
}

// TestCreateDuplicateSQLiteBackend tests Create duplicate with SQLite backend
func TestCreateDuplicateSQLiteBackend(t *testing.T) {
	store := newSQLiteTestStore(t)

	_, err := store.Create("general") // Already exists
	if err == nil {
		t.Error("Create duplicate: expected error")
	}
}

// TestDeleteSQLiteBackend tests Delete with SQLite backend
func TestDeleteSQLiteBackend(t *testing.T) {
	store := newSQLiteTestStore(t)

	// Create a channel first
	if _, err := store.Create("to-delete"); err != nil {
		t.Fatal(err)
	}

	// Delete it
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	_, exists := store.Get("to-delete")
	if exists {
		t.Error("Get after Delete: should not exist")
	}
}
