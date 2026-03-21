package channel

import (
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("test"); err != nil {
		t.Fatal(err)
	}

	// Add some test messages
	for i := 0; i < 10; i++ {
		if err := s.AddHistory("test", "user", "message"); err != nil {
			t.Fatal(err)
		}
	}

	result, err := s.Query("test", QueryOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Messages) != 5 {
		t.Errorf("expected 5 messages, got %d", len(result.Messages))
	}
	if result.Total != 10 {
		t.Errorf("expected total 10, got %d", result.Total)
	}
	if !result.HasMore {
		t.Error("expected HasMore=true")
	}
}

func TestQueryPagination(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("test"); err != nil {
		t.Fatal(err)
	}

	// Add 15 messages
	for i := 0; i < 15; i++ {
		if err := s.AddHistory("test", "user", "msg"); err != nil {
			t.Fatal(err)
		}
	}

	// First page
	result1, err := s.Query("test", QueryOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(result1.Messages) != 10 {
		t.Errorf("page 1: expected 10, got %d", len(result1.Messages))
	}
	if !result1.HasMore {
		t.Error("page 1: expected HasMore=true")
	}

	// Second page
	result2, err := s.Query("test", QueryOptions{Limit: 10, Offset: result1.NextOffset})
	if err != nil {
		t.Fatal(err)
	}
	if len(result2.Messages) != 5 {
		t.Errorf("page 2: expected 5, got %d", len(result2.Messages))
	}
	if result2.HasMore {
		t.Error("page 2: expected HasMore=false")
	}
}

func TestQueryBySender(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("test"); err != nil {
		t.Fatal(err)
	}

	_ = s.AddHistory("test", "alice", "hello")
	_ = s.AddHistory("test", "bob", "hi")
	_ = s.AddHistory("test", "alice", "goodbye")

	result, err := s.Query("test", QueryOptions{Sender: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Messages) != 2 {
		t.Errorf("expected 2 messages from alice, got %d", len(result.Messages))
	}
}

func TestQueryByTime(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("test"); err != nil {
		t.Fatal(err)
	}

	_ = s.AddHistory("test", "user", "old")
	// SQLite stores timestamps with second precision, so sleep >1s to get different timestamps
	time.Sleep(1100 * time.Millisecond)
	midpoint := time.Now()
	time.Sleep(1100 * time.Millisecond)
	_ = s.AddHistory("test", "user", "new")

	// Messages after midpoint
	result, err := s.Query("test", QueryOptions{After: &midpoint})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Messages) != 1 {
		t.Errorf("expected 1 message after midpoint, got %d", len(result.Messages))
	}
}

func TestQueryChannelNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Query("nonexistent", QueryOptions{})
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestSearch(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("search-ch1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("search-ch2"); err != nil {
		t.Fatal(err)
	}

	_ = s.AddHistory("search-ch1", "alice", "hello world")
	_ = s.AddHistory("search-ch1", "bob", "goodbye world")
	_ = s.AddHistory("search-ch2", "charlie", "hello dev")

	results, err := s.Search("hello", SearchOptions{Channels: []string{"search-ch1", "search-ch2"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results for 'hello', got %d", len(results))
	}
}

func TestSearchByChannel(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("search-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("search-b"); err != nil {
		t.Fatal(err)
	}

	_ = s.AddHistory("search-a", "alice", "test message")
	_ = s.AddHistory("search-b", "bob", "test message")

	results, err := s.Search("test", SearchOptions{Channels: []string{"search-a"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result in search-a, got %d", len(results))
	}
	if len(results) > 0 && results[0].Channel != "search-a" {
		t.Errorf("expected channel 'search-a', got %q", results[0].Channel)
	}
}

func TestGetMentions(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("test"); err != nil {
		t.Fatal(err)
	}

	_ = s.AddHistory("test", "manager", "Hey @alice please review")
	_ = s.AddHistory("test", "bob", "Hello everyone")
	_ = s.AddHistory("test", "manager", "@alice and @bob check this")

	results, err := s.GetMentions("alice", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 mentions of alice, got %d", len(results))
	}
}

func TestDefaultQueryOptions(t *testing.T) {
	opts := DefaultQueryOptions()
	if opts.Limit != 50 {
		t.Errorf("expected default limit 50, got %d", opts.Limit)
	}
}
