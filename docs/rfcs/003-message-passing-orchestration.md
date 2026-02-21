# RFC 003: Advanced Multi-Agent Orchestration

**Issue:** #1402
**Author:** eng-04
**Status:** Draft
**Created:** 2026-02-22

## Summary

Analyze bc's existing multi-agent capabilities and propose enhancements for advanced orchestration patterns.

## Current State

bc already has substantial multi-agent orchestration features:

### Existing Capabilities

| Feature | Status | Location |
|---------|--------|----------|
| Channel Communication | ✅ Complete | `pkg/channel/` |
| Team Organization | ✅ Complete | `pkg/team/` |
| Role-Based Routing | ✅ Complete | `pkg/routing/` |
| Agent Permissions (RBAC) | ✅ Complete | `pkg/agent/` |
| Message Types | ✅ Complete | Task, Review, Approval, Merge |
| @Mentions | ✅ Complete | Channel message parsing |
| Reactions | ✅ Complete | SQLite channel store |
| PR Workflow Automation | ✅ Complete | ApprovalHandler |

### Gap Analysis

| Feature | Current | Issue #1402 Request |
|---------|---------|---------------------|
| Direct Agent-to-Agent | ✅ Via channels | Same (exists) |
| Group Communication | ✅ Channels + @all | Same (exists) |
| Role-Based Routing | ✅ Round-robin | Task delegation |
| Context Sharing | ❌ Manual | Shared memory |
| Task Queue | ❌ Manual | Automatic delegation |
| Event Bus | ❌ Polling | Pub/Sub |
| Conflict Resolution | ❌ None | Automatic |

## Proposal: Enhancement Tiers

### Tier 1: Documentation (No Code)

Before adding features, document existing capabilities better:

```markdown
docs/orchestration/
├── CHANNELS.md          # Channel patterns and best practices
├── TEAMS.md             # Team organization strategies
├── ROUTING.md           # Work distribution patterns
├── WORKFLOWS.md         # Common agent workflows
└── RBAC.md              # Permission model explained
```

**Effort:** ~2 PRs
**Impact:** High (discoverability)

### Tier 2: Task Queue System

Add structured task management:

```toml
# .bc/config.toml
[orchestration]
task_queue_enabled = true
task_timeout = "1h"
max_concurrent_per_agent = 3
```

**New Commands:**

```bash
# Create a task
bc task create "Implement login" --assignee eng-01 --priority high

# List tasks
bc task list --status pending

# Claim next task (by agent)
bc task claim

# Complete task
bc task done TASK-123 "PR #456 ready"
```

**Database Schema:**

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending', -- pending, claimed, done, failed
    assignee TEXT,
    created_by TEXT,
    created_at DATETIME,
    claimed_at DATETIME,
    completed_at DATETIME,
    result TEXT
);
```

**Effort:** ~4 PRs
**Impact:** Medium (structured work)

### Tier 3: Pub/Sub Event Bus

Replace polling with event-driven updates:

```go
// Event types
type EventType string

const (
    EventAgentStarted    EventType = "agent.started"
    EventAgentStopped    EventType = "agent.stopped"
    EventTaskCreated     EventType = "task.created"
    EventTaskClaimed     EventType = "task.claimed"
    EventMessageSent     EventType = "channel.message"
    EventReportSubmitted EventType = "agent.report"
)

// Event bus interface
type EventBus interface {
    Publish(event Event) error
    Subscribe(types []EventType, handler EventHandler) error
    Unsubscribe(subscriptionID string) error
}
```

**Implementation Options:**

| Option | Pros | Cons |
|--------|------|------|
| SQLite triggers | No external deps | Limited performance |
| File watching | Simple | Platform-specific |
| Unix sockets | Fast, local | Process management |
| Redis | Full pub/sub | External dependency |

**Recommendation:** SQLite triggers for MVP, optional Redis for scale.

**Effort:** ~6 PRs
**Impact:** High (real-time updates)

### Tier 4: Context Sharing

Enable agents to share context:

```bash
# Agent publishes context
bc context set "project_architecture" "$(cat docs/ARCHITECTURE.md)"

# Other agents can read
bc context get "project_architecture"

# Scoped context
bc context set --scope team-frontend "design_system" "..."
```

**Database Schema:**

```sql
CREATE TABLE shared_context (
    key TEXT NOT NULL,
    scope TEXT DEFAULT 'workspace', -- workspace, team, agent
    scope_id TEXT,
    value TEXT,
    expires_at DATETIME,
    updated_by TEXT,
    updated_at DATETIME,
    PRIMARY KEY (key, scope, scope_id)
);
```

**Effort:** ~3 PRs
**Impact:** Medium (knowledge sharing)

### Tier 5: Conflict Resolution

Detect and resolve conflicting operations:

```go
// Conflict types
type ConflictType string

const (
    ConflictFileEdit     ConflictType = "file.edit"     // Same file edited
    ConflictTaskClaim    ConflictType = "task.claim"    // Race to claim
    ConflictChannelRace  ConflictType = "channel.race"  // Concurrent sends
    ConflictWorktreeMerge ConflictType = "worktree.merge" // Git conflicts
)

// Resolution strategies
type Resolution string

const (
    ResolveFirst    Resolution = "first"     // First wins
    ResolveLast     Resolution = "last"      // Last wins
    ResolveMerge    Resolution = "merge"     // Attempt merge
    ResolveEscalate Resolution = "escalate"  // Ask human/manager
)
```

**Effort:** ~4 PRs
**Impact:** Medium (reduces manual intervention)

## Recommendation

**Phase 1: Documentation First (Tier 1)**
- Document existing channel/team/routing patterns
- Create example workflows
- No code changes, high impact

**Phase 2: Task Queue (Tier 2)**
- Most requested missing feature
- Enables structured work delegation
- Foundation for automation

**Phase 3: Event Bus (Tier 3)**
- Required for real-time TUI updates
- Enables plugin hooks
- Can start with SQLite triggers

**Phase 4: Future (Tier 4-5)**
- Context sharing and conflict resolution
- After core orchestration solidified

## Implementation Plan

### Phase 1: Documentation (2 PRs)
1. Create docs/orchestration/ directory
2. Write comprehensive guide with examples

### Phase 2: Task Queue (4 PRs)
3. Task database schema and store
4. `bc task` command group
5. Task routing integration
6. TUI Tasks view

### Phase 3: Event Bus (6 PRs)
7. Event types and interfaces
8. SQLite trigger implementation
9. Agent subscription management
10. Plugin hook integration
11. TUI real-time updates
12. Documentation

## Success Metrics

- Documentation: 80% fewer "how do I..." questions
- Task Queue: 50% reduction in manual task assignment
- Event Bus: Sub-second TUI updates
- Overall: Agents can coordinate without human mediation

## Open Questions

1. Should task queue be opt-in or default?
2. Event bus: SQLite vs external service?
3. Context sharing: TTL or manual cleanup?
4. Conflict resolution: Automatic or always escalate?
5. How to handle offline agents with pending tasks?

## References

- [Existing Channel Documentation](../channel-conventions.md)
- [Agent Hierarchy](../hierarchical-agents.md)
- [Memory System](../memory-system.md)
- [Celery](https://docs.celeryq.dev/) - Python task queue
- [NATS](https://nats.io/) - Cloud-native messaging
