# Memory System

The bc memory system provides per-agent persistent storage for experiences and learnings. Each agent maintains its own memory directory, enabling agents to learn from past tasks and improve over time.

## Directory Structure

```
.bc/memory/
  <agent-name>/
    experiences.jsonl    # Task outcomes and learnings
    learnings.md         # Accumulated insights (markdown)
```

## File Formats

### experiences.jsonl

A JSON Lines file where each line is a complete JSON object representing one experience.

**Schema:**

```json
{
  "timestamp": "2026-02-08T10:30:00Z",
  "task_id": "TASK-123",
  "task_type": "code",
  "description": "Fixed authentication bug in login flow",
  "outcome": "success",
  "learnings": ["Always validate JWT tokens before use"],
  "pinned": false,
  "metadata": {
    "pr_number": 123,
    "files_changed": ["auth.go", "auth_test.go"]
  }
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timestamp` | ISO 8601 | Auto-set | When the experience was recorded |
| `task_id` | string | No | Associated task identifier |
| `task_type` | string | No | Type: code, review, qa, docs, etc. |
| `description` | string | Yes | What was done |
| `outcome` | string | Yes | success, failure, partial |
| `learnings` | string[] | No | Key takeaways from the task |
| `pinned` | boolean | No | If true, preserved during pruning |
| `metadata` | object | No | Additional structured data |

### learnings.md

A Markdown file for accumulated insights organized by category.

**Format:**

```markdown
# <agent-name> Learnings

This file contains insights and learnings accumulated by <agent-name>.

## Patterns

- Always check error returns in Go
- Use context for cancellation in long-running operations

## Anti-patterns

- Don't ignore errors silently
- Avoid global state when possible

## Tips

- Use table-driven tests for comprehensive coverage
- Prefer composition over inheritance
```

## CLI Commands

### bc memory record

Record a task experience to memory.

```bash
# Basic usage
bc memory record "Fixed authentication bug"

# With outcome
bc memory record --outcome success "Implemented feature X"
bc memory record --outcome failure "Debugging session - root cause not found"

# With task metadata
bc memory record --task-id TASK-123 --task-type code "Completed login feature"
```

**Flags:**
- `--outcome` - Task outcome: success (default), failure, partial
- `--task-id` - Associated task identifier
- `--task-type` - Task type: code, review, qa, docs, etc.

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory learn

Add an insight to the learnings file.

```bash
# Add a pattern
bc memory learn patterns "Always validate input at system boundaries"

# Add a tip
bc memory learn tips "Use context.WithTimeout for API calls"

# Add an anti-pattern
bc memory learn anti-patterns "Don't catch and ignore errors"
```

**Arguments:**
1. `category` - Category heading (patterns, anti-patterns, tips, gotchas, etc.)
2. `learning` - The insight to record

**Note:** Requires `BC_AGENT_ID` environment variable.

### bc memory show

Display memory contents for an agent.

```bash
# Show current agent's memory (uses BC_AGENT_ID)
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

Search across agent memories with relevance ranking.

```bash
# Search all agents
bc memory search "authentication"

# Search specific agent
bc memory search --agent engineer-01 "bug fix"
```

**Flags:**
- `--agent` - Search only this agent's memory

**Relevance Scoring:**
- Matches in description: +10 points
- Exact word matches: +5 bonus
- Matches in task type: +5 points
- Matches in learnings: +7 points
- Header matches (##): +3 bonus

### bc memory prune

Remove old experiences to prevent unbounded growth.

```bash
# Prune experiences older than 30 days
bc memory prune --older-than 30d

# Preview what would be removed (dry run)
bc memory prune --older-than 7d --dry-run

# Prune without creating backup
bc memory prune --older-than 90d --no-backup

# Prune specific agent only
bc memory prune --agent engineer-01 --older-than 30d
```

**Flags:**
- `--older-than` - Duration threshold (e.g., 7d, 30d, 90d, 24h). Default: 30d
- `--dry-run` - Preview changes without deleting
- `--no-backup` - Skip backup creation before pruning
- `--agent` - Prune only this agent's memory

**Note:** Pinned experiences are always preserved regardless of age.

## Memory Injection

When agents are spawned, their memories are automatically injected into the context:

```go
// pkg/memory/memory.go
ctx, err := store.GetMemoryContext(limit)
```

This provides agents with:
- Recent experiences (configurable limit, default 10)
- All accumulated learnings

## Best Practices

### Recording Experiences

1. **Be specific** - Include what was done and why
2. **Note learnings** - Capture insights while fresh
3. **Use task IDs** - Link experiences to trackable work
4. **Pin important experiences** - Set `pinned: true` for critical learnings

### Managing Memory Growth

1. **Regular pruning** - Run `bc memory prune --older-than 30d` periodically
2. **Pin valuable experiences** - Important experiences survive pruning
3. **Use dry-run first** - Preview before deleting with `--dry-run`
4. **Keep backups** - Default behavior creates backup before pruning

### Searching Effectively

1. **Use specific terms** - More specific queries yield better results
2. **Check multiple agents** - Omit `--agent` to search all memories
3. **Review scores** - Higher scores indicate better matches

## Example Agent Workflow

```bash
# At task start
export BC_AGENT_ID=engineer-01

# During work - record progress
bc memory record --task-id "#123" --task-type code "Implemented login API"

# Capture learnings
bc memory learn patterns "Use bcrypt for password hashing"
bc memory learn tips "Check session expiry before API calls"

# On task completion
bc memory record --outcome success "Completed authentication feature with tests"

# Periodic maintenance
bc memory prune --older-than 30d --dry-run  # Preview
bc memory prune --older-than 30d            # Execute
```

## Integration with Agent Spawn

Memories are automatically loaded when spawning agents via `bc spawn`:

1. Agent's memory directory is checked
2. Recent experiences are loaded (last 10 by default)
3. Learnings are included in full
4. Context is formatted and injected into agent prompt

This enables agents to:
- Avoid repeating past mistakes
- Apply learned patterns
- Reference previous work on similar tasks
