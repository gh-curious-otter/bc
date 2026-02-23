package agent

import (
	"context"
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
	state, err := store.Create("manager", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if state.Name != "manager" {
		t.Errorf("Name = %q, want manager", state.Name)
	}
	if state.Role != RoleRoot {
		t.Errorf("Role = %q, want %q", state.Role, RoleRoot)
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
	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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
	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Second create fails
	_, err := store.Create("manager2", RoleRoot, "claude")
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

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !store.Exists() {
		t.Error("Exists should return true after creation")
	}
}

func TestRootStateStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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

	state, created, err := store.GetOrCreate("manager", RoleRoot, "claude")
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
	original, err := store.Create("original", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// GetOrCreate should return existing
	state, created, err := store.GetOrCreate("different", Role("engineer"), "codex")
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

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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

	if _, err := store.Create("manager", RoleRoot, "claude"); err != nil {
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

	state, err := store.Create("manager", RoleRoot, "claude")
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
			Role:    RoleRoot,
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
	if state.Role != RoleRoot {
		t.Errorf("Role = %q, want %q", state.Role, RoleRoot)
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

// mockTmuxChecker implements TmuxChecker for testing.
type mockTmuxChecker struct {
	sessions map[string]bool
}

func (m *mockTmuxChecker) HasSession(_ context.Context, name string) bool {
	return m.sessions[name]
}

func TestRootStateStore_CheckRecovery_NoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)
	mock := &mockTmuxChecker{sessions: map[string]bool{}}

	result, err := store.CheckRecovery(context.Background(), mock)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if !result.NeedsCreate {
		t.Error("NeedsCreate should be true when no root exists")
	}
	if result.NeedsRecover {
		t.Error("NeedsRecover should be false")
	}
	if result.IsRunning {
		t.Error("IsRunning should be false")
	}
	if result.State != nil {
		t.Error("State should be nil")
	}
}

func TestRootStateStore_CheckRecovery_Running(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with session
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if updateErr := store.UpdateSession("bc-root-session"); updateErr != nil {
		t.Fatalf("UpdateSession failed: %v", updateErr)
	}

	// Mock tmux says session is alive
	mock := &mockTmuxChecker{sessions: map[string]bool{"bc-root-session": true}}

	result, err := store.CheckRecovery(context.Background(), mock)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if result.NeedsCreate {
		t.Error("NeedsCreate should be false")
	}
	if result.NeedsRecover {
		t.Error("NeedsRecover should be false")
	}
	if !result.IsRunning {
		t.Error("IsRunning should be true when tmux session is alive")
	}
	if result.State == nil {
		t.Error("State should not be nil")
	}
	if result.State.Name != state.Name {
		t.Errorf("State.Name = %q, want %q", result.State.Name, state.Name)
	}
}

func TestRootStateStore_CheckRecovery_NeedsRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with session and children
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.UpdateSession("bc-dead-session"); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}
	if err := store.AddChild("engineer-01"); err != nil {
		t.Fatalf("AddChild failed: %v", err)
	}
	if err := store.AddChild("engineer-02"); err != nil {
		t.Fatalf("AddChild failed: %v", err)
	}

	// Mock tmux says session is dead
	mock := &mockTmuxChecker{sessions: map[string]bool{}}

	result, err := store.CheckRecovery(context.Background(), mock)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	if result.NeedsCreate {
		t.Error("NeedsCreate should be false")
	}
	if !result.NeedsRecover {
		t.Error("NeedsRecover should be true when tmux session is dead")
	}
	if result.IsRunning {
		t.Error("IsRunning should be false")
	}
	if result.State == nil {
		t.Error("State should not be nil")
	}
	if len(result.State.Children) != 2 {
		t.Errorf("Children count = %d, want 2", len(result.State.Children))
	}
}

func TestRootStateStore_MarkRecovered(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with error state
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.UpdateState(StateError); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	// Mark as recovered with new session
	if err := store.MarkRecovered("bc-new-session"); err != nil {
		t.Fatalf("MarkRecovered failed: %v", err)
	}

	// Verify state
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if state.Session != "bc-new-session" {
		t.Errorf("Session = %q, want bc-new-session", state.Session)
	}
	if state.State != StateIdle {
		t.Errorf("State = %q, want %q", state.State, StateIdle)
	}
}

func TestRootStateStore_GetChildren(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// No root - should error
	_, noRootErr := store.GetChildren()
	if !errors.Is(noRootErr, ErrRootNotFound) {
		t.Errorf("GetChildren without root should return ErrRootNotFound, got: %v", noRootErr)
	}

	// Create root with children
	if _, createErr := store.Create("root", RoleRoot, "claude"); createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}
	if addErr := store.AddChild("engineer-01"); addErr != nil {
		t.Fatalf("AddChild failed: %v", addErr)
	}
	if addErr := store.AddChild("qa-01"); addErr != nil {
		t.Fatalf("AddChild failed: %v", addErr)
	}

	children, err := store.GetChildren()
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("Children count = %d, want 2", len(children))
	}
	// Verify children names
	hasEngineer := false
	hasQA := false
	for _, c := range children {
		if c == "engineer-01" {
			hasEngineer = true
		}
		if c == "qa-01" {
			hasQA = true
		}
	}
	if !hasEngineer || !hasQA {
		t.Errorf("Children = %v, want [engineer-01, qa-01]", children)
	}
}

func TestRootStateStore_CheckRecovery_EmptySession(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root without setting session
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Mock tmux - doesn't matter, session is empty
	mock := &mockTmuxChecker{sessions: map[string]bool{"anything": true}}

	result, err := store.CheckRecovery(context.Background(), mock)
	if err != nil {
		t.Fatalf("CheckRecovery failed: %v", err)
	}

	// Empty session should trigger recovery
	if !result.NeedsRecover {
		t.Error("NeedsRecover should be true when session is empty")
	}
	if result.IsRunning {
		t.Error("IsRunning should be false when session is empty")
	}
}

// =============================================================================
// Error Path Tests
// =============================================================================

func TestRootStateStore_LoadCorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory and write corrupted JSON
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	corruptedJSON := []byte(`{"name": "manager", "role": "manager", invalid json here}`)
	rootPath := filepath.Join(agentsDir, RootFileName)
	if err := os.WriteFile(rootPath, corruptedJSON, 0600); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load should fail with corrupted JSON")
	}

	// Verify error message mentions unmarshal failure
	if !errors.Is(err, err) || err.Error() == "" {
		t.Logf("Got expected error: %v", err)
	}
	// The error should wrap the JSON unmarshal error
	expectedSubstring := "failed to unmarshal root state"
	if !containsSubstring(err.Error(), expectedSubstring) {
		t.Errorf("error should contain %q, got: %v", expectedSubstring, err)
	}
}

func TestRootStateStore_LoadReadPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory and valid root.json
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	validJSON := []byte(`{"name": "manager", "role": "manager"}`)
	rootPath := filepath.Join(agentsDir, RootFileName)
	if err := os.WriteFile(rootPath, validJSON, 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Remove read permissions
	if err := os.Chmod(rootPath, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(rootPath, 0600) }() // Restore for cleanup

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load should fail with permission denied")
	}

	// Should NOT be ErrRootNotFound
	if errors.Is(err, ErrRootNotFound) {
		t.Error("error should NOT be ErrRootNotFound for permission issues")
	}

	// Should contain "failed to read"
	if !containsSubstring(err.Error(), "failed to read root state") {
		t.Errorf("error should mention read failure, got: %v", err)
	}
}

func TestRootStateStore_SavePermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory with no write permission
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Remove write permissions from agents directory
	if err := os.Chmod(agentsDir, 0500); err != nil { //nolint:gosec // testing permission errors
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(agentsDir, 0750) }() //nolint:gosec // restore for cleanup

	state := &RootAgentState{
		AgentState: AgentState{
			Name: "manager",
			Role: RoleRoot,
		},
	}

	err := store.Save(state)
	if err == nil {
		t.Fatal("Save should fail with permission denied")
	}

	// Should contain "failed to write temp file"
	if !containsSubstring(err.Error(), "failed to write temp file") {
		t.Errorf("error should mention write failure, got: %v", err)
	}
}

func TestRootStateStore_SaveDirCreationError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Make the base directory read-only so agents subdir can't be created
	if err := os.Chmod(tmpDir, 0500); err != nil { //nolint:gosec // testing permission errors
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0750) }() //nolint:gosec // restore for cleanup

	store := NewRootStateStore(tmpDir)

	state := &RootAgentState{
		AgentState: AgentState{
			Name: "manager",
			Role: RoleRoot,
		},
	}

	err := store.Save(state)
	if err == nil {
		t.Fatal("Save should fail when agents directory cannot be created")
	}

	// Should contain "failed to create agents directory"
	if !containsSubstring(err.Error(), "failed to create agents directory") {
		t.Errorf("error should mention directory creation failure, got: %v", err)
	}
}

func TestRootStateStore_GetOrCreate_LoadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory and write corrupted JSON
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	corruptedJSON := []byte(`{invalid}`)
	rootPath := filepath.Join(agentsDir, RootFileName)
	if err := os.WriteFile(rootPath, corruptedJSON, 0600); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	// GetOrCreate should return the Load error (not ErrRootNotFound)
	_, _, err := store.GetOrCreate("manager", RoleRoot, "claude")
	if err == nil {
		t.Fatal("GetOrCreate should fail with corrupted existing file")
	}

	// Should NOT be ErrRootNotFound
	if errors.Is(err, ErrRootNotFound) {
		t.Error("should propagate unmarshal error, not ErrRootNotFound")
	}
}

func TestRootStateStore_AddChildLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Don't create any root - AddChild should fail on Load
	err := store.AddChild("engineer-01")
	if err == nil {
		t.Fatal("AddChild should fail when root doesn't exist")
	}

	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("expected ErrRootNotFound, got: %v", err)
	}
}

func TestRootStateStore_RemoveChildLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Don't create any root - RemoveChild should fail on Load
	err := store.RemoveChild("engineer-01")
	if err == nil {
		t.Fatal("RemoveChild should fail when root doesn't exist")
	}

	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("expected ErrRootNotFound, got: %v", err)
	}
}

func TestRootStateStore_UpdateStateLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Don't create any root - UpdateState should fail on Load
	err := store.UpdateState(StateWorking)
	if err == nil {
		t.Fatal("UpdateState should fail when root doesn't exist")
	}

	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("expected ErrRootNotFound, got: %v", err)
	}
}

func TestRootStateStore_UpdateSessionLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Don't create any root - UpdateSession should fail on Load
	err := store.UpdateSession("bc-test")
	if err == nil {
		t.Fatal("UpdateSession should fail when root doesn't exist")
	}

	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("expected ErrRootNotFound, got: %v", err)
	}
}

func TestRootStateStore_CreateSaveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Make the base directory read-only so agents subdir can't be created
	if err := os.Chmod(tmpDir, 0500); err != nil { //nolint:gosec // testing permission errors
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0750) }() //nolint:gosec // restore for cleanup

	store := NewRootStateStore(tmpDir)

	_, err := store.Create("manager", RoleRoot, "claude")
	if err == nil {
		t.Fatal("Create should fail when Save fails")
	}

	// Error should propagate from Save
	if !containsSubstring(err.Error(), "failed to create agents directory") {
		t.Errorf("error should mention directory creation failure, got: %v", err)
	}
}

func TestRootStateStore_SaveRenameError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory normally
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create root.json as a directory (which will cause rename to fail)
	rootPath := filepath.Join(agentsDir, RootFileName)
	if err := os.MkdirAll(rootPath, 0750); err != nil {
		t.Fatalf("failed to create root.json as dir: %v", err)
	}

	state := &RootAgentState{
		AgentState: AgentState{
			Name: "manager",
			Role: RoleRoot,
		},
	}

	err := store.Save(state)
	if err == nil {
		t.Fatal("Save should fail when rename fails (target is directory)")
	}

	// Should contain "failed to rename temp file"
	if !containsSubstring(err.Error(), "failed to rename temp file") {
		t.Errorf("error should mention rename failure, got: %v", err)
	}

	// Temp file should be cleaned up
	tempPath := filepath.Join(agentsDir, "."+RootFileName+".tmp")
	if _, statErr := os.Stat(tempPath); !os.IsNotExist(statErr) {
		t.Error("temp file should be cleaned up after rename failure")
	}
}

func TestRootStateStore_GetOrCreate_CreateError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Make the base directory read-only so agents subdir can't be created
	if err := os.Chmod(tmpDir, 0500); err != nil { //nolint:gosec // testing permission errors
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0750) }() //nolint:gosec // restore for cleanup

	store := NewRootStateStore(tmpDir)

	// GetOrCreate should try to create (no existing root) and fail
	_, _, err := store.GetOrCreate("manager", RoleRoot, "claude")
	if err == nil {
		t.Fatal("GetOrCreate should fail when Create fails")
	}

	// Error should propagate from Create/Save
	if !containsSubstring(err.Error(), "failed to create agents directory") {
		t.Errorf("error should mention directory creation failure, got: %v", err)
	}
}

func TestRootStateStore_DeleteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create agents directory
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create root.json as a non-empty directory (can't be removed with os.Remove)
	rootPath := filepath.Join(agentsDir, RootFileName)
	if err := os.MkdirAll(rootPath, 0750); err != nil {
		t.Fatalf("failed to create root.json as dir: %v", err)
	}
	// Add a file inside so it's not empty
	if err := os.WriteFile(filepath.Join(rootPath, "dummy.txt"), []byte("test"), 0600); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	err := store.Delete()
	if err == nil {
		t.Fatal("Delete should fail when target is non-empty directory")
	}

	// Should contain "failed to delete"
	if !containsSubstring(err.Error(), "failed to delete root state") {
		t.Errorf("error should mention delete failure, got: %v", err)
	}
}

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Crash Handling Tests ---

func TestRootStateStore_RecordCrash(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root with a session
	state, createErr := store.Create("root", RoleRoot, "claude")
	if createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}
	state.Session = "old-session"
	if saveErr := store.Save(state); saveErr != nil {
		t.Fatalf("Save failed: %v", saveErr)
	}

	// Record a crash
	if crashErr := store.RecordCrash("old-session"); crashErr != nil {
		t.Fatalf("RecordCrash failed: %v", crashErr)
	}

	// Verify crash was recorded
	loaded, loadErr := store.Load()
	if loadErr != nil {
		t.Fatalf("Load failed: %v", loadErr)
	}

	if loaded.CrashCount != 1 {
		t.Errorf("CrashCount = %d, want 1", loaded.CrashCount)
	}
	if loaded.LastCrashTime == nil {
		t.Error("LastCrashTime should be set")
	}
	if loaded.RecoveredFrom != "old-session" {
		t.Errorf("RecoveredFrom = %q, want old-session", loaded.RecoveredFrom)
	}
	if loaded.State != StateStopped {
		t.Errorf("State = %q, want %q", loaded.State, StateStopped)
	}
}

func TestRootStateStore_RecordCrash_MultipleCrashes(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Record multiple crashes
	for i := 0; i < 3; i++ {
		if err := store.RecordCrash("session-" + string(rune('a'+i))); err != nil {
			t.Fatalf("RecordCrash %d failed: %v", i, err)
		}
	}

	loaded, loadErr := store.Load()
	if loadErr != nil {
		t.Fatalf("Load failed: %v", loadErr)
	}

	if loaded.CrashCount != 3 {
		t.Errorf("CrashCount = %d, want 3", loaded.CrashCount)
	}
}

func TestRootStateStore_RecoverFromCrash(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	// Create root and simulate crash
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.RecordCrash("crashed-session"); err != nil {
		t.Fatalf("RecordCrash failed: %v", err)
	}

	// Recover with new session
	if err := store.RecoverFromCrash("new-session"); err != nil {
		t.Fatalf("RecoverFromCrash failed: %v", err)
	}

	loaded, loadErr := store.Load()
	if loadErr != nil {
		t.Fatalf("Load failed: %v", loadErr)
	}

	if loaded.Session != "new-session" {
		t.Errorf("Session = %q, want new-session", loaded.Session)
	}
	if loaded.State != StateIdle {
		t.Errorf("State = %q, want %q", loaded.State, StateIdle)
	}
	if loaded.RecoveredFrom != "" {
		t.Errorf("RecoveredFrom should be cleared, got %q", loaded.RecoveredFrom)
	}
	// CrashCount should be preserved for diagnostics
	if loaded.CrashCount != 1 {
		t.Errorf("CrashCount = %d, want 1 (should be preserved)", loaded.CrashCount)
	}
}

func TestRootStateStore_GetCrashInfo(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, createErr := store.Create("root", RoleRoot, "claude"); createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}

	// No crashes yet
	info, infoErr := store.GetCrashInfo()
	if infoErr != nil {
		t.Fatalf("GetCrashInfo failed: %v", infoErr)
	}
	if info.HasCrashed() {
		t.Error("HasCrashed should be false initially")
	}
	if info.TimeSinceLastCrash() != 0 {
		t.Error("TimeSinceLastCrash should be 0 with no crashes")
	}

	// Record a crash
	if crashErr := store.RecordCrash("session-1"); crashErr != nil {
		t.Fatalf("RecordCrash failed: %v", crashErr)
	}

	info, infoErr = store.GetCrashInfo()
	if infoErr != nil {
		t.Fatalf("GetCrashInfo failed: %v", infoErr)
	}
	if !info.HasCrashed() {
		t.Error("HasCrashed should be true after crash")
	}
	if info.CrashCount != 1 {
		t.Errorf("CrashCount = %d, want 1", info.CrashCount)
	}
	if info.TimeSinceLastCrash() <= 0 {
		t.Error("TimeSinceLastCrash should be positive after crash")
	}
}

func TestRootStateStore_ClearCrashHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Record crashes
	for i := 0; i < 5; i++ {
		if err := store.RecordCrash("session"); err != nil {
			t.Fatalf("RecordCrash failed: %v", err)
		}
	}

	// Clear history
	if err := store.ClearCrashHistory(); err != nil {
		t.Fatalf("ClearCrashHistory failed: %v", err)
	}

	info, infoErr := store.GetCrashInfo()
	if infoErr != nil {
		t.Fatalf("GetCrashInfo failed: %v", infoErr)
	}
	if info.HasCrashed() {
		t.Error("HasCrashed should be false after clearing history")
	}
	if info.CrashCount != 0 {
		t.Errorf("CrashCount = %d, want 0", info.CrashCount)
	}
}

func TestRootStateStore_CrashPreservesChildren(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewRootStateStore(tmpDir)

	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add children
	children := []string{"engineer-01", "engineer-02", "qa-01"}
	for _, child := range children {
		if err := store.AddChild(child); err != nil {
			t.Fatalf("AddChild(%s) failed: %v", child, err)
		}
	}

	// Crash and recover
	if err := store.RecordCrash("old-session"); err != nil {
		t.Fatalf("RecordCrash failed: %v", err)
	}
	if err := store.RecoverFromCrash("new-session"); err != nil {
		t.Fatalf("RecoverFromCrash failed: %v", err)
	}

	// Verify children preserved
	loaded, loadErr := store.Load()
	if loadErr != nil {
		t.Fatalf("Load failed: %v", loadErr)
	}

	if len(loaded.Children) != len(children) {
		t.Errorf("Children count = %d, want %d", len(loaded.Children), len(children))
	}
	for i, child := range children {
		if loaded.Children[i] != child {
			t.Errorf("Children[%d] = %q, want %q", i, loaded.Children[i], child)
		}
	}
}
