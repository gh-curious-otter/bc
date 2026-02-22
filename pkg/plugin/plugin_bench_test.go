package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Benchmark helpers ---

// newBenchManager creates a Manager with pre-populated plugins for benchmarking.
func newBenchManager(b *testing.B, pluginCount int) *Manager {
	b.Helper()

	dir := b.TempDir()
	mgr := NewManager(dir)

	// Pre-populate with plugins
	types := []string{TypeTool, TypeAgent, TypeRole, TypeHook}
	states := []string{StateEnabled, StateEnabled, StateDisabled, StateEnabled}

	for i := range pluginCount {
		name := fmt.Sprintf("plugin-%d", i)
		now := time.Now()
		mgr.plugins[name] = &Plugin{
			Manifest: Manifest{
				Name:        name,
				Version:     "1.0.0",
				Description: fmt.Sprintf("Test plugin %d for benchmarking", i),
				Author:      "test",
				Type:        types[i%len(types)],
				Entrypoint:  "main.go",
			},
			State:       states[i%len(states)],
			Path:        filepath.Join(dir, name),
			InstalledAt: now,
		}
	}

	return mgr
}

// --- NewManager benchmarks ---

func BenchmarkNewManager(b *testing.B) {
	for range b.N {
		_ = NewManager("/tmp/bench-workspace")
	}
}

// --- List benchmarks ---

func BenchmarkManagerList_Small(b *testing.B) {
	mgr := newBenchManager(b, 10)

	b.ResetTimer()
	for range b.N {
		_ = mgr.List()
	}
}

func BenchmarkManagerList_Medium(b *testing.B) {
	mgr := newBenchManager(b, 50)

	b.ResetTimer()
	for range b.N {
		_ = mgr.List()
	}
}

func BenchmarkManagerList_Large(b *testing.B) {
	mgr := newBenchManager(b, 200)

	b.ResetTimer()
	for range b.N {
		_ = mgr.List()
	}
}

func BenchmarkManagerList_Empty(b *testing.B) {
	mgr := NewManager("/tmp/bench")

	b.ResetTimer()
	for range b.N {
		_ = mgr.List()
	}
}

// --- Get benchmarks ---

func BenchmarkManagerGet_Hit(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_, _ = mgr.Get("plugin-50")
	}
}

func BenchmarkManagerGet_Miss(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_, _ = mgr.Get("nonexistent")
	}
}

// --- Info benchmarks ---

func BenchmarkManagerInfo_Hit(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_, _ = mgr.Info("plugin-50")
	}
}

func BenchmarkManagerInfo_Miss(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_, _ = mgr.Info("nonexistent")
	}
}

// --- Enabled benchmarks ---

func BenchmarkManagerEnabled_All(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = mgr.Enabled("")
	}
}

func BenchmarkManagerEnabled_ByType(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	for range b.N {
		_ = mgr.Enabled(TypeTool)
	}
}

func BenchmarkManagerEnabled_AllTypes(b *testing.B) {
	mgr := newBenchManager(b, 100)
	types := []string{TypeTool, TypeAgent, TypeRole, TypeHook, TypeCommand, TypeView}

	b.ResetTimer()
	for i := range b.N {
		_ = mgr.Enabled(types[i%len(types)])
	}
}

// --- validateManifest benchmarks ---

func BenchmarkValidateManifest_Valid(b *testing.B) {
	m := &Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    TypeTool,
	}

	b.ResetTimer()
	for range b.N {
		_ = validateManifest(m)
	}
}

func BenchmarkValidateManifest_WithPermissions(b *testing.B) {
	m := &Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    TypeTool,
		Permissions: &Permissions{
			Filesystem: "workspace",
			Network:    true,
			EnvVars:    []string{"HOME", "PATH", "USER"},
		},
	}

	b.ResetTimer()
	for range b.N {
		_ = validateManifest(m)
	}
}

func BenchmarkValidateManifest_Invalid(b *testing.B) {
	m := &Manifest{
		Name:    "",
		Version: "1.0.0",
		Type:    TypeTool,
	}

	b.ResetTimer()
	for range b.N {
		_ = validateManifest(m)
	}
}

// --- Load/Save benchmarks ---

func BenchmarkManagerLoad_Empty(b *testing.B) {
	dir := b.TempDir()
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		mgr := NewManager(dir)
		_ = mgr.Load(ctx)
	}
}

func BenchmarkManagerLoad_WithPlugins(b *testing.B) {
	dir := b.TempDir()
	ctx := context.Background()

	// Create initial state file
	pluginsDir := filepath.Join(dir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		b.Fatal(err)
	}
	stateData := `[{"manifest":{"name":"test1","version":"1.0.0","type":"tool"},"state":"enabled","path":"/tmp/t1"},
{"manifest":{"name":"test2","version":"1.0.0","type":"agent"},"state":"enabled","path":"/tmp/t2"},
{"manifest":{"name":"test3","version":"1.0.0","type":"role"},"state":"disabled","path":"/tmp/t3"}]`
	if err := os.WriteFile(filepath.Join(pluginsDir, "plugins.json"), []byte(stateData), 0600); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		mgr := NewManager(dir)
		_ = mgr.Load(ctx)
	}
}

func BenchmarkManagerSave(b *testing.B) {
	for range b.N {
		b.StopTimer()
		mgr := newBenchManager(b, 10)
		b.StartTimer()
		_ = mgr.Save()
	}
}

// --- Enable/Disable benchmarks ---

func BenchmarkManagerEnable(b *testing.B) {
	for range b.N {
		b.StopTimer()
		mgr := newBenchManager(b, 10)
		// Disable one for testing enable
		mgr.plugins["plugin-0"].State = StateDisabled
		b.StartTimer()
		_ = mgr.Enable("plugin-0")
	}
}

func BenchmarkManagerDisable(b *testing.B) {
	for range b.N {
		b.StopTimer()
		mgr := newBenchManager(b, 10)
		b.StartTimer()
		_ = mgr.Disable("plugin-0")
	}
}

// --- Install benchmarks ---

func BenchmarkManagerInstall(b *testing.B) {
	manifest := `name = "test-plugin"
version = "1.0.0"
description = "A test plugin"
type = "tool"
entrypoint = "main.go"
`
	for i := range b.N {
		b.StopTimer()
		// Setup fresh environment for each iteration
		tmpDir := b.TempDir()
		pluginDir := filepath.Join(tmpDir, fmt.Sprintf("plugin-%d", i))
		if err := os.MkdirAll(pluginDir, 0750); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "plugin.toml"), []byte(manifest), 0600); err != nil {
			b.Fatal(err)
		}
		mgr := NewManager(tmpDir)
		_ = mgr.Load(context.Background())
		b.StartTimer()

		_, _ = mgr.Install(context.Background(), pluginDir)
	}
}

// --- Uninstall benchmarks ---

func BenchmarkManagerUninstall(b *testing.B) {
	for range b.N {
		b.StopTimer()
		mgr := newBenchManager(b, 10)
		b.StartTimer()
		_ = mgr.Uninstall(context.Background(), "plugin-0")
	}
}

// --- Search benchmarks ---

func BenchmarkManagerSearch(b *testing.B) {
	mgr := NewManager("/tmp/bench")
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = mgr.Search(ctx, "test query")
	}
}

// --- Parallel benchmarks ---

func BenchmarkManagerGet_Parallel(b *testing.B) {
	mgr := newBenchManager(b, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = mgr.Get(fmt.Sprintf("plugin-%d", i%100))
			i++
		}
	})
}

func BenchmarkManagerList_Parallel(b *testing.B) {
	mgr := newBenchManager(b, 50)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mgr.List()
		}
	})
}

func BenchmarkManagerEnabled_Parallel(b *testing.B) {
	mgr := newBenchManager(b, 100)
	types := []string{TypeTool, TypeAgent, TypeRole, ""}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = mgr.Enabled(types[i%len(types)])
			i++
		}
	})
}

func BenchmarkValidateManifest_Parallel(b *testing.B) {
	m := &Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    TypeTool,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validateManifest(m)
		}
	})
}
