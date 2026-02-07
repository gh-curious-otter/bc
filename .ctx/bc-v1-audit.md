# bc v1 Architectural Audit

**Date:** 2025-02-06
**Auditor:** Coordinator Agent
**Status:** LINT ZERO Complete (763 violations fixed)

---

## 1. Architecture Overview

### 1.1 System Purpose
bc (beads coordinator) is a multi-agent orchestration system for coordinating AI coding agents (Claude Code, Cursor Agent, Codex) with predictable behavior and cost awareness.

### 1.2 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         bc CLI                                  │
│  (cmd/bc/main.go → internal/cmd/*)                             │
├─────────────────────────────────────────────────────────────────┤
│                    Core Packages (pkg/)                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ │
│  │  agent  │ │  queue  │ │ channel │ │  beads  │ │workspace │ │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬─────┘ │
│       │           │           │           │           │        │
├───────┴───────────┴───────────┴───────────┴───────────┴────────┤
│                    Infrastructure                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐              │
│  │  tmux   │ │   git   │ │  stats  │ │   tui   │              │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 Agent Hierarchy
```
ProductManager (level 0)    ← Product vision, creates epics
    └── Manager (level 1)   ← Breaks down epics, assigns work
        ├── Engineer (level 2)  ← Implements tasks
        └── QA (level 2)        ← Tests implementations
```

---

## 2. Code Organization

### 2.1 Directory Structure
```
bc/
├── cmd/bc/main.go          # Entry point
├── config/                  # Generated config (cfgx)
│   └── config.go           # Static configuration values
├── internal/               # Private implementation
│   ├── cmd/                # CLI commands (cobra)
│   └── tui/                # Application-specific TUI views
├── pkg/                    # Reusable packages
│   ├── agent/              # Agent lifecycle management
│   ├── beads/              # Issue tracker integration
│   ├── channel/            # Broadcast messaging
│   ├── events/             # Event system
│   ├── git/                # Git operations
│   ├── github/             # GitHub API integration
│   ├── log/                # Logging
│   ├── queue/              # Work queue management
│   ├── stats/              # Cost/usage statistics
│   ├── tmux/               # Terminal multiplexer wrapper
│   ├── tui/                # Generic TUI components
│   └── workspace/          # Workspace configuration
├── prompts/                # Role-specific prompts (loaded at spawn)
├── .bc/                    # Workspace state directory
│   ├── agents.json         # Agent state persistence
│   ├── queue.json          # Work queue persistence
│   ├── channels.json       # Channel membership
│   ├── worktrees/          # Per-agent git worktrees
│   └── bin/                # Git wrapper scripts
└── .beads/                 # Issue tracker data (external)
```

### 2.2 Package Responsibilities

| Package | LOC* | Responsibility |
|---------|------|----------------|
| `pkg/agent` | ~1000 | Agent spawn, stop, state machine, worktree management |
| `pkg/queue` | ~350 | Work item CRUD, status transitions, merge tracking |
| `pkg/channel` | ~250 | Named groups, message broadcast, history |
| `pkg/beads` | ~150 | bd CLI wrapper for issue tracking |
| `pkg/workspace` | ~180 | Config loading, directory management |
| `pkg/tmux` | ~200 | Session creation, key sending, capture |
| `pkg/tui` | ~400 | Table, app framework, runtime protocol |
| `internal/cmd` | ~2000 | All CLI commands |
| `internal/tui` | ~800 | Application-specific views |

*Approximate lines of code

---

## 3. Design Patterns

### 3.1 Patterns Used

**Builder Pattern** (`pkg/tui/table.go`)
```go
NewTableView("agents").
    Title("Agents").
    Columns(Col("NAME", 15), Col("STATE", 10)).
    OnSelect(func(r Row) Cmd { ... }).
    Build()
```
- Clean fluent API for complex object construction
- Separates construction from representation

**State Machine** (`pkg/agent/agent.go`)
```go
validTransitions = map[State][]State{
    StateIdle:    {StateWorking, StateDone, StateStuck, ...},
    StateWorking: {StateIdle, StateDone, StateStuck, ...},
    ...
}
```
- Explicit state transitions with validation
- Prevents invalid agent state changes

**Role-Based Access Control** (`pkg/agent/agent.go`)
```go
RoleCapabilities = map[Role][]Capability{...}
RoleHierarchy = map[Role][]Role{...}
```
- Explicit capability definitions per role
- Hierarchical spawn permissions

**Repository Pattern** (`pkg/queue/queue.go`, `pkg/channel/channel.go`)
- JSON file-backed persistence
- Clean CRUD operations with mutex protection
- Separation of storage from business logic

### 3.2 Patterns Missing or Weak

**Dependency Injection** - Currently hardcoded:
- Config is global package variables
- Tmux manager created inline in agent manager
- Makes testing and mocking difficult

**Event Sourcing** - Partial implementation:
- `pkg/events/` exists but not fully utilized
- State changes not captured as events
- No audit trail or replay capability

**Error Handling Strategy** - Inconsistent:
- Some places use `//nolint:errcheck` liberally
- Others return errors but callers ignore them
- No structured error types for different failure modes

---

## 4. Reliability Assessment

### 4.1 Known Issues

**State Synchronization**
- Agent state in memory vs. tmux session can drift
- `RefreshState()` polls but doesn't guarantee consistency
- Race conditions possible between state updates

**Worktree Detection Bug**
- `.bc/worktrees/<agent>/.bc/` directories cause false workspace detection
- `bc home` lists agent worktrees as separate workspaces
- Root cause: `IsWorkspace()` just checks for `.bc` directory existence

**Message Delivery**
- `SendToAgent()` is fire-and-forget via tmux
- No acknowledgment or retry mechanism
- Messages can be lost if agent is at wrong prompt state

**Persistence**
- JSON files not atomic (partial writes possible)
- No locking between multiple bc processes
- `//nolint:errcheck` on save operations hides failures

### 4.2 Reliability Gaps

| Area | Issue | Risk |
|------|-------|------|
| State | No distributed locking | Concurrent bc instances corrupt state |
| Comms | No message acknowledgment | Lost messages, stuck agents |
| Persistence | Non-atomic JSON writes | Data corruption on crash |
| Recovery | No automatic retry/heal | Manual intervention required |
| Monitoring | No health checks | Silent failures undetected |

### 4.3 Performance Concerns

- Tmux capture is synchronous and blocking
- JSON marshal/unmarshal on every state change
- No caching of frequently-read data
- Linear scans through arrays (queue, agents)

---

## 5. Technical Debt

### 5.1 Code Quality (Post Lint-Zero)

| Category | Before | After | Notes |
|----------|--------|-------|-------|
| errcheck | 456 | 0 | All errors handled |
| gosec | 173 | 0 | Security issues fixed |
| govet | 18 | 0 | Shadow variables fixed |
| noctx | 50 | 0 | Context propagation added |
| fieldalignment | 64 | 0 | Struct layout optimized |
| **Total** | **763** | **0** | Clean slate |

### 5.2 Remaining Technical Debt

**Architecture**
- Tight coupling between CLI commands and business logic
- Global configuration makes unit testing difficult
- No clear domain model boundaries

**Testing**
- Integration tests rely on real tmux (flaky in CI)
- No mock implementations for external dependencies
- Test coverage gaps in error paths

**Documentation**
- API documentation incomplete
- No architecture decision records (ADRs)
- Implicit contracts between components

**Observability**
- Logging inconsistent (some debug, some warn)
- No metrics or tracing
- No structured logging format

### 5.3 Deprecation Candidates

- `RoleCoordinator` / `RoleWorker` - Legacy roles, use `RoleProductManager`/`RoleEngineer`
- `//nolint` comments - Should be addressed rather than suppressed
- Multiple tmux manager constructors - Consolidate to single factory

---

## 6. Recommendations for Redesign

### 6.1 High Priority (Reliability)

**1. Implement Proper Message Bus**
```
Current: bc send → tmux.SendKeys → hope it works
Future:  bc send → MessageQueue → Agent polls → Ack → Retry if needed
```
- Use file-based queue or SQLite for durability
- Add acknowledgment protocol
- Implement retry with exponential backoff

**2. Atomic State Persistence**
- Write to temp file, then atomic rename
- Add file locking for concurrent access
- Consider SQLite for ACID guarantees

**3. Health Monitoring**
- Regular heartbeats from agents
- Automatic detection of stuck/dead agents
- Self-healing (restart, reassign work)

### 6.2 Medium Priority (Architecture)

**4. Dependency Injection**
```go
// Current
func NewManager(stateDir string) *Manager {
    return &Manager{
        tmux: tmux.NewManager(...), // Hardcoded
    }
}

// Better
type ManagerConfig struct {
    Tmux    TmuxInterface
    Storage StorageInterface
    Logger  LoggerInterface
}
func NewManager(cfg ManagerConfig) *Manager
```

**5. Event-Driven Architecture**
- All state changes emit events
- Event log enables replay and debugging
- Enables reactive UI updates

**6. Clear Domain Boundaries**
```
domain/
├── agent/      # Agent entity and value objects
├── work/       # Work items and queue
├── comms/      # Channels and messages
└── workspace/  # Configuration and state
```

### 6.3 Lower Priority (Quality of Life)

**7. Structured Error Types**
```go
type AgentError struct {
    Op      string  // Operation that failed
    AgentID string  // Which agent
    Err     error   // Underlying error
}
```

**8. Observability Stack**
- OpenTelemetry tracing
- Prometheus metrics
- Structured JSON logging

**9. Plugin Architecture**
- Agent backends (tmux, docker, k8s)
- Storage backends (file, sqlite, postgres)
- Tool adapters (claude, cursor, codex)

---

## 7. Summary

### What Works Well
- Clean CLI structure with Cobra
- Role hierarchy and capability system
- Git worktree isolation for agents
- Fluent builder APIs for TUI
- Comprehensive lint compliance (post Lint-Zero)

### What Needs Work
- Message delivery reliability
- State synchronization
- Persistence atomicity
- Dependency injection for testing
- Error handling consistency

### Recommended Next Steps

1. **Immediate**: Fix worktree detection bug (false workspace matches)
2. **Short-term**: Implement atomic state persistence with file locking
3. **Medium-term**: Add message acknowledgment protocol
4. **Long-term**: Redesign with dependency injection and event sourcing

---

*This audit was conducted as part of the project hold for redesign planning. All new work is paused pending architectural decisions.*
