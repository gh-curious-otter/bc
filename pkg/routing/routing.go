// Package routing provides task routing logic for assigning work to agents.
package routing

import (
	"fmt"
	"sync"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/queue"
)

// TaskTypeToRole maps task types to the roles that should handle them.
var TaskTypeToRole = map[queue.TaskType]agent.Role{
	queue.TaskTypeCode:   agent.RoleEngineer,
	queue.TaskTypeReview: agent.RoleTechLead,
	queue.TaskTypeMerge:  agent.RoleManager,
	queue.TaskTypeQA:     agent.RoleQA,
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

// RouteTask finds an appropriate agent for the given work item based on its type.
// Returns the agent ID (name) that should handle this task.
// Uses round-robin distribution among agents of the same role.
func (r *Router) RouteTask(item *queue.WorkItem) (string, error) {
	if item == nil {
		return "", fmt.Errorf("nil work item")
	}

	taskType := item.EffectiveType()
	role, ok := TaskTypeToRole[taskType]
	if !ok {
		return "", fmt.Errorf("unknown task type: %s", taskType)
	}

	return r.RouteToRole(role)
}

// RouteToRole finds an available agent with the given role using round-robin.
func (r *Router) RouteToRole(role agent.Role) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agents := r.mgr.ListByRole(role)
	if len(agents) == 0 {
		return "", fmt.Errorf("no agents available with role: %s", role)
	}

	// Filter to only running/idle agents
	var available []*agent.Agent
	for _, a := range agents {
		if a.State == agent.StateIdle || a.State == agent.StateWorking {
			available = append(available, a)
		}
	}

	if len(available) == 0 {
		return "", fmt.Errorf("no running agents available with role: %s", role)
	}

	// Round-robin selection
	lastIdx := r.lastAssigned[role]
	nextIdx := (lastIdx + 1) % len(available)
	r.lastAssigned[role] = nextIdx

	return available[nextIdx].Name, nil
}

// GetRoleForTaskType returns the role that should handle a given task type.
func GetRoleForTaskType(taskType queue.TaskType) (agent.Role, error) {
	role, ok := TaskTypeToRole[taskType]
	if !ok {
		return "", fmt.Errorf("unknown task type: %s", taskType)
	}
	return role, nil
}
