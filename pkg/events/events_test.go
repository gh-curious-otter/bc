package events

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func tempLog(t *testing.T) *Log {
	t.Helper()
	dir := t.TempDir()
	return NewLog(filepath.Join(dir, "events.jsonl"))
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		wantType  EventType
		wantAgent string
	}{
		{
			name:      "agent spawned",
			event:     Event{Type: AgentSpawned, Agent: "worker-01"},
			wantType:  AgentSpawned,
			wantAgent: "worker-01",
		},
		{
			name:      "work assigned with data",
			event:     Event{Type: WorkAssigned, Agent: "worker-02", Data: map[string]any{"work_id": "work-001"}},
			wantType:  WorkAssigned,
			wantAgent: "worker-02",
		},
		{
			name:     "message with no agent",
			event:    Event{Type: QueueLoaded, Message: "loaded 5 items"},
			wantType: QueueLoaded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := tempLog(t)
			if err := log.Append(tt.event); err != nil {
				t.Fatalf("Append() error: %v", err)
			}

			events, err := log.Read()
			if err != nil {
				t.Fatalf("Read() error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			if events[0].Type != tt.wantType {
				t.Errorf("Type = %q, want %q", events[0].Type, tt.wantType)
			}
			if events[0].Agent != tt.wantAgent {
				t.Errorf("Agent = %q, want %q", events[0].Agent, tt.wantAgent)
			}
		})
	}
}

func TestAppend_AutoTimestamp(t *testing.T) {
	log := tempLog(t)
	before := time.Now()
	if err := log.Append(Event{Type: AgentSpawned, Agent: "w1"}); err != nil {
		t.Fatalf("Append() error: %v", err)
	}
	after := time.Now()

	events, _ := log.Read()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	ts := events[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("auto-timestamp %v not in [%v, %v]", ts, before, after)
	}
}

func TestAppend_PreservesExplicitTimestamp(t *testing.T) {
	log := tempLog(t)
	explicit := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if err := log.Append(Event{Type: AgentStopped, Timestamp: explicit}); err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	events, _ := log.Read()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if !events[0].Timestamp.Equal(explicit) {
		t.Errorf("Timestamp = %v, want %v", events[0].Timestamp, explicit)
	}
}

func TestAppend_MultipleEvents(t *testing.T) {
	log := tempLog(t)
	types := []EventType{AgentSpawned, WorkAssigned, WorkCompleted, AgentStopped}
	for _, et := range types {
		if err := log.Append(Event{Type: et, Agent: "w1"}); err != nil {
			t.Fatalf("Append(%s) error: %v", et, err)
		}
	}

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(events) != len(types) {
		t.Fatalf("got %d events, want %d", len(events), len(types))
	}
	for i, et := range types {
		if events[i].Type != et {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, et)
		}
	}
}

func TestAppend_JSONLFormat(t *testing.T) {
	log := tempLog(t)
	if err := log.Append(Event{Type: AgentSpawned, Agent: "w1"}); err != nil {
		t.Fatal(err)
	}
	if err := log.Append(Event{Type: AgentStopped, Agent: "w2"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(log.path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	// Each line should be valid JSON (contain expected type string)
	if !strings.Contains(lines[0], `"agent.spawned"`) {
		t.Errorf("line 0 missing type: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"agent.stopped"`) {
		t.Errorf("line 1 missing type: %s", lines[1])
	}
}

func TestRead_NonExistentFile(t *testing.T) {
	log := NewLog(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if events != nil {
		t.Errorf("expected nil, got %v", events)
	}
}

func TestRead_EmptyFile(t *testing.T) {
	log := tempLog(t)
	if err := os.WriteFile(log.path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestRead_MalformedLines(t *testing.T) {
	log := tempLog(t)
	// Write mix of valid and malformed JSON lines
	content := `{"ts":"2025-01-01T00:00:00Z","type":"agent.spawned","agent":"w1"}
not valid json
{"ts":"2025-01-02T00:00:00Z","type":"agent.stopped","agent":"w2"}
{broken
{"ts":"2025-01-03T00:00:00Z","type":"work.assigned","agent":"w3"}
`
	if err := os.WriteFile(log.path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (2 malformed should be skipped)", len(events))
	}
	if events[0].Agent != "w1" {
		t.Errorf("events[0].Agent = %q, want w1", events[0].Agent)
	}
	if events[1].Agent != "w2" {
		t.Errorf("events[1].Agent = %q, want w2", events[1].Agent)
	}
	if events[2].Agent != "w3" {
		t.Errorf("events[2].Agent = %q, want w3", events[2].Agent)
	}
}

func TestReadLast(t *testing.T) {
	tests := []struct {
		name      string
		numEvents int
		lastN     int
		wantCount int
		wantFirst EventType // first event type in result (ignored if wantCount == 0)
	}{
		{"last 2 of 5", 5, 2, 2, WorkCompleted},
		{"last 5 of 5", 5, 5, 5, AgentSpawned},
		{"last 10 of 3", 3, 10, 3, AgentSpawned},
		{"last 0 of 3", 3, 0, 0, ""},
		{"last 1 of 1", 1, 1, 1, AgentSpawned},
	}

	orderedTypes := []EventType{AgentSpawned, WorkAssigned, WorkStarted, WorkCompleted, AgentStopped}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := tempLog(t)
			for i := 0; i < tt.numEvents; i++ {
				log.Append(Event{Type: orderedTypes[i%len(orderedTypes)]})
			}

			events, err := log.ReadLast(tt.lastN)
			if err != nil {
				t.Fatalf("ReadLast(%d) error: %v", tt.lastN, err)
			}
			if len(events) != tt.wantCount {
				t.Fatalf("got %d events, want %d", len(events), tt.wantCount)
			}
			if tt.wantCount > 0 && len(events) > 0 && events[0].Type != tt.wantFirst {
				t.Errorf("first event Type = %q, want %q", events[0].Type, tt.wantFirst)
			}
		})
	}
}

func TestReadLast_EmptyLog(t *testing.T) {
	log := tempLog(t)
	events, err := log.ReadLast(5)
	if err != nil {
		t.Fatalf("ReadLast() error: %v", err)
	}
	if events != nil {
		t.Errorf("expected nil, got %v", events)
	}
}

func TestReadByAgent(t *testing.T) {
	log := tempLog(t)
	log.Append(Event{Type: AgentSpawned, Agent: "alice"})
	log.Append(Event{Type: AgentSpawned, Agent: "bob"})
	log.Append(Event{Type: WorkAssigned, Agent: "alice"})
	log.Append(Event{Type: WorkCompleted, Agent: "bob"})
	log.Append(Event{Type: AgentStopped, Agent: "alice"})

	tests := []struct {
		agent     string
		wantCount int
	}{
		{"alice", 3},
		{"bob", 2},
		{"charlie", 0},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			events, err := log.ReadByAgent(tt.agent)
			if err != nil {
				t.Fatalf("ReadByAgent(%q) error: %v", tt.agent, err)
			}
			if len(events) != tt.wantCount {
				t.Fatalf("got %d events, want %d", len(events), tt.wantCount)
			}
			for _, ev := range events {
				if ev.Agent != tt.agent {
					t.Errorf("event has Agent=%q, want %q", ev.Agent, tt.agent)
				}
			}
		})
	}
}

func TestReadByAgent_EmptyLog(t *testing.T) {
	log := tempLog(t)
	events, err := log.ReadByAgent("nobody")
	if err != nil {
		t.Fatalf("ReadByAgent() error: %v", err)
	}
	if events != nil {
		t.Errorf("expected nil, got %v", events)
	}
}

func TestAppend_DataFields(t *testing.T) {
	log := tempLog(t)
	data := map[string]any{
		"work_id": "work-042",
		"count":   float64(7),
		"active":  true,
	}
	if err := log.Append(Event{Type: WorkAssigned, Data: data}); err != nil {
		t.Fatal(err)
	}

	events, _ := log.Read()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	for k, want := range data {
		got, ok := events[0].Data[k]
		if !ok {
			t.Errorf("Data[%q] missing", k)
			continue
		}
		if got != want {
			t.Errorf("Data[%q] = %v, want %v", k, got, want)
		}
	}
}

func TestAppend_LargeMessage(t *testing.T) {
	log := tempLog(t)
	// bufio.Scanner default max token is 64KB; use a message that fits within that
	// after JSON encoding overhead (quotes, field names, etc.)
	bigMsg := strings.Repeat("x", 50_000)
	if err := log.Append(Event{Type: MessageSent, Message: bigMsg}); err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if len(events[0].Message) != 50_000 {
		t.Errorf("message length = %d, want 50000", len(events[0].Message))
	}
}

func TestRead_OversizedLine(t *testing.T) {
	// bufio.Scanner has a ~64KB default token limit; lines exceeding it cause a scan error.
	// This documents the current behavior: Read returns whatever was parsed before the error.
	log := tempLog(t)
	log.Append(Event{Type: AgentSpawned, Agent: "before"})
	// Write an oversized line directly (> 64KB)
	f, _ := os.OpenFile(log.path, os.O_APPEND|os.O_WRONLY, 0644)
	huge := `{"ts":"2025-01-01T00:00:00Z","type":"message.sent","message":"` + strings.Repeat("z", 100_000) + "\"}\n"
	f.WriteString(huge)
	f.Close()
	log.Append(Event{Type: AgentStopped, Agent: "after"})

	events, err := log.Read()
	// Scanner error means we get events before the oversized line but lose the rest
	if err != nil {
		t.Logf("Read() returned expected scanner error: %v", err)
	}
	if len(events) < 1 {
		t.Errorf("expected at least 1 event before oversized line, got %d", len(events))
	}
}

func TestAppend_ConcurrentWrites(t *testing.T) {
	log := tempLog(t)
	const n = 50
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Append(Event{Type: AgentReport, Agent: "concurrent"})
		}()
	}
	wg.Wait()

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	// Each append opens/closes the file with O_APPEND, so all events should be present.
	// Some may be malformed if writes interleave at the byte level, but on most OS
	// implementations O_APPEND writes are atomic for reasonable sizes.
	if len(events) < n/2 {
		t.Errorf("got %d events from %d concurrent writes, expected most to succeed", len(events), n)
	}
}

func TestNewLog(t *testing.T) {
	log := NewLog("/some/path/events.jsonl")
	if log.path != "/some/path/events.jsonl" {
		t.Errorf("path = %q, want /some/path/events.jsonl", log.path)
	}
}

func TestAllEventTypes(t *testing.T) {
	// Verify all defined event types can be round-tripped
	types := []EventType{
		AgentSpawned, AgentStopped, AgentReport,
		WorkAssigned, WorkStarted, WorkCompleted, WorkFailed,
		MessageSent, QueueLoaded,
	}

	log := tempLog(t)
	for _, et := range types {
		if err := log.Append(Event{Type: et}); err != nil {
			t.Fatalf("Append(%s) error: %v", et, err)
		}
	}

	events, err := log.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(events) != len(types) {
		t.Fatalf("got %d events, want %d", len(events), len(types))
	}
	for i, et := range types {
		if events[i].Type != et {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, et)
		}
	}
}
