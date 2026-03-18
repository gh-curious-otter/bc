package agent

import (
	"testing"
	"time"
)

func TestParseDockerPct(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"0.50%", 0.50},
		{"100.00%", 100.0},
		{"--", 0},
		{"", 0},
	}
	for _, tc := range tests {
		got := parseDockerPct(tc.in)
		if got != tc.want {
			t.Errorf("parseDockerPct(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDockerBytes(t *testing.T) {
	tests := []struct {
		in   string
		want float64 // MB
	}{
		{"1MB", 1},
		{"1MiB", 1},
		{"1024kB", 1.024},
		{"1GiB", 1024},
		{"512B", 512.0 / (1024 * 1024)},
		{"--", 0},
		{"", 0},
	}
	for _, tc := range tests {
		got := parseDockerBytes(tc.in)
		if got < tc.want*0.99 || got > tc.want*1.01 {
			t.Errorf("parseDockerBytes(%q) = %v, want ~%v", tc.in, got, tc.want)
		}
	}
}

func TestParseDockerMemory(t *testing.T) {
	used, limit := parseDockerMemory("150MiB / 7.77GiB")
	if used < 149 || used > 151 {
		t.Errorf("used = %v, want ~150 MB", used)
	}
	if limit < 7000 || limit > 8000 {
		t.Errorf("limit = %v, want ~7.77 GiB in MB", limit)
	}
}

func TestParseDockerIO(t *testing.T) {
	rx, tx := parseDockerIO("1.5MB / 500kB")
	if rx < 1.4 || rx > 1.6 {
		t.Errorf("rx = %v, want ~1.5 MB", rx)
	}
	if tx < 0.4 || tx > 0.6 {
		t.Errorf("tx = %v, want ~0.5 MB", tx)
	}
}

func TestSaveAndQueryStats(t *testing.T) {
	store, err := NewSQLiteStore(t.TempDir() + "/state.db")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	rec := &AgentStatsRecord{
		AgentName:   "eng-01",
		CollectedAt: time.Now().UTC().Truncate(time.Second),
		CPUPct:      12.5,
		MemUsedMB:   256.0,
		MemLimitMB:  1024.0,
		NetRxMB:     0.1,
		NetTxMB:     0.05,
	}
	if err := store.SaveStats(rec); err != nil {
		t.Fatalf("SaveStats: %v", err)
	}

	records, err := store.QueryStats("eng-01", 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	got := records[0]
	if got.AgentName != rec.AgentName {
		t.Errorf("AgentName = %q, want %q", got.AgentName, rec.AgentName)
	}
	if got.CPUPct != rec.CPUPct {
		t.Errorf("CPUPct = %v, want %v", got.CPUPct, rec.CPUPct)
	}
	if got.MemUsedMB != rec.MemUsedMB {
		t.Errorf("MemUsedMB = %v, want %v", got.MemUsedMB, rec.MemUsedMB)
	}
}

func TestQueryStats_Empty(t *testing.T) {
	store, err := NewSQLiteStore(t.TempDir() + "/state.db")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	records, err := store.QueryStats("nobody", 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestQueryStats_LimitRespected(t *testing.T) {
	store, err := NewSQLiteStore(t.TempDir() + "/state.db")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	for i := range 5 {
		rec := &AgentStatsRecord{
			AgentName:   "worker",
			CollectedAt: time.Now().Add(time.Duration(i) * time.Second),
			CPUPct:      float64(i),
		}
		if err := store.SaveStats(rec); err != nil {
			t.Fatalf("SaveStats #%d: %v", i, err)
		}
	}
	records, err := store.QueryStats("worker", 3)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}
}
