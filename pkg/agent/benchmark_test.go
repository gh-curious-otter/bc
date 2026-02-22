package agent

import (
	"fmt"
	"testing"
)

// BenchmarkGetAgent measures single agent lookup performance.
func BenchmarkGetAgent(b *testing.B) {
	m := NewManager("/tmp/benchmark-agents")

	// Setup: create test agents
	for i := 0; i < 100; i++ {
		m.agents[fmt.Sprintf("agent-%03d", i)] = &Agent{
			Name:  fmt.Sprintf("agent-%03d", i),
			Role:  Role("engineer"),
			State: StateIdle,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetAgent("agent-050")
	}
}

// BenchmarkListAgents measures listing all agents.
func BenchmarkListAgents(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents-%d", size), func(b *testing.B) {
			m := NewManager("/tmp/benchmark-agents")

			// Setup: create test agents
			for i := 0; i < size; i++ {
				m.agents[fmt.Sprintf("agent-%03d", i)] = &Agent{
					Name:  fmt.Sprintf("agent-%03d", i),
					Role:  Role("engineer"),
					State: StateIdle,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.ListAgents()
			}
		})
	}
}

// BenchmarkAgentsByState measures filtering agents by state.
func BenchmarkAgentsByState(b *testing.B) {
	m := NewManager("/tmp/benchmark-agents")

	// Setup: create mixed state agents
	states := []State{StateIdle, StateWorking, StateStopped, StateStarting}
	numStates := len(states)
	for i := 0; i < 100; i++ {
		state := states[i%numStates] //nolint:gosec // index is bounded by numStates
		m.agents[fmt.Sprintf("agent-%03d", i)] = &Agent{
			Name:  fmt.Sprintf("agent-%03d", i),
			Role:  Role("engineer"),
			State: state,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Manual filter by state (simulates common pattern)
		var working []*Agent
		for _, a := range m.agents {
			if a.State == StateWorking {
				working = append(working, a)
			}
		}
		_ = working
	}
}

// BenchmarkManagerCreation measures manager initialization overhead.
func BenchmarkManagerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewManager("/tmp/benchmark-agents")
	}
}

// BenchmarkAgentMapOperations measures map access patterns.
func BenchmarkAgentMapOperations(b *testing.B) {
	b.Run("insert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := NewManager("/tmp/benchmark-agents")
			for j := 0; j < 100; j++ {
				m.agents[fmt.Sprintf("agent-%03d", j)] = &Agent{
					Name: fmt.Sprintf("agent-%03d", j),
				}
			}
		}
	})

	b.Run("lookup", func(b *testing.B) {
		m := NewManager("/tmp/benchmark-agents")
		for j := 0; j < 100; j++ {
			m.agents[fmt.Sprintf("agent-%03d", j)] = &Agent{
				Name: fmt.Sprintf("agent-%03d", j),
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = m.agents["agent-050"]
		}
	})

	b.Run("iterate", func(b *testing.B) {
		m := NewManager("/tmp/benchmark-agents")
		for j := 0; j < 100; j++ {
			m.agents[fmt.Sprintf("agent-%03d", j)] = &Agent{
				Name: fmt.Sprintf("agent-%03d", j),
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range m.agents {
				count++
			}
		}
	})
}
