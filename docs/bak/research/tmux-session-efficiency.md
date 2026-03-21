# Tmux Session Efficiency Research

**Researcher:** eng-04
**Date:** Sprint 12 Phase 2 Prep
**Related Epic:** #962 Performance Optimization

## Executive Summary

The `pkg/tmux/` package spawns a new subprocess for every tmux operation. With health checks running every 30 seconds and multiple agents, this creates significant process overhead. Caching session state with short TTLs could reduce subprocess calls by 80%+.

## Current Implementation Analysis

### Files Reviewed
- `pkg/tmux/session.go` - Core tmux manager (451 lines)
- `pkg/agent/health.go` - Health checker using tmux
- `pkg/agent/agent.go` - Agent manager using tmux

### Key Functions & Subprocess Cost

| Function | Subprocess Command | Calls/Operation |
|----------|-------------------|-----------------|
| `HasSession()` | `tmux has-session -t <name>` | 1 |
| `ListSessions()` | `tmux list-sessions -F ...` | 1 |
| `IsRunning()` | `tmux list-sessions` | 1 |
| `CreateSession()` | `tmux new-session -d ...` | 1 |
| `KillSession()` | `tmux kill-session -t ...` | 1 |
| `Capture()` | `tmux capture-pane -t ...` | 1 |

### Query Patterns Identified

#### 1. Health Check Loop (High Frequency)
```go
// pkg/agent/health.go:116
result.TmuxAlive = h.tmux.HasSession(state.Session)
```
- **Frequency:** Every 30 seconds (DefaultHealthCheckInterval)
- **Per agent:** 1 subprocess per health check
- **With N agents:** N subprocesses every 30 seconds

#### 2. Agent Operations (Medium Frequency)
```go
// pkg/agent/agent.go:412, 493
if m.tmux.HasSession(name) { ... }
```
- Called during: Start, stop, respawn operations
- **Issue:** Multiple HasSession calls in same operation

#### 3. Agent List/Status (On-Demand)
```go
// pkg/agent/agent.go:1135
sessions, err := m.tmux.ListSessions()
```
- Called for: `bc agent list`, TUI updates
- **Issue:** Fetches all sessions even when checking one

#### 4. Worktree Orphan Detection
```go
// internal/cmd/worktree.go:325
sessions, err := tmuxMgr.ListSessions()
```
- Used to determine if agents are running

## Caching Opportunities

### 1. Session Existence Cache (High Impact)

**Current:** Each `HasSession()` spawns `tmux has-session`

**Proposed:**
```go
type Manager struct {
    // ... existing fields ...
    sessionCache    map[string]bool     // session name -> exists
    sessionCacheAt  time.Time           // when cache was populated
    sessionCacheTTL time.Duration       // default 2-5 seconds
    cacheMu         sync.RWMutex
}

func (m *Manager) HasSession(name string) bool {
    // Check cache first
    m.cacheMu.RLock()
    if time.Since(m.sessionCacheAt) < m.sessionCacheTTL {
        if exists, ok := m.sessionCache[name]; ok {
            m.cacheMu.RUnlock()
            return exists
        }
    }
    m.cacheMu.RUnlock()

    // Cache miss - query tmux and update cache
    // ...
}
```

**Benefits:**
- Health checks reuse cached results
- Multiple HasSession calls in same operation share result
- TTL of 2-5 seconds is safe (session state rarely changes mid-check)

**Invalidation:**
- Clear cache on `CreateSession()`, `KillSession()`
- TTL-based expiry for safety

### 2. Session List Cache (Medium Impact)

**Current:** `ListSessions()` always queries tmux

**Proposed:**
```go
type Manager struct {
    // ... existing fields ...
    sessionsCache   []Session
    sessionsCacheAt time.Time
}

func (m *Manager) ListSessions() ([]Session, error) {
    m.cacheMu.RLock()
    if time.Since(m.sessionsCacheAt) < m.sessionCacheTTL {
        result := make([]Session, len(m.sessionsCache))
        copy(result, m.sessionsCache)
        m.cacheMu.RUnlock()
        return result, nil
    }
    m.cacheMu.RUnlock()
    // ... query tmux and cache ...
}
```

**Benefits:**
- TUI polling can use cached session list
- Multiple callers in rapid succession share result

### 3. Batch Session Check (New Pattern)

**Current:** N agents = N `HasSession()` calls

**Proposed:**
```go
func (m *Manager) HasSessions(names []string) map[string]bool {
    // Single ListSessions() call, check all names
    sessions, _ := m.ListSessions()
    sessionSet := make(map[string]bool)
    for _, s := range sessions {
        sessionSet[s.Name] = true
    }
    result := make(map[string]bool)
    for _, name := range names {
        result[name] = sessionSet[name]
    }
    return result
}
```

**Benefits:**
- 1 subprocess instead of N for bulk checks
- Useful for agent list operations

## Performance Impact Estimates

### Current State (10 agents, TUI polling every 5s)
| Operation | Calls/minute | Subprocesses/minute |
|-----------|-------------|---------------------|
| Health checks | 20 (2 per agent) | 20 |
| TUI status | 12 | 120 (10 per call) |
| Agent list | 6 | 60 |
| **Total** | | **~200** |

### With Caching (2s TTL)
| Operation | Cache Hits | Subprocesses/minute |
|-----------|------------|---------------------|
| Health checks | 90% | 2 |
| TUI status | 95% | 6 |
| Agent list | 80% | 12 |
| **Total** | | **~20** |

**Estimated reduction: 90%**

## Implementation Recommendations

### Phase 1: Add TTL Cache to Manager
1. Add cache fields to `Manager` struct
2. Implement cached `HasSession()` with TTL
3. Invalidate on `CreateSession()`/`KillSession()`
4. Default TTL: 2 seconds (configurable)

### Phase 2: Batch Operations
1. Add `HasSessions(names []string)` for bulk checks
2. Refactor health checker to use batch check
3. Update agent list to use single ListSessions

### Phase 3: Smart Refresh
1. Track last-known session states
2. Only refresh when needed (on agent start/stop)
3. Consider inotify/fswatch for tmux socket changes

## Testing Considerations

- Unit tests with mock tmux (already in place)
- Integration tests for cache invalidation
- Benchmark tests comparing cached vs uncached
- Race condition tests for concurrent cache access

## Compatibility Notes

- `TmuxChecker` interface (`pkg/agent/root.go:266`) only requires `HasSession()`
- Changes are internal to Manager, no API changes needed
- Existing tests should continue to work

## Next Steps

1. Create issue for Phase 1 implementation
2. Coordinate with eng-05's polling research
3. Profile current subprocess overhead as baseline
4. Implement caching with feature flag for gradual rollout
