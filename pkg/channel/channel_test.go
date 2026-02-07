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

// --- Message Types ---

func TestValidMessageTypes(t *testing.T) {
	types := ValidMessageTypes()
	if len(types) != 5 {
		t.Errorf("ValidMessageTypes() returned %d types, want 5", len(types))
	}

	expected := []MessageType{TypeMessage, TypeTask, TypeReview, TypeApproval, TypeMerge}
	for _, e := range expected {
		found := false
		for _, mt := range types {
			if mt == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidMessageTypes() missing %q", e)
		}
	}
}

func TestIsValidMessageType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"message", true},
		{"task", true},
		{"review", true},
		{"approval", true},
		{"merge", true},
		{"invalid", false},
		{"", false},
		{"MESSAGE", false}, // case sensitive
	}

	for _, tt := range tests {
		got := IsValidMessageType(tt.input)
		if got != tt.want {
			t.Errorf("IsValidMessageType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestAddHistoryWithType(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	if err := s.AddHistoryWithType("ch", "user", "task message", TypeTask); err != nil {
		t.Fatalf("AddHistoryWithType: %v", err)
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	if history[0].Type != TypeTask {
		t.Errorf("Type = %q, want %q", history[0].Type, TypeTask)
	}
}

func TestAddHistoryWithTypeDefaultsToMessage(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	if err := s.AddHistoryWithType("ch", "user", "msg", ""); err != nil {
		t.Fatalf("AddHistoryWithType: %v", err)
	}

	history, err := s.GetHistory("ch")
	if err != nil {
		t.Fatal(err)
	}
	if history[0].Type != TypeMessage {
		t.Errorf("Type = %q, want %q", history[0].Type, TypeMessage)
	}
}

func TestGetHistoryByType(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	// Add messages of different types
	_ = s.AddHistoryWithType("ch", "user", "msg1", TypeMessage)
	_ = s.AddHistoryWithType("ch", "user", "task1", TypeTask)
	_ = s.AddHistoryWithType("ch", "user", "task2", TypeTask)
	_ = s.AddHistoryWithType("ch", "user", "review1", TypeReview)

	// Filter by task
	tasks, err := s.GetHistoryByType("ch", TypeTask)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Errorf("GetHistoryByType(task) = %d, want 2", len(tasks))
	}

	// Filter by review
	reviews, err := s.GetHistoryByType("ch", TypeReview)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 {
		t.Errorf("GetHistoryByType(review) = %d, want 1", len(reviews))
	}

	// Filter by message
	messages, err := s.GetHistoryByType("ch", TypeMessage)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Errorf("GetHistoryByType(message) = %d, want 1", len(messages))
	}
}

func TestGetHistoryByTypeBackwardCompatibility(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	// Use old AddHistory (no type) - should default to message
	_ = s.AddHistory("ch", "user", "old message")

	// Filter by message should include it
	messages, err := s.GetHistoryByType("ch", TypeMessage)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Errorf("GetHistoryByType(message) = %d, want 1 (backward compat)", len(messages))
	}
}

func TestGetHistoryByTypeChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetHistoryByType("nonexistent", TypeTask)
	if err == nil {
		t.Fatal("GetHistoryByType nonexistent: expected error, got nil")
	}
}
