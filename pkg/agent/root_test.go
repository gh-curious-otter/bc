package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRootStateStore_CreateAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root
	state, err := store.Create("manager", RoleManager, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if state.Name != "manager" {
		t.Errorf("Name = %q, want manager", state.Name)
	}
	if state.Role != RoleManager {
		t.Errorf("Role = %q, want %q", state.Role, RoleManager)
	}
	if !state.IsSingleton {
		t.Error("IsSingleton should be true")
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "agents", RootFileName)
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("Root file not created: %v", statErr)
	}

	// Load
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Name != state.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, state.Name)
	}
	if loaded.Tool != "claude" {
		t.Errorf("Loaded Tool = %q, want claude", loaded.Tool)
	}
}

func TestRootStateStore_EnsureSingleton(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// First call should succeed
	if err := store.EnsureSingleton(); err != nil {
		t.Errorf("First EnsureSingleton should succeed: %v", err)
	}

	// Create root
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Second call should fail
	err := store.EnsureSingleton()
	if !errors.Is(err, ErrRootExists) {
		t.Errorf("Second EnsureSingleton should return ErrRootExists, got: %v", err)
	}
}

func TestRootStateStore_CreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// First create succeeds
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Second create fails
	_, err := store.Create("manager2", RoleManager, "claude")
	if !errors.Is(err, ErrRootExists) {
		t.Errorf("Second Create should return ErrRootExists, got: %v", err)
	}
}

func TestRootStateStore_LoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	_, err := store.Load()
	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("Load on empty should return ErrRootNotFound, got: %v", err)
	}
}

func TestRootStateStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if store.Exists() {
		t.Error("Exists should return false before creation")
	}

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !store.Exists() {
		t.Error("Exists should return true after creation")
	}
}

func TestRootStateStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !store.Exists() {
		t.Fatal("Root should exist after create")
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists() {
		t.Error("Root should not exist after delete")
	}
}

func TestRootStateStore_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Delete of nonexistent should not error
	if err := store.Delete(); err != nil {
		t.Errorf("Delete of nonexistent should not error: %v", err)
	}
}

func TestRootStateStore_GetOrCreate_New(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	state, created, err := store.GetOrCreate("manager", RoleManager, "claude")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if !created {
		t.Error("Should indicate root was created")
	}
	if state.Name != "manager" {
		t.Errorf("Name = %q, want manager", state.Name)
	}
}

func TestRootStateStore_GetOrCreate_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create first
	original, err := store.Create("original", RoleManager, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// GetOrCreate should return existing
	state, created, err := store.GetOrCreate("different", RoleEngineer, "codex")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	if created {
		t.Error("Should indicate root was NOT created")
	}
	if state.Name != original.Name {
		t.Errorf("Should return original root, got Name = %q", state.Name)
	}
}

func TestRootStateStore_Children(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add children
	if err := store.AddChild("engineer-01"); err != nil {
		t.Fatalf("AddChild failed: %v", err)
	}
	if err := store.AddChild("engineer-02"); err != nil {
		t.Fatalf("AddChild failed: %v", err)
	}

	// Verify
	state, _ := store.Load()
	if len(state.Children) != 2 {
		t.Errorf("Children count = %d, want 2", len(state.Children))
	}

	// Add duplicate (should not add again)
	if err := store.AddChild("engineer-01"); err != nil {
		t.Fatalf("AddChild duplicate failed: %v", err)
	}
	state, _ = store.Load()
	if len(state.Children) != 2 {
		t.Errorf("Children count after duplicate = %d, want 2", len(state.Children))
	}

	// Remove one
	if err := store.RemoveChild("engineer-01"); err != nil {
		t.Fatalf("RemoveChild failed: %v", err)
	}
	state, _ = store.Load()
	if len(state.Children) != 1 {
		t.Errorf("Children count after remove = %d, want 1", len(state.Children))
	}
	if state.Children[0] != "engineer-02" {
		t.Errorf("Remaining child = %q, want engineer-02", state.Children[0])
	}
}

func TestRootStateStore_UpdateState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.UpdateState(StateWorking); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	state, _ := store.Load()
	if state.State != StateWorking {
		t.Errorf("State = %q, want %q", state.State, StateWorking)
	}
}

func TestRootStateStore_UpdateSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.UpdateSession("bc-manager-123"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	state, _ := store.Load()
	if state.Session != "bc-manager-123" {
		t.Errorf("Session = %q, want bc-manager-123", state.Session)
	}
}

func TestRootStateStore_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify no temp file left behind
	tempPath := filepath.Join(tmpDir, "agents", "."+RootFileName+".tmp")
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after successful save")
	}

	// Verify actual file exists
	realPath := filepath.Join(tmpDir, "agents", RootFileName)
	if _, err := os.Stat(realPath); err != nil {
		t.Errorf("Real file should exist: %v", err)
	}
}

func TestRootStateStore_UpdatedAtTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	state, err := store.Create("manager", RoleManager, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	originalUpdated := state.UpdatedAt

	// Wait a bit and update
	time.Sleep(10 * time.Millisecond)
	if err := store.UpdateState(StateWorking); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	loaded, _ := store.Load()
	if !loaded.UpdatedAt.After(originalUpdated) {
		t.Error("UpdatedAt should be updated after save")
	}
}

func TestRootAgentState_InheritsAgentState(t *testing.T) {
	state := &RootAgentState{
		AgentState: AgentState{
			Name:    "root",
			Role:    RoleManager,
			Tool:    "claude",
			State:   StateWorking,
			Session: "bc-root",
		},
		IsSingleton: true,
		Children:    []string{"child1", "child2"},
	}

	// Verify embedded fields are accessible
	if state.Name != "root" {
		t.Errorf("Name = %q, want root", state.Name)
	}
	if state.Role != RoleManager {
		t.Errorf("Role = %q, want %q", state.Role, RoleManager)
	}
	if state.Session != "bc-root" {
		t.Errorf("Session = %q, want bc-root", state.Session)
	}
	if !state.IsSingleton {
		t.Error("IsSingleton should be true")
	}
	if len(state.Children) != 2 {
		t.Errorf("Children count = %d, want 2", len(state.Children))
	}
}

// mockTmuxChecker implements TmuxChecker for testing
type mockTmuxChecker struct {
	sessions map[string]bool
}

func (m *mockTmuxChecker) HasSession(name string) bool {
	return m.sessions[name]
}

func TestRootStateStore_CheckRecovery_NoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)
	tmux := &mockTmuxChecker{sessions: map[string]bool{}}

	result, err := store.CheckRecovery(tmux)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if !result.NeedsCreate {
		t.Error("NeedsCreate should be true when no root exists")
	}
	if result.NeedsRecover {
		t.Error("NeedsRecover should be false when no root exists")
	}
	if result.IsRunning {
		t.Error("IsRunning should be false when no root exists")
	}
}

func TestRootStateStore_CheckRecovery_RootRunning(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with session
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.UpdateSession("bc-manager"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	// Mock tmux with session alive
	tmux := &mockTmuxChecker{sessions: map[string]bool{"bc-manager": true}}

	result, err := store.CheckRecovery(tmux)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if result.NeedsCreate {
		t.Error("NeedsCreate should be false when root exists")
	}
	if result.NeedsRecover {
		t.Error("NeedsRecover should be false when session alive")
	}
	if !result.IsRunning {
		t.Error("IsRunning should be true when session alive")
	}
	if result.State == nil {
		t.Error("State should be set")
	}
}

func TestRootStateStore_CheckRecovery_RootDead(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with session
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.UpdateSession("bc-manager"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}
	if err := store.AddChild("engineer-01"); err != nil {
		t.Fatalf("AddChild failed: %v", err)
	}

	// Mock tmux with session dead
	tmux := &mockTmuxChecker{sessions: map[string]bool{}}

	result, err := store.CheckRecovery(tmux)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if result.NeedsCreate {
		t.Error("NeedsCreate should be false when root state exists")
	}
	if !result.NeedsRecover {
		t.Error("NeedsRecover should be true when session dead")
	}
	if result.IsRunning {
		t.Error("IsRunning should be false when session dead")
	}
	if result.State == nil {
		t.Fatal("State should be set for recovery")
	}
	if len(result.State.Children) != 1 {
		t.Errorf("Children should be preserved, got %d", len(result.State.Children))
	}
}

func TestRootStateStore_MarkRecovered(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.UpdateSession("old-session"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}
	if err := store.UpdateState(StateStuck); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Mark recovered with new session
	if err := store.MarkRecovered("new-session"); err != nil {
		t.Fatalf("MarkRecovered failed: %v", err)
	}

	state, _ := store.Load()
	if state.Session != "new-session" {
		t.Errorf("Session = %q, want new-session", state.Session)
	}
	if state.State != StateIdle {
		t.Errorf("State = %q, want %q", state.State, StateIdle)
	}
}

func TestRootStateStore_GetChildren(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with children
	if _, err := store.Create("manager", RoleManager, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_ = store.AddChild("engineer-01")
	_ = store.AddChild("qa-01")

	children, err := store.GetChildren()
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("GetChildren returned %d, want 2", len(children))
	}
}
