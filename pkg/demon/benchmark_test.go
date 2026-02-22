package demon

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkStoreCreate measures demon creation performance.
func BenchmarkStoreCreate(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Create(fmt.Sprintf("demon-%d", i), "0 * * * *", "echo test")
	}
}

// BenchmarkStoreGet measures demon lookup performance.
func BenchmarkStoreGet(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Setup: create demons
	for i := 0; i < 50; i++ {
		_, _ = store.Create(fmt.Sprintf("demon-%03d", i), "0 * * * *", "echo test")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get("demon-025")
	}
}

// BenchmarkStoreList measures listing all demons.
func BenchmarkStoreList(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("demons-%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			store := NewStore(tmpDir)

			// Setup: create demons
			for i := 0; i < size; i++ {
				_, _ = store.Create(fmt.Sprintf("demon-%03d", i), "0 * * * *", "echo test")
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

	// Setup: create demons
	for i := 0; i < 50; i++ {
		_, _ = store.Create(fmt.Sprintf("demon-%03d", i), "0 * * * *", "echo test")
	}

	b.Run("exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Exists("demon-025")
		}
	})

	b.Run("not-exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Exists("nonexistent")
		}
	})
}

// BenchmarkStoreUpdate measures update operation performance.
func BenchmarkStoreUpdate(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_, _ = store.Create("test-demon", "0 * * * *", "echo test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Update("test-demon", func(d *Demon) {
			d.Description = fmt.Sprintf("Updated description %d", i)
		})
	}
}

// BenchmarkStoreDelete measures deletion performance.
func BenchmarkStoreDelete(b *testing.B) {
	b.Run("delete-recreate", func(b *testing.B) {
		tmpDir := b.TempDir()
		store := NewStore(tmpDir)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Create("temp-demon", "0 * * * *", "echo test")
			_ = store.Delete("temp-demon")
		}
	})
}

// BenchmarkStoreEnableDisable measures enable/disable performance.
func BenchmarkStoreEnableDisable(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_, _ = store.Create("test-demon", "0 * * * *", "echo test")

	b.Run("enable", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Enable("test-demon")
		}
	})

	b.Run("disable", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Disable("test-demon")
		}
	})
}

// BenchmarkStoreListEnabled measures listing enabled demons.
func BenchmarkStoreListEnabled(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Setup: create mixed enabled/disabled demons
	for i := 0; i < 50; i++ {
		d, _ := store.Create(fmt.Sprintf("demon-%03d", i), "0 * * * *", "echo test")
		if i%2 == 0 {
			_ = store.Disable(d.Name)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListEnabled()
	}
}

// BenchmarkStoreListByOwner measures listing demons by owner.
func BenchmarkStoreListByOwner(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Setup: create demons with various owners
	owners := []string{"eng-01", "eng-02", "eng-03", "mgr-01"}
	for i := 0; i < 50; i++ {
		d, _ := store.Create(fmt.Sprintf("demon-%03d", i), "0 * * * *", "echo test")
		owner := owners[i%len(owners)] //nolint:gosec // index bounded by len
		_ = store.SetOwner(d.Name, owner)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListByOwner("eng-01")
	}
}

// BenchmarkStoreRecordRun measures recording run completion.
func BenchmarkStoreRecordRun(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_, _ = store.Create("test-demon", "0 * * * *", "echo test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RecordRun("test-demon")
	}
}

// BenchmarkParseCron measures cron expression parsing.
func BenchmarkParseCron(b *testing.B) {
	expressions := []struct {
		name string
		expr string
	}{
		{"simple-hourly", "0 * * * *"},
		{"every-5-min", "*/5 * * * *"},
		{"weekday-9am", "0 9 * * 1-5"},
		{"complex", "0,30 9-17 * 1-6 1-5"},
	}

	for _, tc := range expressions {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = ParseCron(tc.expr)
			}
		})
	}
}

// BenchmarkCronNext measures next run time calculation.
func BenchmarkCronNext(b *testing.B) {
	schedules := []struct {
		name string
		expr string
	}{
		{"hourly", "0 * * * *"},
		{"every-5-min", "*/5 * * * *"},
		{"daily-9am", "0 9 * * *"},
		{"weekday-9am", "0 9 * * 1-5"},
	}

	now := time.Now()

	for _, tc := range schedules {
		cron, _ := ParseCron(tc.expr)
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cron.Next(now)
			}
		})
	}
}

// BenchmarkRecordRunLog measures appending run logs.
func BenchmarkRecordRunLog(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_, _ = store.Create("test-demon", "0 * * * *", "echo test")

	log := RunLog{
		Timestamp: time.Now(),
		Duration:  1500,
		ExitCode:  0,
		Success:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RecordRunLog("test-demon", log)
	}
}

// BenchmarkGetRunLogs measures reading run logs.
func BenchmarkGetRunLogs(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	_, _ = store.Create("test-demon", "0 * * * *", "echo test")

	// Setup: add many log entries
	log := RunLog{
		Timestamp: time.Now(),
		Duration:  1500,
		ExitCode:  0,
		Success:   true,
	}
	for i := 0; i < 100; i++ {
		_ = store.RecordRunLog("test-demon", log)
	}

	b.Run("all-logs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.GetRunLogs("test-demon", 0)
		}
	})

	b.Run("last-10", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.GetRunLogs("test-demon", 10)
		}
	})
}
