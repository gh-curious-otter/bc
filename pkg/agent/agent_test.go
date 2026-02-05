package agent

import (
	"sync"
	"testing"
)

// TestConcurrentSetAgentCommand tests that concurrent SetAgentCommand calls don't race.
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

// TestConcurrentGetAgent tests that concurrent GetAgent calls don't race.
func TestConcurrentGetAgent(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     RoleWorker,
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

// TestConcurrentListAgents tests that concurrent ListAgents calls don't race.
func TestConcurrentListAgents(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["agent1"] = &Agent{Name: "agent1", Role: RoleWorker, State: StateIdle}
	m.agents["agent2"] = &Agent{Name: "agent2", Role: RoleWorker, State: StateWorking}
	m.agents["agent3"] = &Agent{Name: "agent3", Role: RoleCoordinator, State: StateIdle}

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

// TestGetAgentReturnsCopy tests that GetAgent returns a copy, not the original.
func TestGetAgentReturnsCopy(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     RoleWorker,
		State:    StateIdle,
		Children: []string{"child1"},
	}

	// Get a copy
	copy := m.GetAgent("test-agent")

	// Modify the copy
	copy.State = StateWorking
	copy.Children = append(copy.Children, "child2")

	// Verify original is unchanged
	original := m.agents["test-agent"]
	if original.State != StateIdle {
		t.Errorf("original state was modified: expected %s, got %s", StateIdle, original.State)
	}
	if len(original.Children) != 1 {
		t.Errorf("original children was modified: expected 1, got %d", len(original.Children))
	}
}

// TestListAgentsReturnsCopies tests that ListAgents returns copies, not originals.
func TestListAgentsReturnsCopies(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["agent1"] = &Agent{
		Name:     "agent1",
		Role:     RoleWorker,
		State:    StateIdle,
		Children: []string{"child1"},
	}

	// Get copies
	copies := m.ListAgents()
	if len(copies) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(copies))
	}

	// Modify the copy
	copies[0].State = StateWorking
	copies[0].Children = append(copies[0].Children, "child2")

	// Verify original is unchanged
	original := m.agents["agent1"]
	if original.State != StateIdle {
		t.Errorf("original state was modified: expected %s, got %s", StateIdle, original.State)
	}
	if len(original.Children) != 1 {
		t.Errorf("original children was modified: expected 1, got %d", len(original.Children))
	}
}

// TestConcurrentReadWrite tests concurrent reads and writes don't race.
func TestConcurrentReadWrite(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:     "test-agent",
		Role:     RoleWorker,
		State:    StateIdle,
		Children: []string{},
	}

	var wg sync.WaitGroup

	// Readers
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

	// Writers
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

// TestRoleHierarchy tests the role hierarchy functions.
func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		parent   Role
		child    Role
		expected bool
	}{
		{RoleProductManager, RoleManager, true},
		{RoleManager, RoleEngineer, true},
		{RoleManager, RoleQA, true},
		{RoleCoordinator, RoleWorker, true},
		{RoleCoordinator, RoleManager, true},
		{RoleCoordinator, RoleQA, true},
		{RoleEngineer, RoleWorker, false},
		{RoleWorker, RoleEngineer, false},
		{RoleQA, RoleEngineer, false},
	}

	for _, tc := range tests {
		result := CanCreateRole(tc.parent, tc.child)
		if result != tc.expected {
			t.Errorf("CanCreateRole(%s, %s) = %v, expected %v", tc.parent, tc.child, result, tc.expected)
		}
	}
}

// TestHasCapability tests the capability checking function.
func TestHasCapability(t *testing.T) {
	tests := []struct {
		role     Role
		cap      Capability
		expected bool
	}{
		{RoleProductManager, CapCreateAgents, true},
		{RoleProductManager, CapImplementTasks, false},
		{RoleEngineer, CapImplementTasks, true},
		{RoleEngineer, CapCreateAgents, false},
		{RoleWorker, CapImplementTasks, true},
		{RoleQA, CapTestWork, true},
		{RoleQA, CapReviewWork, true},
		{RoleQA, CapImplementTasks, false},
	}

	for _, tc := range tests {
		result := HasCapability(tc.role, tc.cap)
		if result != tc.expected {
			t.Errorf("HasCapability(%s, %s) = %v, expected %v", tc.role, tc.cap, result, tc.expected)
		}
	}
}

// TestRoleLevel tests the role level function.
func TestRoleLevel(t *testing.T) {
	tests := []struct {
		role     Role
		expected int
	}{
		{RoleProductManager, 0},
		{RoleCoordinator, 0},
		{RoleManager, 1},
		{RoleEngineer, 2},
		{RoleWorker, 2},
		{RoleQA, 2},
	}

	for _, tc := range tests {
		result := RoleLevel(tc.role)
		if result != tc.expected {
			t.Errorf("RoleLevel(%s) = %d, expected %d", tc.role, result, tc.expected)
		}
	}
}

// TestValidateTransition tests that valid transitions are allowed and invalid ones rejected.
func TestValidateTransition(t *testing.T) {
	valid := []struct {
		from, to State
	}{
		{StateIdle, StateWorking},
		{StateWorking, StateIdle},
		{StateWorking, StateDone},
		{StateWorking, StateStuck},
		{StateWorking, StateError},
		{StateWorking, StateStopped},
		{StateDone, StateIdle},
		{StateDone, StateWorking},
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
		{StateIdle, StateIdle},
		{StateIdle, StateStarting},
		{StateWorking, StateWorking},
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

// TestUpdateAgentStateValidation tests that UpdateAgentState rejects invalid transitions.
func TestUpdateAgentStateValidation(t *testing.T) {
	m := &Manager{
		agents: make(map[string]*Agent),
	}
	m.agents["test-agent"] = &Agent{
		Name:  "test-agent",
		Role:  RoleWorker,
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
	// State should remain done after rejected transition
	if m.agents["test-agent"].State != StateDone {
		t.Errorf("state should remain done after rejected transition, got %s", m.agents["test-agent"].State)
	}

	// Agent not found
	if err := m.UpdateAgentState("nonexistent", StateWorking, ""); err == nil {
		t.Error("should error for nonexistent agent")
	}
}
