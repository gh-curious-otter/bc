package process

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkNewRegistry measures registry creation.
func BenchmarkNewRegistry(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewRegistry(tmpDir)
	}
}

// BenchmarkRegistryInit measures registry initialization.
func BenchmarkRegistryInit(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewRegistry(tmpDir)
		_ = r.Init()
	}
}

// BenchmarkRegistryRegister measures process registration.
func BenchmarkRegistryRegister(b *testing.B) {
	tmpDir := b.TempDir()
	r := NewRegistry(tmpDir)
	if err := r.Init(); err != nil {
		b.Fatalf("failed to init registry: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := &Process{
			Name:      fmt.Sprintf("process-%d", i),
			Command:   "sleep 1",
			PID:       i + 1000,
			StartedAt: time.Now(),
		}
		_ = r.Register(p)
	}
}

// BenchmarkRegistryGet measures process lookup.
func BenchmarkRegistryGet(b *testing.B) {
	tmpDir := b.TempDir()
	r := NewRegistry(tmpDir)
	if err := r.Init(); err != nil {
		b.Fatalf("failed to init registry: %v", err)
	}

	// Register a process
	p := &Process{
		Name:    "test-process",
		Command: "sleep 1",
		PID:     1234,
	}
	if err := r.Register(p); err != nil {
		b.Fatalf("failed to register process: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Get("test-process")
	}
}

// BenchmarkRegistryList measures process listing.
func BenchmarkRegistryList(b *testing.B) {
	counts := []int{10, 50, 100}

	for _, count := range counts {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			tmpDir := b.TempDir()
			r := NewRegistry(tmpDir)
			if err := r.Init(); err != nil {
				b.Fatalf("failed to init registry: %v", err)
			}

			// Register processes
			for i := 0; i < count; i++ {
				p := &Process{
					Name:    fmt.Sprintf("process-%d", i),
					Command: "sleep 1",
					PID:     i + 1000,
				}
				_ = r.Register(p)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = r.List()
			}
		})
	}
}

// BenchmarkRegistryUnregister measures process removal.
func BenchmarkRegistryUnregister(b *testing.B) {
	tmpDir := b.TempDir()
	r := NewRegistry(tmpDir)
	if err := r.Init(); err != nil {
		b.Fatalf("failed to init registry: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		name := fmt.Sprintf("process-%d", i)
		p := &Process{
			Name:    name,
			Command: "sleep 1",
			PID:     i + 1000,
		}
		_ = r.Register(p)
		b.StartTimer()
		_ = r.Unregister(name)
	}
}

// BenchmarkRegistryListByOwner measures filtering by owner.
func BenchmarkRegistryListByOwner(b *testing.B) {
	tmpDir := b.TempDir()
	r := NewRegistry(tmpDir)
	if err := r.Init(); err != nil {
		b.Fatalf("failed to init registry: %v", err)
	}

	// Register processes with different owners
	owners := []string{"eng-01", "eng-02", "eng-03", "mgr-01"}
	for i := range 40 {
		ownerIdx := i % len(owners)
		p := &Process{
			Name:    fmt.Sprintf("process-%d", i),
			Command: "sleep 1",
			PID:     i + 1000,
			Owner:   owners[ownerIdx], //nolint:gosec // index is always in bounds
		}
		_ = r.Register(p)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ListByOwner("eng-01")
	}
}

// BenchmarkRegistryGetConcurrent measures concurrent lookups.
func BenchmarkRegistryGetConcurrent(b *testing.B) {
	tmpDir := b.TempDir()
	r := NewRegistry(tmpDir)
	if err := r.Init(); err != nil {
		b.Fatalf("failed to init registry: %v", err)
	}

	// Register processes
	for i := 0; i < 10; i++ {
		p := &Process{
			Name:    fmt.Sprintf("process-%d", i),
			Command: "sleep 1",
			PID:     i + 1000,
		}
		_ = r.Register(p)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = r.Get(fmt.Sprintf("process-%d", i%10))
			i++
		}
	})
}
