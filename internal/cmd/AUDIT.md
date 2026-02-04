# internal/cmd Audit Report

**Auditor:** worker-02
**Date:** 2026-02-05
**Files Reviewed:** 15 files in internal/cmd/
**Test Coverage:** Only root.go has tests (3 basic tests)

---

## Executive Summary

The `internal/cmd` package implements the bc CLI commands using Cobra. While generally functional, there are several inconsistencies in error handling, workspace detection, and UX patterns across commands. The most significant issues relate to silent error ignoring, inconsistent agent state checking, and potential data loss scenarios.

---

## Issues Found

### 1. Silent Error Ignoring in `init` (Severity: HIGH)

**Location:** `init.go:62-66`

Registry errors are silently ignored, which means workspace registration can fail without user awareness:

```go
// Register in global registry
reg, err := workspace.LoadRegistry()
if err == nil {
    reg.Register(ws.RootDir, ws.Config.Name)
    reg.Save()  // Error ignored!
}
```

**Impact:** User thinks workspace is registered but `bc home` won't show it.

**Recommendation:**
```go
reg, err := workspace.LoadRegistry()
if err != nil {
    fmt.Printf("Warning: couldn't register workspace globally: %v\n", err)
} else {
    reg.Register(ws.RootDir, ws.Config.Name)
    if err := reg.Save(); err != nil {
        fmt.Printf("Warning: couldn't save registry: %v\n", err)
    }
}
```

---

### 2. Error Ignored in Queue Operations (Severity: HIGH)

**Location:** `queue.go:61-64`, `report.go:100`

The `loadQueue` helper ignores load errors, and `q.Save()` error is ignored in report:

```go
func loadQueue(ws interface{ StateDir() string }) *queue.Queue {
    q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
    q.Load()  // Error ignored - queue could be corrupt
    return q
}
```

```go
// report.go:100
q.Save()  // Error ignored!
```

**Impact:** Corrupted queue files go unnoticed; state changes may be lost.

**Recommendation:** Return error from loadQueue, handle it in callers.

---

### 3. `down --force` Flag Is Unused (Severity: MEDIUM)

**Location:** `down.go:26, 30`

The `--force` flag is defined but never read:

```go
var downForce bool

func init() {
    downCmd.Flags().BoolVar(&downForce, "force", false, "Force kill without cleanup")
    // ...
}

func runDown(cmd *cobra.Command, args []string) error {
    // downForce is never used!
```

**Impact:** Documented feature doesn't work; user expectations not met.

**Recommendation:** Implement force kill logic or remove the flag.

---

### 4. Inconsistent Agent Existence Checks (Severity: MEDIUM)

**Location:** `attach.go:42-44` vs `send.go:46-53`

Different commands check agent existence differently:

```go
// attach.go - checks tmux session directly
if !mgr.Tmux().HasSession(agentName) {
    return fmt.Errorf("agent '%s' not running (session bc-%s not found)", agentName, agentName)
}

// send.go - loads state and checks agent object
mgr.LoadState()
a := mgr.GetAgent(agentName)
if a == nil {
    return fmt.Errorf("agent '%s' not found", agentName)
}
if a.State == agent.StateStopped {
    return fmt.Errorf("agent '%s' is stopped", agentName)
}
```

**Impact:** Inconsistent error messages; attach works when send fails or vice versa.

**Recommendation:** Create a helper function for consistent agent validation.

---

### 5. `attach` Error Message Shows Wrong Session Name (Severity: MEDIUM)

**Location:** `attach.go:43`

The error message hardcodes "bc-" prefix, ignoring workspace hash:

```go
return fmt.Errorf("agent '%s' not running (session bc-%s not found)", agentName, agentName)
```

**Impact:** Misleading error message when using workspace-scoped managers.

**Recommendation:**
```go
return fmt.Errorf("agent '%s' not running (session %s not found)",
    agentName, mgr.Tmux().SessionName(agentName))
```

---

### 6. Magic Sleep Values in `up` (Severity: MEDIUM)

**Location:** `up.go:108, 131`

Hardcoded delays without documentation:

```go
// Give coordinator time to initialize
time.Sleep(500 * time.Millisecond)  // Why 500ms?

// Small delay between spawns
time.Sleep(300 * time.Millisecond)  // Why 300ms?
```

**Impact:** May be too short on slow systems; wasteful on fast systems.

**Recommendation:** Make configurable or add proper readiness checks.

---

### 7. `home` Hardcodes Agent Path (Severity: MEDIUM)

**Location:** `home.go:64-66`

Hardcodes `.bc/agents` path instead of using workspace method:

```go
mgr := agent.NewWorkspaceManager(
    entry.Path+"/.bc/agents",  // Hardcoded!
    entry.Path,
)
```

**Recommendation:**
```go
ws, err := workspace.Load(entry.Path)
if err == nil {
    mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
    // ...
}
```

---

### 8. `queue assign` Doesn't Verify Agent Exists (Severity: MEDIUM)

**Location:** `queue.go:139-166`

Allows assigning work to non-existent agents:

```go
func runQueueAssign(cmd *cobra.Command, args []string) error {
    // ...
    agentName := args[1]
    // No check if agent exists!
    if err := q.Assign(itemID, agentName); err != nil {
```

**Impact:** Work assigned to typo'd agent names; never gets done.

**Recommendation:** Validate agent exists before assignment.

---

### 9. `logs` Filter Logic Bug (Severity: LOW)

**Location:** `logs.go:47-53, 64-66`

When both `--agent` and `--tail` are specified, tail is applied twice:

```go
if logsAgent != "" {
    evts, err = log.ReadByAgent(logsAgent)  // Returns all for agent
} else if logsTail > 0 {
    evts, err = log.ReadLast(logsTail)
} else {
    evts, err = log.Read()
}

// Apply tail after agent filter if both are set
if logsAgent != "" && logsTail > 0 && len(evts) > logsTail {
    evts = evts[len(evts)-logsTail:]  // Applied again
}
```

**Impact:** Minor - works correctly but inefficient (loads all then truncates).

**Recommendation:** Pass tail parameter to ReadByAgent if needed.

---

### 10. `report` Only Updates First Matching Work Item (Severity: LOW)

**Location:** `report.go:75-99`

When an agent has multiple assigned items, only the first matching status gets updated:

```go
for _, item := range agentItems {
    switch state {
    case agent.StateWorking:
        if item.Status == queue.StatusAssigned {
            q.UpdateStatus(item.ID, queue.StatusWorking)
            // continues loop, updates all - actually OK
```

Actually this is fine on closer inspection - it updates all items. No issue here.

---

### 11. `ui` Demo Ignores Pipe Errors (Severity: LOW)

**Location:** `ui.go:54-55`

Pipe creation errors are ignored:

```go
aiToTUI, tuiInput, _ := os.Pipe()
tuiOutput, tuiToAI, _ := os.Pipe()
```

**Impact:** Demo mode could fail mysteriously on resource-constrained systems.

**Recommendation:** Check pipe errors and return meaningful error message.

---

### 12. No Validation of Agent Names (Severity: LOW)

**Location:** Multiple files

Agent names are passed directly without validation. Special characters could cause issues:

```go
// attach.go
agentName := args[0]  // Could be "../../../etc/passwd" or "foo;rm -rf /"

// send.go
agentName := args[0]  // Same issue
```

**Impact:** While tmux will reject invalid names, error messages will be confusing.

**Recommendation:** Validate agent names match expected pattern (alphanumeric, dash, underscore).

---

### 13. `dashboard` Doesn't Handle ReadLast Error (Severity: LOW)

**Location:** `dashboard.go:52`

Error from ReadLast is silently discarded:

```go
recentEvents, _ := log.ReadLast(10)
```

**Impact:** Events section shows empty without explanation if log is corrupted.

**Recommendation:** Log a warning or show "Error loading events" message.

---

### 14. Color Codes Not Disabled for Non-TTY (Severity: LOW)

**Location:** `status.go:122-149`, `queue.go:212-238`, `dashboard.go:190-209`

ANSI color codes are always output, even when stdout isn't a terminal:

```go
func colorState(s agent.State) string {
    const (
        reset  = "\033[0m"
        green  = "\033[32m"
        // Always uses colors, even when piped
```

**Impact:** Garbled output when piping to files or other commands.

**Recommendation:** Check `term.IsTerminal(os.Stdout.Fd())` before applying colors.

---

## UX Inconsistencies

| Area | Inconsistency |
|------|---------------|
| **Error format** | Some use `fmt.Errorf("not in a bc workspace: %w", err)`, others just `err` |
| **Success output** | Some print "✓", others don't; some print "Done", others silent |
| **Help hints** | `status` and `queue` show hints; `attach`, `send`, `report` don't |
| **JSON output** | `--json` works for `status`, `logs`, `queue`, `dashboard`; not for others |
| **Workspace check** | All commands check workspace except `home` (which uses registry) |

---

## Commands Missing Tests

| Command | Test Coverage | Priority |
|---------|---------------|----------|
| init | 0% | High - creates state |
| up | 0% | High - spawns processes |
| down | 0% | High - kills processes |
| status | 0% | Medium |
| attach | 0% | Medium |
| send | 0% | High - sends user data |
| logs | 0% | Low |
| queue | 0% | Medium |
| report | 0% | High - modifies state |
| home | 0% | Low |
| dashboard | 0% | Low |

---

## Summary Table

| # | Issue | Severity | File | Line |
|---|-------|----------|------|------|
| 1 | Registry save error ignored | HIGH | init.go | 65 |
| 2 | Queue load/save errors ignored | HIGH | queue.go, report.go | 63, 100 |
| 3 | --force flag unused | MEDIUM | down.go | 26 |
| 4 | Inconsistent agent checks | MEDIUM | attach.go, send.go | various |
| 5 | Wrong session name in error | MEDIUM | attach.go | 43 |
| 6 | Magic sleep values | MEDIUM | up.go | 108, 131 |
| 7 | Hardcoded agent path | MEDIUM | home.go | 65 |
| 8 | No agent validation on assign | MEDIUM | queue.go | 146 |
| 9 | Inefficient logs filtering | LOW | logs.go | 64 |
| 10 | (removed - false positive) | - | - | - |
| 11 | Pipe errors ignored in demo | LOW | ui.go | 54-55 |
| 12 | No agent name validation | LOW | multiple | - |
| 13 | ReadLast error ignored | LOW | dashboard.go | 52 |
| 14 | Colors without TTY check | LOW | multiple | - |

---

## Recommendations

1. **Add error handling** - Never ignore errors; at minimum log warnings
2. **Create helper functions** - Standardize workspace finding, agent validation
3. **Add TTY detection** - Disable colors when not outputting to terminal
4. **Implement --force** - Or remove the flag to avoid confusion
5. **Add tests** - Prioritize `up`, `down`, `send`, `report` for state modification
6. **Standardize UX** - Consistent success/error messages across all commands
7. **Validate inputs** - Agent names, work IDs, states should be validated
