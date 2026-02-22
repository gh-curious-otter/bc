package team

import (
	"fmt"
	"testing"
)

// BenchmarkStoreCreate measures team creation performance.
func BenchmarkStoreCreate(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Create(fmt.Sprintf("team-%d", i))
	}
}

// BenchmarkStoreGet measures team lookup performance.
func BenchmarkStoreGet(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()

	// Setup: create teams
	for i := 0; i < 50; i++ {
		_, _ = store.Create(fmt.Sprintf("team-%03d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get("team-025")
	}
}

// BenchmarkStoreList measures listing all teams.
func BenchmarkStoreList(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("teams-%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			store := NewStore(tmpDir)
			_ = store.Init()

			// Setup: create teams
			for i := 0; i < size; i++ {
				_, _ = store.Create(fmt.Sprintf("team-%03d", i))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.List()
			}
		})
	}
}

// BenchmarkStoreExists measures existence check performance.
func BenchmarkStoreExists(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()

	// Setup: create teams
	for i := 0; i < 50; i++ {
		_, _ = store.Create(fmt.Sprintf("team-%03d", i))
	}

	b.Run("exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Exists("team-025")
		}
	})

	b.Run("not-exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Exists("nonexistent")
		}
	})
}

// BenchmarkStoreAddMember measures adding members to teams.
func BenchmarkStoreAddMember(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()
	_, _ = store.Create("test-team")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AddMember("test-team", fmt.Sprintf("agent-%d", i))
	}
}

// BenchmarkStoreRemoveMember measures removing members from teams.
func BenchmarkStoreRemoveMember(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()
	_, _ = store.Create("test-team")

	// Setup: add many members
	for i := 0; i < 1000; i++ {
		_ = store.AddMember("test-team", fmt.Sprintf("agent-%04d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RemoveMember("test-team", fmt.Sprintf("agent-%04d", i%1000))
	}
}

// BenchmarkStoreUpdate measures update operation performance.
func BenchmarkStoreUpdate(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()
	_, _ = store.Create("test-team")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Update("test-team", func(t *Team) {
			t.Description = fmt.Sprintf("Updated description %d", i)
		})
	}
}

// BenchmarkStoreDelete measures deletion performance.
func BenchmarkStoreDelete(b *testing.B) {
	b.Run("delete-recreate", func(b *testing.B) {
		tmpDir := b.TempDir()
		store := NewStore(tmpDir)
		_ = store.Init()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Create("temp-team")
			_ = store.Delete("temp-team")
		}
	})
}

// BenchmarkRemoveAgentFromAllTeams measures cross-team agent removal.
func BenchmarkRemoveAgentFromAllTeams(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_ = store.Init()

	// Setup: create teams and add agent to all
	for i := 0; i < 20; i++ {
		_, _ = store.Create(fmt.Sprintf("team-%03d", i))
		_ = store.AddMember(fmt.Sprintf("team-%03d", i), "shared-agent")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RemoveAgentFromAllTeams("shared-agent")
		// Re-add for next iteration
		for j := 0; j < 20; j++ {
			_ = store.AddMember(fmt.Sprintf("team-%03d", j), "shared-agent")
		}
	}
}
