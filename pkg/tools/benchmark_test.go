package tools

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkNewRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRegistry()
	}
}

func BenchmarkRegistry_Register(b *testing.B) {
	r := NewRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := &Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: true,
		}
		_ = r.Register(tool)
	}
}

func BenchmarkRegistry_Register_SameName(b *testing.B) {
	r := NewRegistry()
	tool := &Tool{
		Name:    "test-tool",
		Command: "echo",
		Enabled: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Register(tool)
	}
}

func BenchmarkRegistry_Get_Found(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "test-tool", Command: "echo", Enabled: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("test-tool")
	}
}

func BenchmarkRegistry_Get_NotFound(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "test-tool", Command: "echo", Enabled: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("nonexistent")
	}
}

func BenchmarkRegistry_List_Empty(b *testing.B) {
	r := NewRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.List()
	}
}

func BenchmarkRegistry_List_WithTools(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		_ = r.Register(&Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: i%2 == 0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.List()
	}
}

func BenchmarkRegistry_ListEnabled_Empty(b *testing.B) {
	r := NewRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ListEnabled()
	}
}

func BenchmarkRegistry_ListEnabled_WithTools(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		_ = r.Register(&Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: i%2 == 0, // 5 enabled, 5 disabled
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ListEnabled()
	}
}

func BenchmarkRegistry_Enable(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "test-tool", Command: "echo", Enabled: false})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Enable("test-tool")
	}
}

func BenchmarkRegistry_Disable(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "test-tool", Command: "echo", Enabled: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Disable("test-tool")
	}
}

func BenchmarkTool_IsInstalled_Found(b *testing.B) {
	tool := &Tool{
		Name:    "echo-tool",
		Command: "echo hello", // echo is always available
		Enabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.IsInstalled()
	}
}

func BenchmarkTool_IsInstalled_NotFound(b *testing.B) {
	tool := &Tool{
		Name:    "fake-tool",
		Command: "nonexistent-binary-xyz-123",
		Enabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.IsInstalled()
	}
}

func BenchmarkTool_Status_Ready(b *testing.B) {
	tool := &Tool{
		Name:    "echo-tool",
		Command: "echo",
		Enabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.Status()
	}
}

func BenchmarkTool_Status_Disabled(b *testing.B) {
	tool := &Tool{
		Name:    "echo-tool",
		Command: "echo",
		Enabled: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.Status()
	}
}

func BenchmarkTool_Status_NotInstalled(b *testing.B) {
	tool := &Tool{
		Name:    "fake-tool",
		Command: "nonexistent-binary-xyz-123",
		Enabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.Status()
	}
}

func BenchmarkRegistry_Exec_Simple(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "true", Command: "true", Enabled: true})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Exec(ctx, "true")
	}
}

func BenchmarkRegistry_Exec_WithArgs(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "echo", Command: "echo", Enabled: true})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Exec(ctx, "echo", "hello", "world")
	}
}

// Parallel benchmarks

func BenchmarkRegistry_Get_Parallel(b *testing.B) {
	r := NewRegistry()
	_ = r.Register(&Tool{Name: "test-tool", Command: "echo", Enabled: true})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = r.Get("test-tool")
		}
	})
}

func BenchmarkRegistry_List_Parallel(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		_ = r.Register(&Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: true,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.List()
		}
	})
}

func BenchmarkRegistry_ListEnabled_Parallel(b *testing.B) {
	r := NewRegistry()
	for i := 0; i < 10; i++ {
		_ = r.Register(&Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: i%2 == 0,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.ListEnabled()
		}
	})
}

func BenchmarkRegistry_RegisterAndGet_Parallel(b *testing.B) {
	r := NewRegistry()
	// Pre-register some tools
	for i := 0; i < 5; i++ {
		_ = r.Register(&Tool{
			Name:    fmt.Sprintf("tool%d", i),
			Command: "echo",
			Enabled: true,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				_, _ = r.Get(fmt.Sprintf("tool%d", i%5))
			} else {
				_ = r.Register(&Tool{
					Name:    fmt.Sprintf("new-tool%d", i),
					Command: "echo",
					Enabled: true,
				})
			}
			i++
		}
	})
}

// Default registry benchmarks

func BenchmarkDefaultRegistry_Get(b *testing.B) {
	// Register a tool in the default registry
	_ = Register(&Tool{Name: "default-bench-tool", Command: "echo", Enabled: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Get("default-bench-tool")
	}
}

func BenchmarkDefaultRegistry_List(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = List()
	}
}

func BenchmarkDefaultRegistry_ListEnabled(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ListEnabled()
	}
}
