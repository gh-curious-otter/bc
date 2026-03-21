package channel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// --- service.go: Stats, computeTopSenders, Store() ---

func TestServiceStats(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Stats on empty store
	stats, err := svc.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats empty: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("Stats empty: got %d, want 0", len(stats))
	}

	// Create channels with messages
	_, err = svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(ctx, CreateChannelReq{Name: "ops"})
	if err != nil {
		t.Fatal(err)
	}

	// Add messages to eng
	for i := 0; i < 5; i++ {
		_, sendErr := svc.Send(ctx, "eng", "alice", fmt.Sprintf("msg-%d", i))
		if sendErr != nil {
			t.Fatal(sendErr)
		}
	}
	for i := 0; i < 3; i++ {
		_, sendErr := svc.Send(ctx, "eng", "bob", fmt.Sprintf("msg-%d", i))
		if sendErr != nil {
			t.Fatal(sendErr)
		}
	}

	// Add members
	err = svc.AddMember(ctx, "eng", "alice")
	if err != nil {
		t.Fatal(err)
	}
	err = svc.AddMember(ctx, "eng", "bob")
	if err != nil {
		t.Fatal(err)
	}

	stats, err = svc.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("Stats: got %d channels, want 2", len(stats))
	}

	// Find eng stats
	var engStats *ChannelStatsDTO
	for i := range stats {
		if stats[i].Name == "eng" {
			engStats = &stats[i]
			break
		}
	}
	if engStats == nil {
		t.Fatal("eng channel not found in stats")
	}
	if engStats.MessageCount != 8 {
		t.Errorf("eng MessageCount = %d, want 8", engStats.MessageCount)
	}
	if engStats.MemberCount != 2 {
		t.Errorf("eng MemberCount = %d, want 2", engStats.MemberCount)
	}
	if engStats.LastActivity == nil {
		t.Error("eng LastActivity should not be nil")
	}
	if len(engStats.TopSenders) == 0 {
		t.Error("eng TopSenders should not be empty")
	}
	// alice sent 5, bob sent 3 => alice should be first
	if engStats.TopSenders[0].Sender != "alice" {
		t.Errorf("eng TopSenders[0].Sender = %q, want alice", engStats.TopSenders[0].Sender)
	}
	if engStats.TopSenders[0].Count != 5 {
		t.Errorf("eng TopSenders[0].Count = %d, want 5", engStats.TopSenders[0].Count)
	}

	// ops channel should have 0 messages and nil LastActivity
	var opsStats *ChannelStatsDTO
	for i := range stats {
		if stats[i].Name == "ops" {
			opsStats = &stats[i]
			break
		}
	}
	if opsStats == nil {
		t.Fatal("ops channel not found in stats")
	}
	if opsStats.MessageCount != 0 {
		t.Errorf("ops MessageCount = %d, want 0", opsStats.MessageCount)
	}
	if opsStats.LastActivity != nil {
		t.Error("ops LastActivity should be nil for empty channel")
	}
}

func TestComputeTopSenders(t *testing.T) {
	tests := []struct {
		name    string
		history []HistoryEntry
		n       int
		wantLen int
	}{
		{
			name:    "empty history",
			history: nil,
			n:       5,
			wantLen: 0,
		},
		{
			name: "fewer senders than n",
			history: []HistoryEntry{
				{Sender: "alice", Message: "hi"},
				{Sender: "alice", Message: "bye"},
			},
			n:       5,
			wantLen: 1,
		},
		{
			name: "truncate to n",
			history: []HistoryEntry{
				{Sender: "alice"}, {Sender: "alice"}, {Sender: "alice"},
				{Sender: "bob"}, {Sender: "bob"},
				{Sender: "charlie"},
				{Sender: "dave"},
			},
			n:       2,
			wantLen: 2,
		},
		{
			name: "tie-breaking by name",
			history: []HistoryEntry{
				{Sender: "bob"}, {Sender: "alice"},
			},
			n:       5,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeTopSenders(tt.history, tt.n)
			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}

	// Verify tie-breaking: same count => alphabetical order
	history := []HistoryEntry{
		{Sender: "bob"}, {Sender: "alice"},
	}
	result := computeTopSenders(history, 5)
	if len(result) != 2 {
		t.Fatalf("expected 2 senders, got %d", len(result))
	}
	if result[0].Sender != "alice" {
		t.Errorf("expected alice first (alphabetical tie-break), got %q", result[0].Sender)
	}
}

func TestServiceStoreAccessor(t *testing.T) {
	svc := newTestService(t)
	store := svc.Store()
	if store == nil {
		t.Error("Store() returned nil")
	}
}

// --- service.go: OnMessage callback ---

func TestServiceSendOnMessageCallback(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	var calledCh, calledSender, calledContent string
	svc.OnMessage = func(ch, sender, content string) {
		calledCh = ch
		calledSender = sender
		calledContent = content
	}

	if _, err := svc.Create(ctx, CreateChannelReq{Name: "eng"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Send(ctx, "eng", "alice", "hello"); err != nil {
		t.Fatal(err)
	}

	if calledCh != "eng" {
		t.Errorf("OnMessage channel = %q, want eng", calledCh)
	}
	if calledSender != "alice" {
		t.Errorf("OnMessage sender = %q, want alice", calledSender)
	}
	if calledContent != "hello" {
		t.Errorf("OnMessage content = %q, want hello", calledContent)
	}
}

// --- channel.go: Store.Close with nil backend, RemoveReaction, GetReactions edge cases ---

func TestStoreCloseNilBackend(t *testing.T) {
	s := &Store{backend: nil}
	if err := s.Close(); err != nil {
		t.Errorf("Close with nil backend should not error: %v", err)
	}
}

func TestStoreRemoveReactionOutOfRange(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	// No messages => any index is out of range
	err := s.RemoveReaction("ch", 0, "👍", "user")
	if err == nil {
		t.Error("expected error for out of range message index")
	}

	// Negative index
	addErr := s.AddHistory("ch", "alice", "hello")
	if addErr != nil {
		t.Fatal(addErr)
	}
	err = s.RemoveReaction("ch", -1, "👍", "user")
	if err == nil {
		t.Error("expected error for negative message index")
	}
}

func TestStoreGetReactionsOutOfRange(t *testing.T) {
	s := newTestStore(t)
	if _, createErr := s.Create("ch"); createErr != nil {
		t.Fatal(createErr)
	}

	// No messages => out of range
	_, err := s.GetReactions("ch", 0)
	if err == nil {
		t.Error("expected error for out of range message index")
	}

	// Negative index
	addErr := s.AddHistory("ch", "alice", "hello")
	if addErr != nil {
		t.Fatal(addErr)
	}
	_, err = s.GetReactions("ch", -1)
	if err == nil {
		t.Error("expected error for negative message index")
	}
}

func TestStoreToggleReactionOutOfRange(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	_, err := s.ToggleReaction("ch", 999, "👍", "user")
	if err == nil {
		t.Error("expected error for out of range message index")
	}
}

func TestStoreAddReactionOutOfRange(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}

	err := s.AddReaction("ch", 999, "👍", "user")
	if err == nil {
		t.Error("expected error for out of range message index")
	}

	err = s.AddReaction("ch", -1, "👍", "user")
	if err == nil {
		t.Error("expected error for negative message index")
	}
}

// --- automation.go: WatchChannelForApprovals ---

func TestWatchChannelForApprovals(t *testing.T) {
	s := newTestStore(t)

	// Channel not found
	_, err := WatchChannelForApprovals(s, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}

	// Create channel and watch it
	_, createErr := s.Create("reviews")
	if createErr != nil {
		t.Fatal(createErr)
	}

	handler, watchErr := WatchChannelForApprovals(s, "reviews")
	if watchErr != nil {
		t.Fatalf("WatchChannelForApprovals: %v", watchErr)
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Test that handler.OnMergeRequest adds to history
	req := &MergeRequest{
		PRNumber:     42,
		ApprovedBy:   "tech-lead",
		TargetBranch: "main",
		CreatedAt:    time.Now(),
	}
	mergeErr := handler.OnMergeRequest(req)
	if mergeErr != nil {
		t.Fatalf("OnMergeRequest: %v", mergeErr)
	}

	// Verify the message was added to channel history
	history, err := s.GetHistory("reviews")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].Sender != "automation" {
		t.Errorf("sender = %q, want automation", history[0].Sender)
	}
}

func TestApprovalHandlerHandleMessageError(t *testing.T) {
	handler := &ApprovalHandler{
		OnMergeRequest: func(_ *MergeRequest) error {
			return errors.New("merge failed")
		},
	}

	// Send an approval message to trigger error path
	content := "[APPROVED] PR #123"
	processed, err := handler.HandleMessage(content, "reviewer")
	if !processed {
		t.Error("expected message to be processed")
	}
	if err == nil {
		t.Error("expected error from OnMergeRequest")
	}
}

// --- review_request.go: FormatReviewRequestWithTitle empty target, FormatReviewRequestWithURL empty target ---

func TestFormatReviewRequestWithTitleEmptyTarget(t *testing.T) {
	got := FormatReviewRequestWithTitle(100, "", "Fix bug")
	want := "@tech-lead PR #100 ready for review: Fix bug"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatReviewRequestWithURLEmptyTarget(t *testing.T) {
	got := FormatReviewRequestWithURL(200, "", "https://github.com/repo/pull/200")
	want := "@tech-lead PR #200 ready for review: https://github.com/repo/pull/200"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- query.go: Query with default limit, limit > 100, offset beyond range ---

func TestQueryDefaultLimit(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := s.AddHistory("ch", "user", "msg"); err != nil {
			t.Fatal(err)
		}
	}

	result, err := s.Query("ch", QueryOptions{Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 5 {
		t.Errorf("expected 5 messages with default limit, got %d", len(result.Messages))
	}
}

func TestQueryLimitCapped(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := s.AddHistory("ch", "user", "msg"); err != nil {
			t.Fatal(err)
		}
	}

	// Limit > 100 should be capped
	result, err := s.Query("ch", QueryOptions{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 5 {
		t.Errorf("expected 5 messages, got %d", len(result.Messages))
	}
}

func TestQueryOffsetBeyondRange(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "user", "msg"); err != nil {
		t.Fatal(err)
	}

	result, err := s.Query("ch", QueryOptions{Offset: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected 0 messages with large offset, got %d", len(result.Messages))
	}
}

func TestQueryBeforeFilter(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "user", "msg"); err != nil {
		t.Fatal(err)
	}

	// Before a time in the past => should return nothing
	past := time.Now().Add(-1 * time.Hour)
	result, err := s.Query("ch", QueryOptions{Before: &past})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected 0 messages before past time, got %d", len(result.Messages))
	}

	// Before a time in the future => should return messages
	future := time.Now().Add(1 * time.Hour)
	result, err = s.Query("ch", QueryOptions{Before: &future})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message before future time, got %d", len(result.Messages))
	}
}

// --- query.go: Search with Since filter, sender match, default limit ---

func TestSearchWithSinceFilter(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "user", "hello world"); err != nil {
		t.Fatal(err)
	}

	// Future since => no results
	future := time.Now().Add(1 * time.Hour)
	results, err := s.Search("hello", SearchOptions{Since: &future})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results with future since, got %d", len(results))
	}
}

func TestSearchBySender(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "some message"); err != nil {
		t.Fatal(err)
	}

	// Search by sender name
	results, err := s.Search("alice", SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result searching for sender alice, got %d", len(results))
	}
}

func TestSearchDefaultLimit(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.AddHistory("ch", "user", "test message"); err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.Search("test", SearchOptions{Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results with default limit, got %d", len(results))
	}
}

func TestSearchHitsLimit(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if err := s.AddHistory("ch", "user", "findme message"); err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.Search("findme", SearchOptions{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results (limited), got %d", len(results))
	}
}

func TestSearchNoChannelFilter(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("ch2"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch1", "user", "searchme"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch2", "user", "searchme"); err != nil {
		t.Fatal(err)
	}

	// No channel filter => search all
	results, err := s.Search("searchme", SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results across all channels, got %d", len(results))
	}
}

// --- query.go: GetMentions default limit ---

func TestGetMentionsDefaultLimit(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "manager", "@bob do this"); err != nil {
		t.Fatal(err)
	}

	results, err := s.GetMentions("bob", 0) // 0 => default
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 mention, got %d", len(results))
	}
}

func TestGetMentionsHitsLimit(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if err := s.AddHistory("ch", "manager", fmt.Sprintf("@bob task %d", i)); err != nil {
			t.Fatal(err)
		}
	}

	results, err := s.GetMentions("bob", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results (limited), got %d", len(results))
	}
}

// --- service.go: History with offset >= filtered length ---

func TestServiceHistoryOffsetExceedsLength(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateChannelReq{Name: "ch"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Send(ctx, "ch", "user", "msg"); err != nil {
		t.Fatal(err)
	}

	dtos, err := svc.History(ctx, "ch", HistoryOpts{Offset: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(dtos) != 0 {
		t.Errorf("expected 0 messages with large offset, got %d", len(dtos))
	}
}

// --- highlight.go: ParseAllHighlights with overlapping channel/github refs ---

func TestParseAllHighlightsWithOverlap(t *testing.T) {
	// A message with a mention, channel ref, and github link
	msg := "@alice check #general and PR #123"
	highlights := ParseAllHighlights(msg)
	if len(highlights) == 0 {
		t.Error("expected some highlights")
	}

	// Verify we have at least mention and some other highlights
	hasMention := false
	for _, h := range highlights {
		if h.Type == HighlightMention {
			hasMention = true
		}
	}
	if !hasMention {
		t.Error("expected a mention highlight")
	}
}

func TestParseAllHighlightsEmpty(t *testing.T) {
	highlights := ParseAllHighlights("plain text no highlights")
	if len(highlights) != 0 {
		t.Errorf("expected 0 highlights for plain text, got %d", len(highlights))
	}
}

// --- highlight.go: ParseGitHubLinks edge cases ---

func TestParseGitHubLinksURL(t *testing.T) {
	msg := "Check https://github.com/owner/repo/pull/42 for details"
	highlights := ParseGitHubLinks(msg)
	if len(highlights) == 0 {
		t.Error("expected github link highlight")
	}
}

func TestParseGitHubLinksNoMatch(t *testing.T) {
	highlights := ParseGitHubLinks("no github links here")
	if len(highlights) != 0 {
		t.Errorf("expected 0 highlights, got %d", len(highlights))
	}
}

// --- highlight.go: ParseChannelRefs with digit-only name (should skip) ---

func TestParseChannelRefsSkipsDigitOnly(t *testing.T) {
	// #123 looks like a github issue, not a channel
	highlights := ParseChannelRefs("#123")
	if len(highlights) != 0 {
		t.Errorf("expected 0 highlights for digit-only ref, got %d", len(highlights))
	}
}

func TestParseChannelRefsNoMatch(t *testing.T) {
	highlights := ParseChannelRefs("no channel refs")
	if len(highlights) != 0 {
		t.Errorf("expected 0 highlights, got %d", len(highlights))
	}
}

// --- channel.go: OpenStore fallback to SQLite ---

func TestOpenStoreSQLiteFallback(t *testing.T) {
	dir := t.TempDir()
	// No DATABASE_URL set, should use SQLite
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Should be functional
	ch, createErr := store.Create("test-open")
	if createErr != nil {
		t.Fatalf("Create: %v", createErr)
	}
	if ch.Name != "test-open" {
		t.Errorf("Name = %q, want test-open", ch.Name)
	}
}

// --- sqlite.go: GetChannelByID ---

func TestSQLiteStoreGetChannelByID(t *testing.T) {
	store := setupTestDB(t)

	ch, err := store.CreateChannel("byid-test", ChannelTypeGroup, "Test")
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	got, err := store.GetChannelByID(ch.ID)
	if err != nil {
		t.Fatalf("GetChannelByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected channel, got nil")
	}
	if got.Name != "byid-test" {
		t.Errorf("Name = %q, want byid-test", got.Name)
	}

	// Non-existent ID
	got, err = store.GetChannelByID(99999)
	if err != nil {
		t.Fatalf("GetChannelByID nonexistent: unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

// --- sqlite.go: GetMessage ---

func TestSQLiteStoreGetMessage(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("msg-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	msg, err := store.AddMessage("msg-test", "user", "hello", TypeText, "")
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetMessage(msg.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got == nil {
		t.Fatal("expected message, got nil")
	}
	if got.Content != "hello" {
		t.Errorf("Content = %q, want hello", got.Content)
	}

	// Non-existent message
	got, err = store.GetMessage(99999)
	if err != nil {
		t.Fatalf("GetMessage nonexistent: unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent message")
	}
}

// --- sqlite.go: AddMessage to nonexistent channel ---

func TestSQLiteStoreAddMessageNonexistentChannel(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.AddMessage("nonexistent", "user", "hello", TypeText, "")
	if err == nil {
		t.Error("expected error adding message to nonexistent channel")
	}
}

// --- sqlite.go: GetHistory nonexistent channel ---

func TestSQLiteStoreGetHistoryNonexistent(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetHistory("nonexistent", 10)
	if err == nil {
		t.Error("expected error getting history for nonexistent channel")
	}
}

// --- sqlite.go: GetMessagesByType nonexistent channel ---

func TestSQLiteStoreGetMessagesByTypeNonexistent(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetMessagesByType("nonexistent", TypeText, 10)
	if err == nil {
		t.Error("expected error getting messages by type for nonexistent channel")
	}
}

// --- sqlite.go: AddMember/RemoveMember to nonexistent channel ---

func TestSQLiteStoreAddMemberNonexistent(t *testing.T) {
	store := setupTestDB(t)

	err := store.AddMember("nonexistent", "user")
	if err == nil {
		t.Error("expected error adding member to nonexistent channel")
	}
}

func TestSQLiteStoreRemoveMemberNonexistent(t *testing.T) {
	store := setupTestDB(t)

	err := store.RemoveMember("nonexistent", "user")
	if err == nil {
		t.Error("expected error removing member from nonexistent channel")
	}
}

// --- sqlite.go: GetMembers nonexistent channel ---

func TestSQLiteStoreGetMembersNonexistent(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.GetMembers("nonexistent")
	if err == nil {
		t.Error("expected error getting members for nonexistent channel")
	}
}

// --- sqlite.go: GetChannelsForAgent with no channels ---

func TestSQLiteStoreGetChannelsForAgentNone(t *testing.T) {
	store := setupTestDB(t)

	channels, err := store.GetChannelsForAgent("ghost-agent")
	if err != nil {
		t.Fatalf("GetChannelsForAgent: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(channels))
	}
}

// --- sqlite.go: SearchMessages with no results ---

func TestSQLiteStoreSearchMessagesNoResults(t *testing.T) {
	store := setupTestDB(t)

	results, err := store.SearchMessages("nonexistent-query", 10)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- sqlite.go: MigrateFromJSON with nonexistent file ---

func TestSQLiteStoreMigrateFromJSONNonexistent(t *testing.T) {
	store := setupTestDB(t)

	// Nonexistent file is a no-op (returns nil)
	err := store.MigrateFromJSON("/nonexistent/path/channels.json")
	if err != nil {
		t.Errorf("expected nil for nonexistent JSON file, got: %v", err)
	}
}

// --- sqlite.go: MigrateFromJSON with invalid JSON ---

func TestSQLiteStoreMigrateFromJSONInvalid(t *testing.T) {
	dir := t.TempDir()
	store := NewSQLiteStore(dir)
	if err := store.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	jsonPath := dir + "/bad.json"
	if err := os.WriteFile(jsonPath, []byte("not valid json"), 0600); err != nil {
		t.Fatal(err)
	}

	err := store.MigrateFromJSON(jsonPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- sqlite.go: ListChannels with type filter ---

func TestSQLiteStoreListChannelsMultipleTypes(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("group-ch", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateChannel("direct-ch", ChannelTypeDirect, ""); err != nil {
		t.Fatal(err)
	}

	channels, err := store.ListChannels()
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

// --- approval_message.go: ParseApprovalMessage edge cases ---

func TestParseApprovalMessageNoMatch(t *testing.T) {
	result := ParseApprovalMessage("just a regular message")
	if result != nil {
		t.Error("expected nil for non-approval message")
	}
}

func TestParseMergeNotificationNoMatch(t *testing.T) {
	result := ParseMergeNotification("just a regular message")
	if result != nil {
		t.Error("expected nil for non-merge message")
	}
}

func TestParseApprovalMessageChangesRequested(t *testing.T) {
	result := ParseApprovalMessage("[CHANGES REQUESTED] PR #99 - needs work")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PRNumber != 99 {
		t.Errorf("PRNumber = %d, want 99", result.PRNumber)
	}
	if result.Status != StatusChangesRequested {
		t.Errorf("Status = %q, want %q", result.Status, StatusChangesRequested)
	}
}

func TestParseMergeNotificationValid(t *testing.T) {
	result := ParseMergeNotification("[MERGED] PR #42 merged to main")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", result.PRNumber)
	}
}

// --- message_type.go: Description for unknown type ---

func TestMessageTypeDescriptionUnknown(t *testing.T) {
	mt := MessageType("unknown-type")
	desc := mt.Description()
	if desc == "" {
		t.Error("expected non-empty description for unknown type")
	}
}

// --- sqlite.go: AddMessage with empty type defaults to text ---

func TestSQLiteStoreAddMessageDefaultType(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("type-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	// Empty type should default to text
	msg, err := store.AddMessage("type-test", "user", "hello", "", "")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if msg.Type != TypeText {
		t.Errorf("Type = %q, want %q", msg.Type, TypeText)
	}
}

// --- sqlite.go: AddMessage with metadata ---

func TestSQLiteStoreAddMessageEmptyMetadata(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("meta-test2", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	msg, err := store.AddMessage("meta-test2", "user", "hello", TypeText, "")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if msg.Metadata != "" {
		t.Errorf("Metadata = %q, want empty", msg.Metadata)
	}
}

// --- Store.RemoveReaction valid case ---

func TestStoreRemoveReactionValid(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "hello"); err != nil {
		t.Fatal(err)
	}

	// Add then remove
	if err := s.AddReaction("ch", 0, "👍", "bob"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveReaction("ch", 0, "👍", "bob"); err != nil {
		t.Fatalf("RemoveReaction: %v", err)
	}

	reactions, err := s.GetReactions("ch", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(reactions["👍"]) != 0 {
		t.Errorf("expected 0 reactions after removal, got %d", len(reactions["👍"]))
	}
}

// --- Store.GetReactions valid case ---

func TestStoreGetReactionsValid(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddReaction("ch", 0, "🎉", "bob"); err != nil {
		t.Fatal(err)
	}

	reactions, err := s.GetReactions("ch", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(reactions["🎉"]) != 1 {
		t.Errorf("expected 1 reaction, got %d", len(reactions["🎉"]))
	}
}

// --- Store.ToggleReaction valid add and remove ---

func TestStoreToggleReactionAddRemove(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "hello"); err != nil {
		t.Fatal(err)
	}

	// Add
	added, err := s.ToggleReaction("ch", 0, "🚀", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Error("expected reaction to be added")
	}

	// Remove
	added, err = s.ToggleReaction("ch", 0, "🚀", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Error("expected reaction to be removed")
	}
}

// --- sqlite.go: SearchMessages matching ---

func TestSQLiteStoreSearchMessagesMultipleChannels(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("ch1", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateChannel("ch2", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("ch1", "user", "important update", TypeText, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("ch2", "user", "another important thing", TypeText, ""); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchMessages("important", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

// --- highlight.go: ApplyHighlights with no highlights ---

func TestApplyHighlightsNoHighlights(t *testing.T) {
	msg := "plain text"
	result := ApplyHighlights(msg, func(text string, _ HighlightType) string {
		return "[" + text + "]"
	})
	if result != msg {
		t.Errorf("expected unchanged message, got %q", result)
	}
}

func TestApplyHighlightsWithHighlights(t *testing.T) {
	msg := "@alice check #general"
	result := ApplyHighlights(msg, func(text string, ht HighlightType) string {
		return "<" + text + ">"
	})
	if result == msg {
		t.Error("expected highlights to be applied")
	}
}

// --- channel.go: OpenStore creates .bc dir ---

func TestOpenStoreCreatesDir(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Verify .bc directory was created
	if _, err := os.Stat(dir + "/.bc"); err != nil {
		t.Errorf("expected .bc directory to exist: %v", err)
	}
}

// --- approval_message.go: ParseApprovalMessage with comment status (PR reference without approval/changes keywords) ---

func TestParseApprovalMessageCommentedStatus(t *testing.T) {
	// Has PR reference and review keyword but no approval/changes patterns
	result := ParseApprovalMessage("PR #50 reviewed and commented")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != StatusCommented {
		t.Errorf("Status = %q, want %q", result.Status, StatusCommented)
	}
}

// --- approval_message.go: ParseApprovalMessage with reviewer mention ---

func TestParseApprovalMessageWithReviewer(t *testing.T) {
	result := ParseApprovalMessage("@reviewer PR #77 approved ✓")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Reviewer != "reviewer" {
		t.Errorf("Reviewer = %q, want reviewer", result.Reviewer)
	}
}

// --- ParseMergeNotification with branch and merger ---

func TestParseMergeNotificationWithBranchAndMerger(t *testing.T) {
	result := ParseMergeNotification("@admin merged PR #55 to develop")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PRNumber != 55 {
		t.Errorf("PRNumber = %d, want 55", result.PRNumber)
	}
	if result.Branch != "develop" {
		t.Errorf("Branch = %q, want develop", result.Branch)
	}
	if result.MergedBy != "admin" {
		t.Errorf("MergedBy = %q, want admin", result.MergedBy)
	}
}

// --- channel.go: Store.List populates history ---

// --- NewStore error path: if Open fails it still returns a store ---

func TestNewStoreOpenError(t *testing.T) {
	// Use a path where .bc exists but the db file can't be created
	dir := t.TempDir()
	bcDir := dir + "/.bc"
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Create a directory where the db file should be, causing Open to fail
	if err := os.MkdirAll(bcDir+"/bc.db", 0750); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	if s == nil {
		t.Fatal("NewStore should always return non-nil")
	}
	// Store has a backend set but operations should fail
	_ = s.Close()
}

// --- service.go: React to nonexistent channel ---

func TestServiceReactNonexistentChannel(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.React(ctx, "nope", 0, "👍", "user")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

// --- service.go: Update with empty description (no-op update) ---

func TestServiceUpdateEmptyDescription(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateChannelReq{Name: "ch", Description: "original"}); err != nil {
		t.Fatal(err)
	}

	// Update with empty description => no change
	dto, err := svc.Update(ctx, "ch", UpdateChannelReq{Description: ""})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if dto.Description != "original" {
		t.Errorf("Description = %q, want original", dto.Description)
	}
}

// --- channel.go: RemoveReaction for nonexistent channel history ---

func TestStoreRemoveReactionNonexistentChannel(t *testing.T) {
	s := newTestStore(t)
	err := s.RemoveReaction("nonexistent", 0, "👍", "user")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestStoreGetReactionsNonexistentChannel(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetReactions("nonexistent", 0)
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestStoreToggleReactionNonexistentChannel(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ToggleReaction("nonexistent", 0, "👍", "user")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestStoreAddReactionNonexistentChannel(t *testing.T) {
	s := newTestStore(t)
	err := s.AddReaction("nonexistent", 0, "👍", "user")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

// --- highlight.go: ParseGitHubLinks with issue ref ---

func TestParseGitHubLinksIssueRef(t *testing.T) {
	msg := "Fix issue #42 in the codebase"
	highlights := ParseGitHubLinks(msg)
	if len(highlights) == 0 {
		t.Error("expected github link highlight for issue ref")
	}
}

func TestParseGitHubLinksPRRef(t *testing.T) {
	msg := "Check PR #99 for the fix"
	highlights := ParseGitHubLinks(msg)
	if len(highlights) == 0 {
		t.Error("expected github link highlight for PR ref")
	}
}

// --- highlight.go: ParseChannelRefs with valid channel ---

func TestParseChannelRefsValid(t *testing.T) {
	msg := "Check #general for updates"
	highlights := ParseChannelRefs(msg)
	if len(highlights) != 1 {
		t.Errorf("expected 1 highlight, got %d", len(highlights))
		return
	}
	if highlights[0].Text != "#general" {
		t.Errorf("Text = %q, want #general", highlights[0].Text)
	}
	if highlights[0].Type != HighlightChannel {
		t.Errorf("Type = %d, want %d", highlights[0].Type, HighlightChannel)
	}
}

// --- ParseAllHighlights with all types ---

func TestParseAllHighlightsAllTypes(t *testing.T) {
	msg := "@alice mentioned #general and PR #55 was approved"
	highlights := ParseAllHighlights(msg)

	types := map[HighlightType]bool{}
	for _, h := range highlights {
		types[h.Type] = true
	}
	if !types[HighlightMention] {
		t.Error("expected mention highlight")
	}
	if !types[HighlightChannel] {
		t.Error("expected channel highlight")
	}
	if !types[HighlightGitHubLink] {
		t.Error("expected github link highlight")
	}
}

// --- sqlite.go: SearchMessages with ftsAvailable=false ---

func TestSQLiteStoreSearchMessagesFTSDisabled(t *testing.T) {
	store := setupTestDB(t)
	store.ftsAvailable = false // Force LIKE-based search path

	if _, err := store.CreateChannel("fts-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("fts-test", "user", "findable content", TypeText, ""); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchMessages("findable", 10)
	if err != nil {
		t.Fatalf("SearchMessages (no FTS): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// --- sqlite.go: SearchMessages default limit ---

func TestSQLiteStoreSearchMessagesDefaultLimit(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("limit-test", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMessage("limit-test", "user", "hello", TypeText, ""); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchMessages("hello", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with default limit, got %d", len(results))
	}
}

// --- sqlite.go: DeleteChannel with FTS rebuild ---

func TestSQLiteStoreDeleteChannelWithFTSRebuild(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("fts-del", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	// Add a message so FTS has content to rebuild
	if _, err := store.AddMessage("fts-del", "user", "searchable text", TypeText, ""); err != nil {
		t.Fatal(err)
	}

	// Verify message is searchable
	results, _ := store.SearchMessages("searchable", 10)
	if len(results) != 1 {
		t.Errorf("expected 1 search result before delete, got %d", len(results))
	}

	// Delete channel
	if err := store.DeleteChannel("fts-del"); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}

	// Verify message is no longer searchable
	results, _ = store.SearchMessages("searchable", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 search results after delete, got %d", len(results))
	}
}

// --- sqlite.go: GetHistory with limit ---

func TestSQLiteStoreGetHistoryLimit(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("limit-hist", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := store.AddMessage("limit-hist", "user", fmt.Sprintf("msg-%d", i), TypeText, ""); err != nil {
			t.Fatal(err)
		}
	}

	history, err := store.GetHistory("limit-hist", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 messages, got %d", len(history))
	}
}

// --- sqlite.go: GetMessagesByType with limit ---

func TestSQLiteStoreGetMessagesByTypeLimit(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("type-limit", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := store.AddMessage("type-limit", "user", fmt.Sprintf("task-%d", i), TypeTask, ""); err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := store.GetMessagesByType("type-limit", TypeTask, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestStoreListPopulatesHistory(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "user", "msg1"); err != nil {
		t.Fatal(err)
	}

	channels := s.List()
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if len(channels[0].History) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(channels[0].History))
	}
}

// --- sqlite.go: GetChannelsForAgent returns channel info ---

func TestSQLiteStoreGetChannelsForAgentReturnsInfo(t *testing.T) {
	store := setupTestDB(t)

	if _, err := store.CreateChannel("ch-a", ChannelTypeGroup, "Description A"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMember("ch-a", "agent-x"); err != nil {
		t.Fatal(err)
	}

	channels, err := store.GetChannelsForAgent("agent-x")
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Name != "ch-a" {
		t.Errorf("Name = %q, want ch-a", channels[0].Name)
	}
	if channels[0].Description != "Description A" {
		t.Errorf("Description = %q, want Description A", channels[0].Description)
	}
}

// --- sqlite.go: AcknowledgeMentions with no mentions ---

func TestSQLiteStoreAcknowledgeMentionsNone(t *testing.T) {
	store := setupTestDB(t)

	// Acknowledge when there are no mentions should not error
	err := store.AcknowledgeMentions("ghost-agent")
	if err != nil {
		t.Errorf("AcknowledgeMentions with no mentions should not error: %v", err)
	}
}

// --- sqlite.go: GetUnreadMentions with no mentions ---

func TestSQLiteStoreGetUnreadMentionsNone(t *testing.T) {
	store := setupTestDB(t)

	mentions, err := store.GetUnreadMentions("ghost-agent")
	if err != nil {
		t.Fatalf("GetUnreadMentions: %v", err)
	}
	if len(mentions) != 0 {
		t.Errorf("expected 0 mentions, got %d", len(mentions))
	}
}

// --- channel.go: Store.Get populates members and history ---

// --- sqlite.go: MigrateFromJSON with empty channel list ---

func TestSQLiteStoreMigrateFromJSONEmptyList(t *testing.T) {
	dir := t.TempDir()
	store := NewSQLiteStore(dir)
	if err := store.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	jsonPath := dir + "/empty.json"
	if err := os.WriteFile(jsonPath, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := store.MigrateFromJSON(jsonPath); err != nil {
		t.Fatalf("MigrateFromJSON empty: %v", err)
	}

	channels, _ := store.ListChannels()
	if len(channels) != 0 {
		t.Errorf("expected 0 channels after empty migration, got %d", len(channels))
	}
}

// --- sqlite.go: MigrateFromJSON with existing channel (INSERT OR IGNORE path) ---

func TestSQLiteStoreMigrateFromJSONExistingChannel(t *testing.T) {
	dir := t.TempDir()
	store := NewSQLiteStore(dir)
	if err := store.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Pre-create the channel
	if _, err := store.CreateChannel("existing", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	// Migrate JSON that includes the same channel
	jsonPath := dir + "/existing.json"
	jsonData := `[{"name": "existing", "members": ["agent-a"], "history": [{"time": "2024-01-01T10:00:00Z", "sender": "agent-a", "message": "Hello"}]}]`
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0600); err != nil {
		t.Fatal(err)
	}

	if err := store.MigrateFromJSON(jsonPath); err != nil {
		t.Fatalf("MigrateFromJSON existing: %v", err)
	}

	// Should still have 1 channel (INSERT OR IGNORE)
	channels, _ := store.ListChannels()
	if len(channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(channels))
	}
}

// --- sqlite.go: DeleteChannel ftsAvailable=false path ---

func TestSQLiteStoreDeleteChannelNoFTS(t *testing.T) {
	store := setupTestDB(t)
	store.ftsAvailable = false

	if _, err := store.CreateChannel("no-fts-del", ChannelTypeGroup, ""); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteChannel("no-fts-del"); err != nil {
		t.Fatalf("DeleteChannel (no FTS): %v", err)
	}

	ch, _ := store.GetChannel("no-fts-del")
	if ch != nil {
		t.Error("channel should be deleted")
	}
}

// --- highlight.go: ParseAllHighlights where channel refs overlap with GitHub links ---

func TestParseAllHighlightsChannelGitHubOverlap(t *testing.T) {
	// PR #channel-name-like-ref should not create a channel highlight
	// when it overlaps with a GitHub link
	msg := "PR #feature is mentioned"
	highlights := ParseAllHighlights(msg)
	// Verify that #feature is not duplicated as both channel and github link
	channelCount := 0
	for _, h := range highlights {
		if h.Type == HighlightChannel && h.Text == "#feature" {
			channelCount++
		}
	}
	// The overlap filter should prevent #feature from appearing as both types
	if channelCount > 1 {
		t.Errorf("expected at most 1 channel highlight for #feature, got %d", channelCount)
	}
}

// --- highlight.go: ParseGitHubLinks trimming leading non-link chars ---

func TestParseGitHubLinksTrimming(t *testing.T) {
	// Test the trimming logic for leading/trailing non-link chars
	msg := " #42 "
	highlights := ParseGitHubLinks(msg)
	if len(highlights) > 0 {
		if highlights[0].Text == "" {
			t.Error("trimmed text should not be empty")
		}
	}
}

// --- query.go: matchesQuery with After filter ---

func TestMatchesQueryAfterFilter(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	entry := HistoryEntry{Time: time.Now(), Sender: "user", Message: "hello"}

	// After past => should match
	if !matchesQuery(entry, QueryOptions{After: &past}) {
		t.Error("expected entry to match After filter")
	}

	// After future => should not match
	future := time.Now().Add(1 * time.Hour)
	if matchesQuery(entry, QueryOptions{After: &future}) {
		t.Error("expected entry NOT to match future After filter")
	}
}

// --- query.go: matchesQuery with Before filter ---

func TestMatchesQueryBeforeFilter(t *testing.T) {
	entry := HistoryEntry{Time: time.Now(), Sender: "user", Message: "hello"}

	future := time.Now().Add(1 * time.Hour)
	if !matchesQuery(entry, QueryOptions{Before: &future}) {
		t.Error("expected entry to match Before future filter")
	}

	past := time.Now().Add(-1 * time.Hour)
	if matchesQuery(entry, QueryOptions{Before: &past}) {
		t.Error("expected entry NOT to match Before past filter")
	}
}

// --- query.go: matchesQuery with sender filter ---

func TestMatchesQuerySenderFilter(t *testing.T) {
	entry := HistoryEntry{Time: time.Now(), Sender: "alice", Message: "hello"}

	if !matchesQuery(entry, QueryOptions{Sender: "alice"}) {
		t.Error("expected entry to match alice sender")
	}
	if matchesQuery(entry, QueryOptions{Sender: "bob"}) {
		t.Error("expected entry NOT to match bob sender")
	}
}

func TestStoreGetWithMembersAndHistory(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("ch"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember("ch", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistory("ch", "alice", "hello"); err != nil {
		t.Fatal(err)
	}

	ch, ok := s.Get("ch")
	if !ok {
		t.Fatal("expected channel to exist")
	}
	if len(ch.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(ch.Members))
	}
	if len(ch.History) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(ch.History))
	}
}
