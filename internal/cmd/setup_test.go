package cmd

import (
	"os"
	"testing"

	"github.com/gh-curious-otter/bc/pkg/agent"
)

func TestMain(m *testing.M) {
	// Setup roles for tests - mirrors pkg/agent/agent_test.go TestMain
	agent.RoleCapabilities[agent.Role("engineer")] = []agent.Capability{agent.CapImplementTasks}
	agent.RoleCapabilities[agent.Role("manager")] = []agent.Capability{agent.CapAssignWork, agent.CapCreateAgents}
	agent.RoleCapabilities[agent.Role("qa")] = []agent.Capability{agent.CapTestWork, agent.CapReviewWork}
	agent.RoleCapabilities[agent.Role("product-manager")] = []agent.Capability{agent.CapCreateEpics, agent.CapCreateAgents}
	agent.RoleCapabilities[agent.Role("worker")] = []agent.Capability{agent.CapImplementTasks}
	agent.RoleCapabilities[agent.Role("tech-lead")] = []agent.Capability{agent.CapReviewWork, agent.CapCreateAgents}

	agent.RoleHierarchy[agent.Role("manager")] = []agent.Role{
		agent.Role("engineer"),
		agent.Role("qa"),
		agent.Role("tech-lead"),
	}
	agent.RoleHierarchy[agent.Role("tech-lead")] = []agent.Role{
		agent.Role("engineer"),
	}
	agent.RoleHierarchy[agent.Role("product-manager")] = []agent.Role{agent.Role("manager")}
	agent.RoleHierarchy[agent.RoleRoot] = []agent.Role{
		agent.Role("product-manager"),
		agent.Role("manager"),
		agent.Role("engineer"),
		agent.Role("qa"),
		agent.Role("worker"),
		agent.Role("tech-lead"),
	}

	os.Exit(m.Run())
}
