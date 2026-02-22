package remote

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkNewManager measures manager creation performance.
func BenchmarkNewManager(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewManager("/tmp/test-workspace")
	}
}

// BenchmarkListHosts measures listing all hosts.
func BenchmarkListHosts(b *testing.B) {
	sizes := []int{0, 5, 20, 50}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("hosts-%d", size), func(b *testing.B) {
			mgr := NewManager("/tmp/test-workspace")

			// Populate with mock hosts
			now := time.Now()
			for i := 0; i < size; i++ {
				name := fmt.Sprintf("host-%02d", i)
				mgr.hosts[name] = &Host{
					Name:     name,
					Hostname: fmt.Sprintf("192.168.1.%d", i+1),
					Port:     22,
					User:     "deploy",
					Status:   StatusConnected,
					AddedAt:  now,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = mgr.ListHosts()
			}
		})
	}
}

// BenchmarkGetHost measures getting a host by name.
func BenchmarkGetHost(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add hosts
	now := time.Now()
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("host-%02d", i)
		mgr.hosts[name] = &Host{
			Name:     name,
			Hostname: fmt.Sprintf("192.168.1.%d", i+1),
			Port:     22,
			User:     "deploy",
			Status:   StatusConnected,
			AddedAt:  now,
		}
	}

	b.Run("existing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.GetHost("host-10")
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.GetHost("nonexistent")
		}
	})
}

// BenchmarkListAgents measures listing all agents.
func BenchmarkListAgents(b *testing.B) {
	sizes := []int{0, 10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents-%d", size), func(b *testing.B) {
			mgr := NewManager("/tmp/test-workspace")

			// Populate with mock agents
			now := time.Now()
			for i := 0; i < size; i++ {
				name := fmt.Sprintf("agent-%03d", i)
				mgr.agents[name] = &RemoteAgent{
					Name:      name,
					Host:      fmt.Sprintf("host-%02d", i%5),
					Role:      "engineer",
					Status:    "running",
					StartedAt: now,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = mgr.ListAgents()
			}
		})
	}
}

// BenchmarkGetAgent measures getting an agent by name.
func BenchmarkGetAgent(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add agents
	now := time.Now()
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("agent-%03d", i)
		mgr.agents[name] = &RemoteAgent{
			Name:      name,
			Host:      fmt.Sprintf("host-%02d", i%5),
			Role:      "engineer",
			Status:    "running",
			StartedAt: now,
		}
	}

	b.Run("existing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.GetAgent("agent-025")
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.GetAgent("nonexistent")
		}
	})
}

// BenchmarkListAgentsByHost measures filtering agents by host.
func BenchmarkListAgentsByHost(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add 100 agents across 5 hosts
	now := time.Now()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("agent-%03d", i)
		mgr.agents[name] = &RemoteAgent{
			Name:      name,
			Host:      fmt.Sprintf("host-%02d", i%5),
			Role:      "engineer",
			Status:    "running",
			StartedAt: now,
		}
	}

	b.Run("existing-host", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.ListAgentsByHost("host-02")
		}
	})

	b.Run("nonexistent-host", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.ListAgentsByHost("nonexistent")
		}
	})
}

// BenchmarkSSHCommand measures SSH command generation.
func BenchmarkSSHCommand(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add hosts with different configs
	now := time.Now()
	mgr.hosts["simple"] = &Host{
		Name:     "simple",
		Hostname: "192.168.1.1",
		Port:     22,
		User:     "deploy",
		AddedAt:  now,
	}
	mgr.hosts["with-key"] = &Host{
		Name:     "with-key",
		Hostname: "192.168.1.2",
		Port:     2222,
		User:     "admin",
		KeyPath:  "/home/user/.ssh/custom_key",
		AddedAt:  now,
	}

	b.Run("simple", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.SSHCommand("simple")
		}
	})

	b.Run("with-key", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.SSHCommand("with-key")
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.SSHCommand("nonexistent")
		}
	})
}

// BenchmarkHostCreation measures Host struct creation.
func BenchmarkHostCreation(b *testing.B) {
	now := time.Now()

	b.Run("minimal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Host{
				Name:     "test-host",
				Hostname: "192.168.1.1",
				Port:     22,
				User:     "deploy",
				Status:   StatusUnknown,
				AddedAt:  now,
			}
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Host{
				Name:        "production-server",
				Hostname:    "prod.example.com",
				Port:        2222,
				User:        "deploy",
				KeyPath:     "/home/user/.ssh/prod_key",
				Description: "Main production server for agent deployment",
				AddedAt:     now,
				LastUsed:    now,
				Status:      StatusConnected,
			}
		}
	})
}

// BenchmarkRemoteAgentCreation measures RemoteAgent struct creation.
func BenchmarkRemoteAgentCreation(b *testing.B) {
	now := time.Now()

	b.Run("minimal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = RemoteAgent{
				Name:      "eng-01",
				Host:      "server-01",
				Role:      "engineer",
				Status:    "running",
				StartedAt: now,
			}
		}
	})

	b.Run("with-pid", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = RemoteAgent{
				Name:      "eng-01",
				Host:      "server-01",
				Role:      "engineer",
				PID:       12345,
				Status:    "running",
				StartedAt: now,
			}
		}
	})
}

// BenchmarkStatusConstants measures status constant comparisons.
func BenchmarkStatusConstants(b *testing.B) {
	statuses := []string{StatusUnknown, StatusConnected, StatusUnreachable, StatusError}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)] //nolint:gosec // index bounded by modulo
		_ = status == StatusConnected
	}
}
