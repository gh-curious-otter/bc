# Agent Memory System

The bc memory system provides persistent storage for agent experiences and learnings, enabling agents to build knowledge over time and apply past insights to new tasks.

## Overview

Each agent has a dedicated memory directory at `.bc/memory/<agent-name>/` containing:

- `experiences.jsonl` - Recorded task outcomes (JSON Lines format)
- `learnings.md` - Accumulated insights and best practices

Memory is automatically loaded and injected into agent context at spawn time, helping agents leverage past experiences.

## File Formats

### experiences.jsonl

Each line is a JSON object representing a completed task:

```json
{
  "timestamp": "2026-02-08T12:30:00Z",
  "task_id": "TASK-123",
  "task_type": "bugfix",
  "description": "Fixed authentication timeout issue",
  "outcome": "success",
  "learnings": ["Use context.WithTimeout for API calls"],
  "pinned": false,
  "metadata": {
    "pr_number": 456,
    "files_changed": 3
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string | ISO 8601 timestamp when recorded |
| `task_id` | string | Optional task/issue identifier |
| `task_type` | string | Category: code, review, bugfix, feature, etc. |
| `description` | string | What was accomplished |
| `outcome` | string | Result: success, failure, partial |
| `learnings` | array | Insights gained from the task |
| `pinned` | bool | If true, preserved during pruning |
| `metadata` | object | Additional context (PR numbers, etc.) |

### learnings.md

Markdown file with categorized insights:

```markdown
# agent-name Learnings

Agent insights and lessons learned.

## Patterns

- Always check error returns before proceeding
- Use context for cancellation in long-running operations

## Anti-Patterns

- Don't ignore validation errors
- Avoid global state in concurrent code

## Tips

- Use table-driven tests for comprehensive coverage
- Prefer composition over inheritance
```

## CLI Commands

### bc memory record

Record a task experience to memory.

```bash
# Basic usage
bc memory record "Fixed login timeout bug"

# With outcome
bc memory record --outcome success "Implemented caching layer"
bc memory record --outcome failure "Migration failed - rollback needed"

# With task context
bc memory record --task-id TASK-123 --task-type bugfix "Resolved race condition"
```

**Flags:**
- `--outcome` - Task result: success (default), failure, partial
- `--task-id` - Associated task/issue ID
- `--task-type` - Task category (code, review, bugfix, feature, etc.)

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory learn

Add a learning or insight to memory.

```bash
bc memory learn patterns "Always validate input at API boundaries"
bc memory learn tips "Use defer for cleanup operations"
bc memory learn anti-patterns "Don't swallow errors silently"
bc memory learn gotchas "Time zones affect date comparisons"
```

**Categories:** patterns, anti-patterns, tips, gotchas

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory show

Display agent memory contents.

```bash
# Current agent's memory
bc memory show

# Specific agent
bc memory show engineer-01

# Filter by type
bc memory show --experiences    # Only experiences
bc memory show --learnings      # Only learnings
```

### bc memory search

Search across all agent memories with relevance ranking.

```bash
# Search all agents
bc memory search "authentication"

# Search specific agent
bc memory search --agent engineer-01 "database"
```

Results are ranked by relevance:
- Description matches score highest (10 pts)
- Task type/ID matches (5 pts)
- Outcome matches (3 pts)
- Learning matches (7 pts)

### bc memory prune

Remove old experiences to prevent unbounded growth.

```bash
# Preview what would be deleted
bc memory prune --older-than 30d --dry-run

# Actually prune
bc memory prune --older-than 30d

# Prune specific agent
bc memory prune --agent engineer-01 --older-than 7d
```

**Flags:**
- `--older-than` - Duration cutoff (e.g., 7d, 30d, 24h)
- `--dry-run` - Preview without deleting
- `--agent` - Target specific agent

**Note:** Creates backup (`experiences.jsonl.bak`) before pruning. Pinned experiences are preserved.

## Automatic Features

### Memory Injection at Spawn

When an agent spawns, recent experiences and learnings are automatically loaded and injected into the agent's prompt context. This provides:

- Context from previous similar tasks
- Known patterns and anti-patterns
- Accumulated team knowledge

Configure the injection limit (default: 10 experiences) via the memory store.

### Auto-Recording on Task Completion

When an agent reports completion with `bc report done "message"`, the experience is automatically recorded to memory:

```bash
bc report done "Fixed authentication bug in login flow"
# → Automatically records experience with outcome=success
```

Deduplication prevents recording the same description twice in recent history.

## Best Practices

### For Recording Experiences

1. **Be specific** - Include what was done and how
2. **Capture learnings** - Note insights while fresh
3. **Use consistent task types** - Enables filtering and analysis
4. **Pin important experiences** - Prevent loss during pruning

### For Learnings

1. **Categorize appropriately** - Use patterns/tips/gotchas consistently
2. **Be actionable** - Write learnings that can be applied
3. **Include context** - Why is this important?

### For Memory Management

1. **Prune regularly** - Keep memory focused and relevant
2. **Use dry-run first** - Verify before deleting
3. **Backup before major changes** - Automatic backups help
4. **Pin critical experiences** - Protect important knowledge

## Directory Structure

```
.bc/
└── memory/
    ├── engineer-01/
    │   ├── experiences.jsonl
    │   ├── experiences.jsonl.bak
    │   └── learnings.md
    ├── engineer-02/
    │   ├── experiences.jsonl
    │   └── learnings.md
    └── manager/
        ├── experiences.jsonl
        └── learnings.md
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BC_AGENT_ID` | Current agent identifier (required for record/learn) |
| `BC_AGENT_MEMORY` | Path to agent's memory directory (set at spawn) |
