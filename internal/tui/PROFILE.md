# bc home / dashboard — profiling and fixes (#311)

## Root causes (profiling)

### Slowness

1. **Heavy work on every tick (2s)**  
   `HomeModel.Update(TickMsg)` called `WorkspaceModel.refresh()` and `AgentModel.refresh()` on the main Bubble Tea goroutine.  
   - **WorkspaceModel.refresh()** did a full reload every 2s: `RefreshState()` (tmux list + tmux capture for every running agent), `beads.ListIssues()`, `loadChannels()`, `loadMemoryInfo()` (memory store per agent), `loadQueue()` (includes `git rev-parse` per working agent), `loadRecentEvents()`, `computeStats()`, `loadPkgStats()`.  
   - **AgentModel.refresh()** did `RefreshState()`, `loadRecentActivity()`, `loadMemoryInfo()` every 2s.  
   Result: UI blocked every 2s on disk I/O and tmux/git subprocesses.

2. **Startup blocking** (fixed in #323)  
   Previously: `runHome` built the workspace list synchronously (LoadState + RefreshState per workspace) before starting the TUI. Now the TUI starts immediately with a “Loading workspaces…” state and loads workspace info in a goroutine; when done, a `WorkspacesLoadedMsg` updates the model. Time-to-first-frame is minimal (registry load + first paint only).

3. **captureLiveTask**  
   `RefreshState()` calls `captureLiveTask(name)` for every running agent, each doing `tmux.Capture(name, 15)`. With several agents this is multiple tmux invocations per refresh.

### Crashes

- No nil-dereference or index panics identified in the hot paths. `issuesByAgent` is initialized in `computeStats()` before use; reading from a nil map in Go returns zero.

## Fixes applied

1. **Lightweight tick refresh**  
   - **WorkspaceModel**: On `TickMsg`, call new `refreshLight()` instead of `refresh()`.  
     `refreshLight()` does only: `RefreshState()`, `ListAgents()`, `computeStats()`, cursor clamp.  
     Full reload (issues, channels, queue, events, memory, pkg stats) runs only on explicit 'r' via `refresh()`.  
   - **AgentModel**: On `TickMsg`, call new `refreshLight()` instead of `refresh()`.  
     `refreshLight()` does only: `RefreshState()`, `GetAgent()` to refresh in-memory agent, no file I/O.  
     Full reload (recent activity, memory info) runs only on 'r' via `refresh()`.  
   Effect: TUI stays responsive; agent state and task still update every 2s; heavy I/O only on manual refresh.

2. **Documentation**  
   This file records root causes and the above fixes for future profiling work.

3. **Per-screen lazy-load (#324 / epic #322)**  
   - **Workspace**: `NewWorkspaceModel` loads only manager + agents. Issues, channels, queue, events, stats load on first focus of each tab via `ensureTabDataLoaded(tab)` in `View()`. Stats bar shows zeros until Issues or Dashboard is loaded; full reload on 'r' sets all flags.  
   - **Agent**: `NewAgentModel` no longer calls `loadRecentActivity()` / `loadMemoryInfo()`; `ensureHeavyDataLoaded()` runs on first `View()`.  
   - **Channel**: `store.Load()` is deferred from home drill-down to first `ChannelModel.View()`; first paint loads store and refreshes channel.  
   Effect: Opening a workspace shows Agents tab immediately; heavy data loads when user switches to that tab or screen.

## Time-to-first-frame (#323)

With non-blocking startup, time-to-first-frame is the time until the first `View()` is rendered (registry load + first paint). No per-workspace I/O runs before the TUI is shown. Measure with:

```bash
go test -bench=BenchmarkHomeView_AsyncLoadFirstFrame -benchmem ./internal/tui/...
```

## Possible follow-ups

- ~~**Throttle captureLiveTask**~~: Done (epic #322). Manager now captures live task for one agent per `RefreshState` (round-robin) instead of all agents, reducing tmux.Capture calls when many agents run.

## Fixes applied (non-blocking startup, #303 / #311)

- **Startup**: TUI now appears immediately with “Loading workspaces…”. Workspace list is built in a background goroutine (`loadWorkspacesAndSend`); when done, `WorkspacesLoadedMsg` updates the model so the list appears without blocking.
- **Crash hardening**: `refreshWorkspaces()` guards on nil `m.workspaces` and wraps each workspace update in a recover so a bad path or beads panic cannot crash the TUI.
