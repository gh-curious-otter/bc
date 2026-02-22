package process

import (
	"fmt"
	"testing"
)

// --- Benchmark helpers ---

// newBenchRegistry creates a Registry with pre-populated processes for benchmarking.
func newBenchRegistry(b *testing.B, processCount int) *Registry {
	b.Helper()

	dir := b.TempDir()
	reg := NewRegistry(dir)
	if err := reg.Init(); err != nil {
		b.Fatal(err)
	}

	// Pre-populate with processes
	owners := []string{"engineer-01", "engineer-02", "manager-01", "qa-01"}
	for i := range processCount {
		p := &Process{
			Name:    fmt.Sprintf("proc-%d", i),
			Command: fmt.Sprintf("echo %d", i),
			PID:     1000 + i,
			Port:    8000 + i,
			Owner:   owners[i%len(owners)],
		}
		if err := reg.Register(p); err != nil {
			b.Fatal(err)
		}
	}

	return reg
}

// --- NewRegistry benchmarks ---

func BenchmarkNewRegistry(b *testing.B) {
	for range b.N {
		_ = NewRegistry("/tmp/bench")
	}
}

// --- Init benchmarks ---

func BenchmarkRegistryInit(b *testing.B) {
	dir := b.TempDir()

	b.ResetTimer()
	for range b.N {
		reg := NewRegistry(dir)
		if err := reg.Init(); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Register benchmarks ---

func BenchmarkRegistryRegister(b *testing.B) {
	dir := b.TempDir()
	reg := NewRegistry(dir)
	if err := reg.Init(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := range b.N {
		p := &Process{
			Name:    fmt.Sprintf("bench-proc-%d", i),
			Command: "echo test",
			PID:     i,
			Port:    10000 + i,
		}
		_ = reg.Register(p)
	}
}

// --- Get benchmarks ---

func BenchmarkRegistryGet_Small(b *testing.B) {
	reg := newBenchRegistry(b, 10)

	b.ResetTimer()
	for range b.N {
		_ = reg.Get("proc-5")
	}
}

func BenchmarkRegistryGet_Medium(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.Get("proc-50")
	}
}

func BenchmarkRegistryGet_Large(b *testing.B) {
	reg := newBenchRegistry(b, 1000)

	b.ResetTimer()
	for range b.N {
		_ = reg.Get("proc-500")
	}
}

func BenchmarkRegistryGet_Miss(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.Get("nonexistent")
	}
}

// --- List benchmarks ---

func BenchmarkRegistryList_Small(b *testing.B) {
	reg := newBenchRegistry(b, 10)

	b.ResetTimer()
	for range b.N {
		_ = reg.List()
	}
}

func BenchmarkRegistryList_Medium(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.List()
	}
}

func BenchmarkRegistryList_Large(b *testing.B) {
	reg := newBenchRegistry(b, 1000)

	b.ResetTimer()
	for range b.N {
		_ = reg.List()
	}
}

// --- ListByOwner benchmarks ---

func BenchmarkRegistryListByOwner_Small(b *testing.B) {
	reg := newBenchRegistry(b, 10)

	b.ResetTimer()
	for range b.N {
		_ = reg.ListByOwner("engineer-01")
	}
}

func BenchmarkRegistryListByOwner_Medium(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.ListByOwner("engineer-01")
	}
}

func BenchmarkRegistryListByOwner_Large(b *testing.B) {
	reg := newBenchRegistry(b, 1000)

	b.ResetTimer()
	for range b.N {
		_ = reg.ListByOwner("engineer-01")
	}
}

// --- MarkStopped benchmarks ---

func BenchmarkRegistryMarkStopped(b *testing.B) {
	dir := b.TempDir()
	reg := NewRegistry(dir)
	if err := reg.Init(); err != nil {
		b.Fatal(err)
	}

	// Create processes for each iteration
	for i := range b.N {
		p := &Process{
			Name:    fmt.Sprintf("stop-proc-%d", i),
			Command: "echo test",
			PID:     i,
		}
		if err := reg.Register(p); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := range b.N {
		_ = reg.MarkStopped(fmt.Sprintf("stop-proc-%d", i))
	}
}

// --- UpdatePID benchmarks ---

func BenchmarkRegistryUpdatePID(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for i := range b.N {
		_ = reg.UpdatePID("proc-50", i)
	}
}

// --- IsPortInUse benchmarks ---

func BenchmarkRegistryIsPortInUse_Hit(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.IsPortInUse(8050)
	}
}

func BenchmarkRegistryIsPortInUse_Miss(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.IsPortInUse(99999)
	}
}

// --- GetByPort benchmarks ---

func BenchmarkRegistryGetByPort_Hit(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.GetByPort(8050)
	}
}

func BenchmarkRegistryGetByPort_Miss(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = reg.GetByPort(99999)
	}
}

// --- Unregister benchmarks ---

func BenchmarkRegistryUnregister(b *testing.B) {
	dir := b.TempDir()
	reg := NewRegistry(dir)
	if err := reg.Init(); err != nil {
		b.Fatal(err)
	}

	// Create processes for each iteration
	for i := range b.N {
		p := &Process{
			Name:    fmt.Sprintf("unreg-proc-%d", i),
			Command: "echo test",
		}
		if err := reg.Register(p); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := range b.N {
		_ = reg.Unregister(fmt.Sprintf("unreg-proc-%d", i))
	}
}

// --- Clear benchmarks ---

func BenchmarkRegistryClear_Small(b *testing.B) {
	for range b.N {
		b.StopTimer()
		reg := newBenchRegistry(b, 10)
		b.StartTimer()
		_ = reg.Clear()
	}
}

func BenchmarkRegistryClear_Large(b *testing.B) {
	for range b.N {
		b.StopTimer()
		reg := newBenchRegistry(b, 100)
		b.StartTimer()
		_ = reg.Clear()
	}
}

// --- LogPath benchmarks ---

func BenchmarkRegistryLogPath(b *testing.B) {
	reg := NewRegistry("/tmp/bench")

	b.ResetTimer()
	for range b.N {
		_ = reg.LogPath("test-proc")
	}
}

// --- ProcessesDir benchmarks ---

func BenchmarkRegistryProcessesDir(b *testing.B) {
	reg := NewRegistry("/tmp/bench")

	b.ResetTimer()
	for range b.N {
		_ = reg.ProcessesDir()
	}
}

// --- Parallel benchmarks ---

func BenchmarkRegistryGet_Parallel(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = reg.Get(fmt.Sprintf("proc-%d", i%100))
			i++
		}
	})
}

func BenchmarkRegistryList_Parallel(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = reg.List()
		}
	})
}

func BenchmarkRegistryIsPortInUse_Parallel(b *testing.B) {
	reg := newBenchRegistry(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = reg.IsPortInUse(8000 + i%100)
			i++
		}
	})
}

func BenchmarkRegistryListByOwner_Parallel(b *testing.B) {
	reg := newBenchRegistry(b, 100)
	owners := []string{"engineer-01", "engineer-02", "manager-01", "qa-01"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = reg.ListByOwner(owners[i%len(owners)])
			i++
		}
	})
}
