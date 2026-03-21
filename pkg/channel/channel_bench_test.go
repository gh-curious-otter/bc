package channel

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- Benchmark helpers ---

// newBenchStore creates a SQLite-backed store for benchmarking.
func newBenchStore(b *testing.B) *Store {
	b.Helper()
	dir := b.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		b.Fatal(err)
	}
	s := NewStore(dir)
	b.Cleanup(func() { _ = s.Close() })
	return s
}

// --- AddHistory (Send) benchmarks ---

func BenchmarkAddHistory_JSON(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := range b.N {
		if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddHistory_SQLite(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := range b.N {
		if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddHistory_JSON_Parallel(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkAddHistory_SQLite_Parallel(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// --- GetHistory (Read) benchmarks ---

func BenchmarkGetHistory_JSON_10(b *testing.B) {
	benchGetHistoryJSON(b, 10)
}

func BenchmarkGetHistory_JSON_50(b *testing.B) {
	benchGetHistoryJSON(b, 50)
}

func BenchmarkGetHistory_JSON_100(b *testing.B) {
	benchGetHistoryJSON(b, 100)
}

func benchGetHistoryJSON(b *testing.B, msgCount int) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}
	for i := range msgCount {
		if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := store.GetHistory("bench"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetHistory_SQLite_10(b *testing.B) {
	benchGetHistorySQLite(b, 10)
}

func BenchmarkGetHistory_SQLite_50(b *testing.B) {
	benchGetHistorySQLite(b, 50)
}

func BenchmarkGetHistory_SQLite_100(b *testing.B) {
	benchGetHistorySQLite(b, 100)
}

func benchGetHistorySQLite(b *testing.B, msgCount int) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}
	for i := range msgCount {
		if err := store.AddHistory("bench", "sender", fmt.Sprintf("message-%d", i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := store.GetHistory("bench"); err != nil {
			b.Fatal(err)
		}
	}
}

// --- List benchmarks ---

func BenchmarkList_JSON_5(b *testing.B) {
	benchListJSON(b, 5)
}

func BenchmarkList_JSON_20(b *testing.B) {
	benchListJSON(b, 20)
}

func BenchmarkList_JSON_50(b *testing.B) {
	benchListJSON(b, 50)
}

func benchListJSON(b *testing.B, channelCount int) {
	store := newBenchStore(b)
	for i := range channelCount {
		if _, err := store.Create(fmt.Sprintf("channel-%d", i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		_ = store.List()
	}
}

func BenchmarkList_SQLite_5(b *testing.B) {
	benchListSQLite(b, 5)
}

func BenchmarkList_SQLite_20(b *testing.B) {
	benchListSQLite(b, 20)
}

func BenchmarkList_SQLite_50(b *testing.B) {
	benchListSQLite(b, 50)
}

func benchListSQLite(b *testing.B, channelCount int) {
	store := newBenchStore(b)
	for i := range channelCount {
		if _, err := store.Create(fmt.Sprintf("channel-%d", i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		_ = store.List()
	}
}

// --- Get benchmarks ---

func BenchmarkGet_JSON(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		_, _ = store.Get("bench")
	}
}

func BenchmarkGet_SQLite(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		_, _ = store.Get("bench")
	}
}

// --- AddMember benchmarks ---

func BenchmarkAddMember_JSON(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		// Create fresh channel each iteration to avoid duplicate errors
		b.StopTimer()
		_ = store.Delete("bench")
		_, _ = store.Create("bench")
		b.StartTimer()

		for j := range 10 {
			if err := store.AddMember("bench", fmt.Sprintf("agent-%d", j)); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkAddMember_SQLite(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		// Create fresh channel each iteration to avoid duplicate errors
		b.StopTimer()
		_ = store.Delete("bench")
		_, _ = store.Create("bench")
		b.StartTimer()

		for j := range 10 {
			if err := store.AddMember("bench", fmt.Sprintf("agent-%d", j)); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// --- Reaction benchmarks ---

func BenchmarkToggleReaction_JSON(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}
	if err := store.AddHistory("bench", "sender", "message"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = store.ToggleReaction("bench", 0, "👍", fmt.Sprintf("user-%d", i%10))
	}
}

func BenchmarkToggleReaction_SQLite(b *testing.B) {
	store := newBenchStore(b)
	if _, err := store.Create("bench"); err != nil {
		b.Fatal(err)
	}
	if err := store.AddHistory("bench", "sender", "message"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = store.ToggleReaction("bench", 0, "👍", fmt.Sprintf("user-%d", i%10))
	}
}
