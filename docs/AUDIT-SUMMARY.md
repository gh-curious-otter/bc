# BC Codebase Audit Summary

**Date:** 2026-02-05
**Bead:** bc-34b.12 (work-007)
**Auditors:** worker-01, worker-02, worker-03

---

## Executive Summary

A comprehensive audit of the bc codebase was conducted across 8 packages and modules. The audit identified **47 issues** across all severity levels, with the most significant being:

1. **Pervasive silent error swallowing** - Errors are returned as `nil` or ignored throughout the codebase, making debugging impossible
2. **Race conditions** - Multiple data races in pkg/agent and pkg/tmux threaten data integrity
3. **Zero test coverage** - Most packages have 0% test coverage
4. **No observability** - No structured logging, verbose flag unused, no metrics

The codebase is functional for development but requires significant hardening before production use.

### Summary Statistics

| Severity | Count |
|----------|-------|
| Critical | 5 |
| High | 18 |
| Medium | 16 |
| Low | 8 |
| **Total** | **47** |

---

## Issues by Severity

### Critical Issues (5)

| # | Package | Issue | Impact |
|---|---------|-------|--------|
| C1 | pkg/agent | Data race in SetAgentCommand/SetAgentByName | Undefined behavior, crashes, wrong agent commands |
| C2 | pkg/beads | All list functions return nil on error | Failures invisible, no debugging possible |
| C3 | pkg/github | All list functions return nil on error | gh auth failures invisible |
| C4 | internal/cmd | --verbose flag defined but never used | Misleading UX, no debug capability |
| C5 | (codebase) | No structured logging library | No log levels, no tracing, no external logging |

### High Issues (18)

| # | Package | Issue |
|---|---------|-------|
| H1 | pkg/agent | Pointer escape in GetAgent/ListAgents - data race |
| H2 | pkg/agent | SpawnAgent idempotency incorrectly resets working state |
| H3 | pkg/agent | No thread safety in tmux.Manager calls |
| H4 | pkg/tmux | Race condition in SendKeys using global tmux buffer |
| H5 | pkg/events | No concurrency protection for writes |
| H6 | pkg/events | Unbounded log growth - Read() loads entire file to memory |
| H7 | pkg/workspace | Registry Prune() not persisted in home.go |
| H8 | pkg/beads | Silent error in ListIssues |
| H9 | pkg/beads | Silent error in ReadyIssues |
| H10 | pkg/github | Silent error in ListIssues |
| H11 | pkg/github | Silent error in ListPRs |
| H12 | internal/cmd | Silent error ignoring in init (registry save) |
| H13 | internal/cmd | Queue load/save errors ignored |
| H14 | internal/cmd | os.Pipe errors ignored in ui.go |
| H15 | internal/tui | Home screen 'r' refresh is a no-op |
| H16 | (codebase) | events.Log.Append() errors ignored everywhere |
| H17 | (codebase) | queue.Load() errors ignored in most places |
| H18 | (codebase) | mgr.LoadState/RefreshState errors never checked |

### Medium Issues (16)

| # | Package | Issue |
|---|---------|-------|
| M1 | pkg/agent | Tmux() exposes internal manager, bypasses locks |
| M2 | pkg/agent | No state machine validation |
| M3 | pkg/agent | Stale task not cleared in RefreshState |
| M4 | pkg/tmux | Ignored MkdirAll error |
| M5 | pkg/tmux | Magic 500ms sleep in SendKeys |
| M6 | pkg/tmux | Shell injection risk via env var keys |
| M7 | pkg/events | ReadLast does not validate input (n<=0 panics) |
| M8 | pkg/events | Malformed lines silently skipped |
| M9 | pkg/workspace | Init() overwrites existing config if called directly |
| M10 | pkg/workspace | GlobalDir() returns empty string on error |
| M11 | pkg/beads | Partial data returned on JSONL parse error |
| M12 | pkg/beads | AddIssue loses stderr context |
| M13 | pkg/github | CreateIssue loses stderr context |
| M14 | pkg/github | Hardcoded limit of 50 items |
| M15 | internal/cmd | --force flag defined but unused |
| M16 | internal/tui | Issues/PRs tabs have no drill-down (ActionDrillIssue unused) |

### Low Issues (8)

| # | Package | Issue |
|---|---------|-------|
| L1 | pkg/agent | Fragile task capture patterns (Unicode chars) |
| L2 | pkg/agent | LoadState reads file before acquiring lock |
| L3 | pkg/tmux | Workspace hash collision risk (24-bit) |
| L4 | pkg/tmux | CreateSession doesn't check for existing session |
| L5 | pkg/tmux | Windows count not parsed in ListSessions |
| L6 | pkg/workspace | Registry path comparison not normalized |
| L7 | pkg/workspace | No concurrent access protection for Registry |
| L8 | internal/cmd | ANSI colors output without TTY check |

---

## Issues by Package

| Package | Critical | High | Medium | Low | Total |
|---------|----------|------|--------|-----|-------|
| pkg/agent | 1 | 3 | 3 | 2 | 9 |
| pkg/tmux | 0 | 1 | 3 | 3 | 7 |
| pkg/events | 0 | 2 | 2 | 0 | 4 |
| pkg/workspace | 0 | 1 | 2 | 2 | 5 |
| pkg/beads | 1 | 2 | 2 | 0 | 5 |
| pkg/github | 1 | 2 | 2 | 0 | 5 |
| internal/cmd | 1 | 3 | 1 | 1 | 6 |
| internal/tui | 0 | 1 | 1 | 0 | 2 |
| Cross-cutting | 1 | 3 | 0 | 0 | 4 |

---

## Test Coverage Gaps

### Current State

| Package | Test Coverage | Files Tested |
|---------|---------------|--------------|
| pkg/agent | 0% | None |
| pkg/tmux | 0% | None |
| pkg/events | 0% | None |
| pkg/workspace | 0% | None |
| pkg/beads | 0% | None |
| pkg/github | 0% | None |
| pkg/queue | 0% | None |
| internal/cmd | ~5% | root.go only (3 tests) |
| internal/tui | 0% | None |

### Priority Test Cases Needed

**Immediate (Critical/High paths):**
1. `TestSpawnAgent_ConcurrentCalls` - Race detection
2. `TestSendKeys_ConcurrentLongMessages` - Buffer race
3. `TestListIssues_CommandFailure` - Error propagation
4. `TestQueueLoad_CorruptedFile` - Error handling
5. `TestEventAppend_ConcurrentWrites` - Write atomicity

**Short-term (Core functionality):**
6. `TestStateMachine_Transitions` - Valid state changes
7. `TestRefreshState_SessionDied` - State detection
8. `TestWorkspaceFind_UpwardSearch` - Path resolution
9. `TestRegistryPrune_Persistence` - Pruned entries saved
10. `TestReadLast_EdgeCases` - n=0, n<0, n>len

**Medium-term (Edge cases):**
11. `TestCaptureLiveTask_AllPatterns` - Unicode handling
12. `TestCreateSessionWithEnv_Injection` - Shell safety
13. `TestGlobalDir_NoHome` - Error handling
14. `TestListPRs_Pagination` - Large repos

---

## Architecture Concerns

### 1. Error Handling Philosophy

**Current:** Silent failures throughout - functions return nil/empty on error
**Impact:** Debugging is impossible; users see empty data without knowing why
**Recommendation:** Adopt explicit error returns: `func ListX() ([]T, error)`

### 2. Concurrency Model

**Current:** Mixed locking - some methods locked, some not; pointers escape locks
**Impact:** Data races under concurrent access
**Recommendation:**
- Return copies, not pointers from locked methods
- Add mutex to tmux.Manager
- Consider read-write lock granularity

### 3. Observability

**Current:** No logging library, no metrics, no tracing
**Impact:** Cannot diagnose issues in production
**Recommendation:**
- Add `log/slog` for structured logging
- Implement --verbose flag
- Add prometheus metrics for key operations

### 4. State Persistence

**Current:** JSON files with no atomicity guarantees
**Impact:** Concurrent access can corrupt state
**Recommendation:**
- Use atomic write pattern (temp file + rename)
- Consider file locking for cross-process safety

### 5. External CLI Dependencies

**Current:** Hard dependency on `bd`, `gh`, `tmux` CLIs
**Impact:** Failures invisible when tools not installed/configured
**Recommendation:**
- Add tool detection at startup
- Return errors when tools unavailable
- Consider fallback behaviors

---

## Prioritized Recommendations

### Phase 1: Critical Fixes (Before Production)

1. **Add structured logging** - Integrate `log/slog`, implement --verbose
2. **Fix data races in pkg/agent** - Lock SetAgentCommand, return copies not pointers
3. **Fix SendKeys race condition** - Use named tmux buffers
4. **Change list functions to return errors** - `([]T, error)` pattern
5. **Check all ignored errors** - At minimum log warnings

**Estimated effort:** 2-3 days

### Phase 2: High Priority Fixes

6. **Add concurrency tests** - `go test -race` on all packages
7. **Add events log rotation** - Prevent unbounded growth
8. **Fix home.go prune persistence** - Save after Prune()
9. **Fix SpawnAgent idempotency** - Don't overwrite working state
10. **Implement --force flag** - Or remove from docs

**Estimated effort:** 3-4 days

### Phase 3: Medium Priority Improvements

11. **Add state machine validation** - Prevent invalid transitions
12. **Add input validation** - Agent names, env var keys, work IDs
13. **Fix ReadLast edge cases** - Validate n parameter
14. **Add TTY detection** - Disable colors when piped
15. **Implement issue/PR drill-down** - Or remove tabs

**Estimated effort:** 2-3 days

### Phase 4: Quality & Polish

16. **Add comprehensive test suite** - Target 70% coverage
17. **Add metrics** - Agent counts, queue stats, error rates
18. **Normalize registry paths** - Handle symlinks, trailing slashes
19. **Make limits configurable** - GitHub 50-item limit
20. **Document CLI output format** - Task capture patterns

**Estimated effort:** 1-2 weeks

---

## Appendix: Audit Sources

| Work Item | Package | Auditor | Branch |
|-----------|---------|---------|--------|
| work-004 | pkg/agent | worker-01 | (commit 44cc2fd) |
| work-005 | pkg/tmux | worker-02 | worker-02/audit-pkg-tmux |
| work-006 | Error handling | worker-01 | worker-03/audit-internal-tui |
| work-008 | pkg/events | worker-03 | worker-03/audit-pkg-events |
| work-009 | pkg/workspace | worker-03 | worker-03/audit-pkg-workspace |
| work-010 | pkg/beads, pkg/github | worker-01 | worker-03/audit-internal-tui |
| work-011 | internal/cmd | worker-02 | worker-02/audit-internal-cmd |
| work-012 | internal/tui | worker-03 | worker-03/audit-internal-tui |

---

## Conclusion

The bc codebase provides solid core functionality for multi-agent orchestration but has significant quality gaps that must be addressed before production use. The most critical issues are:

1. **Silent error swallowing** makes debugging impossible
2. **Race conditions** threaten data integrity
3. **Zero test coverage** means bugs go undetected
4. **No observability** prevents production monitoring

With focused effort on the Phase 1 recommendations (2-3 days), the codebase can reach a stable state. Full remediation across all phases would take approximately 2-3 weeks.
