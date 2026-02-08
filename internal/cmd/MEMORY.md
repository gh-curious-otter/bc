# bc CLI memory — profiling and reductions (#310)

## Profile summary

- **TUI benchmarks** (`go test -bench=. -benchmem ./internal/tui/...`): Dashboard tab view is the heaviest path (~10 KB/op, ~248 allocs/op per render). Other tabs and home view are in the 3–7 KB/op range. See `internal/tui/benchmark_test.go`.
- **CLI entry points**: `bc home` builds a workspace list (one Manager per workspace); `bc status` / `bc dashboard` load one Manager and agents once. `pkg/agent.ListAgents()` returns copies for thread safety and already pre-allocates the slice.
- **Config**: Loaded once at init (global vars in `config` package). No per-command config reload.

## Reductions applied

1. **internal/cmd/home.go**  
   Pre-allocate the workspaces slice: `workspaces := make([]itui.WorkspaceInfo, 0, len(reg.List()))` so the loop doesn’t cause multiple slice growths.

2. **internal/cmd/agent.go**  
   Pre-allocate the filtered agent list when filtering by role: `filtered := make([]*agent.Agent, 0, len(agents))` to avoid reallocations.

## Hotspots (documented, not changed)

- **ListAgents copies**: `pkg/agent.Manager.ListAgents()` allocates a new slice and agent copies by design to avoid data races. Callers should not hold large references longer than needed.
- **Dashboard view**: `WorkspaceModel.View()` for TabDashboard does more formatting and allocations than other tabs; acceptable for interactive use. Further reduction would require refactoring render helpers.
- **runHome workspace loop**: One Manager per workspace is created and discarded after counting; could be revisited if many workspaces cause high memory (e.g. lazy or pooled loading).

## Verification

- `go build ./...` and `go test ./...` pass.
- TUI benchmarks remain unchanged in behavior; allocs/op may be slightly lower on home path with pre-allocation.
