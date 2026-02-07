# bc CLI Reference

The `bc` command-line interface provides control over agent orchestration, work queue management, and workspace diagnostics.

## Command Categories

| Category | Commands | Purpose |
|----------|----------|---------|
| **Agent Lifecycle** | spawn, up, down, status, attach | Manage agent sessions |
| **Work Queue** | queue | Manage work items |
| **Communication** | send, channel | Inter-agent messaging |
| **Workspace** | init, home, worktree | Workspace management |
| **Diagnostics** | logs, stats, dashboard | Monitoring and debugging |
| **Merge** | merge | Merge queue processing |

---

## Agent Lifecycle

### bc up

Start the workspace coordinator and optionally restore agents.

```bash
bc up                          # Start coordinator
bc up --restore                # Restore previous agents
bc up --agent claude           # Use specific AI tool
```

**Flags:**
- `--restore` - Restore agents from previous session
- `--agent <name>` - AI tool to use (default: from config)

**Behavior:**
1. Creates `.bc/` directory structure if needed
2. Starts coordinator agent in tmux session
3. Loads workspace configuration

### bc down

Stop all agents and optionally clean up.

```bash
bc down                        # Stop all agents
bc down --force                # Force stop without cleanup
```

**Flags:**
- `--force` - Skip graceful shutdown
- `--clean` - Remove worktrees after stop

**Behavior:**
1. Sends stop signal to all agents
2. Waits for graceful shutdown
3. Kills tmux sessions
4. Optionally removes worktrees

### bc spawn

Spawn a new agent with the specified role.

```bash
bc spawn <name> --role <role>
bc spawn eng-01 --role engineer
bc spawn pm-01 --role product-manager
bc spawn qa-01 --role qa --parent mgr-01
bc spawn eng-02 --role engineer --tool cursor-agent
```

**Arguments:**
- `<name>` - Unique agent identifier

**Flags:**
- `--role <role>` - Agent role: product-manager, manager, engineer, qa (required)
- `--parent <id>` - Parent agent ID for hierarchy
- `--tool <name>` - AI tool override (claude, cursor-agent)

**Roles:**
| Role | Description |
|------|-------------|
| `product-manager` | Top-level coordinator, creates epics |
| `manager` | Breaks down work, manages team |
| `engineer` | Implements code |
| `qa` | Tests and validates |
| `coordinator` | Legacy: maps to product-manager |
| `worker` | Legacy: maps to engineer |

**Behavior:**
1. Validates role and parent hierarchy
2. Creates git worktree at `.bc/worktrees/<name>/`
3. Starts tmux session with environment variables
4. Loads role prompt from `prompts/<role>.md`

### bc status

Show agent status overview.

```bash
bc status                      # Show all agents
bc status --json               # JSON output
bc status <agent>              # Show specific agent
```

**Flags:**
- `--json` - Output in JSON format
- `--watch` - Continuous refresh

**Output:**
```
AGENT      ROLE       STATE     TASK
pm-01      product    idle
mgr-01     manager    working   Planning sprint
eng-01     engineer   working   Implementing login
qa-01      qa         idle
```

### bc attach

Attach to an agent's tmux session.

```bash
bc attach <agent>
bc attach eng-01
```

**Arguments:**
- `<agent>` - Agent identifier

**Usage:**
- Use `Ctrl+b d` to detach from session
- Session continues running after detach

---

## Work Queue

### bc queue

Manage the work queue.

#### bc queue add

Add a work item to the queue.

```bash
bc queue add "Title"
bc queue add "Fix login bug" --description "Users can't login with SSO"
bc queue add "Implement feature" --beads-id gt-123
```

**Arguments:**
- `<title>` - Work item title (required)

**Flags:**
- `--description <text>` - Detailed description
- `--beads-id <id>` - Link to beads issue

#### bc queue list

List work items.

```bash
bc queue list                  # List all items
bc queue list --status pending # Filter by status
bc queue list --agent eng-01   # Filter by assignee
bc queue list --json           # JSON output
```

**Flags:**
- `--status <status>` - Filter: pending, assigned, working, done, failed
- `--agent <id>` - Filter by assigned agent
- `--json` - JSON output

**Output:**
```
ID         STATUS    TITLE                    ASSIGNED TO
work-001   pending   Fix login bug
work-002   working   Implement auth           eng-01
work-003   done      Update docs              eng-02
```

#### bc queue assign

Assign a work item to an agent.

```bash
bc queue assign <work-id> <agent>
bc queue assign work-001 eng-01
```

#### bc queue status

Update work item status.

```bash
bc queue status <work-id> <status>
bc queue status work-001 working
bc queue status work-001 done
```

**Statuses:**
| Status | Description |
|--------|-------------|
| `pending` | Available for assignment |
| `assigned` | Claimed by agent |
| `working` | Being executed |
| `done` | Completed successfully |
| `failed` | Execution failed |

---

## Communication

### bc send

Send a message to an agent's tmux session.

```bash
bc send <agent> <message>
bc send eng-01 "Check your work queue"
bc send pm-01 "Status update needed"
```

**Arguments:**
- `<agent>` - Target agent identifier
- `<message>` - Message text

**Behavior:**
1. Finds agent's tmux session
2. Sends message via `tmux send-keys`
3. Sends Enter to submit

### bc channel

Manage communication channels.

#### bc channel list

List channels.

```bash
bc channel list
bc channel list --json
```

#### bc channel send

Broadcast to a channel.

```bash
bc channel send <channel> <message>
bc channel send announcements "System maintenance at 5pm"
```

#### bc channel create

Create a new channel.

```bash
bc channel create <name>
bc channel create team-updates
```

---

## Workspace

### bc init

Initialize a new bc workspace.

```bash
bc init                        # Initialize current directory
bc init <path>                 # Initialize specific path
```

**Creates:**
```
.bc/
├── agents/
├── bin/
├── logs/
├── worktrees/
├── config.json
├── queue.json
├── channels.json
└── events.jsonl
```

### bc home

Open the bc home screen TUI.

```bash
bc home                        # Launch home TUI
```

**Features:**
- Workspace registry
- Quick workspace switching
- Recent activity

### bc worktree

Manage per-agent git worktrees.

#### bc worktree list

List all worktrees.

```bash
bc worktree list
bc worktree list --json
```

#### bc worktree clean

Clean up orphaned worktrees.

```bash
bc worktree clean
bc worktree clean --dry-run
```

---

## Diagnostics

### bc logs

View the event log.

```bash
bc logs                        # Recent events
bc logs --follow               # Tail events
bc logs -n 50                  # Last 50 events
bc logs --json                 # JSON output
```

**Flags:**
- `--follow`, `-f` - Continuous streaming
- `-n <count>` - Number of events
- `--json` - JSON output

**Event Types:**
- `spawn` - Agent spawned
- `stop` - Agent stopped
- `state_change` - Agent state changed
- `work_assigned` - Work item assigned
- `work_completed` - Work item completed

### bc stats

Show workspace statistics.

```bash
bc stats                       # Summary stats
bc stats --json                # JSON output
```

**Output:**
```
Agents:     5 total (3 running, 2 stopped)
Work Queue: 12 items (3 pending, 5 working, 4 done)
Worktrees:  3 active
Uptime:     2h 15m
```

### bc dashboard

Show workspace dashboard with rich stats.

```bash
bc dashboard                   # Launch dashboard
```

**Features:**
- Real-time agent status
- Work queue summary
- Event feed

### bc report

Report agent state (called by agents).

```bash
bc report <state> [message]
bc report working "Implementing login"
bc report done "Completed implementation"
bc report stuck "Need help with auth flow"
```

**States:**
| State | Description |
|-------|-------------|
| `idle` | Ready for work |
| `working` | Actively working |
| `done` | Task completed |
| `stuck` | Needs assistance |
| `error` | Error occurred |

**Environment:**
Expects `BC_AGENT_ID` environment variable to identify caller.

---

## Merge Queue

### bc merge

Manage the merge queue.

#### bc merge list

List items ready for merge.

```bash
bc merge list
bc merge list --json
```

#### bc merge process

Process the next item in merge queue.

```bash
bc merge process
bc merge process --dry-run
```

#### bc merge status

Show merge queue status.

```bash
bc merge status
```

**Merge Statuses:**
| Status | Description |
|--------|-------------|
| `unmerged` | Ready for merge |
| `merging` | Currently being merged |
| `merged` | Successfully merged |
| `conflict` | Has merge conflicts |

---

## Global Flags

These flags work with all commands:

| Flag | Description |
|------|-------------|
| `--verbose`, `-v` | Enable verbose output |
| `--json` | Output in JSON format |
| `--help`, `-h` | Show help |

---

## Configuration

### Config Files

| File | Location | Purpose |
|------|----------|---------|
| `config.toml` | `~/.config/bc/config.toml` | Global config |
| `config.json` | `.bc/config.json` | Workspace config |

### Global Config (config.toml)

```toml
[agent]
command = "claude --dangerously-skip-permissions"

[[agents]]
name = "claude"
command = "claude --dangerously-skip-permissions"

[[agents]]
name = "cursor-agent"
command = "cursor-agent --force --print"

[tmux]
session_prefix = "bc"

[tui]
refresh_interval = "1s"
theme = "ayu-dark"
```

### Workspace Config (.bc/config.json)

```json
{
  "workspace": "/path/to/project",
  "agent_command": "claude --dangerously-skip-permissions",
  "created_at": "2026-01-15T10:00:00Z"
}
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `BC_WORKSPACE` | Workspace root path |
| `BC_AGENT_ID` | Current agent identifier |
| `BC_AGENT_ROLE` | Current agent role |
| `BC_AGENT_WORKTREE` | Agent's worktree directory |
| `BC_AGENT_TOOL` | AI tool name |
| `BC_PARENT_ID` | Parent agent ID |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Agent not found |
| 4 | Workspace not initialized |

---

## Typical Workflows

### Starting a Session

```bash
# Initialize workspace (first time)
bc init

# Start coordinator
bc up

# Check status
bc status

# Spawn workers
bc spawn eng-01 --role engineer
bc spawn eng-02 --role engineer
```

### Assigning Work

```bash
# Add work to queue
bc queue add "Implement user auth"

# Assign to agent
bc queue assign work-001 eng-01

# Agent reports progress
bc report working "Starting implementation"
bc report done "Auth implemented"
```

### Monitoring

```bash
# Watch status
bc status --watch

# View logs
bc logs --follow

# Check stats
bc stats
```

### Ending Session

```bash
# Stop all agents
bc down

# Clean up worktrees
bc worktree clean
```

---

## See Also

- [Agent Types](02-agent-types.md) - Role definitions and capabilities
- [Data Models](04-data-models.md) - Data structures and files
- [Workflows](05-workflows.md) - Common workflow patterns
