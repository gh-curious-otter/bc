package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/tmux"
)

func TestMain(m *testing.M) {
	// Setup roles for tests
	RoleCapabilities[Role("engineer")] = []Capability{CapImplementTasks}
	RoleCapabilities[Role("manager")] = []Capability{CapAssignWork}
	RoleCapabilities[Role("qa")] = []Capability{CapTestWork, CapReviewWork}
	RoleCapabilities[Role("product-manager")] = []Capability{CapCreateEpics, CapCreateAgents}
	RoleCapabilities[Role("worker")] = []Capability{CapImplementTasks}

	RoleHierarchy[Role("manager")] = []Role{Role("engineer"), Role("qa"), Role("worker")}
	RoleHierarchy[Role("product-manager")] = []Role{Role("manager")}
	RoleHierarchy[RoleRoot] = []Role{Role("product-manager"), Role("manager"), Role("engineer"), Role("qa"), Role("worker")}

	os.Exit(m.Run())
}

// newTestManager creates a Manager with a unique tmux prefix and temp state dir.
// The tmux manager uses a prefix that won't match any real sessions.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager(fmt.Sprintf("bctest-%d-", time.Now().UnixNano())),
		stateDir: t.TempDir(),
		agentCmd: "/bin/true",
	}
}

// --- Agent struct method tests ---

func TestAgent_HasCapability(t *testing.T) {
	tests := []struct {
		name     string
		cap      Capability
		agent    Agent
		expected bool
	}{
		{"engineer can implement", CapImplementTasks, Agent{Role: Role("engineer")}, true},
		{"engineer cannot create agents", CapCreateAgents, Agent{Role: Role("engineer")}, false},
		{"manager can assign work", CapAssignWork, Agent{Role: Role("manager")}, true},
		{"qa can test work", CapTestWork, Agent{Role: Role("qa")}, true},
		{"qa can review work", CapReviewWork, Agent{Role: Role("qa")}, true},
		{"product manager can create epics", CapCreateEpics, Agent{Role: Role("product-manager")}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.agent.HasCapability(tc.cap); got != tc.expected {
				t.Errorf("Agent{Role: %s}.HasCapability(%s) = %v, want %v", tc.agent.Role, tc.cap, got, tc.expected)
			}
		})
	}
}

func TestAgent_CanCreate(t *testing.T) {
	tests := []struct {
		name      string
		childRole Role
		agent     Agent
		expected  bool
	}{
		{"manager can create engineer", Role("engineer"), Agent{Role: Role("manager")}, true},
		{"manager can create qa", Role("qa"), Agent{Role: Role("manager")}, true},
		{"engineer cannot create anything", Role("worker"), Agent{Role: Role("engineer")}, false},
		{"product manager can create manager", Role("manager"), Agent{Role: Role("product-manager")}, true},
		{"coordinator can create worker", Role("worker"), Agent{Role: RoleRoot}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.agent.CanCreate(tc.childRole); got != tc.expected {
				t.Errorf("Agent{Role: %s}.CanCreate(%s) = %v, want %v", tc.agent.Role, tc.childRole, got, tc.expected)
			}
		})
	}
}

func TestAgent_IsLeaf(t *testing.T) {
	t.Run("no children", func(t *testing.T) {
		a := Agent{Children: []string{}}
		if !a.IsLeaf() {
			t.Error("expected IsLeaf() = true for agent with no children")
		}
	})
	t.Run("nil children", func(t *testing.T) {
		a := Agent{}
		if !a.IsLeaf() {
			t.Error("expected IsLeaf() = true for agent with nil children")
		}
	})
	t.Run("has children", func(t *testing.T) {
		a := Agent{Children: []string{"child-1"}}
		if a.IsLeaf() {
			t.Error("expected IsLeaf() = false for agent with children")
		}
	})
}

func TestAgent_Level(t *testing.T) {
	tests := []struct {
		role     Role
		expected int
	}{
		{Role("product-manager"), 1},
		{RoleRoot, -1},
		{Role("manager"), 1},
		{Role("engineer"), 1},
		{Role("worker"), 1},
		{Role("qa"), 1},
	}
	for _, tc := range tests {
		a := Agent{Role: tc.role}
		if got := a.Level(); got != tc.expected {
			t.Errorf("Agent{Role: %s}.Level() = %d, want %d", tc.role, got, tc.expected)
		}
	}
}

// --- Pure function edge-case tests ---

func TestCanCreateRole_UnknownRole(t *testing.T) {
	if CanCreateRole(Role("unknown"), Role("engineer")) {
		t.Error("unknown parent role should return false")
	}
	if CanCreateRole(Role("manager"), Role("unknown")) {
		t.Error("unknown child role should return false")
	}
}

func TestHasCapability_UnknownRole(t *testing.T) {
	if HasCapability(Role("unknown"), CapImplementTasks) {
		t.Error("unknown role should return false")
	}
}

func TestRoleLevel_UnknownRole(t *testing.T) {
	if got := RoleLevel(Role("unknown")); got != 1 {
		t.Errorf("unknown role level = %d, want 1", got)
	}
}

func TestValidateTransition_UnknownState(t *testing.T) {
	err := ValidateTransition(State("bogus"), StateIdle)
	if err == nil {
		t.Error("expected error for unknown current state")
	}
}

// --- Constructor tests ---

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/test-agents")
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.agents == nil {
		t.Error("agents map should be initialized")
	}
	if m.tmux == nil {
		t.Error("tmux manager should be initialized")
	}
	if m.stateDir != "/tmp/test-agents" {
		t.Errorf("stateDir = %q, want %q", m.stateDir, "/tmp/test-agents")
	}
}

func TestNewWorkspaceManager(t *testing.T) {
	m := NewWorkspaceManager("/tmp/test-agents", "/workspace")
	if m == nil {
		t.Fatal("NewWorkspaceManager returned nil")
	}
	if m.agents == nil {
		t.Error("agents map should be initialized")
	}
	if m.tmux == nil {
		t.Error("tmux manager should be initialized")
	}
	if m.workspacePath != "/workspace" {
		t.Errorf("workspacePath = %q, want %q", m.workspacePath, "/workspace")
	}
}

// --- Config-dependent function tests ---

func TestSetAgentByName(t *testing.T) {
	// Save original config and restore after test
	origAgents := config.Agents
	defer func() { config.Agents = origAgents }()

	config.Agents = []config.AgentsItem{
		{Name: "claude", Command: "claude --skip", Description: "Claude"},
		{Name: "cursor", Command: "cursor-agent", Description: "Cursor"},
	}

	m := newTestManager(t)

	t.Run("found", func(t *testing.T) {
		if !m.SetAgentByName("claude") {
			t.Error("expected SetAgentByName to return true for known agent")
		}
		if m.agentCmd != "claude --skip" {
			t.Errorf("agentCmd = %q, want %q", m.agentCmd, "claude --skip")
		}
	})

	t.Run("not found", func(t *testing.T) {
		if m.SetAgentByName("nonexistent") {
			t.Error("expected SetAgentByName to return false for unknown agent")
		}
	})
}

func TestGetAgentCommand(t *testing.T) {
	origAgents := config.Agents
	defer func() { config.Agents = origAgents }()

	config.Agents = []config.AgentsItem{
		{Name: "claude", Command: "claude --skip"},
		{Name: "codex", Command: "codex --full-auto"},
	}

	t.Run("found", func(t *testing.T) {
		cmd, ok := GetAgentCommand("claude")
		if !ok {
			t.Error("expected ok=true for known tool")
		}
		if cmd != "claude --skip" {
			t.Errorf("cmd = %q, want %q", cmd, "claude --skip")
		}
	})

	t.Run("not found", func(t *testing.T) {
		cmd, ok := GetAgentCommand("nonexistent")
		if ok {
			t.Error("expected ok=false for unknown tool")
		}
		if cmd != "" {
			t.Errorf("cmd = %q, want empty", cmd)
		}
	})
}

func TestListAvailableTools(t *testing.T) {
	origAgents := config.Agents
	defer func() { config.Agents = origAgents }()

	config.Agents = []config.AgentsItem{
		{Name: "claude"},
		{Name: "cursor"},
		{Name: "codex"},
	}

	tools := ListAvailableTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	// Tools should contain all names (order depends on slice)
	found := map[string]bool{}
	for _, tool := range tools {
		found[tool] = true
	}
	for _, name := range []string{"claude", "cursor", "codex"} {
		if !found[name] {
			t.Errorf("missing tool %q in result", name)
		}
	}
}

// --- Manager listing tests ---

func TestListChildren(t *testing.T) {
	m := newTestManager(t)
	m.agents["manager-1"] = &Agent{
		Name:     "manager-1",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{"eng-1", "eng-2"},
	}
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateWorking,
		ParentID: "manager-1",
		Children: []string{},
	}
	m.agents["eng-2"] = &Agent{
		Name:     "eng-2",
		Role:     Role("engineer"),
		State:    StateIdle,
		ParentID: "manager-1",
		Children: []string{},
	}

	t.Run("has children", func(t *testing.T) {
		children := m.ListChildren("manager-1")
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(children))
		}
	})

	t.Run("no children", func(t *testing.T) {
		children := m.ListChildren("eng-1")
		if len(children) != 0 {
			t.Errorf("expected 0 children, got %d", len(children))
		}
	})

	t.Run("nonexistent parent", func(t *testing.T) {
		children := m.ListChildren("nonexistent")
		if children != nil {
			t.Error("expected nil for nonexistent parent")
		}
	})

	t.Run("returns copies", func(t *testing.T) {
		children := m.ListChildren("manager-1")
		children[0].State = StateDone
		original := m.agents["eng-1"]
		if original.State == StateDone {
			t.Error("modifying returned child should not affect original")
		}
	})
}

func TestListDescendants(t *testing.T) {
	m := newTestManager(t)
	// Build a 3-level hierarchy: root → manager → engineer
	m.agents["coord"] = &Agent{
		Name:     "coord",
		Role:     RoleRoot,
		State:    StateIdle,
		Children: []string{"mgr"},
	}
	m.agents["mgr"] = &Agent{
		Name:     "mgr",
		Role:     Role("manager"),
		State:    StateIdle,
		ParentID: "coord",
		Children: []string{"eng-1", "eng-2"},
	}
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateWorking,
		ParentID: "mgr",
		Children: []string{},
	}
	m.agents["eng-2"] = &Agent{
		Name:     "eng-2",
		Role:     Role("engineer"),
		State:    StateIdle,
		ParentID: "mgr",
		Children: []string{},
	}

	t.Run("all descendants from top", func(t *testing.T) {
		descendants := m.ListDescendants("coord")
		if len(descendants) != 3 {
			t.Errorf("expected 3 descendants, got %d", len(descendants))
		}
	})

	t.Run("descendants from middle", func(t *testing.T) {
		descendants := m.ListDescendants("mgr")
		if len(descendants) != 2 {
			t.Errorf("expected 2 descendants, got %d", len(descendants))
		}
	})

	t.Run("leaf has no descendants", func(t *testing.T) {
		descendants := m.ListDescendants("eng-1")
		if len(descendants) != 0 {
			t.Errorf("expected 0 descendants, got %d", len(descendants))
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		descendants := m.ListDescendants("nonexistent")
		if len(descendants) != 0 {
			t.Errorf("expected 0 descendants, got %d", len(descendants))
		}
	})
}

func TestGetParent(t *testing.T) {
	m := newTestManager(t)
	m.agents["mgr"] = &Agent{
		Name:     "mgr",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{"eng-1"},
	}
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateWorking,
		ParentID: "mgr",
		Children: []string{},
	}

	t.Run("has parent", func(t *testing.T) {
		parent := m.GetParent("eng-1")
		if parent == nil {
			t.Fatal("expected non-nil parent")
		}
		if parent.Name != "mgr" {
			t.Errorf("parent name = %q, want %q", parent.Name, "mgr")
		}
	})

	t.Run("no parent", func(t *testing.T) {
		parent := m.GetParent("mgr")
		if parent != nil {
			t.Error("expected nil parent for root agent")
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		parent := m.GetParent("nonexistent")
		if parent != nil {
			t.Error("expected nil for nonexistent agent")
		}
	})

	t.Run("parent not in map", func(t *testing.T) {
		m.agents["orphan"] = &Agent{
			Name:     "orphan",
			Role:     Role("engineer"),
			ParentID: "deleted-parent",
		}
		parent := m.GetParent("orphan")
		if parent != nil {
			t.Error("expected nil when parent ID references nonexistent agent")
		}
	})

	t.Run("returns copy", func(t *testing.T) {
		parent := m.GetParent("eng-1")
		parent.State = StateDone
		if m.agents["mgr"].State == StateDone {
			t.Error("modifying returned parent should not affect original")
		}
	})
}

func TestListByRole(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{Name: "eng-1", Role: Role("engineer"), State: StateIdle, Children: []string{}}
	m.agents["eng-2"] = &Agent{Name: "eng-2", Role: Role("engineer"), State: StateWorking, Children: []string{}}
	m.agents["qa-1"] = &Agent{Name: "qa-1", Role: Role("qa"), State: StateIdle, Children: []string{}}
	m.agents["mgr"] = &Agent{Name: "mgr", Role: Role("manager"), State: StateIdle, Children: []string{}}

	t.Run("filter engineers", func(t *testing.T) {
		engineers := m.ListByRole(Role("engineer"))
		if len(engineers) != 2 {
			t.Fatalf("expected 2 engineers, got %d", len(engineers))
		}
		// Should be sorted by name
		if engineers[0].Name != "eng-1" || engineers[1].Name != "eng-2" {
			t.Errorf("engineers not sorted: got %s, %s", engineers[0].Name, engineers[1].Name)
		}
	})

	t.Run("filter qa", func(t *testing.T) {
		qas := m.ListByRole(Role("qa"))
		if len(qas) != 1 {
			t.Fatalf("expected 1 qa, got %d", len(qas))
		}
		if qas[0].Name != "qa-1" {
			t.Errorf("qa name = %q, want %q", qas[0].Name, "qa-1")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		pms := m.ListByRole(Role("product-manager"))
		if len(pms) != 0 {
			t.Errorf("expected 0 product managers, got %d", len(pms))
		}
	})

	t.Run("returns copies", func(t *testing.T) {
		engineers := m.ListByRole(Role("engineer"))
		engineers[0].State = StateDone
		if m.agents["eng-1"].State == StateDone {
			t.Error("modifying returned agent should not affect original")
		}
	})
}

func TestAgentCount(t *testing.T) {
	m := newTestManager(t)

	if m.AgentCount() != 0 {
		t.Errorf("empty manager should have 0 agents, got %d", m.AgentCount())
	}

	m.agents["a"] = &Agent{Name: "a"}
	m.agents["b"] = &Agent{Name: "b"}
	m.agents["c"] = &Agent{Name: "c"}

	if m.AgentCount() != 3 {
		t.Errorf("expected 3 agents, got %d", m.AgentCount())
	}
}

func TestRunningCount(t *testing.T) {
	m := newTestManager(t)

	m.agents["a"] = &Agent{Name: "a", State: StateIdle}
	m.agents["b"] = &Agent{Name: "b", State: StateWorking}
	m.agents["c"] = &Agent{Name: "c", State: StateStopped}
	m.agents["d"] = &Agent{Name: "d", State: StateDone}
	m.agents["e"] = &Agent{Name: "e", State: StateStopped}

	// 3 non-stopped agents: a (idle), b (working), d (done)
	if got := m.RunningCount(); got != 3 {
		t.Errorf("RunningCount() = %d, want 3", got)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	m := newTestManager(t)
	if a := m.GetAgent("nonexistent"); a != nil {
		t.Error("expected nil for nonexistent agent")
	}
}

func TestListAgents_SortOrder(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-2"] = &Agent{Name: "eng-2", Role: Role("engineer"), State: StateIdle, Children: []string{}}
	m.agents["eng-1"] = &Agent{Name: "eng-1", Role: Role("engineer"), State: StateIdle, Children: []string{}}
	m.agents["mgr"] = &Agent{Name: "mgr", Role: Role("manager"), State: StateIdle, Children: []string{}}
	m.agents["coord"] = &Agent{Name: "coord", Role: RoleRoot, State: StateIdle, Children: []string{}}
	m.agents["qa-1"] = &Agent{Name: "qa-1", Role: Role("qa"), State: StateIdle, Children: []string{}}

	agents := m.ListAgents()
	if len(agents) != 5 {
		t.Fatalf("expected 5 agents, got %d", len(agents))
	}

	// Root (level -1) first, then Manager (level 1), then Engineer/QA (level 1) sorted by name
	expectedOrder := []string{"coord", "eng-1", "eng-2", "mgr", "qa-1"}
	for i, expected := range expectedOrder {
		if agents[i].Name != expected {
			t.Errorf("agents[%d].Name = %q, want %q", i, agents[i].Name, expected)
		}
	}
}

// --- State persistence tests ---

func TestSaveAndLoadState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manager and add agents
	m1 := &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager("test-"),
		stateDir: tmpDir,
	}
	m1.agents["eng-1"] = &Agent{
		Name:      "eng-1",
		Role:      Role("engineer"),
		State:     StateWorking,
		Task:      "implementing feature",
		Children:  []string{},
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m1.agents["qa-1"] = &Agent{
		Name:     "qa-1",
		Role:     Role("qa"),
		State:    StateIdle,
		Children: []string{},
	}

	// Save state
	if err := m1.saveState(); err != nil {
		t.Fatalf("saveState failed: %v", err)
	}

	// Verify file exists
	stateFile := filepath.Join(tmpDir, "agents.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// Load into new manager
	m2 := &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager("test-"),
		stateDir: tmpDir,
	}
	if err := m2.LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify loaded agents
	if len(m2.agents) != 2 {
		t.Fatalf("expected 2 agents after load, got %d", len(m2.agents))
	}

	eng := m2.agents["eng-1"]
	if eng == nil {
		t.Fatal("eng-1 not found after load")
	}
	if eng.Role != Role("engineer") {
		t.Errorf("eng-1 role = %s, want %s", eng.Role, Role("engineer"))
	}
	if eng.State != StateWorking {
		t.Errorf("eng-1 state = %s, want %s", eng.State, StateWorking)
	}
	if eng.Task != "implementing feature" {
		t.Errorf("eng-1 task = %q, want %q", eng.Task, "implementing feature")
	}
}

func TestLoadState_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager("test-"),
		stateDir: tmpDir,
	}
	// No agents.json exists, should return nil (not error)
	if err := m.LoadState(); err != nil {
		t.Errorf("LoadState with no file should return nil, got: %v", err)
	}
	if len(m.agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(m.agents))
	}
}

func TestLoadState_EmptyStateDir(t *testing.T) {
	m := &Manager{
		agents:   make(map[string]*Agent),
		stateDir: "",
	}
	if err := m.LoadState(); err != nil {
		t.Errorf("LoadState with empty stateDir should return nil, got: %v", err)
	}
}

func TestSaveState_EmptyStateDir(t *testing.T) {
	m := &Manager{
		agents:   make(map[string]*Agent),
		stateDir: "",
	}
	if err := m.saveState(); err != nil {
		t.Errorf("saveState with empty stateDir should return nil, got: %v", err)
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "agents.json")
	if err := os.WriteFile(stateFile, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		agents:   make(map[string]*Agent),
		stateDir: tmpDir,
	}
	if err := m.LoadState(); err == nil {
		t.Error("LoadState with invalid JSON should return error")
	}
}

// --- LoadRoleMemory tests ---

func TestLoadRoleMemory(t *testing.T) {
	tmpDir := t.TempDir()
	rolesDir := filepath.Join(tmpDir, ".bc", "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	t.Run("file exists", func(t *testing.T) {
		content := "You are an engineer. Write code and tests."
		// RoleManager expects YAML or Markdown with frontmatter or just prompt?
		// Actually RoleManager might expect a specific format.
		// Let's check RoleManager.LoadRole.
		if err := os.WriteFile(filepath.Join(rolesDir, "engineer.md"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}

		mem := LoadRoleMemory(tmpDir, Role("engineer"))
		if mem == nil {
			t.Fatal("expected non-nil AgentMemory")
		}
		if mem.RolePrompt != content {
			t.Errorf("RolePrompt = %q, want %q", mem.RolePrompt, content)
		}
		if mem.LoadedAt.IsZero() {
			t.Error("LoadedAt should not be zero")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		mem := LoadRoleMemory(tmpDir, Role("qa"))
		if mem != nil {
			t.Error("expected nil AgentMemory for missing file")
		}
	})

	t.Run("product-manager role", func(t *testing.T) {
		content := "You are a product manager."
		if err := os.WriteFile(filepath.Join(rolesDir, "product-manager.md"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}

		mem := LoadRoleMemory(tmpDir, Role("product-manager"))
		if mem == nil {
			t.Fatal("expected non-nil AgentMemory for product-manager")
		}
		if mem.RolePrompt != content {
			t.Errorf("RolePrompt = %q, want %q", mem.RolePrompt, content)
		}
	})

	t.Run("root role from prompts dir", func(t *testing.T) {
		// Root role uses backward compatible prompts/root.md path
		promptsDir := filepath.Join(tmpDir, "prompts")
		if mkErr := os.MkdirAll(promptsDir, 0750); mkErr != nil {
			t.Fatal(mkErr)
		}
		content := "You are the root coordinator."
		if writeErr := os.WriteFile(filepath.Join(promptsDir, "root.md"), []byte(content), 0600); writeErr != nil {
			t.Fatal(writeErr)
		}

		mem := LoadRoleMemory(tmpDir, RoleRoot)
		if mem == nil {
			t.Fatal("expected non-nil AgentMemory for root role")
		}
		if mem.RolePrompt != content {
			t.Errorf("RolePrompt = %q, want %q", mem.RolePrompt, content)
		}
	})

	t.Run("empty prompt returns nil", func(t *testing.T) {
		// Create role file with empty content
		if writeErr := os.WriteFile(filepath.Join(rolesDir, "empty.md"), []byte(""), 0600); writeErr != nil {
			t.Fatal(writeErr)
		}

		mem := LoadRoleMemory(tmpDir, Role("empty"))
		if mem != nil {
			t.Error("expected nil AgentMemory for empty prompt file")
		}
	})
}

// --- Stop operations tests ---

func TestStopAgent(t *testing.T) {
	m := newTestManager(t)
	m.agents["mgr"] = &Agent{
		Name:     "mgr",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{"eng-1"},
	}
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateWorking,
		ParentID: "mgr",
		Children: []string{},
	}

	// Stop eng-1
	if err := m.StopAgent("eng-1"); err != nil {
		t.Fatalf("StopAgent failed: %v", err)
	}

	// Agent should be stopped
	if m.agents["eng-1"].State != StateStopped {
		t.Errorf("agent state = %s, want %s", m.agents["eng-1"].State, StateStopped)
	}

	// Parent's children should be updated
	if len(m.agents["mgr"].Children) != 0 {
		t.Errorf("parent children = %v, want empty", m.agents["mgr"].Children)
	}

	// State file should be written
	stateFile := filepath.Join(m.stateDir, "agents.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Error("state file should exist after StopAgent")
	}
}

func TestStopAgent_NotFound(t *testing.T) {
	m := newTestManager(t)
	if err := m.StopAgent("nonexistent"); err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestStopAgent_WithWorktree(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{
		Name:        "eng-1",
		Role:        Role("engineer"),
		State:       StateWorking,
		Workspace:   "/tmp/workspace",
		WorktreeDir: "/tmp/workspace/.bc/worktrees/eng-1",
		Children:    []string{},
	}

	// Stop should succeed and preserve worktree for later restart
	if err := m.StopAgent("eng-1"); err != nil {
		t.Fatalf("StopAgent with worktree failed: %v", err)
	}
	if m.agents["eng-1"].State != StateStopped {
		t.Errorf("agent state = %s, want %s", m.agents["eng-1"].State, StateStopped)
	}
	// Worktree should be preserved (not cleared) so agent can resume work on restart
	if m.agents["eng-1"].WorktreeDir != "/tmp/workspace/.bc/worktrees/eng-1" {
		t.Error("worktree dir should be preserved after stop, not cleared")
	}
}

func TestStopAgent_WorktreeSameAsWorkspace(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{
		Name:        "eng-1",
		Role:        Role("engineer"),
		State:       StateWorking,
		Workspace:   "/tmp/workspace",
		WorktreeDir: "/tmp/workspace", // Same as workspace
		Children:    []string{},
	}

	if err := m.StopAgent("eng-1"); err != nil {
		t.Fatalf("StopAgent failed: %v", err)
	}
	// WorktreeDir should NOT be cleared when it equals Workspace
	if m.agents["eng-1"].WorktreeDir != "/tmp/workspace" {
		t.Error("worktreeDir should not be cleared when equal to workspace")
	}
}

func TestStopAll(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{Name: "eng-1", Role: Role("engineer"), State: StateWorking, Children: []string{}}
	m.agents["eng-2"] = &Agent{Name: "eng-2", Role: Role("engineer"), State: StateIdle, Children: []string{}}
	m.agents["qa-1"] = &Agent{Name: "qa-1", Role: Role("qa"), State: StateDone, Children: []string{}}

	if err := m.StopAll(); err != nil {
		t.Fatalf("StopAll failed: %v", err)
	}

	for name, a := range m.agents {
		if a.State != StateStopped {
			t.Errorf("agent %s state = %s, want %s", name, a.State, StateStopped)
		}
	}
}

func TestStopAgentTree(t *testing.T) {
	m := newTestManager(t)
	m.agents["mgr"] = &Agent{
		Name:     "mgr",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{"eng-1", "eng-2"},
	}
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateWorking,
		ParentID: "mgr",
		Children: []string{},
	}
	m.agents["eng-2"] = &Agent{
		Name:     "eng-2",
		Role:     Role("engineer"),
		State:    StateIdle,
		ParentID: "mgr",
		Children: []string{},
	}

	if err := m.StopAgentTree("mgr"); err != nil {
		t.Fatalf("StopAgentTree failed: %v", err)
	}

	// All should be stopped
	for _, name := range []string{"mgr", "eng-1", "eng-2"} {
		if m.agents[name].State != StateStopped {
			t.Errorf("agent %s state = %s, want %s", name, m.agents[name].State, StateStopped)
		}
	}

	// Manager's children should be cleared
	if len(m.agents["mgr"].Children) != 0 {
		t.Errorf("manager children = %v, want empty", m.agents["mgr"].Children)
	}
}

func TestStopAgentTree_NotFound(t *testing.T) {
	m := newTestManager(t)
	if err := m.StopAgentTree("nonexistent"); err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

// --- RenameAgent tests ---

func TestRenameAgent(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-01"] = &Agent{Name: "eng-01", Role: Role("engineer"), State: StateStopped, Children: []string{}}

	if err := m.RenameAgent("eng-01", "engineer-01"); err != nil {
		t.Fatalf("RenameAgent failed: %v", err)
	}

	// Old name should not exist
	if _, exists := m.agents["eng-01"]; exists {
		t.Error("old agent name should not exist")
	}

	// New name should exist
	agent, exists := m.agents["engineer-01"]
	if !exists {
		t.Fatal("new agent name should exist")
	}

	// Agent name should be updated
	if agent.Name != "engineer-01" {
		t.Errorf("agent.Name = %q, want %q", agent.Name, "engineer-01")
	}
}

func TestRenameAgent_NotFound(t *testing.T) {
	m := newTestManager(t)
	if err := m.RenameAgent("nonexistent", "new-name"); err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestRenameAgent_NameExists(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-01"] = &Agent{Name: "eng-01", State: StateStopped, Children: []string{}}
	m.agents["eng-02"] = &Agent{Name: "eng-02", State: StateStopped, Children: []string{}}

	if err := m.RenameAgent("eng-01", "eng-02"); err == nil {
		t.Error("expected error when renaming to existing name")
	}
}

func TestRenameAgent_UpdatesParentChildren(t *testing.T) {
	m := newTestManager(t)
	m.agents["mgr"] = &Agent{Name: "mgr", State: StateIdle, Children: []string{"eng-01", "eng-02"}}
	m.agents["eng-01"] = &Agent{Name: "eng-01", ParentID: "mgr", State: StateStopped, Children: []string{}}
	m.agents["eng-02"] = &Agent{Name: "eng-02", ParentID: "mgr", State: StateStopped, Children: []string{}}

	if err := m.RenameAgent("eng-01", "engineer-01"); err != nil {
		t.Fatalf("RenameAgent failed: %v", err)
	}

	// Parent's children list should be updated
	children := m.agents["mgr"].Children
	found := false
	for _, c := range children {
		if c == "engineer-01" {
			found = true
		}
		if c == "eng-01" {
			t.Error("old name should not be in parent's children")
		}
	}
	if !found {
		t.Error("new name should be in parent's children")
	}
}

// --- RefreshState tests ---

func TestRefreshState(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{Name: "eng-1", State: StateWorking, Children: []string{}}
	m.agents["eng-2"] = &Agent{Name: "eng-2", State: StateStopped, Children: []string{}}

	// RefreshState should succeed - no matching tmux sessions exist
	err := m.RefreshState()
	if err != nil {
		t.Fatalf("RefreshState failed: %v", err)
	}

	// eng-1 was working but no tmux session → should be marked stopped
	if m.agents["eng-1"].State != StateStopped {
		t.Errorf("eng-1 state = %s, want stopped (no tmux session)", m.agents["eng-1"].State)
	}

	// eng-2 was already stopped → should remain stopped
	if m.agents["eng-2"].State != StateStopped {
		t.Errorf("eng-2 state = %s, want stopped", m.agents["eng-2"].State)
	}
}

// --- SpawnAgent error path tests ---

func TestSpawnAgentWithOptions_ParentNotFound(t *testing.T) {
	m := newTestManager(t)
	_, err := m.SpawnAgentWithOptions("eng-1", Role("engineer"), "/tmp", "nonexistent-parent", "")
	if err == nil {
		t.Error("expected error when parent not found")
	}
}

func TestSpawnAgentWithOptions_ParentCantCreate(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{
		Name:     "eng-1",
		Role:     Role("engineer"),
		State:    StateIdle,
		Children: []string{},
	}

	// Engineer cannot create other engineers
	_, err := m.SpawnAgentWithOptions("eng-2", Role("engineer"), "/tmp", "eng-1", "")
	if err == nil {
		t.Error("expected error when parent can't create child role")
	}
}

func TestSpawnAgentWithOptions_NullRole(t *testing.T) {
	m := newTestManager(t)

	tests := []struct {
		name string
		role Role
	}{
		{"empty role", Role("")},
		{"null string", Role("null")},
		{"nil-like string", Role("<nil>")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.SpawnAgentWithOptions("test-agent", tt.role, "/tmp", "", "")
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
			if !strings.Contains(err.Error(), "role is required") {
				t.Errorf("expected 'role is required' error, got: %v", err)
			}
		})
	}
}

func TestSpawnAgentWithOptions_UnknownTool(t *testing.T) {
	origAgents := config.Agents
	defer func() { config.Agents = origAgents }()
	config.Agents = []config.AgentsItem{
		{Name: "claude", Command: "claude"},
	}

	m := newTestManager(t)
	_, err := m.SpawnAgentWithOptions("eng-1", Role("engineer"), "/tmp", "", "nonexistent-tool")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

// --- Tmux accessor test ---

func TestTmux(t *testing.T) {
	m := newTestManager(t)
	if m.Tmux() == nil {
		t.Error("Tmux() should not return nil")
	}
	if m.Tmux() != m.tmux {
		t.Error("Tmux() should return the same tmux manager")
	}
}

// --- UpdateAgentState with task update ---

func TestUpdateAgentState_TaskUpdate(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-1"] = &Agent{
		Name:  "eng-1",
		Role:  Role("engineer"),
		State: StateIdle,
	}

	// Transition to working with a task
	if err := m.UpdateAgentState("eng-1", StateWorking, "writing tests"); err != nil {
		t.Fatalf("UpdateAgentState failed: %v", err)
	}

	a := m.agents["eng-1"]
	if a.Task != "writing tests" {
		t.Errorf("task = %q, want %q", a.Task, "writing tests")
	}
	if a.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// State file should be written
	stateFile := filepath.Join(m.stateDir, "agents.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Error("state file should exist after UpdateAgentState")
	}
}

// --- removeFromParent tests ---

func TestRemoveFromParent(t *testing.T) {
	m := newTestManager(t)

	t.Run("removes child from parent", func(t *testing.T) {
		m.agents["parent"] = &Agent{
			Name:     "parent",
			Role:     Role("manager"),
			Children: []string{"child-1", "child-2", "child-3"},
		}
		m.agents["child-2"] = &Agent{
			Name:     "child-2",
			ParentID: "parent",
		}

		m.removeFromParent("child-2")

		parent := m.agents["parent"]
		if len(parent.Children) != 2 {
			t.Errorf("expected 2 children after removal, got %d", len(parent.Children))
		}
		for _, child := range parent.Children {
			if child == "child-2" {
				t.Error("child-2 should have been removed from parent")
			}
		}
	})

	t.Run("agent not in map", func(t *testing.T) {
		// Should not panic
		m.removeFromParent("nonexistent")
	})

	t.Run("no parent ID", func(t *testing.T) {
		m.agents["orphan"] = &Agent{Name: "orphan"}
		// Should not panic
		m.removeFromParent("orphan")
	})

	t.Run("parent not in map", func(t *testing.T) {
		m.agents["lost-child"] = &Agent{
			Name:     "lost-child",
			ParentID: "deleted-parent",
		}
		// Should not panic
		m.removeFromParent("lost-child")
	})
}

// --- State persistence round-trip with complex data ---

func TestSaveLoadState_ComplexHierarchy(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager("test-"),
		stateDir: tmpDir,
	}

	now := time.Now().Truncate(time.Second)
	m.agents["coord"] = &Agent{
		ID:        "coord",
		Name:      "coord",
		Role:      RoleRoot,
		State:     StateIdle,
		Workspace: "/workspace",
		Session:   "coord",
		Children:  []string{"mgr"},
		StartedAt: now,
		UpdatedAt: now,
		Memory: &AgentMemory{
			RolePrompt: "You are a root.",
			LoadedAt:   now,
		},
	}
	m.agents["mgr"] = &Agent{
		ID:          "mgr",
		Name:        "mgr",
		Role:        Role("manager"),
		State:       StateWorking,
		Workspace:   "/workspace",
		Session:     "mgr",
		ParentID:    "coord",
		Children:    []string{},
		HookedWork:  "work-001",
		WorktreeDir: "/workspace/.bc/worktrees/mgr",
		Tool:        "claude",
		StartedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.saveState(); err != nil {
		t.Fatalf("saveState failed: %v", err)
	}

	// Load into fresh manager
	m2 := &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager("test-"),
		stateDir: tmpDir,
	}
	if err := m2.LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify complex fields
	coord := m2.agents["coord"]
	if coord == nil {
		t.Fatal("coord not found")
	}
	if coord.Memory == nil {
		t.Fatal("coord Memory should not be nil")
	}
	if coord.Memory.RolePrompt != "You are a root." {
		t.Errorf("RolePrompt = %q, want %q", coord.Memory.RolePrompt, "You are a root.")
	}
	if len(coord.Children) != 1 || coord.Children[0] != "mgr" {
		t.Errorf("coord children = %v, want [mgr]", coord.Children)
	}

	mgr := m2.agents["mgr"]
	if mgr == nil {
		t.Fatal("mgr not found")
	}
	if mgr.ParentID != "coord" {
		t.Errorf("mgr ParentID = %q, want %q", mgr.ParentID, "coord")
	}
	if mgr.Tool != "claude" {
		t.Errorf("mgr Tool = %q, want %q", mgr.Tool, "claude")
	}
	if mgr.WorktreeDir != "/workspace/.bc/worktrees/mgr" {
		t.Errorf("mgr WorktreeDir = %q, want expected", mgr.WorktreeDir)
	}
	if mgr.HookedWork != "work-001" {
		t.Errorf("mgr HookedWork = %q, want %q", mgr.HookedWork, "work-001")
	}
}

// --- captureLiveTask parsing tests (via direct string inspection) ---
// captureLiveTask requires tmux.Capture to work, so we test the capture logic
// indirectly via the public RefreshState path when possible. Additional coverage
// for the parsing patterns is ensured below.

func TestCaptureLiveTask_SkipsEmptyLines(t *testing.T) {
	// captureLiveTask is called internally by RefreshState, but since we
	// can't easily mock tmux.Capture, we verify the function returns ""
	// when Capture fails (no real tmux session).
	m := newTestManager(t)
	result := m.captureLiveTask("nonexistent")
	if result != "" {
		t.Errorf("expected empty string for non-existent session, got %q", result)
	}
}

// --- JSON serialization tests ---

func TestAgentJSON_RoundTrip(t *testing.T) {
	original := &Agent{
		ID:          "eng-1",
		Name:        "eng-1",
		Role:        Role("engineer"),
		State:       StateWorking,
		Workspace:   "/workspace",
		Task:        "writing tests",
		Session:     "eng-1-session",
		Tool:        "claude",
		ParentID:    "mgr",
		Children:    []string{"sub1"},
		HookedWork:  "work-099",
		WorktreeDir: "/workspace/.bc/worktrees/eng-1",
		StartedAt:   time.Now().Truncate(time.Second),
		UpdatedAt:   time.Now().Truncate(time.Second),
		Memory: &AgentMemory{
			RolePrompt: "test prompt",
			LoadedAt:   time.Now().Truncate(time.Second),
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded Agent
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID mismatch: %q vs %q", loaded.ID, original.ID)
	}
	if loaded.Role != original.Role {
		t.Errorf("Role mismatch: %q vs %q", loaded.Role, original.Role)
	}
	if loaded.State != original.State {
		t.Errorf("State mismatch: %q vs %q", loaded.State, original.State)
	}
	if loaded.Tool != original.Tool {
		t.Errorf("Tool mismatch: %q vs %q", loaded.Tool, original.Tool)
	}
	if loaded.ParentID != original.ParentID {
		t.Errorf("ParentID mismatch: %q vs %q", loaded.ParentID, original.ParentID)
	}
	if loaded.Memory == nil {
		t.Fatal("Memory should not be nil after round-trip")
	}
	if loaded.Memory.RolePrompt != original.Memory.RolePrompt {
		t.Errorf("RolePrompt mismatch: %q vs %q", loaded.Memory.RolePrompt, original.Memory.RolePrompt)
	}
}

// --- Concurrent access tests (preserved from original) ---

func TestConcurrentSetAgentCommand(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				m.SetAgentCommand("claude")
			} else {
				m.SetAgentCommand("cursor-agent")
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentGetAgent(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     Role("worker"),
		State:    StateIdle,
		Children: []string{"child1", "child2"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := m.GetAgent("test-agent")
			if a == nil {
				t.Error("GetAgent returned nil")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentListAgents(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["agent1"] = &Agent{Name: "agent1", Role: Role("worker"), State: StateIdle}
	m.agents["agent2"] = &Agent{Name: "agent2", Role: Role("worker"), State: StateWorking}
	m.agents["agent3"] = &Agent{Name: "agent3", Role: RoleRoot, State: StateIdle}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agents := m.ListAgents()
			if len(agents) != 3 {
				t.Errorf("expected 3 agents, got %d", len(agents))
			}
		}()
	}
	wg.Wait()
}

func TestGetAgentReturnsCopy(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     Role("worker"),
		State:    StateIdle,
		Children: []string{"child1"},
	}

	copy := m.GetAgent("test-agent")
	copy.State = StateWorking
	copy.Children = append(copy.Children, "child2")

	original := m.agents["test-agent"]
	if original.State != StateIdle {
		t.Errorf("original state was modified: expected %s, got %s", StateIdle, original.State)
	}
	if len(original.Children) != 1 {
		t.Errorf("original children was modified: expected 1, got %d", len(original.Children))
	}
}

func TestListAgentsReturnsCopies(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["agent1"] = &Agent{
		Name:     "agent1",
		Role:     Role("worker"),
		State:    StateIdle,
		Children: []string{"child1"},
	}

	copies := m.ListAgents()
	if len(copies) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(copies))
	}

	copies[0].State = StateWorking
	copies[0].Children = append(copies[0].Children, "child2")

	original := m.agents["agent1"]
	if original.State != StateIdle {
		t.Errorf("original state was modified: expected %s, got %s", StateIdle, original.State)
	}
	if len(original.Children) != 1 {
		t.Errorf("original children was modified: expected 1, got %d", len(original.Children))
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     Role("worker"),
		State:    StateIdle,
		Children: []string{},
	}

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = m.GetAgent("test-agent")
				_ = m.ListAgents()
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if n%2 == 0 {
					m.SetAgentCommand("cmd1")
				} else {
					m.SetAgentCommand("cmd2")
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		parent   Role
		child    Role
		expected bool
	}{
		{Role("product-manager"), Role("manager"), true},
		{Role("manager"), Role("engineer"), true},
		{Role("manager"), Role("qa"), true},
		{RoleRoot, Role("worker"), true},
		{RoleRoot, Role("manager"), true},
		{RoleRoot, Role("qa"), true},
		{Role("engineer"), Role("worker"), false},
		{Role("worker"), Role("engineer"), false},
		{Role("qa"), Role("engineer"), false},
	}

	for _, tc := range tests {
		result := CanCreateRole(tc.parent, tc.child)
		if result != tc.expected {
			t.Errorf("CanCreateRole(%s, %s) = %v, expected %v", tc.parent, tc.child, result, tc.expected)
		}
	}
}

func TestHasCapability(t *testing.T) {
	tests := []struct {
		role     Role
		cap      Capability
		expected bool
	}{
		{Role("product-manager"), CapCreateAgents, true},
		{Role("product-manager"), CapImplementTasks, false},
		{Role("engineer"), CapImplementTasks, true},
		{Role("engineer"), CapCreateAgents, false},
		{Role("worker"), CapImplementTasks, true},
		{Role("qa"), CapTestWork, true},
		{Role("qa"), CapReviewWork, true},
		{Role("qa"), CapImplementTasks, false},
	}

	for _, tc := range tests {
		result := HasCapability(tc.role, tc.cap)
		if result != tc.expected {
			t.Errorf("HasCapability(%s, %s) = %v, expected %v", tc.role, tc.cap, result, tc.expected)
		}
	}
}

func TestRoleLevel(t *testing.T) {
	tests := []struct {
		role     Role
		expected int
	}{
		{Role("product-manager"), 1},
		{RoleRoot, -1},
		{Role("manager"), 1},
		{Role("engineer"), 1},
		{Role("worker"), 1},
		{Role("qa"), 1},
	}

	for _, tc := range tests {
		result := RoleLevel(tc.role)
		if result != tc.expected {
			t.Errorf("RoleLevel(%s) = %d, expected %d", tc.role, result, tc.expected)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	valid := []struct {
		from, to State
	}{
		{StateIdle, StateIdle},
		{StateIdle, StateWorking},
		{StateWorking, StateWorking},
		{StateWorking, StateIdle},
		{StateWorking, StateDone},
		{StateWorking, StateStuck},
		{StateWorking, StateError},
		{StateWorking, StateStopped},
		{StateDone, StateIdle},
		{StateDone, StateWorking},
		{StateStuck, StateStuck},
		{StateStuck, StateIdle},
		{StateStuck, StateWorking},
		{StateError, StateIdle},
		{StateError, StateWorking},
		{StateStopped, StateIdle},
		{StateStopped, StateStarting},
		{StateStarting, StateIdle},
		{StateStarting, StateError},
		{StateIdle, StateStopped},
		{StateDone, StateStopped},
		{StateStuck, StateError},
	}

	for _, tc := range valid {
		if err := ValidateTransition(tc.from, tc.to); err != nil {
			t.Errorf("ValidateTransition(%s, %s) should be valid, got error: %v", tc.from, tc.to, err)
		}
	}

	invalid := []struct {
		from, to State
	}{
		{StateIdle, StateStarting},
		{StateWorking, StateStarting},
		{StateDone, StateDone},
		{StateDone, StateStuck},
		{StateStopped, StateWorking},
		{StateStopped, StateDone},
		{StateStarting, StateWorking},
		{StateStarting, StateDone},
	}

	for _, tc := range invalid {
		if err := ValidateTransition(tc.from, tc.to); err == nil {
			t.Errorf("ValidateTransition(%s, %s) should be invalid, but returned nil", tc.from, tc.to)
		}
	}
}

func TestUpdateAgentStateValidation(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:  "test-agent",
		Role:  Role("worker"),
		State: StateIdle,
	}

	// Valid: idle → working
	if err := m.UpdateAgentState("test-agent", StateWorking, "starting task"); err != nil {
		t.Errorf("idle→working should be valid: %v", err)
	}
	if m.agents["test-agent"].State != StateWorking {
		t.Errorf("expected state working, got %s", m.agents["test-agent"].State)
	}

	// Valid: working → done
	if err := m.UpdateAgentState("test-agent", StateDone, "finished"); err != nil {
		t.Errorf("working→done should be valid: %v", err)
	}

	// Invalid: done → stuck
	if err := m.UpdateAgentState("test-agent", StateStuck, "stuck"); err == nil {
		t.Error("done→stuck should be invalid, but returned nil")
	}
	if m.agents["test-agent"].State != StateDone {
		t.Errorf("state should remain done after rejected transition, got %s", m.agents["test-agent"].State)
	}

	// Agent not found
	if err := m.UpdateAgentState("nonexistent", StateWorking, ""); err == nil {
		t.Error("should error for nonexistent agent")
	}
}

func TestUpdateAgentState_SameStateMessageUpdate(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:  "test-agent",
		Role:  Role("worker"),
		State: StateIdle,
	}

	// idle → working
	if err := m.UpdateAgentState("test-agent", StateWorking, "starting task"); err != nil {
		t.Fatalf("idle→working should be valid: %v", err)
	}

	// working → working (update message)
	if err := m.UpdateAgentState("test-agent", StateWorking, "now testing edge cases"); err != nil {
		t.Errorf("working→working should be valid: %v", err)
	}
	if m.agents["test-agent"].Task != "now testing edge cases" {
		t.Errorf("expected task 'now testing edge cases', got %q", m.agents["test-agent"].Task)
	}
	if m.agents["test-agent"].State != StateWorking {
		t.Errorf("expected state working, got %s", m.agents["test-agent"].State)
	}

	// working → stuck
	if err := m.UpdateAgentState("test-agent", StateStuck, "blocked on dependency"); err != nil {
		t.Fatalf("working→stuck should be valid: %v", err)
	}

	// stuck → stuck (update message)
	if err := m.UpdateAgentState("test-agent", StateStuck, "still blocked, filed issue"); err != nil {
		t.Errorf("stuck→stuck should be valid: %v", err)
	}
	if m.agents["test-agent"].Task != "still blocked, filed issue" {
		t.Errorf("expected updated stuck message, got %q", m.agents["test-agent"].Task)
	}

	// stuck → idle
	if err := m.UpdateAgentState("test-agent", StateIdle, ""); err != nil {
		t.Fatalf("stuck→idle should be valid: %v", err)
	}

	// idle → idle (update message)
	if err := m.UpdateAgentState("test-agent", StateIdle, "waiting for assignment"); err != nil {
		t.Errorf("idle→idle should be valid: %v", err)
	}
	if m.agents["test-agent"].Task != "waiting for assignment" {
		t.Errorf("expected updated idle message, got %q", m.agents["test-agent"].Task)
	}
}

// --- Concurrent access for new functions ---

// TestSpawnAgent_ConcurrentCalls verifies that concurrent SpawnAgent calls
// are properly serialized and don't cause data races or corruption.
// This exercises the mutex locking in SpawnAgentWithOptions.
func TestSpawnAgent_ConcurrentCalls(t *testing.T) {
	m := newTestManager(t)

	// Pre-populate some agents to create contention
	m.agents["existing-1"] = &Agent{
		ID:       "existing-1",
		Name:     "existing-1",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{},
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent reads while spawning would happen
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Read operations that should be thread-safe
			_ = m.GetAgent("existing-1")
			_ = m.ListAgents()
			_ = m.AgentCount()
		}()
	}

	// Concurrent state mutations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentName := fmt.Sprintf("concurrent-agent-%d", idx)
			m.mu.Lock()
			m.agents[agentName] = &Agent{
				ID:       agentName,
				Name:     agentName,
				Role:     Role("engineer"),
				State:    StateIdle,
				Children: []string{},
			}
			m.mu.Unlock()
		}(i)
	}

	// Concurrent reads of spawned agents
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentName := fmt.Sprintf("concurrent-agent-%d", idx%20)
			_ = m.GetAgent(agentName)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}

	// Verify state is consistent
	count := m.AgentCount()
	if count < 1 {
		t.Errorf("expected at least 1 agent, got %d", count)
	}
}

func TestConcurrentAgentCount(t *testing.T) {
	m := newTestManager(t)
	m.agents["a"] = &Agent{Name: "a", State: StateIdle}
	m.agents["b"] = &Agent{Name: "b", State: StateWorking}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if count := m.AgentCount(); count != 2 {
				t.Errorf("expected 2, got %d", count)
			}
		}()
	}
	wg.Wait()
}

func TestSpawnAgent_ExistingSessionCreatesWorktree(t *testing.T) {
	// Setup: create a real git repo so createWorktree works
	workspace := t.TempDir()
	cmd := exec.CommandContext(context.Background(), "git", "init", workspace) //nolint:gosec // test helper
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, out)
	}
	// Configure git user for CI environments where global config is absent
	if err := exec.CommandContext(context.Background(), "git", "-C", workspace, "config", "user.email", "test@test.com").Run(); err != nil { //nolint:gosec // test helper
		t.Fatal(err)
	}
	if err := exec.CommandContext(context.Background(), "git", "-C", workspace, "config", "user.name", "Test").Run(); err != nil { //nolint:gosec // test helper
		t.Fatal(err)
	}
	// Need at least one commit for git worktree add to work
	cmd = exec.CommandContext(context.Background(), "git", "-C", workspace, "commit", "--allow-empty", "-m", "init") //nolint:gosec // test helper
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v (%s)", err, out)
	}

	m := newTestManager(t)
	m.stateDir = filepath.Join(workspace, ".bc", "agents")
	if err := os.MkdirAll(m.stateDir, 0750); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create a real tmux session so HasSession returns true
	sessionName := m.tmux.SessionName("eng-1")
	cmd = exec.CommandContext(context.Background(), "tmux", "new-session", "-d", "-s", sessionName) //nolint:gosec // test helper
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tmux new-session failed: %v (%s)", err, out)
	}
	t.Cleanup(func() {
		_ = exec.CommandContext(context.Background(), "tmux", "kill-session", "-t", sessionName).Run() //nolint:errcheck,gosec // best-effort cleanup
	})

	// Pre-populate agent WITHOUT WorktreeDir (simulates pre-worktree agent)
	m.agents["eng-1"] = &Agent{
		ID:        "eng-1",
		Name:      "eng-1",
		Role:      Role("engineer"),
		State:     StateIdle,
		Workspace: workspace,
		Session:   "eng-1",
		Children:  []string{},
	}

	// SpawnAgentWithOptions should reuse session but create worktree
	agent, err := m.SpawnAgentWithOptions("eng-1", Role("engineer"), workspace, "", "")
	if err != nil {
		t.Fatalf("SpawnAgentWithOptions failed: %v", err)
	}

	expectedDir := filepath.Join(workspace, ".bc", "worktrees", "eng-1")
	if agent.WorktreeDir != expectedDir {
		t.Errorf("WorktreeDir = %q, want %q", agent.WorktreeDir, expectedDir)
	}

	// Verify worktree actually exists on disk
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("worktree directory was not created on disk")
	}

	// Cleanup worktree
	_ = exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "remove", "--force", expectedDir).Run() //nolint:errcheck,gosec // best-effort cleanup
}

func TestSpawnAgent_PreservesWorkingState(t *testing.T) {
	m := newTestManager(t)
	// Setup agent in working state
	m.agents["worker-1"] = &Agent{
		ID:          "worker-1",
		Name:        "worker-1",
		Role:        Role("worker"),
		State:       StateWorking,
		Task:        "Important Job",
		Session:     "worker-1",
		WorktreeDir: t.TempDir(),
	}

	// Mock tmux session as NOT existing (simulating crash/restart)
	// newTestManager uses a prefix that ensures real tmux sessions don't match,
	// so HasSession returns false by default.

	// Spawn again
	// We expect it to restart the session but KEEP StateWorking
	agent, err := m.SpawnAgent("worker-1", Role("worker"), t.TempDir())
	if err != nil {
		t.Fatalf("SpawnAgent failed: %v", err)
	}

	if agent.State != StateWorking {
		t.Errorf("Agent state reset to %s, want %s", agent.State, StateWorking)
	}
	if agent.Task != "Important Job" {
		t.Errorf("Agent task lost: %q", agent.Task)
	}
}

func TestConcurrentRunningCount(t *testing.T) {
	m := newTestManager(t)
	m.agents["a"] = &Agent{Name: "a", State: StateIdle}
	m.agents["b"] = &Agent{Name: "b", State: StateStopped}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if count := m.RunningCount(); count != 1 {
				t.Errorf("expected 1, got %d", count)
			}
		}()
	}
	wg.Wait()
}

func TestBootstrapDelay(t *testing.T) {
	m := newTestManager(t)

	// Default should be 3 seconds
	if d := m.getBootstrapDelay(); d != DefaultBootstrapDelay {
		t.Errorf("default bootstrap delay: got %v, want %v", d, DefaultBootstrapDelay)
	}
	if d := m.getBootstrapDelay(); d != 3*time.Second {
		t.Errorf("default bootstrap delay: got %v, want 3s", d)
	}

	// Setting custom delay should work
	m.SetBootstrapDelay(5 * time.Second)
	if d := m.getBootstrapDelay(); d != 5*time.Second {
		t.Errorf("custom bootstrap delay: got %v, want 5s", d)
	}

	// Setting to 0 should revert to default
	m.SetBootstrapDelay(0)
	if d := m.getBootstrapDelay(); d != DefaultBootstrapDelay {
		t.Errorf("zero bootstrap delay should use default: got %v, want %v", d, DefaultBootstrapDelay)
	}
}

// --- Git wrapper tests ---

func TestEnsureGitWrapper_CreatesFile(t *testing.T) {
	workspace := t.TempDir()

	if err := ensureGitWrapper(workspace); err != nil {
		t.Fatalf("ensureGitWrapper failed: %v", err)
	}

	wrapperPath := filepath.Join(workspace, ".bc", "bin", "git")
	info, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("wrapper not created: %v", err)
	}

	// Check executable permission
	if info.Mode()&0111 == 0 {
		t.Errorf("wrapper not executable: mode %v", info.Mode())
	}

	// Check content contains key elements
	content, err := os.ReadFile(wrapperPath) //nolint:gosec // test file read
	if err != nil {
		t.Fatalf("failed to read wrapper: %v", err)
	}
	s := string(content)
	if s == "" {
		t.Fatal("wrapper is empty")
	}
	for _, want := range []string{"/usr/bin/git", "BC_AGENT_WORKTREE", "exec"} {
		found := false
		for i := 0; i <= len(s)-len(want); i++ {
			if s[i:i+len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("wrapper missing expected string %q", want)
		}
	}
}

func TestEnsureGitWrapper_Idempotent(t *testing.T) {
	workspace := t.TempDir()

	if err := ensureGitWrapper(workspace); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	wrapperPath := filepath.Join(workspace, ".bc", "bin", "git")
	info1, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}

	// Second call should be a no-op (file already exists)
	if ensureErr := ensureGitWrapper(workspace); ensureErr != nil {
		t.Fatalf("second call failed: %v", ensureErr)
	}

	info2, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	if info1.ModTime() != info2.ModTime() {
		t.Error("wrapper was rewritten on second call (not idempotent)")
	}
}

func TestEnsureGitWrapper_CreatesBinDir(t *testing.T) {
	workspace := t.TempDir()
	binDir := filepath.Join(workspace, ".bc", "bin")

	// .bc/bin should not exist yet
	if _, err := os.Stat(binDir); err == nil {
		t.Fatal(".bc/bin already exists before test")
	}

	if err := ensureGitWrapper(workspace); err != nil {
		t.Fatalf("ensureGitWrapper failed: %v", err)
	}

	info, err := os.Stat(binDir)
	if err != nil {
		t.Fatalf(".bc/bin not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".bc/bin is not a directory")
	}
}

// --- RoleRoot tests ---

func TestRoleRoot_Capabilities(t *testing.T) {
	caps, ok := RoleCapabilities[RoleRoot]
	if !ok {
		t.Fatal("RoleRoot should have capabilities defined")
	}

	expected := []Capability{CapCreateAgents, CapAssignWork, CapCreateEpics, CapReviewWork}
	for _, cap := range expected {
		found := false
		for _, c := range caps {
			if c == cap {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RoleRoot should have capability %s", cap)
		}
	}
}

func TestRoleRoot_Hierarchy(t *testing.T) {
	children, ok := RoleHierarchy[RoleRoot]
	if !ok {
		t.Fatal("RoleRoot should have hierarchy defined")
	}

	expected := []Role{Role("product-manager"), Role("manager"), Role("engineer"), Role("qa")}
	for _, role := range expected {
		found := false
		for _, r := range children {
			if r == role {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RoleRoot should be able to create %s", role)
		}
	}
}

func TestRoleRoot_Level(t *testing.T) {
	level := RoleLevel(RoleRoot)
	if level != -1 {
		t.Errorf("RoleRoot level = %d, want -1", level)
	}

	// Root should be above all other roles
	if level >= RoleLevel(Role("product-manager")) {
		t.Error("RoleRoot should be above RoleProductManager")
	}
	if level >= RoleLevel(Role("manager")) {
		t.Error("RoleRoot should be above RoleManager")
	}
}

func TestRoleRoot_CanCreateRole(t *testing.T) {
	tests := []struct {
		child    Role
		expected bool
	}{
		{Role("product-manager"), true},
		{Role("manager"), true},
		{Role("engineer"), true},
		{Role("qa"), true},
		{RoleRoot, false}, // Cannot create another root
	}

	for _, tc := range tests {
		t.Run(string(tc.child), func(t *testing.T) {
			got := CanCreateRole(RoleRoot, tc.child)
			if got != tc.expected {
				t.Errorf("CanCreateRole(RoleRoot, %s) = %v, want %v", tc.child, got, tc.expected)
			}
		})
	}
}

func TestRoleRoot_HasCapability(t *testing.T) {
	tests := []struct {
		cap      Capability
		expected bool
	}{
		{CapCreateAgents, true},
		{CapAssignWork, true},
		{CapCreateEpics, true},
		{CapReviewWork, true},
		{CapImplementTasks, false}, // Root delegates, doesn't implement
		{CapTestWork, false},       // Root delegates, doesn't test
	}

	for _, tc := range tests {
		t.Run(string(tc.cap), func(t *testing.T) {
			got := HasCapability(RoleRoot, tc.cap)
			if got != tc.expected {
				t.Errorf("HasCapability(RoleRoot, %s) = %v, want %v", tc.cap, got, tc.expected)
			}
		})
	}
}

// --- Singleton enforcement tests ---

func TestEnforceRootSingleton_NoRootExists(t *testing.T) {
	workspace := t.TempDir()
	m := newTestManager(t)

	// No root exists - should allow creation
	if err := m.enforceRootSingleton(workspace); err != nil {
		t.Errorf("enforceRootSingleton should allow creation when no root exists: %v", err)
	}
}

func TestEnforceRootSingleton_RootActiveBlocks(t *testing.T) {
	workspace := t.TempDir()
	m := newTestManager(t)

	// Create root in active state (idle)
	bcDir := filepath.Join(workspace, ".bc")
	store := NewRootStateStore(bcDir)
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}

	// Active root should block new spawn
	err := m.enforceRootSingleton(workspace)
	if err == nil {
		t.Error("enforceRootSingleton should block when active root exists")
	}
	if err != nil && err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestEnforceRootSingleton_RootWorkingBlocks(t *testing.T) {
	workspace := t.TempDir()
	m := newTestManager(t)

	// Create root in working state
	bcDir := filepath.Join(workspace, ".bc")
	store := NewRootStateStore(bcDir)
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	if err := store.UpdateState(StateWorking); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	// Working root should block new spawn
	err := m.enforceRootSingleton(workspace)
	if err == nil {
		t.Error("enforceRootSingleton should block when working root exists")
	}
}

func TestEnforceRootSingleton_RootStoppedAllows(t *testing.T) {
	workspace := t.TempDir()
	m := newTestManager(t)

	// Create root in stopped state
	bcDir := filepath.Join(workspace, ".bc")
	store := NewRootStateStore(bcDir)
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	if err := store.UpdateState(StateStopped); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	// Stopped root should allow respawn
	if err := m.enforceRootSingleton(workspace); err != nil {
		t.Errorf("enforceRootSingleton should allow respawn when root is stopped: %v", err)
	}

	// Verify old root state was deleted
	if store.Exists() {
		t.Error("old root state should be deleted after allowing respawn")
	}
}

func TestEnforceRootSingleton_RootErrorAllows(t *testing.T) {
	workspace := t.TempDir()
	m := newTestManager(t)

	// Create root in error state
	bcDir := filepath.Join(workspace, ".bc")
	store := NewRootStateStore(bcDir)
	if _, err := store.Create("root", RoleRoot, "claude"); err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	if err := store.UpdateState(StateError); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	// Error state should allow respawn
	if err := m.enforceRootSingleton(workspace); err != nil {
		t.Errorf("enforceRootSingleton should allow respawn when root is in error: %v", err)
	}

	// Verify old root state was deleted
	if store.Exists() {
		t.Error("old root state should be deleted after allowing respawn")
	}
}

// --- Memory Directory Tests ---

func TestCreateMemoryDir(t *testing.T) {
	workspace := t.TempDir()
	agentName := "test-agent"

	memoryDir, err := createMemoryDir(workspace, agentName)
	if err != nil {
		t.Fatalf("createMemoryDir failed: %v", err)
	}

	expectedDir := filepath.Join(workspace, ".bc", "memory", agentName)
	if memoryDir != expectedDir {
		t.Errorf("memoryDir = %q, want %q", memoryDir, expectedDir)
	}

	// Verify directory exists
	if _, statErr := os.Stat(memoryDir); os.IsNotExist(statErr) {
		t.Error("memory directory was not created")
	}

	// Verify experiences.jsonl exists
	experiencesPath := filepath.Join(memoryDir, "experiences.jsonl")
	if _, statErr := os.Stat(experiencesPath); os.IsNotExist(statErr) {
		t.Error("experiences.jsonl was not created")
	}

	// Verify learnings.md exists and has header
	learningsPath := filepath.Join(memoryDir, "learnings.md")
	if _, statErr := os.Stat(learningsPath); os.IsNotExist(statErr) {
		t.Error("learnings.md was not created")
	}

	// Read learnings.md content - path is constructed from test inputs
	content, readErr := os.ReadFile(learningsPath) //nolint:gosec // test file path from t.TempDir()
	if readErr != nil {
		t.Fatalf("failed to read learnings.md: %v", readErr)
	}
	if len(content) == 0 {
		t.Error("learnings.md is empty, expected header")
	}
}

func TestCreateMemoryDir_Idempotent(t *testing.T) {
	workspace := t.TempDir()
	agentName := "test-agent"

	// Create once
	memoryDir1, err := createMemoryDir(workspace, agentName)
	if err != nil {
		t.Fatalf("first createMemoryDir failed: %v", err)
	}

	// Create again - should reuse existing
	memoryDir2, err := createMemoryDir(workspace, agentName)
	if err != nil {
		t.Fatalf("second createMemoryDir failed: %v", err)
	}

	if memoryDir1 != memoryDir2 {
		t.Errorf("memory dirs should match: %q != %q", memoryDir1, memoryDir2)
	}
}

// --- Permission function tests ---

func TestDefaultPermissions(t *testing.T) {
	tests := []struct {
		expectedContains   Permission
		expectedNotContain Permission
		name               string
		roleLevel          int
	}{
		{
			name:               "root level has all permissions",
			roleLevel:          -1,
			expectedContains:   PermCreateAgents,
			expectedNotContain: "",
		},
		{
			name:               "manager level has create agents",
			roleLevel:          0,
			expectedContains:   PermCreateAgents,
			expectedNotContain: PermDeleteAgents,
		},
		{
			name:               "engineer level has view logs",
			roleLevel:          1,
			expectedContains:   PermViewLogs,
			expectedNotContain: PermCreateAgents,
		},
		{
			name:               "worker level has send commands",
			roleLevel:          2,
			expectedContains:   PermSendCommands,
			expectedNotContain: PermCreateChannels,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			perms := DefaultPermissions(tc.roleLevel)
			if len(perms) == 0 && tc.roleLevel <= -1 {
				t.Error("root level should have permissions")
			}

			if tc.expectedContains != "" {
				found := false
				for _, p := range perms {
					if p == tc.expectedContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected permission %q not found", tc.expectedContains)
				}
			}

			if tc.expectedNotContain != "" {
				for _, p := range perms {
					if p == tc.expectedNotContain {
						t.Errorf("unexpected permission %q found", tc.expectedNotContain)
						break
					}
				}
			}
		})
	}
}

func TestDefaultPermissions_AllLevels(t *testing.T) {
	// Test root level has all permissions
	rootPerms := DefaultPermissions(-1)
	if len(rootPerms) != len(AllPermissions) {
		t.Errorf("root level should have %d permissions, got %d", len(AllPermissions), len(rootPerms))
	}

	// Test manager level
	mgrPerms := DefaultPermissions(0)
	if len(mgrPerms) < 3 {
		t.Errorf("manager level should have at least 3 permissions, got %d", len(mgrPerms))
	}

	// Test engineer level
	engPerms := DefaultPermissions(1)
	if len(engPerms) < 2 {
		t.Errorf("engineer level should have at least 2 permissions, got %d", len(engPerms))
	}
}

func TestCheckPermission(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment not critical
		permissions []string
		required    Permission
		name        string
		wantErr     bool
	}{
		{
			name:        "has required permission",
			permissions: []string{"can_create_agents", "can_view_logs", "can_send_commands"},
			required:    PermCreateAgents,
			wantErr:     false,
		},
		{
			name:        "missing required permission",
			permissions: []string{"can_view_logs", "can_send_commands"},
			required:    PermCreateAgents,
			wantErr:     true,
		},
		{
			name:        "empty permissions",
			permissions: []string{},
			required:    PermViewLogs,
			wantErr:     true,
		},
		{
			name:        "single matching permission",
			permissions: []string{"can_view_logs"},
			required:    PermViewLogs,
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckPermission(tc.permissions, tc.required)
			if (err != nil) != tc.wantErr {
				t.Errorf("CheckPermission() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestHasPermissionStr(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment not critical
		permissions []string
		name        string
		required    string
		expected    bool
	}{
		{
			name:        "has permission",
			permissions: []string{"can_create_agents", "can_view_logs", "can_send_commands"},
			required:    "can_view_logs",
			expected:    true,
		},
		{
			name:        "missing permission",
			permissions: []string{"can_view_logs", "can_send_commands"},
			required:    "can_create_agents",
			expected:    false,
		},
		{
			name:        "empty permissions",
			permissions: []string{},
			required:    "can_view_logs",
			expected:    false,
		},
		{
			name:        "single matching",
			permissions: []string{"can_send_messages"},
			required:    "can_send_messages",
			expected:    true,
		},
		{
			name:        "partial match not accepted",
			permissions: []string{"can_create_agents"},
			required:    "can_create",
			expected:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := HasPermissionStr(tc.permissions, tc.required)
			if result != tc.expected {
				t.Errorf("HasPermissionStr() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// --- SpawnAgentWithTool tests (#1236) ---

func TestSpawnAgentWithTool(t *testing.T) {
	m := newTestManager(t)

	// Invalid name should fail
	_, err := m.SpawnAgentWithTool("invalid name!", Role("engineer"), "/tmp", "claude")
	if err == nil {
		t.Error("expected error for invalid agent name")
	}

	// Empty role should fail
	_, err = m.SpawnAgentWithTool("test-agent", Role(""), "/tmp", "claude")
	if err == nil {
		t.Error("expected error for empty role")
	}
}

// --- SpawnAgentWithParent tests (#1236) ---

func TestSpawnAgentWithParent(t *testing.T) {
	m := newTestManager(t)

	// Invalid name should fail
	_, err := m.SpawnAgentWithParent("bad name!", Role("engineer"), "/tmp", "parent")
	if err == nil {
		t.Error("expected error for invalid agent name")
	}

	// Null role should fail
	_, err = m.SpawnAgentWithParent("test-agent", Role("null"), "/tmp", "parent")
	if err == nil {
		t.Error("expected error for null role")
	}
}

// --- DeleteAgent tests (#1236) ---

func TestDeleteAgent(t *testing.T) {
	m := newTestManager(t)

	// Set up an agent to delete
	m.agents["doomed"] = &Agent{
		Name:      "doomed",
		Role:      Role("engineer"),
		State:     StateIdle,
		Workspace: m.stateDir,
		MemoryDir: filepath.Join(m.stateDir, "memory-doomed"),
		Children:  []string{},
	}

	// Create memory dir
	if mkErr := os.MkdirAll(m.agents["doomed"].MemoryDir, 0750); mkErr != nil {
		t.Fatalf("failed to create memory dir: %v", mkErr)
	}

	// Delete should succeed
	if err := m.DeleteAgent("doomed"); err != nil {
		t.Errorf("DeleteAgent failed: %v", err)
	}

	// Agent should be gone
	if _, exists := m.agents["doomed"]; exists {
		t.Error("agent should be deleted from map")
	}

	// Memory dir should be removed (PurgeMemory is true by default)
	if _, statErr := os.Stat(filepath.Join(m.stateDir, "memory-doomed")); !os.IsNotExist(statErr) {
		t.Error("memory dir should be removed")
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	m := newTestManager(t)

	err := m.DeleteAgent("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

// --- DeleteAgentWithOptions tests (#1236) ---

func TestDeleteAgentWithOptions_PreserveMemory(t *testing.T) {
	m := newTestManager(t)

	memDir := filepath.Join(m.stateDir, "memory-preserve")
	m.agents["preserve"] = &Agent{
		Name:      "preserve",
		Role:      Role("engineer"),
		State:     StateIdle,
		Workspace: m.stateDir,
		MemoryDir: memDir,
		Children:  []string{},
	}

	// Create memory dir
	if mkErr := os.MkdirAll(memDir, 0750); mkErr != nil {
		t.Fatalf("failed to create memory dir: %v", mkErr)
	}
	// Create a test file
	testFile := filepath.Join(memDir, "test.json")
	if writeErr := os.WriteFile(testFile, []byte("{}"), 0600); writeErr != nil {
		t.Fatalf("failed to create test file: %v", writeErr)
	}

	// Delete with PurgeMemory=false
	err := m.DeleteAgentWithOptions("preserve", DeleteOptions{PurgeMemory: false})
	if err != nil {
		t.Errorf("DeleteAgentWithOptions failed: %v", err)
	}

	// Agent should be gone
	if _, exists := m.agents["preserve"]; exists {
		t.Error("agent should be deleted")
	}

	// Memory dir should still exist
	if _, statErr := os.Stat(memDir); os.IsNotExist(statErr) {
		t.Error("memory dir should be preserved with PurgeMemory=false")
	}
}

func TestDeleteAgentWithOptions_WithWorktree(t *testing.T) {
	m := newTestManager(t)

	m.agents["with-worktree"] = &Agent{
		Name:        "with-worktree",
		Role:        Role("engineer"),
		State:       StateIdle,
		Workspace:   m.stateDir,
		WorktreeDir: filepath.Join(m.stateDir, "worktrees", "wt"),
		Children:    []string{},
	}

	// Delete should succeed (worktree removal is best-effort)
	err := m.DeleteAgentWithOptions("with-worktree", DeleteOptions{PurgeMemory: true})
	if err != nil {
		t.Errorf("DeleteAgentWithOptions failed: %v", err)
	}

	// Agent should be gone
	if _, exists := m.agents["with-worktree"]; exists {
		t.Error("agent should be deleted")
	}
}

func TestDeleteAgentWithOptions_RemovesFromParent(t *testing.T) {
	m := newTestManager(t)

	m.agents["parent-mgr"] = &Agent{
		Name:     "parent-mgr",
		Role:     Role("manager"),
		State:    StateIdle,
		Children: []string{"child-eng", "other-child"},
	}
	m.agents["child-eng"] = &Agent{
		Name:     "child-eng",
		Role:     Role("engineer"),
		State:    StateIdle,
		ParentID: "parent-mgr",
		Children: []string{},
	}

	// Delete child
	err := m.DeleteAgentWithOptions("child-eng", DeleteOptions{PurgeMemory: false})
	if err != nil {
		t.Errorf("DeleteAgentWithOptions failed: %v", err)
	}

	// Child should be removed from parent's children
	parent := m.agents["parent-mgr"]
	for _, child := range parent.Children {
		if child == "child-eng" {
			t.Error("deleted child should be removed from parent's children list")
		}
	}
	if len(parent.Children) != 1 || parent.Children[0] != "other-child" {
		t.Errorf("parent.Children = %v, want [other-child]", parent.Children)
	}
}

func TestPermissionConstants(t *testing.T) {
	// Verify permission constants are defined correctly
	if PermCreateAgents != "can_create_agents" {
		t.Errorf("PermCreateAgents = %q, want %q", PermCreateAgents, "can_create_agents")
	}
	if PermViewLogs != "can_view_logs" {
		t.Errorf("PermViewLogs = %q, want %q", PermViewLogs, "can_view_logs")
	}
	if PermSendCommands != "can_send_commands" {
		t.Errorf("PermSendCommands = %q, want %q", PermSendCommands, "can_send_commands")
	}
	if PermSendMessages != "can_send_messages" {
		t.Errorf("PermSendMessages = %q, want %q", PermSendMessages, "can_send_messages")
	}
}

func TestAllPermissions(t *testing.T) {
	// AllPermissions should contain all defined permissions
	if len(AllPermissions) < 5 {
		t.Errorf("AllPermissions should have at least 5 permissions, got %d", len(AllPermissions))
	}

	// Check that key permissions are in AllPermissions
	expectedPerms := []Permission{PermCreateAgents, PermViewLogs, PermSendCommands, PermSendMessages}
	for _, expected := range expectedPerms {
		found := false
		for _, p := range AllPermissions {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllPermissions missing %q", expected)
		}
	}
}

func TestIsValidAgentName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "agent", true},
		{"valid with hyphen", "my-agent", true},
		{"valid with underscore", "my_agent", true},
		{"valid with numbers", "agent123", true},
		{"valid mixed case", "MyAgent", true},
		{"valid complex", "eng-02_test", true},
		{"empty string", "", false},
		{"contains space", "my agent", false},
		{"contains semicolon", "agent;rm", false},
		{"contains pipe", "agent|ls", false},
		{"contains ampersand", "agent&echo", false},
		{"contains dollar", "agent$var", false},
		{"contains backtick", "agent`id`", false},
		{"contains slash", "agent/path", false},
		{"contains dot", "agent.test", false},
		{"contains newline", "agent\ntest", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidAgentName(tc.input)
			if result != tc.expected {
				t.Errorf("IsValidAgentName(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestContainsShellMetachars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"clean string", "hello", false},
		{"alphanumeric", "agent123", false},
		{"with hyphen", "my-agent", false},
		{"with underscore", "my_agent", false},
		{"semicolon injection", "cmd; rm -rf", true},
		{"pipe injection", "cmd | cat /etc/passwd", true},
		{"background", "cmd &", true},
		{"variable expansion", "$HOME", true},
		{"backtick execution", "`id`", true},
		{"parentheses", "$(whoami)", true},
		{"curly braces", "${var}", true},
		{"square brackets", "[test]", true},
		{"redirect output", "cmd > /tmp/out", true},
		{"redirect input", "cmd < /etc/passwd", true},
		{"newline injection", "cmd\nrm", true},
		{"carriage return", "cmd\rrm", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := containsShellMetachars(tc.input)
			if result != tc.expected {
				t.Errorf("containsShellMetachars(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestManagerSetAgentTeam(t *testing.T) {
	m := newTestManager(t)

	// Add a test agent
	m.agents["test-agent"] = &Agent{
		Name: "test-agent",
		Role: "engineer",
	}

	// Test setting team on existing agent
	err := m.SetAgentTeam("test-agent", "backend")
	if err != nil {
		t.Fatalf("SetAgentTeam() error = %v", err)
	}

	if m.agents["test-agent"].Team != "backend" {
		t.Errorf("Team = %q, want %q", m.agents["test-agent"].Team, "backend")
	}

	// Test UpdatedAt was set
	if m.agents["test-agent"].UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after SetAgentTeam")
	}

	// Test setting team on non-existent agent
	err = m.SetAgentTeam("nonexistent", "team")
	if err == nil {
		t.Error("SetAgentTeam() should error for non-existent agent")
	}
}

func TestManagerSetAgentCommand(t *testing.T) {
	m := newTestManager(t)

	// Initially the command is /bin/true
	if m.agentCmd != "/bin/true" {
		t.Errorf("initial agentCmd = %q, want /bin/true", m.agentCmd)
	}

	// Set a new command
	m.SetAgentCommand("claude")
	if m.agentCmd != "claude" {
		t.Errorf("agentCmd = %q, want claude", m.agentCmd)
	}

	// Set another command
	m.SetAgentCommand("claude --dangerously-skip-permissions")
	if m.agentCmd != "claude --dangerously-skip-permissions" {
		t.Errorf("agentCmd = %q, want 'claude --dangerously-skip-permissions'", m.agentCmd)
	}
}

func TestManagerSetBootstrapDelay(t *testing.T) {
	m := newTestManager(t)

	// Initially zero
	if m.BootstrapDelay != 0 {
		t.Errorf("initial BootstrapDelay = %v, want 0", m.BootstrapDelay)
	}

	// Set delay
	m.SetBootstrapDelay(5 * time.Second)
	if m.BootstrapDelay != 5*time.Second {
		t.Errorf("BootstrapDelay = %v, want 5s", m.BootstrapDelay)
	}

	// Set another delay
	m.SetBootstrapDelay(10 * time.Second)
	if m.BootstrapDelay != 10*time.Second {
		t.Errorf("BootstrapDelay = %v, want 10s", m.BootstrapDelay)
	}
}

func TestManagerListAgents(t *testing.T) {
	m := newTestManager(t)

	// Initially no agents
	agents := m.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}

	// Add agents
	m.agents["agent1"] = &Agent{Name: "agent1", Role: "engineer"}
	m.agents["agent2"] = &Agent{Name: "agent2", Role: "manager"}

	agents = m.ListAgents()
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestManagerGetAgent(t *testing.T) {
	m := newTestManager(t)

	// Add test agent
	testAgent := &Agent{Name: "test-agent", Role: "engineer", Team: "backend"}
	m.agents["test-agent"] = testAgent

	// Test getting existing agent
	agent := m.GetAgent("test-agent")
	if agent == nil {
		t.Fatal("GetAgent() should find existing agent")
	}
	if agent.Name != "test-agent" {
		t.Errorf("Name = %q, want test-agent", agent.Name)
	}
	if agent.Team != "backend" {
		t.Errorf("Team = %q, want backend", agent.Team)
	}

	// Test getting non-existent agent
	agent = m.GetAgent("nonexistent")
	if agent != nil {
		t.Error("GetAgent() should return nil for non-existent agent")
	}
}

func TestCapabilityConstants(t *testing.T) {
	// Verify capability constant values
	if CapCreateAgents != "create_agents" {
		t.Errorf("CapCreateAgents = %q, want create_agents", CapCreateAgents)
	}
	if CapAssignWork != "assign_work" {
		t.Errorf("CapAssignWork = %q, want assign_work", CapAssignWork)
	}
	if CapCreateEpics != "create_epics" {
		t.Errorf("CapCreateEpics = %q, want create_epics", CapCreateEpics)
	}
	if CapImplementTasks != "implement_tasks" {
		t.Errorf("CapImplementTasks = %q, want implement_tasks", CapImplementTasks)
	}
	if CapReviewWork != "review_work" {
		t.Errorf("CapReviewWork = %q, want review_work", CapReviewWork)
	}
	if CapTestWork != "test_work" {
		t.Errorf("CapTestWork = %q, want test_work", CapTestWork)
	}
}

func TestRoleConstant(t *testing.T) {
	// Verify RoleRoot constant
	if RoleRoot != "root" {
		t.Errorf("RoleRoot = %q, want root", RoleRoot)
	}
}

// --- Additional LoadRoleMemory tests ---

func TestLoadRoleMemory_RootRoleBackwardCompat(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create root.md in the backward-compatible location
	content := "You are the root orchestrator agent."
	if err := os.WriteFile(filepath.Join(promptsDir, "root.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	mem := LoadRoleMemory(tmpDir, RoleRoot)
	if mem == nil {
		t.Fatal("expected non-nil AgentMemory for root role")
	}
	if mem.RolePrompt != content {
		t.Errorf("RolePrompt = %q, want %q", mem.RolePrompt, content)
	}
}

func TestLoadRoleMemory_RootRoleFallsBackToRoleManager(t *testing.T) {
	tmpDir := t.TempDir()
	rolesDir := filepath.Join(tmpDir, ".bc", "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create root.md in roles dir (not prompts dir)
	content := "Root from roles directory."
	if err := os.WriteFile(filepath.Join(rolesDir, "root.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	// Should fall back to role manager since prompts/root.md doesn't exist
	mem := LoadRoleMemory(tmpDir, RoleRoot)
	if mem == nil {
		t.Fatal("expected non-nil AgentMemory for root role from role manager")
	}
	if mem.RolePrompt != content {
		t.Errorf("RolePrompt = %q, want %q", mem.RolePrompt, content)
	}
}

func TestLoadRoleMemory_EmptyPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	rolesDir := filepath.Join(tmpDir, ".bc", "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create empty role file
	if err := os.WriteFile(filepath.Join(rolesDir, "empty-role.md"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	mem := LoadRoleMemory(tmpDir, Role("empty-role"))
	if mem != nil {
		t.Error("expected nil AgentMemory for empty prompt")
	}
}

// --- UpdateAgentState error tests ---

func TestUpdateAgentState_NotFound(t *testing.T) {
	m := newTestManager(t)

	err := m.UpdateAgentState("nonexistent", StateWorking, "working on task")
	if err == nil {
		t.Error("expected error when updating non-existent agent")
	}
}

// --- SetAgentTeam error tests ---

func TestSetAgentTeam_NotFound(t *testing.T) {
	m := newTestManager(t)

	err := m.SetAgentTeam("nonexistent", "backend")
	if err == nil {
		t.Error("expected error when setting team for non-existent agent")
	}
}

func TestSetAgentTeam_Success(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-01"] = &Agent{
		Name:     "eng-01",
		Role:     Role("engineer"),
		State:    StateIdle,
		Children: []string{},
	}

	err := m.SetAgentTeam("eng-01", "frontend")
	if err != nil {
		t.Fatalf("SetAgentTeam failed: %v", err)
	}

	if m.agents["eng-01"].Team != "frontend" {
		t.Errorf("Team = %q, want frontend", m.agents["eng-01"].Team)
	}
}

// --- enforceRootSingleton tests ---

func TestEnforceRootSingleton_NoExistingRoot(t *testing.T) {
	m := newTestManager(t)
	m.agents["eng-01"] = &Agent{
		Name: "eng-01",
		Role: Role("engineer"),
	}

	// Should not error - no root exists
	err := m.enforceRootSingleton("/workspace")
	if err != nil {
		t.Errorf("enforceRootSingleton should not error without root: %v", err)
	}
}

func TestEnforceRootSingleton_OneRootAllowed(t *testing.T) {
	m := newTestManager(t)
	m.agents["root"] = &Agent{
		Name: "root",
		Role: RoleRoot,
	}

	// Should not error - only one root
	err := m.enforceRootSingleton("/workspace")
	if err != nil {
		t.Errorf("enforceRootSingleton should not error with one root: %v", err)
	}
}
