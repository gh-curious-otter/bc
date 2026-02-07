package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "engineer-01",
		Role:      RoleEngineer,
		Tool:      "claude",
		Team:      "backend",
		Parent:    "manager-01",
		State:     StateWorking,
		Worktree:  ".bc/worktrees/engineer-01",
		StartedAt: time.Now().Truncate(time.Second),
	}

	// Save
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "agents", "engineer-01.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("State file not created: %v", err)
	}

	// Load
	loaded, err := store.Load("engineer-01")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}

	// Verify fields
	if loaded.Name != state.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, state.Name)
	}
	if loaded.Role != state.Role {
		t.Errorf("Role = %q, want %q", loaded.Role, state.Role)
	}
	if loaded.Tool != state.Tool {
		t.Errorf("Tool = %q, want %q", loaded.Tool, state.Tool)
	}
	if loaded.Team != state.Team {
		t.Errorf("Team = %q, want %q", loaded.Team, state.Team)
	}
	if loaded.Parent != state.Parent {
		t.Errorf("Parent = %q, want %q", loaded.Parent, state.Parent)
	}
	if loaded.State != state.State {
		t.Errorf("State = %q, want %q", loaded.State, state.State)
	}
}

func TestStateStore_LoadNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	loaded, err := store.Load("nonexistent")
	if err != nil {
		t.Fatalf("Load should not error for nonexistent: %v", err)
	}
	if loaded != nil {
		t.Error("Load should return nil for nonexistent agent")
	}
}

func TestStateStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "to-delete",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !store.Exists("to-delete") {
		t.Fatal("Agent should exist after save")
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists("to-delete") {
		t.Error("Agent should not exist after delete")
	}
}

func TestStateStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Create several agents
	agents := []string{"engineer-01", "engineer-02", "manager-01"}
	for _, name := range agents {
		state := &AgentState{
			Name:      name,
			Role:      RoleEngineer,
			StartedAt: time.Now(),
		}
		if err := store.Save(state); err != nil {
			t.Fatalf("Save %s failed: %v", name, err)
		}
	}

	// List
	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(names) != len(agents) {
		t.Errorf("List returned %d agents, want %d", len(names), len(agents))
	}

	// Verify all agents are listed
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range agents {
		if !nameSet[expected] {
			t.Errorf("Agent %q not in list", expected)
		}
	}
}

func TestStateStore_LoadAll(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Create several agents
	for i, name := range []string{"agent-a", "agent-b", "agent-c"} {
		state := &AgentState{
			Name:      name,
			Role:      RoleEngineer,
			State:     State([]string{"idle", "working", "done"}[i]),
			StartedAt: time.Now(),
		}
		if err := store.Save(state); err != nil {
			t.Fatalf("Save %s failed: %v", name, err)
		}
	}

	// LoadAll
	states, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(states) != 3 {
		t.Errorf("LoadAll returned %d states, want 3", len(states))
	}
}

func TestStateStore_UpdateState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "engineer-01",
		Role:      RoleEngineer,
		State:     StateIdle,
		StartedAt: time.Now(),
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.UpdateState("engineer-01", StateWorking); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	loaded, _ := store.Load("engineer-01")
	if loaded.State != StateWorking {
		t.Errorf("State = %q, want %q", loaded.State, StateWorking)
	}
}

func TestStateStore_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "atomic-test",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify no temp file left behind
	tempPath := filepath.Join(tmpDir, "agents", ".atomic-test.json.tmp")
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after successful save")
	}

	// Verify actual file exists
	realPath := filepath.Join(tmpDir, "agents", "atomic-test.json")
	if _, err := os.Stat(realPath); err != nil {
		t.Errorf("Real file should exist: %v", err)
	}
}

func TestToAgentState_Conversion(t *testing.T) {
	agent := &Agent{
		Name:        "test-agent",
		ID:          "test-agent",
		Role:        RoleEngineer,
		Tool:        "claude",
		ParentID:    "manager-01",
		State:       StateWorking,
		WorktreeDir: ".bc/worktrees/test-agent",
		Session:     "bc-test-agent",
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	state := ToAgentState(agent)

	if state.Name != agent.Name {
		t.Errorf("Name = %q, want %q", state.Name, agent.Name)
	}
	if state.Role != agent.Role {
		t.Errorf("Role = %q, want %q", state.Role, agent.Role)
	}
	if state.Parent != agent.ParentID {
		t.Errorf("Parent = %q, want %q", state.Parent, agent.ParentID)
	}
}

func TestAgentState_ToAgent(t *testing.T) {
	state := &AgentState{
		Name:      "test-agent",
		Role:      RoleEngineer,
		Tool:      "claude",
		Parent:    "manager-01",
		State:     StateWorking,
		Worktree:  ".bc/worktrees/test-agent",
		Session:   "bc-test-agent",
		StartedAt: time.Now(),
	}

	agent := state.ToAgent("/workspace")

	if agent.Name != state.Name {
		t.Errorf("Name = %q, want %q", agent.Name, state.Name)
	}
	if agent.Workspace != "/workspace" {
		t.Errorf("Workspace = %q, want /workspace", agent.Workspace)
	}
	if agent.ParentID != state.Parent {
		t.Errorf("ParentID = %q, want %q", agent.ParentID, state.Parent)
	}
}

func TestStateStore_ListEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// List on non-existent directory should return empty, not error
	names, err := store.List()
	if err != nil {
		t.Fatalf("List on empty dir should not error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("List on empty dir should return empty slice, got %d items", len(names))
	}
}

func TestStateStore_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Delete of nonexistent should not error
	if err := store.Delete("nonexistent"); err != nil {
		t.Errorf("Delete of nonexistent should not error: %v", err)
	}
}

func TestStateStore_UpdateStateNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Ensure agents directory exists but agent doesn't
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// UpdateState on nonexistent agent should error
	err := store.UpdateState("nonexistent", StateWorking)
	if err == nil {
		t.Error("UpdateState on nonexistent agent should error")
	}
	if err != nil && err.Error() != "agent nonexistent not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStateStore_ListSkipsTempFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Create agents directory
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create a real agent
	state := &AgentState{
		Name:      "real-agent",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create a temp file (should be skipped)
	tempPath := filepath.Join(tmpDir, "agents", ".temp-agent.json.tmp")
	if err := os.WriteFile(tempPath, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// List should only return real agent
	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(names) != 1 {
		t.Errorf("List returned %d names, want 1", len(names))
	}
	if len(names) > 0 && names[0] != "real-agent" {
		t.Errorf("List returned %q, want 'real-agent'", names[0])
	}
}

func TestStateStore_ListSkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Create agents directory
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Create a real agent
	state := &AgentState{
		Name:      "real-agent",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create a subdirectory (should be skipped)
	subDir := filepath.Join(tmpDir, "agents", "subdir")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// List should only return real agent
	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(names) != 1 {
		t.Errorf("List returned %d names, want 1", len(names))
	}
}
