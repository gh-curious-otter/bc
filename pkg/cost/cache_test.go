package cost

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewCache(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	data := json.RawMessage(`{"daily":[{"date":"2026-03-01","totalCost":1.23}]}`)

	if saveErr := cache.Save("daily", data); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	loaded, fetchedAt, err := cache.Load("daily")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}
	if fetchedAt.IsZero() {
		t.Error("fetchedAt should not be zero")
	}

	var parsed map[string]any
	if err := json.Unmarshal(loaded, &parsed); err != nil {
		t.Fatalf("Unmarshal loaded: %v", err)
	}
	if _, ok := parsed["daily"]; !ok {
		t.Error("expected 'daily' key in loaded data")
	}
}

func TestCache_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewCache(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	data, _, err := cache.Load("nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if data != nil {
		t.Error("expected nil for missing key")
	}
}

func TestCache_Overwrite(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewCache(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	defer func() { _ = cache.Close() }()

	_ = cache.Save("k", json.RawMessage(`"old"`))
	_ = cache.Save("k", json.RawMessage(`"new"`))

	data, _, _ := cache.Load("k")
	if string(data) != `"new"` {
		t.Errorf("expected overwritten value, got %s", string(data))
	}
}
