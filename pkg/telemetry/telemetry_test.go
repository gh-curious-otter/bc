package telemetry

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistry(t *testing.T) {
	r := &Registry{enabled: true}

	var received []Event
	var mu sync.Mutex

	r.Register(ObserverFunc(func(_ context.Context, e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	ctx := context.Background()
	r.Emit(ctx, Event{Type: EventAgentSpawn, Agent: "test-agent"})
	r.Emit(ctx, Event{Type: EventAgentStop, Agent: "test-agent"})

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Errorf("expected 2 events, got %d", len(received))
	}
	if received[0].Type != EventAgentSpawn {
		t.Errorf("expected EventAgentSpawn, got %s", received[0].Type)
	}
}

func TestRegistry_Disabled(t *testing.T) {
	r := &Registry{enabled: false}

	var count int
	r.Register(ObserverFunc(func(_ context.Context, _ Event) {
		count++
	}))

	r.Emit(context.Background(), Event{Type: EventAgentSpawn})

	if count != 0 {
		t.Errorf("expected 0 events when disabled, got %d", count)
	}
}

func TestRegistry_EmitAsync(t *testing.T) {
	r := &Registry{enabled: true}

	var count atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)

	r.Register(ObserverFunc(func(_ context.Context, _ Event) {
		count.Add(1)
		wg.Done()
	}))

	ctx := context.Background()
	r.EmitAsync(ctx, Event{Type: EventAgentSpawn})
	r.EmitAsync(ctx, Event{Type: EventAgentStop})

	wg.Wait()

	if count.Load() != 2 {
		t.Errorf("expected 2 async events, got %d", count.Load())
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := &Registry{enabled: true}
	r.Register(ObserverFunc(func(_ context.Context, _ Event) {}))
	r.Register(ObserverFunc(func(_ context.Context, _ Event) {}))

	if r.ObserverCount() != 2 {
		t.Errorf("expected 2 observers, got %d", r.ObserverCount())
	}

	r.Clear()

	if r.ObserverCount() != 0 {
		t.Errorf("expected 0 observers after clear, got %d", r.ObserverCount())
	}
}

func TestEvent_TimestampAutoSet(t *testing.T) {
	r := &Registry{enabled: true}

	var received Event
	r.Register(ObserverFunc(func(_ context.Context, e Event) {
		received = e
	}))

	before := time.Now()
	r.Emit(context.Background(), Event{Type: EventAgentSpawn})
	after := time.Now()

	if received.Timestamp.Before(before) || received.Timestamp.After(after) {
		t.Errorf("timestamp not auto-set correctly")
	}
}

func TestFilterObserver(t *testing.T) {
	var received []Event
	inner := ObserverFunc(func(_ context.Context, e Event) {
		received = append(received, e)
	})

	filtered := NewFilterObserver(inner, TypeFilter(EventAgentSpawn, EventAgentStop))

	ctx := context.Background()
	filtered.OnEvent(ctx, Event{Type: EventAgentSpawn})
	filtered.OnEvent(ctx, Event{Type: EventChannelSend})
	filtered.OnEvent(ctx, Event{Type: EventAgentStop})

	if len(received) != 2 {
		t.Errorf("expected 2 filtered events, got %d", len(received))
	}
}

func TestAgentFilter(t *testing.T) {
	var received []Event
	inner := ObserverFunc(func(_ context.Context, e Event) {
		received = append(received, e)
	})

	filtered := NewFilterObserver(inner, AgentFilter("agent-1", "agent-2"))

	ctx := context.Background()
	filtered.OnEvent(ctx, Event{Agent: "agent-1"})
	filtered.OnEvent(ctx, Event{Agent: "agent-3"})
	filtered.OnEvent(ctx, Event{Agent: "agent-2"})

	if len(received) != 2 {
		t.Errorf("expected 2 filtered events, got %d", len(received))
	}
}

func TestBufferedObserver(t *testing.T) {
	var flushed []Event
	var mu sync.Mutex

	bo := NewBufferedObserver(3, func(events []Event) {
		mu.Lock()
		flushed = append(flushed, events...)
		mu.Unlock()
	})

	ctx := context.Background()
	bo.OnEvent(ctx, Event{Type: "1"})
	bo.OnEvent(ctx, Event{Type: "2"})

	// Not full yet
	mu.Lock()
	if len(flushed) != 0 {
		t.Errorf("expected 0 flushed events, got %d", len(flushed))
	}
	mu.Unlock()

	bo.OnEvent(ctx, Event{Type: "3"})

	// Give async callback time to run
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(flushed) != 3 {
		t.Errorf("expected 3 flushed events, got %d", len(flushed))
	}
	mu.Unlock()
}

func TestBufferedObserver_Flush(t *testing.T) {
	bo := NewBufferedObserver(10, nil)

	ctx := context.Background()
	bo.OnEvent(ctx, Event{Type: "1"})
	bo.OnEvent(ctx, Event{Type: "2"})

	events := bo.Flush()
	if len(events) != 2 {
		t.Errorf("expected 2 flushed events, got %d", len(events))
	}

	// Buffer should be empty now
	events = bo.Flush()
	if len(events) != 0 {
		t.Errorf("expected 0 events after second flush, got %d", len(events))
	}
}

func TestMultiObserver(t *testing.T) {
	var count1, count2 int

	obs1 := ObserverFunc(func(_ context.Context, _ Event) { count1++ })
	obs2 := ObserverFunc(func(_ context.Context, _ Event) { count2++ })

	multi := NewMultiObserver(obs1, obs2)
	multi.OnEvent(context.Background(), Event{Type: EventAgentSpawn})

	if count1 != 1 || count2 != 1 {
		t.Errorf("expected both observers called once, got %d and %d", count1, count2)
	}
}
