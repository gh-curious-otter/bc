package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

// --- Benchmark helpers ---

// newBenchRouter creates a Router with pre-populated agents for benchmarking.
func newBenchRouter(b *testing.B, agentsPerRole int) *Router {
	b.Helper()

	dir := b.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		b.Fatal(err)
	}

	// Create agents state with multiple agents per role
	agentsState := make(map[string]*agent.Agent)
	roles := []string{"engineer", "tech-lead", "manager", "qa"}

	for _, role := range roles {
		for i := range agentsPerRole {
			name := role + "-" + string(rune('0'+i))
			agentsState[name] = &agent.Agent{
				Name:  name,
				ID:    name,
				Role:  agent.Role(role),
				State: agent.StateIdle,
			}
		}
	}

	stateData, err := json.Marshal(agentsState)
	if err != nil {
		b.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), stateData, 0600); err != nil {
		b.Fatal(err)
	}

	mgr := agent.NewManager(agentsDir)
	if err := mgr.LoadState(); err != nil {
		b.Fatal(err)
	}

	return NewRouter(mgr)
}

// --- NewRouter benchmarks ---

func BenchmarkNewRouter(b *testing.B) {
	dir := b.TempDir()
	agentsDir := filepath.Join(dir, ".bc", "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		b.Fatal(err)
	}

	mgr := agent.NewManager(agentsDir)

	b.ResetTimer()
	for range b.N {
		_ = NewRouter(mgr)
	}
}

// --- RouteTaskType benchmarks ---

func BenchmarkRouteTaskType_Code(b *testing.B) {
	router := newBenchRouter(b, 5)

	b.ResetTimer()
	for range b.N {
		_, _ = router.RouteTaskType(TaskTypeCode)
	}
}

func BenchmarkRouteTaskType_Review(b *testing.B) {
	router := newBenchRouter(b, 5)

	b.ResetTimer()
	for range b.N {
		_, _ = router.RouteTaskType(TaskTypeReview)
	}
}

func BenchmarkRouteTaskType_AllTypes(b *testing.B) {
	router := newBenchRouter(b, 5)
	types := []TaskType{TaskTypeCode, TaskTypeReview, TaskTypeMerge, TaskTypeQA}

	b.ResetTimer()
	for i := range b.N {
		_, _ = router.RouteTaskType(types[i%len(types)])
	}
}

// --- RouteToRole benchmarks ---

func BenchmarkRouteToRole_SingleAgent(b *testing.B) {
	router := newBenchRouter(b, 1)

	b.ResetTimer()
	for range b.N {
		_, _ = router.RouteToRole("engineer")
	}
}

func BenchmarkRouteToRole_FiveAgents(b *testing.B) {
	router := newBenchRouter(b, 5)

	b.ResetTimer()
	for range b.N {
		_, _ = router.RouteToRole("engineer")
	}
}

func BenchmarkRouteToRole_TenAgents(b *testing.B) {
	router := newBenchRouter(b, 10)

	b.ResetTimer()
	for range b.N {
		_, _ = router.RouteToRole("engineer")
	}
}

func BenchmarkRouteToRole_RoundRobin(b *testing.B) {
	router := newBenchRouter(b, 5)

	b.ResetTimer()
	for range b.N {
		// This exercises the round-robin logic repeatedly
		_, _ = router.RouteToRole("engineer")
	}
}

// --- GetRoleForTaskType benchmarks ---

func BenchmarkGetRoleForTaskType(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_, _ = GetRoleForTaskType(TaskTypeCode)
	}
}

func BenchmarkGetRoleForTaskType_AllTypes(b *testing.B) {
	types := []TaskType{TaskTypeCode, TaskTypeReview, TaskTypeMerge, TaskTypeQA}

	b.ResetTimer()
	for i := range b.N {
		_, _ = GetRoleForTaskType(types[i%len(types)])
	}
}

// --- Parallel benchmarks ---

func BenchmarkRouteToRole_Parallel(b *testing.B) {
	router := newBenchRouter(b, 5)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = router.RouteToRole("engineer")
		}
	})
}

func BenchmarkRouteTaskType_Parallel(b *testing.B) {
	router := newBenchRouter(b, 5)
	types := []TaskType{TaskTypeCode, TaskTypeReview, TaskTypeMerge, TaskTypeQA}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = router.RouteTaskType(types[i%len(types)])
			i++
		}
	})
}
