# Agent Memory System

The bc agent memory system enables agents to persist and recall experiences and learnings across sessions. This document describes the memory storage format, CLI commands, and best practices.

## Overview

Each agent has a dedicated memory directory at `.bc/memory/<agent-name>/` containing:

- `experiences.jsonl` - Recorded task outcomes (JSON Lines format)
- `learnings.md` - Accumulated insights and patterns (Markdown)

## Experiences vs Learnings: When to Use Each

Understanding the distinction between experiences and learnings is crucial for effective memory management.

### Experiences: Transient Task Records

**What they are:** Time-stamped records of specific task outcomes - what you did, when, and how it turned out.

**Lifecycle:**
- **Transient** - Automatically pruned by age (default: 30 days)
- **Time-bound** - Include timestamp, can be filtered by date
- **High volume** - Expected to accumulate rapidly
- **Ephemeral context** - Useful for recent history, less valuable over time

**When to use experiences:**
- Recording task completions (`bc memory record "Fixed bug #123"`)
- Tracking success/failure outcomes
- Creating audit trail of work done
- Capturing context that's relevant short-term

**Examples:**
```bash
# Good experience records
bc memory record --outcome success "Merged PR #456 - auth refactor"
bc memory record --outcome failure --task-id 789 "Build failed - missing dependency"
bc memory record --outcome partial "Implemented 3 of 5 endpoints"
```

### Learnings: Permanent Knowledge Base

**What they are:** Enduring insights, patterns, and guidelines that remain valuable regardless of when they were learned.

**Lifecycle:**
- **Permanent** - Not automatically pruned
- **Timeless** - Value doesn't decay with age
- **Curated** - Should be reviewed and maintained manually
- **Shared wisdom** - Applicable across many future tasks

**When to use learnings:**
- Documenting patterns that work (`bc memory learn patterns "..."`)
- Recording anti-patterns to avoid
- Capturing tips and best practices
- Building institutional knowledge

**Examples:**
```bash
# Good learning records
bc memory learn patterns "Always run tests before committing"
bc memory learn anti-patterns "Never store secrets in config files"
bc memory learn gotchas "SQLite requires explicit Close() to avoid leaks"
bc memory learn best-practices "Use defer for cleanup in Go"
```

### Decision Guide

| Question | If Yes → | If No → |
|----------|----------|---------|
| Is this about a specific task? | Experience | Learning |
| Will this be relevant in 6 months? | Learning | Experience |
| Does it include a timestamp/date? | Experience | Learning |
| Is it a general principle or rule? | Learning | Experience |
| Would pruning it lose valuable info? | Learning | Experience |

### Lifecycle Comparison

| Aspect | Experiences | Learnings |
|--------|-------------|-----------|
| Storage format | JSONL (structured) | Markdown (human-readable) |
| Pruning | Automatic by age | Manual curation |
| Default retention | 30 days | Permanent |
| Volume expectation | High (hundreds) | Low (tens) |
| Search priority | Recent first | All equally weighted |
| Update frequency | Every task | Occasionally |

### Retention Policies

**Experiences:**
- Default: Prune after 30 days
- Aggressive: Prune after 7 days (high-volume agents)
- Conservative: Prune after 90 days (audit requirements)
- Use `--pinned` flag to preserve critical experiences from pruning

**Learnings:**
- Review quarterly for relevance
- Remove outdated guidance (e.g., deprecated APIs)
- Consolidate duplicate entries
- Keep categories organized and consistent

```bash
# Recommended maintenance schedule
# Weekly: Record experiences as you work
# Monthly: bc memory prune --older-than 30d --dry-run
# Quarterly: Review learnings.md for accuracy
```

## File Formats

### experiences.jsonl

Experiences are stored as JSON Lines (one JSON object per line):

```json
{"timestamp":"2026-02-08T12:00:00Z","description":"Fixed auth bug using JWT tokens","outcome":"success","task_id":"TASK-123","task_type":"bugfix","learnings":["Always validate token expiry"]}
```

**Schema:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timestamp` | ISO 8601 | Yes | When the experience was recorded |
| `description` | string | Yes | What happened |
| `outcome` | string | Yes | Result: `success`, `failure`, `partial` |
| `task_id` | string | No | Associated task identifier |
| `task_type` | string | No | Category: `bugfix`, `feature`, `refactor`, `review` |
| `learnings` | string[] | No | Key takeaways from the experience |
| `metadata` | object | No | Additional context |

### learnings.md

Learnings are stored as Markdown with categorized sections:

```markdown
# agent-name Learnings

This file contains insights and learnings accumulated by agent-name.

## patterns

- Always check error returns in Go
- Use context for cancellation

## anti-patterns

- Don't ignore linter warnings
- Avoid global state

## tips

- Run tests before committing
- Use --dry-run for destructive operations
```

## CLI Commands

### bc memory record

Record a task outcome or experience.

```bash
# Basic usage
bc memory record "Fixed authentication bug"

# With outcome
bc memory record --outcome success "Implemented feature X"
bc memory record --outcome failure "Build failed due to dependency"

# With task metadata
bc memory record --task-id TASK-123 --task-type bugfix "Resolved issue"
```

**Flags:**
- `--outcome` - Result of the task: `success` (default), `failure`, `partial`
- `--task-id` - Task identifier for correlation
- `--task-type` - Category: `bugfix`, `feature`, `refactor`, `review`, `qa`

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory learn

Add an insight or learning to memory.

```bash
# Add a pattern
bc memory learn patterns "Always validate input at boundaries"

# Add an anti-pattern
bc memory learn anti-patterns "Never commit secrets to git"

# Add a tip
bc memory learn tips "Use go test -race for concurrency bugs"
```

**Categories:** `patterns`, `anti-patterns`, `tips`, `gotchas`, `best-practices`

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory show

Display memory contents for an agent.

```bash
# Show current agent's memory
bc memory show

# Show specific agent's memory
bc memory show engineer-01

# Show only experiences
bc memory show --experiences

# Show only learnings
bc memory show --learnings
```

**Flags:**
- `--experiences` - Show only experiences
- `--learnings` - Show only learnings

### bc memory search

Search through agent memories with relevance ranking.

```bash
# Search all agents
bc memory search "authentication"

# Search specific agent
bc memory search --agent engineer-01 "bug"
```

**Flags:**
- `--agent` - Search specific agent's memory only

Results are ranked by relevance:
- Exact word matches score higher
- Description matches score higher than metadata
- Multiple occurrences increase score

### bc memory prune

Remove old memory entries to prevent unbounded growth.

```bash
# Remove entries older than 30 days (default)
bc memory prune --older-than 30d

# Preview what would be deleted
bc memory prune --older-than 7d --dry-run

# Prune all agents
bc memory prune --older-than 90d --all-agents

# Skip backup (not recommended)
bc memory prune --older-than 14d --no-backup
```

**Flags:**
- `--older-than` - Duration threshold (e.g., `7d`, `30d`, `24h`). Default: `30d`
- `--dry-run` - Preview without deleting
- `--no-backup` - Skip creating backup before pruning
- `--all-agents` - Prune all agent memories

**Note:** By default, creates a backup at `.bc/memory/<agent>/experiences.<timestamp>.jsonl.bak`

## Memory Injection

When agents spawn, their accumulated memory is automatically injected into their context. The system:

1. Loads the most recent experiences (default: 10)
2. Includes all learnings
3. Formats as a "Agent Memory" section in the prompt

This enables agents to:
- Recall past successes and failures
- Apply learned patterns
- Avoid repeating mistakes

## Best Practices

### Recording Experiences

1. **Be specific** - Include what you did and why it worked/failed
2. **Note learnings** - Capture key takeaways with `--outcome`
3. **Use task IDs** - Enable correlation with issue tracking

```bash
# Good
bc memory record --outcome success --task-id 123 "Fixed race condition by adding mutex - the shared map wasn't thread-safe"

# Less useful
bc memory record "Fixed bug"
```

### Managing Learnings

1. **Categorize properly** - Use consistent categories
2. **Be actionable** - Write learnings as guidance
3. **Update periodically** - Prune outdated learnings

### Memory Hygiene

1. **Prune regularly** - Run `bc memory prune --older-than 30d` monthly
2. **Use dry-run first** - Preview before deleting
3. **Keep backups** - Don't use `--no-backup` unless necessary

### Storage Limits

- Experiences: Keep last 100-500 entries per agent
- Learnings: Keep under 50KB per agent
- Total memory: Monitor `.bc/memory/` size

## Directory Structure

```
.bc/
└── memory/
    ├── engineer-01/
    │   ├── experiences.jsonl
    │   ├── experiences.20260208-120000.jsonl.bak
    │   └── learnings.md
    ├── engineer-02/
    │   ├── experiences.jsonl
    │   └── learnings.md
    └── coordinator/
        ├── experiences.jsonl
        └── learnings.md
```

## Integration with Agent Lifecycle

| Event | Memory Action |
|-------|---------------|
| Agent spawn | Inject memory context |
| Task completion | Auto-record experience |
| `bc agent report done` | Record with outcome |
| Session end | Prune old entries |

## Troubleshooting

### "BC_AGENT_ID not set"

Memory commands require the agent ID. Set it manually for testing:

```bash
export BC_AGENT_ID=engineer-01
bc memory show
```

### "No memory found for agent"

The agent hasn't recorded any experiences yet. Initialize with:

```bash
bc memory record "Initial experience"
```

### Memory files corrupted

Restore from backup:

```bash
cp .bc/memory/agent/experiences.*.bak .bc/memory/agent/experiences.jsonl
```
