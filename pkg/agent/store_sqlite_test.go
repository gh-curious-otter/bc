package agent

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStore_SaveLoadDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().Truncate(time.Second)
	a := &Agent{
		Name:      "eng-01",
		ID:        "eng-01",
		Role:      Role("engineer"),
		State:     StateIdle,
		Tool:      "claude",
		Workspace: "/tmp/ws",
		CreatedAt: now,
		StartedAt: now,
		SessionID: "ses-abc123",
		Children:  []string{"child-1", "child-2"},
	}

	// Save
	if saveErr := store.Save(a); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	// Load
	loaded, err := store.Load("eng-01")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}
	if loaded.Name != "eng-01" {
		t.Errorf("Name = %q, want eng-01", loaded.Name)
	}
	if loaded.Role != Role("engineer") {
		t.Errorf("Role = %q, want engineer", loaded.Role)
	}
	if loaded.Tool != "claude" {
		t.Errorf("Tool = %q, want claude", loaded.Tool)
	}
	if len(loaded.Children) != 2 {
		t.Errorf("Children len = %d, want 2", len(loaded.Children))
	}
	if loaded.SessionID != "ses-abc123" {
		t.Errorf("SessionID = %q, want ses-abc123", loaded.SessionID)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// Load non-existent
	missing, err := store.Load("nonexistent")
	if err != nil {
		t.Fatalf("Load nonexistent: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for nonexistent agent")
	}

	// Delete
	if err := store.Delete("eng-01"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	after, _ := store.Load("eng-01")
	if after != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestSQLiteStore_LoadAll(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	for _, name := range []string{"a", "b", "c"} {
		_ = store.Save(&Agent{
			Name:      name,
			Role:      Role("worker"),
			State:     StateIdle,
			Workspace: "/tmp/ws",
			StartedAt: time.Now(),
		})
	}

	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("LoadAll returned %d agents, want 3", len(all))
	}
}

func TestSQLiteStore_SaveAll(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	agents := map[string]*Agent{
		"x": {Name: "x", Role: "worker", State: StateIdle, Workspace: "/ws", StartedAt: time.Now()},
		"y": {Name: "y", Role: "engineer", State: StateWorking, Workspace: "/ws", StartedAt: time.Now()},
	}
	if err := store.SaveAll(agents); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	all, _ := store.LoadAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(all))
	}
}

func TestSQLiteStore_UpdateState(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	_ = store.Save(&Agent{Name: "a", Role: "worker", State: StateIdle, Workspace: "/ws", StartedAt: time.Now()})

	if err := store.UpdateState("a", StateWorking); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}

	a, _ := store.Load("a")
	if a.State != StateWorking {
		t.Errorf("State = %q, want working", a.State)
	}

	// Non-existent agent
	if err := store.UpdateState("zzz", StateIdle); err == nil {
		t.Fatal("expected error for non-existent agent")
	}
}

func TestSQLiteStore_UpdateField(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	_ = store.Save(&Agent{Name: "a", Role: "worker", State: StateIdle, Workspace: "/ws", StartedAt: time.Now()})

	if err := store.UpdateField("a", "team", "alpha"); err != nil {
		t.Fatalf("UpdateField: %v", err)
	}

	a, _ := store.Load("a")
	if a.Team != "alpha" {
		t.Errorf("Team = %q, want alpha", a.Team)
	}

	// Disallowed field
	if err := store.UpdateField("a", "name", "evil"); err == nil {
		t.Fatal("expected error for disallowed field")
	}
}

func TestSQLiteStore_RootFields(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().Truncate(time.Second)
	a := &Agent{
		Name:          "root",
		Role:          RoleRoot,
		State:         StateIdle,
		Workspace:     "/ws",
		StartedAt:     now,
		IsRoot:        true,
		CrashCount:    2,
		LastCrashTime: &now,
		RecoveredFrom: "old-session",
		Children:      []string{"eng-01"},
	}

	if err := store.Save(a); err != nil {
		t.Fatalf("Save root: %v", err)
	}

	loaded, _ := store.Load("root")
	if !loaded.IsRoot {
		t.Error("IsRoot should be true")
	}
	if loaded.CrashCount != 2 {
		t.Errorf("CrashCount = %d, want 2", loaded.CrashCount)
	}
	if loaded.LastCrashTime == nil {
		t.Error("LastCrashTime should not be nil")
	}
	if loaded.RecoveredFrom != "old-session" {
		t.Errorf("RecoveredFrom = %q, want old-session", loaded.RecoveredFrom)
	}
}

func TestSQLiteStore_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")

	// Two stores sharing the same DB
	s1, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("s1: %v", err)
	}
	defer func() { _ = s1.Close() }()

	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("s2: %v", err)
	}
	defer func() { _ = s2.Close() }()

	// s1 saves agent A
	_ = s1.Save(&Agent{Name: "a", Role: "worker", State: StateIdle, Workspace: "/ws", StartedAt: time.Now()})

	// s2 saves agent B
	_ = s2.Save(&Agent{Name: "b", Role: "engineer", State: StateWorking, Workspace: "/ws", StartedAt: time.Now()})

	// Both should see both agents
	all1, _ := s1.LoadAll()
	all2, _ := s2.LoadAll()

	if len(all1) != 2 {
		t.Errorf("s1 sees %d agents, want 2", len(all1))
	}
	if len(all2) != 2 {
		t.Errorf("s2 sees %d agents, want 2", len(all2))
	}
}
