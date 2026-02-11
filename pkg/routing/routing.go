// Package routing provides task routing logic for assigning work to agents.
package routing

import (
	"fmt"
	"sync"

	"github.com/rpuneet/bc/pkg/agent"
)

// TaskType distinguishes between different kinds of work items.
type TaskType string

const (
	TaskTypeCode   TaskType = "code"   // Implementation work (default)
	TaskTypeReview TaskType = "review" // PR review work
	TaskTypeMerge  TaskType = "merge"  // Merge approved PRs
	TaskTypeQA     TaskType = "qa"     // Testing/validation work
)

// TaskTypeToRole maps task types to the roles that should handle them.
// These are role name strings that match .bc/roles/<role>.md files
var TaskTypeToRole = map[TaskType]string{
	TaskTypeCode:   "engineer",
	TaskTypeReview: "tech-lead",
	TaskTypeMerge:  "manager",
	TaskTypeQA:     "qa",
}

// Router assigns tasks to agents based on task type and agent availability.
type Router struct {
	// Track last assigned agent per role for round-robin
	lastAssigned map[agent.Role]int
	mgr          *agent.Manager
	mu           sync.Mutex
}

// NewRouter creates a Router that uses the given agent manager to find agents.
func NewRouter(mgr *agent.Manager) *Router {
	return &Router{
		mgr:          mgr,
		lastAssigned: make(map[agent.Role]int),
	}
}

// RouteTaskType finds an appropriate agent for the given task type.
// Returns the agent ID (name) that should handle this task.
// Uses round-robin distribution among agents of the same role.
func (r *Router) RouteTaskType(taskType TaskType) (string, error) {
	role, ok := TaskTypeToRole[taskType]
	if !ok {
		return "", fmt.Errorf("unknown task type: %s", taskType)
	}

	return r.RouteToRole(role)
}

// RouteToRole finds an available agent with the given role name using round-robin.
func (r *Router) RouteToRole(roleName string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agents := r.mgr.ListByRole(agent.Role(roleName))
	if len(agents) == 0 {
		return "", fmt.Errorf("no agents available with role: %s", roleName)
	}

	// Filter to only running/idle agents
	var available []*agent.Agent
	for _, a := range agents {
		if a.State == agent.StateIdle || a.State == agent.StateWorking {
			available = append(available, a)
		}
	}

	if len(available) == 0 {
		return "", fmt.Errorf("no running agents available with role: %s", roleName)
	}

	// Round-robin selection (use index to track across string role names)
	lastIdx := r.lastAssigned[agent.Role(roleName)]
	nextIdx := (lastIdx + 1) % len(available)
	r.lastAssigned[agent.Role(roleName)] = nextIdx

	return available[nextIdx].Name, nil
}

// GetRoleForTaskType returns the role name that should handle a given task type.
func GetRoleForTaskType(taskType TaskType) (string, error) {
	roleName, ok := TaskTypeToRole[taskType]
	if !ok {
		return "", fmt.Errorf("unknown task type: %s", taskType)
	}
	return roleName, nil
}
