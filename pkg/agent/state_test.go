package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/queue"
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

func TestStateStore_WithQueues(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "engineer-01",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
		WorkQueue: []queue.WorkItem{
			{ID: "work-1", Title: "Fix bug", Status: queue.StatusWorking},
			{ID: "work-2", Title: "Add feature", Status: queue.StatusPending},
		},
		MergeQueue: []queue.WorkItem{
			{ID: "merge-1", Title: "PR #123", Status: queue.StatusDone},
		},
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load("engineer-01")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.WorkQueue) != 2 {
		t.Errorf("WorkQueue has %d items, want 2", len(loaded.WorkQueue))
	}
	if len(loaded.MergeQueue) != 1 {
		t.Errorf("MergeQueue has %d items, want 1", len(loaded.MergeQueue))
	}
	if loaded.WorkQueue[0].ID != "work-1" {
		t.Errorf("First work item ID = %q, want work-1", loaded.WorkQueue[0].ID)
	}
}

func TestStateStore_AddToWorkQueue(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	// Create agent
	state := &AgentState{
		Name:      "engineer-01",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Add work item
	item := queue.WorkItem{ID: "work-new", Title: "New task", Status: queue.StatusPending}
	if err := store.AddToWorkQueue("engineer-01", item); err != nil {
		t.Fatalf("AddToWorkQueue failed: %v", err)
	}

	// Verify
	loaded, _ := store.Load("engineer-01")
	if len(loaded.WorkQueue) != 1 {
		t.Errorf("WorkQueue has %d items, want 1", len(loaded.WorkQueue))
	}
	if loaded.WorkQueue[0].ID != "work-new" {
		t.Errorf("Work item ID = %q, want work-new", loaded.WorkQueue[0].ID)
	}
}

func TestStateStore_RemoveFromWorkQueue(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStateStore(tmpDir)

	state := &AgentState{
		Name:      "engineer-01",
		Role:      RoleEngineer,
		StartedAt: time.Now(),
		WorkQueue: []queue.WorkItem{
			{ID: "keep", Title: "Keep this"},
			{ID: "remove", Title: "Remove this"},
		},
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.RemoveFromWorkQueue("engineer-01", "remove"); err != nil {
		t.Fatalf("RemoveFromWorkQueue failed: %v", err)
	}

	loaded, _ := store.Load("engineer-01")
	if len(loaded.WorkQueue) != 1 {
		t.Errorf("WorkQueue has %d items, want 1", len(loaded.WorkQueue))
	}
	if loaded.WorkQueue[0].ID != "keep" {
		t.Errorf("Remaining item ID = %q, want keep", loaded.WorkQueue[0].ID)
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
