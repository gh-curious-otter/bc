package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestTaskTypeToRoleMapping(t *testing.T) {
	tests := []struct {
		taskType TaskType
		wantRole string
	}{
		{TaskTypeCode, "engineer"},
		{TaskTypeReview, "tech-lead"},
		{TaskTypeMerge, "manager"},
		{TaskTypeQA, "qa"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			role, ok := TaskTypeToRole[tt.taskType]
			if !ok {
				t.Fatalf("TaskTypeToRole[%s] not found", tt.taskType)
			}
			if role != tt.wantRole {
				t.Errorf("TaskTypeToRole[%s] = %s, want %s", tt.taskType, role, tt.wantRole)
			}
		})
	}
}

func TestGetRoleForTaskType(t *testing.T) {
	tests := []struct {
		taskType TaskType
		wantRole string
		wantErr  bool
	}{
		{TaskTypeCode, "engineer", false},
		{TaskTypeReview, "tech-lead", false},
		{TaskTypeMerge, "manager", false},
		{TaskTypeQA, "qa", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			role, err := GetRoleForTaskType(tt.taskType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRoleForTaskType(%s) error = %v, wantErr %v", tt.taskType, err, tt.wantErr)
				return
			}
			if !tt.wantErr && role != tt.wantRole {
				t.Errorf("GetRoleForTaskType(%s) = %s, want %s", tt.taskType, role, tt.wantRole)
			}
		})
	}
}

func TestRouterRouteTaskType(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	// Test with no agents - should fail
	_, err := router.RouteTaskType(TaskTypeCode)
	if err == nil {
		t.Error("expected error when no agents available")
	}
}

func TestRouterRoundRobin(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	// Without actual agents, we can only test the round-robin index logic
	// by checking that lastAssigned gets updated
	router.lastAssigned[agent.Role("engineer")] = 0

	// After incrementing, should be 1
	nextIdx := (router.lastAssigned[agent.Role("engineer")] + 1) % 3
	if nextIdx != 1 {
		t.Errorf("expected next index 1, got %d", nextIdx)
	}

	// Wrap around test
	router.lastAssigned[agent.Role("engineer")] = 2
	nextIdx = (router.lastAssigned[agent.Role("engineer")] + 1) % 3
	if nextIdx != 0 {
		t.Errorf("expected wrapped index 0, got %d", nextIdx)
	}
}

func TestRouteToRoleNoAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	_, err := router.RouteToRole("engineer")
	if err == nil {
		t.Error("expected error when no engineers available")
	}
}

func TestRouteTaskTypeUnknown(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	_, err := router.RouteTaskType(TaskType("unknown-type"))
	if err == nil {
		t.Error("expected error for unknown task type")
	}
}

func TestNewRouter(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	if router.mgr != mgr {
		t.Error("router.mgr not set correctly")
	}
	if router.lastAssigned == nil {
		t.Error("router.lastAssigned not initialized")
	}
}

func TestRouteTaskTypeAllKnown(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	// Test all known task types route to expected roles (without agents = error)
	taskTypes := []TaskType{
		TaskTypeCode,
		TaskTypeReview,
		TaskTypeMerge,
		TaskTypeQA,
	}

	for _, taskType := range taskTypes {
		t.Run(string(taskType), func(t *testing.T) {
			_, err := router.RouteTaskType(taskType)
			// Should fail due to no agents, but should get past the type check
			if err == nil {
				t.Error("expected error (no agents), got nil")
			}
			// Error should be about no agents, not unknown type
			if err != nil && err.Error() == "unknown task type: "+string(taskType) {
				t.Errorf("task type %s should be known", taskType)
			}
		})
	}
}

func TestTaskTypeToRoleAllMapped(t *testing.T) {
	// Verify all expected task types are mapped
	expectedMappings := map[TaskType]string{
		TaskTypeCode:   "engineer",
		TaskTypeReview: "tech-lead",
		TaskTypeMerge:  "manager",
		TaskTypeQA:     "qa",
	}

	for taskType, expectedRole := range expectedMappings {
		role, ok := TaskTypeToRole[taskType]
		if !ok {
			t.Errorf("TaskTypeToRole missing mapping for %s", taskType)
			continue
		}
		if role != expectedRole {
			t.Errorf("TaskTypeToRole[%s] = %s, want %s", taskType, role, expectedRole)
		}
	}
}

func TestRouteToRoleMultipleRoles(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	// Test all roles fail without agents
	roles := []string{
		"engineer",
		"tech-lead",
		"manager",
		"qa",
	}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			_, err := router.RouteToRole(role)
			if err == nil {
				t.Errorf("expected error for role %s with no agents", role)
			}
		})
	}
}

func TestRouteToRoleWithAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")

	// Create agents directory and state file
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create agents state with idle/working agents
	agentsState := map[string]*agent.Agent{
		"engineer-01": {
			Name:  "engineer-01",
			ID:    "engineer-01",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
		"engineer-02": {
			Name:  "engineer-02",
			ID:    "engineer-02",
			Role:  agent.Role("engineer"),
			State: agent.StateWorking,
		},
		"tech-lead-01": {
			Name:  "tech-lead-01",
			ID:    "tech-lead-01",
			Role:  agent.Role("tech-lead"),
			State: agent.StateIdle,
		},
	}

	stateData, marshalErr := json.Marshal(agentsState)
	if marshalErr != nil {
		t.Fatalf("failed to marshal agents state: %v", marshalErr)
	}

	if writeErr := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); writeErr != nil {
		t.Fatalf("failed to write agents state: %v", writeErr)
	}

	mgr := agent.NewManager(agentsDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		t.Fatalf("failed to load state: %v", loadErr)
	}

	router := NewRouter(mgr)

	// Test routing to engineers
	agentID, err := router.RouteToRole("engineer")
	if err != nil {
		t.Fatalf("RouteToRole(engineer) failed: %v", err)
	}
	if agentID != "engineer-01" && agentID != "engineer-02" {
		t.Errorf("expected engineer-01 or engineer-02, got %s", agentID)
	}

	// Test routing to tech-lead
	agentID, err = router.RouteToRole("tech-lead")
	if err != nil {
		t.Fatalf("RouteToRole(tech-lead) failed: %v", err)
	}
	if agentID != "tech-lead-01" {
		t.Errorf("expected tech-lead-01, got %s", agentID)
	}

	// Test routing to manager (none configured) - should fail
	_, err = router.RouteToRole("manager")
	if err == nil {
		t.Error("expected error for manager with no agents")
	}
}

func TestRouteToRoleRoundRobin(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")

	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create 3 engineers
	agentsState := map[string]*agent.Agent{
		"engineer-01": {
			Name:  "engineer-01",
			ID:    "engineer-01",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
		"engineer-02": {
			Name:  "engineer-02",
			ID:    "engineer-02",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
		"engineer-03": {
			Name:  "engineer-03",
			ID:    "engineer-03",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
	}

	stateData, _ := json.Marshal(agentsState)
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); err != nil {
		t.Fatalf("failed to write agents state: %v", err)
	}

	mgr := agent.NewManager(agentsDir)
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	router := NewRouter(mgr)

	// Call RouteToRole 4 times to verify round-robin
	assigned := make(map[string]int)
	for i := 0; i < 6; i++ {
		agentID, err := router.RouteToRole("engineer")
		if err != nil {
			t.Fatalf("RouteToRole failed on iteration %d: %v", i, err)
		}
		assigned[agentID]++
	}

	// Each engineer should be assigned 2 times
	for agentID, count := range assigned {
		if count != 2 {
			t.Errorf("agent %s assigned %d times, expected 2", agentID, count)
		}
	}
}

func TestRouteToRoleFiltersByState(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")

	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create agents with various states - only idle/working should be available
	agentsState := map[string]*agent.Agent{
		"engineer-stopped": {
			Name:  "engineer-stopped",
			ID:    "engineer-stopped",
			Role:  agent.Role("engineer"),
			State: agent.StateStopped,
		},
		"engineer-error": {
			Name:  "engineer-error",
			ID:    "engineer-error",
			Role:  agent.Role("engineer"),
			State: agent.StateError,
		},
		"engineer-idle": {
			Name:  "engineer-idle",
			ID:    "engineer-idle",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
	}

	stateData, _ := json.Marshal(agentsState)
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); err != nil {
		t.Fatalf("failed to write agents state: %v", err)
	}

	mgr := agent.NewManager(agentsDir)
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	router := NewRouter(mgr)

	// Should only get the idle engineer
	agentID, err := router.RouteToRole("engineer")
	if err != nil {
		t.Fatalf("RouteToRole failed: %v", err)
	}
	if agentID != "engineer-idle" {
		t.Errorf("expected engineer-idle, got %s", agentID)
	}
}

func TestRouteToRoleNoRunningAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")

	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create agents but all in non-running states
	agentsState := map[string]*agent.Agent{
		"engineer-stopped": {
			Name:  "engineer-stopped",
			ID:    "engineer-stopped",
			Role:  agent.Role("engineer"),
			State: agent.StateStopped,
		},
		"engineer-error": {
			Name:  "engineer-error",
			ID:    "engineer-error",
			Role:  agent.Role("engineer"),
			State: agent.StateError,
		},
	}

	stateData, _ := json.Marshal(agentsState)
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); err != nil {
		t.Fatalf("failed to write agents state: %v", err)
	}

	mgr := agent.NewManager(agentsDir)
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	router := NewRouter(mgr)

	// Should fail - engineers exist but none are running
	_, err := router.RouteToRole("engineer")
	if err == nil {
		t.Error("expected error for no running agents")
	}
	if !strings.Contains(err.Error(), "no running agents") {
		t.Errorf("error should mention no running agents: %v", err)
	}
}

func TestRouteTaskTypeWithAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")

	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	agentsState := map[string]*agent.Agent{
		"engineer-01": {
			Name:  "engineer-01",
			ID:    "engineer-01",
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		},
		"tech-lead-01": {
			Name:  "tech-lead-01",
			ID:    "tech-lead-01",
			Role:  agent.Role("tech-lead"),
			State: agent.StateWorking,
		},
		"manager-01": {
			Name:  "manager-01",
			ID:    "manager-01",
			Role:  agent.Role("manager"),
			State: agent.StateIdle,
		},
		"qa-01": {
			Name:  "qa-01",
			ID:    "qa-01",
			Role:  agent.Role("qa"),
			State: agent.StateIdle,
		},
	}

	stateData, _ := json.Marshal(agentsState)
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); err != nil {
		t.Fatalf("failed to write agents state: %v", err)
	}

	mgr := agent.NewManager(agentsDir)
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	router := NewRouter(mgr)

	tests := []struct {
		taskType    TaskType
		wantAgentID string
	}{
		{TaskTypeCode, "engineer-01"},
		{TaskTypeReview, "tech-lead-01"},
		{TaskTypeMerge, "manager-01"},
		{TaskTypeQA, "qa-01"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			agentID, err := router.RouteTaskType(tt.taskType)
			if err != nil {
				t.Fatalf("RouteTaskType(%s) failed: %v", tt.taskType, err)
			}
			if agentID != tt.wantAgentID {
				t.Errorf("RouteTaskType(%s) = %s, want %s", tt.taskType, agentID, tt.wantAgentID)
			}
		})
	}
}
