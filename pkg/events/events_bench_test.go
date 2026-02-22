package events

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// --- Benchmark helpers ---

// newBenchLog creates a Log for benchmarking.
func newBenchLog(b *testing.B) *Log {
	b.Helper()
	dir := b.TempDir()
	return NewLog(filepath.Join(dir, "events.jsonl"))
}

// --- Append benchmarks ---

func BenchmarkAppend(b *testing.B) {
	log := newBenchLog(b)
	event := Event{
		Type:      AgentSpawned,
		Agent:     "worker-01",
		Message:   "Agent started",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for range b.N {
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppend_WithData(b *testing.B) {
	log := newBenchLog(b)
	event := Event{
		Type:      WorkAssigned,
		Agent:     "worker-01",
		Message:   "Work assigned",
		Timestamp: time.Now(),
		Data: map[string]any{
			"work_id":     "work-001",
			"priority":    1,
			"description": "Implement feature X",
		},
	}

	b.ResetTimer()
	for range b.N {
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppend_Parallel(b *testing.B) {
	log := newBenchLog(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			event := Event{
				Type:      AgentReport,
				Agent:     fmt.Sprintf("worker-%d", i%10),
				Message:   fmt.Sprintf("Report %d", i),
				Timestamp: time.Now(),
			}
			if err := log.Append(event); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// --- Read benchmarks ---

func BenchmarkRead_10(b *testing.B) {
	benchRead(b, 10)
}

func BenchmarkRead_100(b *testing.B) {
	benchRead(b, 100)
}

func BenchmarkRead_1000(b *testing.B) {
	benchRead(b, 1000)
}

func benchRead(b *testing.B, eventCount int) {
	log := newBenchLog(b)
	for i := range eventCount {
		event := Event{
			Type:      AgentReport,
			Agent:     fmt.Sprintf("worker-%d", i%5),
			Message:   fmt.Sprintf("Event %d", i),
			Timestamp: time.Now(),
		}
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := log.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

// --- ReadLast benchmarks ---

func BenchmarkReadLast_10_from_100(b *testing.B) {
	benchReadLast(b, 100, 10)
}

func BenchmarkReadLast_10_from_1000(b *testing.B) {
	benchReadLast(b, 1000, 10)
}

func BenchmarkReadLast_50_from_1000(b *testing.B) {
	benchReadLast(b, 1000, 50)
}

func benchReadLast(b *testing.B, totalEvents, lastN int) {
	log := newBenchLog(b)
	for i := range totalEvents {
		event := Event{
			Type:      AgentReport,
			Agent:     fmt.Sprintf("worker-%d", i%5),
			Message:   fmt.Sprintf("Event %d", i),
			Timestamp: time.Now(),
		}
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := log.ReadLast(lastN); err != nil {
			b.Fatal(err)
		}
	}
}

// --- ReadByAgent benchmarks ---

func BenchmarkReadByAgent_100(b *testing.B) {
	benchReadByAgent(b, 100)
}

func BenchmarkReadByAgent_1000(b *testing.B) {
	benchReadByAgent(b, 1000)
}

func benchReadByAgent(b *testing.B, eventCount int) {
	log := newBenchLog(b)
	for i := range eventCount {
		event := Event{
			Type:      AgentReport,
			Agent:     fmt.Sprintf("worker-%d", i%5),
			Message:   fmt.Sprintf("Event %d", i),
			Timestamp: time.Now(),
		}
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := log.ReadByAgent("worker-0"); err != nil {
			b.Fatal(err)
		}
	}
}

// --- NewLog benchmark ---

func BenchmarkNewLog(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	b.ResetTimer()
	for range b.N {
		_ = NewLog(path)
	}
}

// --- Event marshaling benchmark ---

func BenchmarkEventMarshal(b *testing.B) {
	log := newBenchLog(b)
	// Pre-create file so Append doesn't include file creation overhead
	if err := log.Append(Event{Type: AgentSpawned}); err != nil {
		b.Fatal(err)
	}

	event := Event{
		Type:      WorkCompleted,
		Agent:     "worker-01",
		Message:   "Task completed successfully",
		Timestamp: time.Now(),
		Data: map[string]any{
			"duration_ms": 1234,
			"files_changed": []string{
				"main.go",
				"handler.go",
				"service.go",
			},
		},
	}

	b.ResetTimer()
	for range b.N {
		if err := log.Append(event); err != nil {
			b.Fatal(err)
		}
	}
}
