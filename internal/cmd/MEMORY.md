# bc CLI memory — profiling and reductions (#310)

## Profile summary

- **TUI benchmarks** (`go test -bench=. -benchmem ./internal/tui/...`): Dashboard tab view is the heaviest path (~10 KB/op, ~248 allocs/op per render). Other tabs and home view are in the 3–7 KB/op range. See `internal/tui/benchmark_test.go`.
- **CLI entry points**: `bc home` builds a minimal workspace list (no Manager/agent load at startup; TUI fills counts on first tick). `bc status` / `bc home` load one Manager and agents once. `pkg/agent.ListAgents()` returns copies for thread safety and already pre-allocates the slice.
- **Config**: Loaded once at init (global vars in `config` package). No per-command config reload.

## Reductions applied

1. **internal/cmd/home.go**  
   Pre-allocate the workspaces slice: `workspaces := make([]itui.WorkspaceInfo, 0, len(reg.List()))` so the loop doesn’t cause multiple slice growths.

2. **internal/cmd/agent.go**  
   Pre-allocate the filtered agent list when filtering by role: `filtered := make([]*agent.Agent, 0, len(agents))` to avoid reallocations.

3. **internal/cmd/home.go + internal/tui/home.go (#310)**  
   Defer workspace agent/beads loading: `runHome` no longer creates a Manager per workspace or calls `LoadState()`/`RefreshState()` at CLI startup. It builds the workspace list with only `Entry` and `MaxWorkers` (zero counts). The TUI populates agent/issue counts on the first tick via `refreshWorkspaces()` when on the home screen (`homeWorkspacesRefreshed` ensures this runs once). Effect: lower memory and no blocking I/O before the TUI appears; counts show after the first refresh interval (~2s) or on manual `r` refresh.

## Hotspots (documented, not changed)

- **ListAgents copies**: `pkg/agent.Manager.ListAgents()` allocates a new slice and agent copies by design to avoid data races. Callers should not hold large references longer than needed.
- **Dashboard view**: `WorkspaceModel.View()` for TabDashboard does more formatting and allocations than other tabs; acceptable for interactive use. Further reduction would require refactoring render helpers.
- **refreshWorkspaces**: Still creates one Manager per workspace when run (on first tick or on `r`); runs only when on home screen, once per session plus manual refresh.

## Verification

- `go build ./...` and `go test ./...` pass.
- TUI benchmarks remain unchanged in behavior; allocs/op may be slightly lower on home path with pre-allocation.
