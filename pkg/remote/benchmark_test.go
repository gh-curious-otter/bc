package remote

import (
	"context"
	"fmt"
	"testing"
)

func setupBenchmarkManager(b *testing.B) *Manager {
	b.Helper()
	dir := b.TempDir()
	m := NewManager(dir)
	return m
}

func BenchmarkNewManager(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewManager(dir)
	}
}

func BenchmarkManager_Load_NoFile(b *testing.B) {
	m := setupBenchmarkManager(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Load()
	}
}

func BenchmarkManager_Load_WithData(b *testing.B) {
	m := setupBenchmarkManager(b)
	// Add some data and save
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test server")
	_, _ = m.AddHost("server2", "192.168.1.2", 22, "admin", "/path/to/key", "Production")
	_, _ = m.SpawnAgent(context.Background(), "agent1", "server1", "engineer")
	_ = m.Save()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create fresh manager to force load from disk
		m2 := NewManager(m.configPath[:len(m.configPath)-len("/.bc/remote.json")])
		m2.configPath = m.configPath
		_ = m2.Load()
	}
}

func BenchmarkManager_Save_Empty(b *testing.B) {
	m := setupBenchmarkManager(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Save()
	}
}

func BenchmarkManager_Save_WithData(b *testing.B) {
	m := setupBenchmarkManager(b)
	// Add some data
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test server")
	_, _ = m.AddHost("server2", "192.168.1.2", 22, "admin", "/path/to/key", "Production")
	_, _ = m.SpawnAgent(context.Background(), "agent1", "server1", "engineer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Save()
	}
}

func BenchmarkManager_AddHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := setupBenchmarkManager(b)
		b.StartTimer()
		_, _ = m.AddHost(fmt.Sprintf("server%d", i), "192.168.1.1", 22, "root", "", "Test")
	}
}

func BenchmarkManager_GetHost_Found(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetHost("server1")
	}
}

func BenchmarkManager_GetHost_NotFound(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetHost("nonexistent")
	}
}

func BenchmarkManager_ListHosts_Empty(b *testing.B) {
	m := setupBenchmarkManager(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListHosts()
	}
}

func BenchmarkManager_ListHosts_WithData(b *testing.B) {
	m := setupBenchmarkManager(b)
	for i := 0; i < 10; i++ {
		_, _ = m.AddHost(fmt.Sprintf("server%d", i), fmt.Sprintf("192.168.1.%d", i), 22, "root", "", "Test")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListHosts()
	}
}

func BenchmarkManager_RemoveHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := setupBenchmarkManager(b)
		_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
		b.StartTimer()
		_ = m.RemoveHost("server1")
	}
}

func BenchmarkManager_SpawnAgent(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := setupBenchmarkManager(b)
		_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
		b.StartTimer()
		_, _ = m.SpawnAgent(ctx, fmt.Sprintf("agent%d", i), "server1", "engineer")
	}
}

func BenchmarkManager_GetAgent_Found(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	_, _ = m.SpawnAgent(context.Background(), "agent1", "server1", "engineer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetAgent("agent1")
	}
}

func BenchmarkManager_GetAgent_NotFound(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	_, _ = m.SpawnAgent(context.Background(), "agent1", "server1", "engineer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetAgent("nonexistent")
	}
}

func BenchmarkManager_ListAgents_Empty(b *testing.B) {
	m := setupBenchmarkManager(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListAgents()
	}
}

func BenchmarkManager_ListAgents_WithData(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	for i := 0; i < 10; i++ {
		_, _ = m.SpawnAgent(context.Background(), fmt.Sprintf("agent%d", i), "server1", "engineer")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListAgents()
	}
}

func BenchmarkManager_ListAgentsByHost_Empty(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListAgentsByHost("server1")
	}
}

func BenchmarkManager_ListAgentsByHost_WithData(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	_, _ = m.AddHost("server2", "192.168.1.2", 22, "root", "", "Test")
	for i := 0; i < 10; i++ {
		host := "server1"
		if i%2 == 0 {
			host = "server2"
		}
		_, _ = m.SpawnAgent(context.Background(), fmt.Sprintf("agent%d", i), host, "engineer")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.ListAgentsByHost("server1")
	}
}

func BenchmarkManager_StopAgent(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := setupBenchmarkManager(b)
		_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
		_, _ = m.SpawnAgent(ctx, "agent1", "server1", "engineer")
		b.StartTimer()
		_ = m.StopAgent(ctx, "agent1")
	}
}

func BenchmarkManager_SSHCommand_Simple(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.SSHCommand("server1")
	}
}

func BenchmarkManager_SSHCommand_WithKey(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "admin", "/home/user/.ssh/id_rsa", "Test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.SSHCommand("server1")
	}
}

func BenchmarkManager_TestConnection(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := setupBenchmarkManager(b)
		_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
		b.StartTimer()
		_ = m.TestConnection(ctx, "server1")
	}
}

func BenchmarkManager_Parallel_GetHost(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = m.GetHost("server1")
		}
	})
}

func BenchmarkManager_Parallel_ListHosts(b *testing.B) {
	m := setupBenchmarkManager(b)
	for i := 0; i < 10; i++ {
		_, _ = m.AddHost(fmt.Sprintf("server%d", i), fmt.Sprintf("192.168.1.%d", i), 22, "root", "", "Test")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = m.ListHosts()
		}
	})
}

func BenchmarkManager_Parallel_GetAgent(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	_, _ = m.SpawnAgent(context.Background(), "agent1", "server1", "engineer")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = m.GetAgent("agent1")
		}
	})
}

func BenchmarkManager_Parallel_ListAgents(b *testing.B) {
	m := setupBenchmarkManager(b)
	_, _ = m.AddHost("server1", "192.168.1.1", 22, "root", "", "Test")
	for i := 0; i < 10; i++ {
		_, _ = m.SpawnAgent(context.Background(), fmt.Sprintf("agent%d", i), "server1", "engineer")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = m.ListAgents()
		}
	})
}
