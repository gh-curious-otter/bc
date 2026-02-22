package plugin

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

// BenchmarkValidateManifest measures manifest validation performance.
func BenchmarkValidateManifest(b *testing.B) {
	//nolint:govet // fieldalignment: test struct ordering for readability
	manifests := []struct {
		name string
		m    Manifest
	}{
		{
			name: "valid-minimal",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    TypeTool,
			},
		},
		{
			name: "valid-full",
			m: Manifest{
				Name:        "test-plugin",
				Version:     "1.0.0",
				Description: "A comprehensive test plugin for benchmarking",
				Author:      "Test Author",
				License:     "MIT",
				Homepage:    "https://example.com",
				Repository:  "https://github.com/example/plugin",
				Type:        TypeAgent,
				Entrypoint:  "main.go",
				BCVersion:   ">=1.0.0",
				Capabilities: []string{
					"create_agents",
					"assign_work",
					"implement_tasks",
				},
				Dependencies: []Dependency{
					{Name: "dep1", Version: "1.0.0"},
					{Name: "dep2", Version: "2.0.0"},
				},
			},
		},
		{
			name: "with-permissions",
			m: Manifest{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    TypeHook,
				Permissions: &Permissions{
					EnvVars:    []string{"HOME", "PATH", "BC_WORKSPACE"},
					Filesystem: "workspace",
					Network:    true,
				},
			},
		},
	}

	for _, tc := range manifests {
		b.Run(tc.name, func(b *testing.B) {
			m := tc.m
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = validateManifest(&m)
			}
		})
	}
}

// BenchmarkValidateManifestInvalid measures validation of invalid manifests.
func BenchmarkValidateManifestInvalid(b *testing.B) {
	//nolint:govet // fieldalignment: test struct ordering for readability
	manifests := []struct {
		name string
		m    Manifest
	}{
		{
			name: "missing-name",
			m: Manifest{
				Version: "1.0.0",
				Type:    TypeTool,
			},
		},
		{
			name: "invalid-type",
			m: Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    "invalid",
			},
		},
		{
			name: "invalid-filesystem",
			m: Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    TypeTool,
				Permissions: &Permissions{
					Filesystem: "invalid",
				},
			},
		},
	}

	for _, tc := range manifests {
		b.Run(tc.name, func(b *testing.B) {
			m := tc.m
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = validateManifest(&m)
			}
		})
	}
}

// BenchmarkManagerList measures listing all plugins.
func BenchmarkManagerList(b *testing.B) {
	sizes := []int{0, 10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("plugins-%d", size), func(b *testing.B) {
			mgr := NewManager("/tmp/test-workspace")

			// Populate with mock plugins
			for i := 0; i < size; i++ {
				name := fmt.Sprintf("plugin-%03d", i)
				mgr.plugins[name] = &Plugin{
					Manifest: Manifest{
						Name:    name,
						Version: "1.0.0",
						Type:    TypeTool,
					},
					State:       StateEnabled,
					Path:        fmt.Sprintf("/tmp/plugins/%s", name),
					InstalledAt: time.Now(),
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = mgr.List()
			}
		})
	}
}

// BenchmarkManagerGet measures getting a plugin by name.
func BenchmarkManagerGet(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add 50 plugins
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("plugin-%03d", i)
		mgr.plugins[name] = &Plugin{
			Manifest: Manifest{
				Name:    name,
				Version: "1.0.0",
				Type:    TypeTool,
			},
			State: StateEnabled,
		}
	}

	b.Run("existing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.Get("plugin-025")
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.Get("nonexistent-plugin")
		}
	})
}

// BenchmarkManagerEnabled measures filtering enabled plugins.
func BenchmarkManagerEnabled(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	// Add plugins with mixed states and types
	types := []string{TypeTool, TypeAgent, TypeRole, TypeHook, TypeCommand}
	states := []string{StateEnabled, StateDisabled, StateEnabled, StateEnabled, StateError}

	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("plugin-%03d", i)
		mgr.plugins[name] = &Plugin{
			Manifest: Manifest{
				Name:    name,
				Version: "1.0.0",
				Type:    types[i%len(types)], //nolint:gosec // index bounded by modulo
			},
			State: states[i%len(states)], //nolint:gosec // index bounded by modulo
		}
	}

	b.Run("all-types", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.Enabled("")
		}
	})

	b.Run("type-tool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.Enabled(TypeTool)
		}
	})

	b.Run("type-agent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.Enabled(TypeAgent)
		}
	})

	b.Run("type-hook", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mgr.Enabled(TypeHook)
		}
	})
}

// BenchmarkManagerInfo measures getting plugin info.
func BenchmarkManagerInfo(b *testing.B) {
	mgr := NewManager("/tmp/test-workspace")

	mgr.plugins["test-plugin"] = &Plugin{
		Manifest: Manifest{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "A test plugin",
			Author:      "Test Author",
			Type:        TypeTool,
		},
		State:       StateEnabled,
		Path:        "/tmp/test-plugin",
		InstalledAt: time.Now(),
	}

	b.Run("existing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.Info("test-plugin")
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mgr.Info("nonexistent")
		}
	})
}

// BenchmarkPluginCreation measures Plugin struct creation.
func BenchmarkPluginCreation(b *testing.B) {
	now := time.Now()

	b.Run("minimal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Plugin{
				Manifest: Manifest{
					Name:    "test",
					Version: "1.0.0",
					Type:    TypeTool,
				},
				State: StateEnabled,
			}
		}
	})

	b.Run("full", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Plugin{
				Manifest: Manifest{
					Name:        "test-plugin",
					Version:     "1.0.0",
					Description: "A comprehensive test plugin",
					Author:      "Test Author",
					License:     "MIT",
					Type:        TypeAgent,
					Entrypoint:  "main.go",
					Capabilities: []string{
						"create_agents",
						"assign_work",
					},
				},
				State:       StateEnabled,
				Path:        "/tmp/plugins/test-plugin",
				InstalledAt: now,
			}
		}
	})
}

// BenchmarkManifestCreation measures Manifest struct creation.
func BenchmarkManifestCreation(b *testing.B) {
	b.Run("minimal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    TypeTool,
			}
		}
	})

	b.Run("with-hooks", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    TypeHook,
				Hooks: map[string]HookDef{
					"agent.start":  {Script: "hooks/start.sh", Description: "On agent start"},
					"agent.stop":   {Script: "hooks/stop.sh", Description: "On agent stop"},
					"channel.send": {Script: "hooks/send.sh", Description: "On message send"},
				},
			}
		}
	})

	b.Run("with-commands", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Manifest{
				Name:    "test",
				Version: "1.0.0",
				Type:    TypeCommand,
				Commands: map[string]CommandDef{
					"sync":   {Script: "cmd/sync.sh", Description: "Sync data"},
					"export": {Script: "cmd/export.sh", Description: "Export data"},
				},
			}
		}
	})
}

// BenchmarkHookEventCreation measures HookEvent struct creation.
func BenchmarkHookEventCreation(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = HookEvent{
				Name:      "agent.start",
				Timestamp: time.Now(),
			}
		}
	})

	b.Run("with-payload", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = HookEvent{
				Name:      "agent.start",
				Timestamp: time.Now(),
				Payload: map[string]interface{}{
					"agent_name": "eng-01",
					"role":       "engineer",
					"workspace":  "/path/to/workspace",
				},
			}
		}
	})
}

// BenchmarkSearchResultCreation measures SearchResult struct creation.
func BenchmarkSearchResultCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SearchResult{
			Name:        "example-plugin",
			Version:     "1.0.0",
			Description: "An example plugin for demonstration",
			Author:      "Example Author",
			Type:        TypeTool,
			Tags:        []string{"tool", "utility", "automation"},
			Downloads:   1500,
			Stars:       42,
		}
	}
}
