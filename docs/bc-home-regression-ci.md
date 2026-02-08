# bc home: regression tests and CI for performance (#313)

Plan for catching behavior and performance regressions in bc home TUI/dashboard (epic #311).

## Current state

### Behavior regression tests

- **Location:** `internal/tui/benchmark_test.go` (test functions only; no benchmarks required to pass).
- **Coverage:**
  - `TestHomeView_Regression_NoPanic` — Home view (empty, with workspaces, help active) does not panic.
  - `TestHomeView_Regression_ExpectedSections` — Home view contains expected labels/sections (bc, Workspaces, NAME, PATH, AGENTS).
  - `TestWorkspaceView_Regression_NoPanic` — Workspace view does not panic for every tab (Agents, Issues, Channels, Queue, Dashboard, Stats).
  - `TestWorkspaceView_Regression_ExpectedSections` — Workspace view has tab bar and Dashboard content (Issue Overview).
  - `TestHomeView_Regression_AllTabsRender` — Full flow (home + workspace screen) renders without panic.
- **CI:** These run in CI on every push/PR via the **Test** job (`make test`). Any failure fails the build.

### Benchmarks (performance)

- **Location:** `internal/tui/benchmark_test.go` (functions named `Benchmark*`).
- **Coverage:** HomeView (empty, with workspaces), WorkspaceView per tab (Agents, Issues, Channels, Queue, Dashboard, Stats).
- **Run locally:** `make bench` or `go test -bench=. -benchmem ./internal/tui/...`
- **CI:** A **Benchmark** job runs `go test -bench=. -benchmem ./internal/tui/...` on every push/PR and uploads the benchmark output as an artifact (TUI-only to avoid other packages’ tests, e.g. config). This does not fail the build (no threshold enforced yet); it provides a record for investigating performance regressions.

## How to run (actionable)

| Goal | Command | When |
|------|---------|------|
| Run all tests (including regression) | `make test` | Before commit; runs in CI |
| Run TUI regression tests only | `go test -run Regression ./internal/tui/...` | When changing TUI |
| Run benchmarks | `make bench` | Before release or when changing TUI; also in CI |
| Run TUI benchmarks only | `go test -bench=. -benchmem ./internal/tui/...` | When tuning TUI performance |

## Adding new regression tests or benchmarks

1. **Regression test (behavior):** Add a `Test*_Regression_*` in `internal/tui/benchmark_test.go` (or in the appropriate `*_test.go`). Use existing helpers (`newTestHomeModel()`, `newTestModel()`) to avoid disk I/O. CI will run it with `make test`.
2. **Benchmark (performance):** Add a `Benchmark*` in `internal/tui/benchmark_test.go`. Call `b.ResetTimer()` after setup so only the measured path is timed. CI will run it in the Benchmark job and include it in the uploaded artifact.

## Future: performance regression gates

- **Option A:** In CI, run `make bench`, parse output, and compare key metrics (ns/op, B/op, allocs/op) to a stored baseline; fail the build if regression exceeds a threshold (e.g. +20%).
- **Option B:** Periodic (e.g. nightly) benchmark job that publishes results; team reviews trends.
- **Option C:** Document “run `make bench` before release and investigate any large regressions” as a release checklist item.

Until one of these is implemented, the Benchmark CI job provides visibility without blocking merges.

## Summary

- **Behavior:** Regression tests are in place and run in CI; they block the build on failure.
- **Performance:** Benchmarks run in CI and results are stored as an artifact; no threshold yet.
- **Actionable:** Use `make test` and `make bench` as above; add tests/benchmarks in `internal/tui/benchmark_test.go` (or existing `*_test.go` for tests only).
