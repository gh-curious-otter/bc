package stats

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkNew measures Stats creation overhead.
func BenchmarkNew(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New("/tmp/test-state")
	}
}

// BenchmarkFormatDuration measures duration formatting performance.
func BenchmarkFormatDuration(b *testing.B) {
	durations := []struct {
		name string
		d    time.Duration
	}{
		{"seconds", 45 * time.Second},
		{"minutes", 5*time.Minute + 30*time.Second},
		{"hours", 2*time.Hour + 15*time.Minute},
		{"long", 24*time.Hour + 30*time.Minute + 45*time.Second},
	}

	for _, tc := range durations {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = formatDuration(tc.d)
			}
		})
	}
}

// BenchmarkUtilization measures utilization calculation.
func BenchmarkUtilization(b *testing.B) {
	s := &Stats{
		Agents: AgentMetrics{
			ActiveAgents: 10,
			Working:      5,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Utilization()
	}
}

// BenchmarkUtilizationZero measures utilization with no active agents.
func BenchmarkUtilizationZero(b *testing.B) {
	s := &Stats{
		Agents: AgentMetrics{
			ActiveAgents: 0,
			Working:      0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Utilization()
	}
}

// BenchmarkSummary measures summary string generation.
func BenchmarkSummary(b *testing.B) {
	sizes := []int{5, 20, 50}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents-%d", size), func(b *testing.B) {
			s := &Stats{
				CollectedAt: time.Now(),
				Agents: AgentMetrics{
					TotalAgents:  size,
					ActiveAgents: size - 2,
					Coordinators: 1,
					Workers:      size - 1,
					Idle:         size / 2,
					Working:      size / 4,
					Done:         size / 8,
					Stuck:        1,
					Stopped:      2,
					AgentStats:   make([]AgentStat, size),
				},
			}
			// Populate agent stats
			for i := 0; i < size; i++ {
				s.Agents.AgentStats[i] = AgentStat{
					Name:   fmt.Sprintf("agent-%03d", i),
					Role:   "engineer",
					State:  "idle",
					Uptime: time.Duration(i) * time.Minute,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.Summary()
			}
		})
	}
}

// BenchmarkSummaryEmpty measures summary with no agents.
func BenchmarkSummaryEmpty(b *testing.B) {
	s := &Stats{
		CollectedAt: time.Now(),
		Agents:      AgentMetrics{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Summary()
	}
}

// BenchmarkCollectAgentMetrics measures metric collection with mock data.
func BenchmarkCollectAgentMetrics(b *testing.B) {
	// Note: This benchmark uses pre-populated AgentStats rather than
	// actually loading agents, as that would require workspace setup.
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents-%d", size), func(b *testing.B) {
			stats := make([]AgentStat, size)
			for i := 0; i < size; i++ {
				role := "engineer"
				if i == 0 {
					role = "root"
				}
				state := "idle"
				if i%3 == 0 {
					state = "working"
				} else if i%5 == 0 {
					state = "stopped"
				}
				stats[i] = AgentStat{
					Name:   fmt.Sprintf("agent-%03d", i),
					Role:   role,
					State:  state,
					Uptime: time.Duration(i) * time.Minute,
				}
			}

			s := &Stats{
				Agents: AgentMetrics{
					AgentStats: stats,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate metric recalculation
				s.Agents.TotalAgents = len(stats)
				s.Agents.ActiveAgents = 0
				for _, stat := range stats {
					if stat.State != "stopped" && stat.State != "error" {
						s.Agents.ActiveAgents++
					}
				}
			}
		})
	}
}

// BenchmarkAgentStatCreation measures AgentStat struct creation.
func BenchmarkAgentStatCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AgentStat{
			Name:   "test-agent",
			Role:   "engineer",
			State:  "working",
			Uptime: 5 * time.Minute,
		}
	}
}
