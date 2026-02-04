# Audit Report: internal/tui

**Package:** `internal/tui/` (home.go, workspace.go, agent.go, actions.go)
**Related:** `pkg/tui/style/theme.go`
**Auditor:** worker-03
**Date:** 2026-02-05
**Work Item:** work-012, bead bc-34b.7

## Summary

The `internal/tui` package implements a Bubble Tea-based TUI for the `bc home` command. All referenced screen files exist and are implemented. The package has **zero tests**. This audit identified **5 issues** ranging from low to high severity, plus architectural observations.

---

## Issue 1: Home Screen Refresh is a No-Op

**Severity:** HIGH

**Description:**
Pressing 'r' on the home screen sets a status message but does not actually refresh workspace data.

**Problematic Code:**
```go
// internal/tui/home.go:159-161
case "r":
    m.statusMsg = "Refreshed"
    // No actual data refresh happens!
```

**Expected Behavior:**
The 'r' key should reload workspace data (agent counts, issue counts, etc.) similar to how `WorkspaceModel.refresh()` works.

**Impact:**
Users expect 'r' to refresh data. The "Refreshed" status message is misleading.

**Recommendation:**
```go
case "r":
    // Refresh all workspace data
    for i, ws := range m.workspaces {
        mgr := agent.NewWorkspaceManager(ws.Entry.Path+"/.bc/agents", ws.Entry.Path)
        mgr.LoadState()
        mgr.RefreshState()
        m.workspaces[i].Running = mgr.RunningCount()
        m.workspaces[i].Total = mgr.AgentCount()
        m.workspaces[i].Issues = len(beads.ListIssues(ws.Entry.Path))
    }
    m.statusMsg = "Refreshed"
```

---

## Issue 2: Issues and PRs Tabs Have No Drill-Down

**Severity:** MEDIUM

**Description:**
`ActionDrillIssue` is defined in `actions.go` but never used. Pressing Enter on Issues or PRs tabs does nothing.

**Problematic Code:**
```go
// internal/tui/actions.go:10
ActionDrillIssue  // Defined but unused

// internal/tui/workspace.go:135-143
func (m *WorkspaceModel) selectCurrent() Action {
    switch m.tab {
    case TabAgents:
        if m.cursor < len(m.agents) {
            return Action{Type: ActionDrillAgent, Data: m.agents[m.cursor]}
        }
    }
    // TabIssues and TabPRs cases missing!
    return NoAction
}
```

**Impact:**
- Users cannot view issue or PR details from the TUI
- Feature appears incomplete

**Recommendation:**
Implement issue and PR detail views, or add placeholder actions with status message explaining the feature is not yet available.

---

## Issue 3: Agent Model Not Refreshed on Tick

**Severity:** MEDIUM

**Description:**
The `TickMsg` handler refreshes `wsModel` but not `agentModel`, so agent details become stale while viewing.

**Problematic Code:**
```go
// internal/tui/home.go:96-100
case TickMsg:
    if m.wsModel != nil {
        m.wsModel.refresh()
    }
    // m.agentModel not refreshed!
    return m, tickCmd()
```

**Impact:**
When viewing an agent's details, the state, task, and uptime become stale.

**Recommendation:**
```go
case TickMsg:
    if m.wsModel != nil {
        m.wsModel.refresh()
    }
    if m.agentModel != nil {
        // Re-fetch agent data
        m.agentModel.agent = m.wsModel.manager.GetAgent(m.agentModel.agent.Name)
    }
    return m, tickCmd()
```

---

## Issue 4: Potential Nil Pointer in Agent Session Display

**Severity:** LOW

**Description:**
`AgentModel.View()` calls `m.manager.Tmux().SessionName()` without nil checks.

**Problematic Code:**
```go
// internal/tui/agent.go:93
{"Session", m.manager.Tmux().SessionName(m.agent.Session), "code"},
```

**Risk:**
If `Tmux()` returns nil (e.g., tmux not configured), this will panic.

**Recommendation:**
Add nil check or ensure `Tmux()` never returns nil.

---

## Issue 5: StatusStyle Mapping Inconsistent with Agent States

**Severity:** LOW

**Description:**
The `mapState` function in workspace.go maps `StateIdle` to "info" and `StateStopped` to "stopped", but `StatusStyle` in theme.go maps "stopped" to `Info` style and "idle" to `Info` style - both result in the same color.

**Code Reference:**
```go
// internal/tui/workspace.go:351-368
func mapState(s agent.State) string {
    switch s {
    case agent.StateIdle:
        return "info"      // Maps to Info style (blue)
    case agent.StateStopped:
        return "stopped"   // Also maps to Info style (blue)
    // ...
    }
}

// pkg/tui/style/theme.go:158-159
case "info", "idle", "stopped":
    return s.Info
```

**Impact:**
Idle and stopped agents are visually indistinguishable. This may be intentional but could confuse users.

**Recommendation:**
Consider using different visual styles for idle (agent exists, ready) vs stopped (agent terminated).

---

## Screen Existence Verification

All referenced screens exist and are implemented:

| File | Status | Lines | Completeness |
|------|--------|-------|--------------|
| `home.go` | Exists | 338 | Fully implemented |
| `workspace.go` | Exists | 386 | Mostly complete (missing issue/PR drill-down) |
| `agent.go` | Exists | 153 | Fully implemented |
| `actions.go` | Exists | 23 | Complete (has unused ActionDrillIssue) |

---

## Navigation Flow Analysis

```
HomeScreen
    │
    ├── [j/k] Navigate workspaces
    ├── [enter] → WorkspaceScreen
    │                 │
    │                 ├── [tab/shift+tab] Switch tabs (Agents/Issues/PRs)
    │                 ├── [j/k] Navigate items
    │                 ├── [enter] → AgentScreen (only on Agents tab)
    │                 │                 │
    │                 │                 ├── [p] Peek output
    │                 │                 ├── [a] Attach → Exits TUI
    │                 │                 └── [esc] → Back to WorkspaceScreen
    │                 │
    │                 └── [esc] → Back to HomeScreen
    │
    └── [q/ctrl+c] Quit
```

**Navigation Issues:**
1. Enter on Issues/PRs tabs does nothing (documented above)
2. No way to start/stop agents from TUI
3. No keyboard shortcut to jump directly home from agent view

---

## Architecture Assessment: Hardcoded TUI vs Streaming

### Current Approach: Polling-Based Static TUI

The TUI uses Bubble Tea with a tick-based refresh:
```go
// internal/tui/home.go:69-73
func tickCmd() tea.Cmd {
    return tea.Tick(config.Tui.RefreshInterval, func(time.Time) tea.Msg {
        return TickMsg{}
    })
}
```

**Pros:**
- Simple to implement and reason about
- No complex event plumbing required
- Works without persistent connections to agents
- Easy to test (deterministic refresh cycles)

**Cons:**
- Not real-time (updates only on tick interval)
- Inefficient: refreshes everything even when nothing changed
- Can miss rapid state changes between ticks
- Each refresh reloads all data (agents, issues, PRs, queue)

### Alternative: Streaming/Event-Driven Approach

An event-driven architecture would use channels or file watchers to push updates.

**Pros:**
- Instant updates when state changes
- More efficient (only processes actual changes)
- Better user experience for monitoring

**Cons:**
- Significantly more complex
- Requires event infrastructure (pub/sub, file watchers)
- Harder to test
- More failure modes (connection drops, event ordering)

### Recommendation

The current polling approach is **appropriate for the current scope**:
1. The TUI is for monitoring, not real-time critical operations
2. Complexity tradeoff favors simplicity
3. The refresh interval can be tuned if needed

Consider event-driven only if:
- Users report missing important state changes
- Refresh overhead becomes measurable
- Real-time agent output streaming is required

---

## Missing Functionality Summary

| Feature | Status | Priority |
|---------|--------|----------|
| Home screen refresh | Broken | High |
| Issue detail view | Not implemented | Medium |
| PR detail view | Not implemented | Medium |
| Agent real-time output streaming | Not implemented | Low |
| Start/stop agents from TUI | Not implemented | Low |
| Search/filter workspaces | Not implemented | Low |
| Keyboard shortcut customization | Not implemented | Low |

---

## No Tests

The package has zero test coverage. Key areas to test:
- Navigation state transitions
- Key handling for each screen
- Cursor bounds checking
- Tick refresh behavior
- Action propagation between screens

---

## Summary Table

| Issue | Severity | Effort to Fix |
|-------|----------|---------------|
| Home refresh is no-op | HIGH | Low |
| Issues/PRs no drill-down | MEDIUM | Medium |
| Agent not refreshed on tick | MEDIUM | Low |
| Potential nil in session display | LOW | Low |
| State color mapping overlap | LOW | Low |

---

## Recommended Priority

1. **Immediate:** Fix home screen 'r' refresh to actually reload data
2. **Immediate:** Add agent refresh on tick
3. **Short-term:** Implement issue/PR drill-down or remove misleading tabs
4. **Short-term:** Add nil checks for tmux session
5. **Ongoing:** Add unit tests for navigation and state management
