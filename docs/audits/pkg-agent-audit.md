# Audit Report: pkg/agent

**Date:** 2026-02-05
**Auditor:** worker-01
**Files Reviewed:** `pkg/agent/agent.go`, `pkg/tmux/session.go`

---

## Executive Summary

The `pkg/agent` package provides agent lifecycle management but has several thread-safety issues, a flawed idempotency implementation, missing state machine validation, and fragile task capture logic. **Zero test coverage** means these issues have not been caught.

---

## Issues Found

### 1. CRITICAL: Data Race in SetAgentCommand/SetAgentByName

**Severity:** Critical

**Description:** `SetAgentCommand` and `SetAgentByName` modify `m.agentCmd` without acquiring any lock, while `SpawnAgent` reads this field under lock. Concurrent calls create a data race.

**Code:**
```go
// agent.go:94-96 - NO LOCK
func (m *Manager) SetAgentCommand(cmd string) {
    m.agentCmd = cmd  // Race condition!
}

// agent.go:99-107 - NO LOCK
func (m *Manager) SetAgentByName(name string) bool {
    for _, a := range config.Agents {
        if a.Name == name {
            m.agentCmd = cmd  // Race condition!
            return true
        }
    }
    return false
}

// agent.go:153 - Under lock, reads m.agentCmd
if err := m.tmux.CreateSessionWithEnv(name, workspace, m.agentCmd, env); err != nil {
```

**Impact:** Undefined behavior, potential crashes, agents spawned with wrong command.

**Recommendation:** Acquire `m.mu.Lock()` in both methods before modifying `m.agentCmd`.

---

### 2. HIGH: Pointer Escape in GetAgent/ListAgents

**Severity:** High

**Description:** `GetAgent` and `ListAgents` return pointers to `Agent` structs that can be modified after the lock is released. Concurrent access to returned agents creates data races.

**Code:**
```go
// agent.go:207-211
func (m *Manager) GetAgent(name string) *Agent {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.agents[name]  // Pointer escapes lock!
}

// agent.go:214-235
func (m *Manager) ListAgents() []*Agent {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // ...
    for _, a := range m.agents {
        agents = append(agents, a)  // Pointers escape lock!
    }
    return agents
}
```

**Impact:** If caller holds a pointer while `RefreshState` or `UpdateAgentState` modifies the same agent, data race occurs.

**Recommendation:** Return copies of Agent structs, not pointers:
```go
func (m *Manager) GetAgent(name string) (Agent, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if a, ok := m.agents[name]; ok {
        return *a, true  // Return copy
    }
    return Agent{}, false
}
```

---

### 3. HIGH: SpawnAgent Idempotency Incorrectly Resets State

**Severity:** High

**Description:** When `SpawnAgent` is called for an existing agent with a live tmux session, it unconditionally sets `State = StateIdle`. This is incorrect if the agent was actually `StateWorking`.

**Code:**
```go
// agent.go:116-123
if existing, exists := m.agents[name]; exists {
    if m.tmux.HasSession(name) {
        existing.State = StateIdle  // BUG: Overwrites working state!
        existing.UpdatedAt = time.Now()
        m.saveState()
        return existing, nil
    }
}
```

**Impact:** If coordinator calls `SpawnAgent` to "ensure" a worker exists while it's working, the state gets corrupted to `Idle`.

**Recommendation:** Preserve existing state when reusing:
```go
if existing, exists := m.agents[name]; exists {
    if m.tmux.HasSession(name) {
        existing.UpdatedAt = time.Now()
        // Don't touch existing.State - preserve it
        m.saveState()
        return existing, nil
    }
}
```

---

### 4. HIGH: No Thread Safety in tmux.Manager

**Severity:** High

**Description:** The underlying `tmux.Manager` has no synchronization. Methods like `SendToAgent` and `CaptureOutput` call tmux methods without holding the agent manager's lock.

**Code:**
```go
// agent.go:359-361 - NO LOCK
func (m *Manager) SendToAgent(name, message string) error {
    return m.tmux.SendKeys(name, message)
}

// agent.go:364-366 - NO LOCK
func (m *Manager) CaptureOutput(name string, lines int) (string, error) {
    return m.tmux.Capture(name, lines)
}
```

**Impact:** Concurrent calls to different agent methods can interleave tmux commands unpredictably.

**Recommendation:** Either add locking to `tmux.Manager` or ensure agent.Manager holds appropriate locks when calling tmux methods.

---

### 5. MEDIUM: Tmux() Exposes Internal Manager

**Severity:** Medium

**Description:** `Tmux()` returns the internal tmux manager, allowing external code to bypass all synchronization.

**Code:**
```go
// agent.go:416-418
func (m *Manager) Tmux() *tmux.Manager {
    return m.tmux  // Bypasses all locks!
}
```

**Impact:** External code can call tmux methods while agent methods are running, causing races.

**Recommendation:** Remove this method or document that callers must handle synchronization.

---

### 6. MEDIUM: No State Machine Validation

**Severity:** Medium

**Description:** There is no validation of state transitions. Any state can transition to any other state, which can mask bugs.

**Valid states:** `Idle`, `Starting`, `Working`, `Done`, `Stuck`, `Error`, `Stopped`

**Code:**
```go
// agent.go:341-356 - Accepts any state
func (m *Manager) UpdateAgentState(name string, state State, task string) error {
    // ...
    agent.State = state  // No validation!
}
```

**Expected transitions:**
- `Starting` → `Idle` (spawn complete)
- `Idle` → `Working` (task assigned)
- `Working` → `Done` | `Stuck` | `Error` (task finished)
- Any → `Stopped` (explicit stop)

**Impact:** Invalid transitions (e.g., `Done` → `Working`) go undetected, masking bugs.

**Recommendation:** Add a transition validation function:
```go
func validTransition(from, to State) bool {
    // Define allowed transitions
}
```

---

### 7. MEDIUM: Stale Task Not Cleared in RefreshState

**Severity:** Medium

**Description:** `RefreshState` only updates `Task` if `captureLiveTask` returns non-empty. Old task strings persist indefinitely.

**Code:**
```go
// agent.go:266-268
if live := m.captureLiveTask(name); live != "" {
    a.Task = live  // Only updates if non-empty
}
// If live == "", old a.Task persists
```

**Impact:** Status output shows stale task information, misleading operators.

**Recommendation:** Clear task when nothing is captured:
```go
a.Task = m.captureLiveTask(name)  // Always update, even if empty
```

---

### 8. LOW: Fragile Task Capture Patterns

**Severity:** Low

**Description:** `captureLiveTask` relies on specific Unicode characters and string patterns from Claude CLI output. These may change with CLI updates.

**Code:**
```go
// agent.go:296-304
if strings.HasPrefix(line, "✻") ||
   strings.HasPrefix(line, "✳") ||
   strings.HasPrefix(line, "✽") ||
   strings.HasPrefix(line, "·") {
    // ...
}
```

**Impact:** CLI output format changes will break task detection silently.

**Recommendation:** Document the assumed output format. Consider making patterns configurable or adding version detection.

---

### 9. LOW: LoadState Reads File Before Acquiring Lock

**Severity:** Low

**Description:** `LoadState` reads from disk before acquiring the mutex, while `saveState` writes under the lock. This is technically a race, though unlikely to cause issues in practice.

**Code:**
```go
// agent.go:396-413
func (m *Manager) LoadState() error {
    // ...
    data, err := os.ReadFile(...)  // No lock
    if err != nil { ... }

    m.mu.Lock()  // Lock acquired after read
    defer m.mu.Unlock()
    return json.Unmarshal(data, &m.agents)
}
```

**Impact:** Minimal - LoadState is typically called at startup.

**Recommendation:** Acquire lock before reading file.

---

## Summary Table

| # | Severity | Issue | Line(s) |
|---|----------|-------|---------|
| 1 | Critical | Data race in SetAgentCommand/SetAgentByName | 94-107 |
| 2 | High | Pointer escape in GetAgent/ListAgents | 207-235 |
| 3 | High | SpawnAgent overwrites working state | 119 |
| 4 | High | No thread safety in tmux.Manager | 359-366 |
| 5 | Medium | Tmux() exposes internal manager | 416-418 |
| 6 | Medium | No state machine validation | 341-356 |
| 7 | Medium | Stale task not cleared | 266-268 |
| 8 | Low | Fragile task capture patterns | 296-304 |
| 9 | Low | LoadState reads before lock | 401 |

---

## Recommendations Priority

1. **Immediate:** Fix Critical and High issues (1-4) before production use
2. **Short-term:** Add comprehensive unit tests with race detector (`go test -race`)
3. **Medium-term:** Implement state machine validation (5-6)
4. **Long-term:** Consider making task capture patterns configurable (8)

---

## Test Coverage Gaps

The package has **zero tests**. Minimum test coverage should include:

1. `TestSpawnAgent_Basic` - Spawn and verify state
2. `TestSpawnAgent_Idempotent` - Call twice, verify behavior
3. `TestSpawnAgent_ConcurrentCalls` - Race detection
4. `TestGetAgent_ConcurrentModification` - Race detection
5. `TestRefreshState_SessionDied` - Verify stopped state
6. `TestStateMachine_Transitions` - Validate transitions
7. `TestCaptureLiveTask_Patterns` - All pattern matching
