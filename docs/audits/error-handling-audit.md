# Audit Report: Error Handling and Observability

**Date:** 2026-02-05
**Auditor:** worker-01
**Bead:** bc-34b.11
**Scope:** Entire codebase error handling patterns

---

## Executive Summary

The codebase has **systemic error handling issues** across all packages. Errors are frequently swallowed, ignored, or lost. There is **no structured logging**, **no debug mode** (despite a flag being defined), and **no observability instrumentation**. The event log exists for user-facing audit trails, not debugging.

---

## Part 1: Silent Error Swallowing

### 1.1 CRITICAL: pkg/beads and pkg/github Return nil on Error

**Severity:** Critical

All list functions return `nil` instead of errors, making failures invisible.

**pkg/beads/beads.go:**
```go
// Line 46-48
output, err := cmd.Output()
if err != nil {
    return nil  // bd command failed silently
}

// Line 108-109
if err := json.Unmarshal(output, &issues); err != nil {
    return nil  // JSON parse failed silently
}
```

**pkg/github/github.go:**
```go
// Line 69-70
if err != nil {
    return nil  // gh command failed silently
}

// Line 74-76
if err := json.Unmarshal(output, &raw); err != nil {
    return nil  // JSON parse failed silently
}
```

**Impact:** When `bd` or `gh` fails (not installed, not authenticated, API error), the system appears to work but returns empty data. Users cannot diagnose problems.

---

### 1.2 HIGH: events.Log.Append() Errors Ignored Everywhere

**Severity:** High

The event log's `Append` method returns an error, but it is **never checked** across the entire codebase.

**Locations (all ignoring error):**
| File | Line |
|------|------|
| internal/cmd/up.go | 86, 102, 125 |
| internal/cmd/send.go | 62 |
| internal/cmd/queue.go | 158, 202 |
| internal/cmd/report.go | 81, 91, 103 |

**Example:**
```go
// internal/cmd/send.go:62
log.Append(events.Event{
    Type:    events.MessageSent,
    Agent:   agentName,
    Message: message,
})  // Error ignored!
```

**Impact:** If the events.jsonl file is read-only, disk is full, or path is invalid, events are silently lost with no indication.

---

### 1.3 HIGH: queue.Load() Errors Ignored

**Severity:** High

`q.Load()` return values are ignored in most places.

**Locations:**
| File | Line | Pattern |
|------|------|---------|
| internal/cmd/up.go | 68 | `q.Load()` |
| internal/cmd/dashboard.go | 47 | `q.Load()` |
| internal/cmd/report.go | 70 | `q.Load()` |
| internal/cmd/queue.go | 62 | `q.Load()` (via loadQueue) |

**Code:**
```go
// internal/cmd/queue.go:60-64
func loadQueue(ws interface{ StateDir() string }) *queue.Queue {
    q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
    q.Load()  // Error ignored!
    return q
}
```

**Impact:** Corrupted queue.json files cause silent data loss.

---

### 1.4 MEDIUM: queue.Save() Sometimes Ignored

**Severity:** Medium

Some `q.Save()` calls are checked, others are not.

**Ignored:**
```go
// internal/cmd/up.go:84
q.Save()  // Error ignored

// internal/cmd/report.go:100
q.Save()  // Error ignored
```

**Checked (good):**
```go
// internal/cmd/queue.go:131
if err := q.Save(); err != nil {
    return fmt.Errorf("failed to save queue: %w", err)
}
```

**Impact:** Inconsistent behavior - some operations fail silently, others report errors.

---

### 1.5 MEDIUM: Registry.Save() Errors Ignored

**Severity:** Medium

**Locations:**
```go
// internal/cmd/init.go:65
reg.Save()  // Error ignored

// internal/cmd/home.go:51
reg.Save()  // Error ignored
```

**Impact:** Workspace registration may silently fail.

---

## Part 2: Explicitly Ignored Errors

### 2.1 MEDIUM: tmux.KillSession Error Ignored

**Severity:** Medium

**pkg/agent/agent.go:130:**
```go
if m.tmux.HasSession(name) {
    _ = m.tmux.KillSession(name)  // Intentionally ignored
}
```

**Context:** This is during cleanup before respawn. However, the `_` pattern makes it invisible whether failures are expected or not.

**Recommendation:** At minimum, log at debug level.

---

### 2.2 HIGH: os.Pipe Errors Ignored

**Severity:** High

**internal/cmd/ui.go:54-55:**
```go
aiToTUI, tuiInput, _ := os.Pipe()
tuiOutput, tuiToAI, _ := os.Pipe()
```

**Impact:** If pipe creation fails (fd exhaustion), the program will crash later with a confusing error instead of a clear "failed to create pipe" message.

---

## Part 3: Missing Error Checks

### 3.1 MEDIUM: LoadState/RefreshState Errors Ignored

**Severity:** Medium

`mgr.LoadState()` and `mgr.RefreshState()` return errors that are never checked.

**Locations:**
| File | Lines |
|------|-------|
| internal/cmd/send.go | 43 |
| internal/cmd/down.go | 41 |
| internal/cmd/status.go | 38, 41 |
| internal/cmd/dashboard.go | 41, 42 |
| internal/cmd/report.go | 63 |
| internal/cmd/home.go | 68, 69 |
| internal/tui/workspace.go | 63, 64, 121 |

**Example:**
```go
// internal/cmd/status.go:37-42
mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
mgr.LoadState()     // Error ignored
mgr.RefreshState()  // Error ignored
```

**Impact:** Corrupted state files or tmux communication errors are invisible.

---

### 3.2 LOW: GlobalDir() Returns Empty String on Error

**Severity:** Low

**pkg/workspace/registry.go:25-30:**
```go
func GlobalDir() string {
    home, err := os.UserHomeDir()
    if err != nil {
        return ""  // Returns empty string, no error
    }
    return filepath.Join(home, ".bc")
}
```

**Impact:** Downstream code using `GlobalDir()` may fail with confusing path errors.

---

## Part 4: Observability Gaps

### 4.1 CRITICAL: --verbose Flag Defined But Never Used

**Severity:** Critical (misleading user interface)

**internal/cmd/root.go:58:**
```go
rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
```

**Usage:** Zero. The flag is never read with `GetBool("verbose")`.

**Impact:** Users expecting verbose output for debugging get nothing. This is misleading UX.

---

### 4.2 CRITICAL: No Structured Logging Library

**Severity:** Critical

The codebase has **no logging library** (logrus, zap, slog, etc.). All "logging" is:
- `fmt.Printf` for user output
- `events.Log.Append` for audit trail (not debugging)

**Impact:**
- No log levels (debug, info, warn, error)
- No structured fields for filtering
- No way to trace internal operations
- No timestamps on debug output
- No way to send logs to external systems

---

### 4.3 HIGH: Events Log is Audit Trail, Not Debug Log

**Severity:** High (design gap)

The `pkg/events` package is designed for user-facing audit trails:
- High-level events: agent.spawned, work.assigned, etc.
- Stored in workspace-specific `.bc/events.jsonl`
- No debug-level detail

**Missing from events:**
- Error details
- tmux command failures
- File I/O operations
- Configuration loading
- Internal state changes

---

### 4.4 HIGH: No Metrics or Tracing

**Severity:** High

No instrumentation exists for:
- Operation timing
- Error rates
- Queue throughput
- Agent health metrics
- tmux session statistics

---

## Part 5: Error Context Loss

### 5.1 MEDIUM: CLI Commands Lose stderr Output

**Severity:** Medium

When external commands fail, stderr is lost.

**pkg/beads/beads.go:91:**
```go
func AddIssue(workspacePath, title, description string) error {
    cmd := exec.Command("bd", args...)
    return cmd.Run()  // stderr lost!
}
```

**Recommendation:** Use `cmd.CombinedOutput()`:
```go
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("bd add failed: %w\n%s", err, output)
}
```

---

## Summary Tables

### By Severity

| Severity | Count | Category |
|----------|-------|----------|
| Critical | 4 | Silent CLI returns, unused verbose flag, no logging |
| High | 6 | Ignored errors, os.Pipe, observability gaps |
| Medium | 6 | Inconsistent error checking, context loss |
| Low | 1 | GlobalDir empty return |

### By Package

| Package | Issues |
|---------|--------|
| pkg/beads | Silent error returns |
| pkg/github | Silent error returns |
| pkg/agent | LoadState/RefreshState ignored, KillSession ignored |
| pkg/events | Append errors ignored everywhere |
| pkg/queue | Load/Save errors inconsistent |
| pkg/workspace | Registry.Save ignored, GlobalDir returns "" |
| internal/cmd/* | Pervasive ignored errors |

---

## Recommendations

### Immediate (Before Production)

1. **Add structured logging library** (recommend `log/slog` for stdlib compatibility):
```go
import "log/slog"

var logger = slog.Default()

// Usage
logger.Debug("loading workspace", "path", path)
logger.Error("failed to load queue", "error", err)
```

2. **Implement verbose flag**:
```go
func init() {
    cobra.OnInitialize(func() {
        if verbose, _ := rootCmd.Flags().GetBool("verbose"); verbose {
            slog.SetLogLoggerLevel(slog.LevelDebug)
        }
    })
}
```

3. **Fix os.Pipe error handling**:
```go
aiToTUI, tuiInput, err := os.Pipe()
if err != nil {
    return fmt.Errorf("failed to create pipe: %w", err)
}
```

### Short-term

4. **Change list functions to return errors**:
```go
func ListIssues(workspacePath string) ([]Issue, error)
```

5. **Check all Load/Save errors**:
```go
if err := mgr.LoadState(); err != nil {
    logger.Warn("failed to load agent state", "error", err)
}
```

6. **Check log.Append errors** (at least log them):
```go
if err := log.Append(event); err != nil {
    logger.Error("failed to append event", "error", err)
}
```

### Medium-term

7. **Add debug logging to key operations**:
   - tmux command execution
   - File I/O
   - External CLI calls

8. **Add metrics** (recommend prometheus/client_golang):
   - `bc_agents_total` gauge
   - `bc_queue_items_total` by status
   - `bc_errors_total` counter by type

9. **Add tracing** for multi-agent operations

---

## Test Coverage

Add tests to verify error handling:

1. `TestListIssues_CommandFailure` - bd not installed
2. `TestListIssues_JSONParseError` - malformed output
3. `TestQueueLoad_CorruptedFile` - invalid JSON
4. `TestEventAppend_ReadOnlyFile` - permission denied
5. `TestOsPipe_FdExhaustion` - if feasible to simulate
