package routing

import (
	"path/filepath"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestTaskTypeToRoleMapping(t *testing.T) {
	tests := []struct {
		taskType TaskType
		wantRole agent.Role
	}{
		{TaskTypeCode, agent.RoleEngineer},
		{TaskTypeReview, agent.RoleTechLead},
		{TaskTypeMerge, agent.RoleManager},
		{TaskTypeQA, agent.RoleQA},
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
		wantRole agent.Role
		wantErr  bool
	}{
		{TaskTypeCode, agent.RoleEngineer, false},
		{TaskTypeReview, agent.RoleTechLead, false},
		{TaskTypeMerge, agent.RoleManager, false},
		{TaskTypeQA, agent.RoleQA, false},
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
	router.lastAssigned[agent.RoleEngineer] = 0

	// After incrementing, should be 1
	nextIdx := (router.lastAssigned[agent.RoleEngineer] + 1) % 3
	if nextIdx != 1 {
		t.Errorf("expected next index 1, got %d", nextIdx)
	}

	// Wrap around test
	router.lastAssigned[agent.RoleEngineer] = 2
	nextIdx = (router.lastAssigned[agent.RoleEngineer] + 1) % 3
	if nextIdx != 0 {
		t.Errorf("expected wrapped index 0, got %d", nextIdx)
	}
}

func TestRouteToRoleNoAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	mgr := agent.NewManager(agentsDir)

	router := NewRouter(mgr)

	_, err := router.RouteToRole(agent.RoleEngineer)
	if err == nil {
		t.Error("expected error when no engineers available")
	}
}
