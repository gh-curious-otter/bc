package cost

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupBenchmarkStore creates a temporary store for benchmarking.
func setupBenchmarkStore(b *testing.B) (*Store, func()) {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "cost-bench-*")
	if err != nil {
		b.Fatal(err)
	}

	// Create .bc directory
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // cleanup on error
		b.Fatal(err)
	}

	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // cleanup on error
		b.Fatal(err)
	}

	cleanup := func() {
		_ = store.Close()        //nolint:errcheck // cleanup
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // cleanup
	}

	return store, cleanup
}

// seedBenchmarkData populates the store with test data.
func seedBenchmarkData(b *testing.B, store *Store, numAgents, recordsPerAgent int) {
	b.Helper()
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		teamID := fmt.Sprintf("team-%d", i%5)
		for j := 0; j < recordsPerAgent; j++ {
			_, err := store.Record(agentID, teamID, "claude-3-opus", 1000, 500, 0.05)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkRecord(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Record("agent-01", "team-01", "claude-3-opus", 1000, 500, 0.05)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRecordParallel(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			agentID := fmt.Sprintf("agent-%d", i%10)
			_, err := store.Record(agentID, "team-01", "claude-3-opus", 1000, 500, 0.05)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkGetByAgent_100Records(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 1, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetByAgent("agent-0", 100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetByAgent_1000Records(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 1, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetByAgent("agent-0", 1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAll_100Records(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetAll(100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAll_1000Records(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetAll(1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSummaryByAgent(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.SummaryByAgent()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSummaryByTeam(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.SummaryByTeam()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWorkspaceSummary(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.WorkspaceSummary()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAgentSummary(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.AgentSummary("agent-0")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetBudget(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope := fmt.Sprintf("agent:agent-%d", i)
		_, err := store.SetBudget(scope, BudgetPeriodMonthly, 100.0, 0.8, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheckBudget(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 1, 100)
	_, _ = store.SetBudget("agent:agent-0", BudgetPeriodMonthly, 100.0, 0.8, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.CheckBudget("agent:agent-0")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetDailyCosts(b *testing.B) {
	store, cleanup := setupBenchmarkStore(b)
	defer cleanup()

	seedBenchmarkData(b, store, 10, 100)

	since := time.Now().AddDate(0, 0, -30) // 30 days lookback

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetDailyCosts(since)
		if err != nil {
			b.Fatal(err)
		}
	}
}
