# bc CLI Reference Guide

Complete command reference for the bc multi-agent orchestration system.

## Table of Contents

1. [Workspace Commands](#workspace-commands)
2. [Agent Management](#agent-management)
3. [Work Queue](#work-queue)
4. [Git & Merging](#git--merging)
5. [Channels & Communication](#channels--communication)
6. [Configuration](#configuration)
7. [Monitoring & Logs](#monitoring--logs)
8. [Admin Commands](#admin-commands)

---

## Workspace Commands

Commands for managing the bc workspace.

### bc init
Initialize a new bc workspace in the current directory.

```bash
bc init
```

**Output:**
```
✓ Workspace initialized
✓ Config created: .bc/config.toml
✓ Queue initialized: .bc/queue.json
✓ Events log created: .bc/events.jsonl
```

**Options:**
```bash
bc init --name "my-project"           # Set workspace name
bc init --root /path/to/project       # Initialize in different directory
```

**Creates:**
- `.bc/` - State directory
- `.bc/config.toml` - Configuration file
- `.bc/queue.json` - Work queue
- `.bc/events.jsonl` - Event log

---

### bc up
Start the root coordinator agent and prepare workspace.

```bash
bc up
```

**Output:**
```
✓ Root agent started
✓ Workspace ready
✓ Use 'bc home' to view dashboard
✓ Use 'bc spawn' to create agents
```

**What it does:**
- Starts root coordinator
- Initializes tmux session
- Validates workspace
- Prepares for agent spawning

---

### bc down
Stop all agents and shut down the workspace.

```bash
bc down
```

**Output:**
```
✓ Stopping all agents...
✓ Killed 5 agents
✓ Saving workspace state
✓ All sessions closed
```

**Options:**
```bash
bc down --force      # Force kill all sessions
bc down --clean      # Remove temporary files
```

---

### bc status
Show current status of workspace and all agents.

```bash
bc status
```

**Output:**
```
Workspace: my-project
Status: running

AGENTS (5)
├── pm-01    ProductManager  idle       (1h 23m)
├── mgr-01   Manager         working    (45m on work-0001)
├── eng-01   Engineer        working    (2h 15m on work-0002)
├── eng-02   Engineer        idle       (awaiting assignment)
└── qa-01    QA              idle       (ready)

WORK QUEUE
├── work-0001  assigned  2h in      eng-01
├── work-0002  working   1h in      (from work-0001)
└── work-0003  pending   (unassigned)

Recent: eng-01 "Implementing user auth"
```

**Options:**
```bash
bc status --json          # Output as JSON
bc status --agent eng-01  # Status of single agent
bc status --queue         # Show only work queue
```

---

## Agent Management

Commands for spawning, managing, and controlling agents.

### bc spawn
Create and start a new agent.

```bash
bc spawn eng-01 --role engineer
```

**Output:**
```
✓ Agent eng-01 spawned
✓ Worktree created: .bc/worktrees/eng-01/
✓ tmux session: bc-xxxx-eng-01
✓ Ready to accept work
```

**Roles:**
- `product-manager` - Create epics, spawn managers, assign work
- `manager` - Spawn engineers/QA, assign work, review
- `engineer` - Implement code
- `qa` - Test and validate

**Options:**
```bash
bc spawn eng-01 --role engineer        # Basic spawn
bc spawn eng-01 --role engineer --parent mgr-01  # With parent
bc spawn eng-01 --role engineer --tool claude-code # Specify AI tool
```

**Example - Spawn hierarchical team:**
```bash
bc spawn pm-01 --role product-manager

bc spawn mgr-01 --role manager --parent pm-01
bc spawn mgr-02 --role manager --parent pm-01

bc spawn eng-01 --role engineer --parent mgr-01
bc spawn eng-02 --role engineer --parent mgr-01
bc spawn qa-01 --role qa --parent mgr-01

bc spawn eng-03 --role engineer --parent mgr-02
bc spawn qa-02 --role qa --parent mgr-02
```

---

### bc attach
Connect to an agent's tmux session and monitor work.

```bash
bc attach eng-01
```

**Inside session:**
- Type normally to send commands to agent
- Press `Ctrl+B` then `D` to detach without stopping agent
- Type `exit` to stop agent
- Agent continues work in background if you detach

**Options:**
```bash
bc attach eng-01 --read-only     # View-only mode
bc attach eng-01 --follow        # Stream mode (watch output)
```

---

### bc stop
Gracefully stop an agent.

```bash
bc stop eng-01
```

**Output:**
```
✓ Sent stop signal to eng-01
✓ Agent gracefully shutting down
✓ Work saved to .bc/
```

---

### bc kill
Forcefully kill an agent session.

```bash
bc kill eng-01
```

**Output:**
```
✓ Killed tmux session for eng-01
⚠ Work may be incomplete
✓ State saved
```

**Difference:**
- `bc stop` - Graceful shutdown
- `bc kill` - Immediate termination (use if agent stuck)

---

### bc list
List all agents in workspace.

```bash
bc list
```

**Output:**
```
AGENTS
┌─────────────────────────────────────────────────────────────┐
│ ID      │ Role              │ Status     │ Uptime    │ Task  │
├─────────────────────────────────────────────────────────────┤
│ pm-01   │ ProductManager    │ idle       │ 2h 45m    │ -     │
│ mgr-01  │ Manager           │ idle       │ 2h 40m    │ -     │
│ eng-01  │ Engineer          │ working    │ 2h 30m    │ 0001  │
│ eng-02  │ Engineer          │ idle       │ 1h 15m    │ -     │
│ qa-01   │ QA                │ idle       │ 45m       │ -     │
└─────────────────────────────────────────────────────────────┘
```

**Options:**
```bash
bc list --json              # JSON output
bc list --role engineer     # Filter by role
bc list --status working    # Filter by status
```

---

## Work Queue

Commands for managing the work queue and task assignments.

### bc queue add
Add a new task to the work queue.

```bash
bc queue add "Implement user authentication"
```

**Output:**
```
✓ Task created: work-0001
✓ Status: pending
✓ Ready for assignment
```

**Options:**
```bash
bc queue add "Title" --priority high          # Set priority
bc queue add "Title" --desc "Description"     # With description
bc queue add "Title" --estimate 4h            # Time estimate
```

---

### bc queue list
List all tasks in work queue.

```bash
bc queue list
```

**Output:**
```
WORK QUEUE
┌────────────────────────────────────────────────────────────────┐
│ ID    │ Status    │ Title                  │ Assigned │ Duration│
├────────────────────────────────────────────────────────────────┤
│ 0001  │ done      │ User authentication    │ eng-01   │ 2h 15m │
│ 0002  │ working   │ Payment integration    │ eng-02   │ 1h 45m │
│ 0003  │ assigned  │ Email notifications    │ eng-03   │ -      │
│ 0004  │ pending   │ Documentation          │ -        │ -      │
└────────────────────────────────────────────────────────────────┘
```

**Options:**
```bash
bc queue list --status working               # Filter by status
bc queue list --assigned-to eng-01           # Agent's tasks
bc queue list --json                         # JSON output
```

---

### bc queue show
Show detailed information about a specific task.

```bash
bc queue show work-0001
```

**Output:**
```
Task: work-0001
Title: User authentication implementation
Description: Implement JWT-based user auth with refresh tokens
Status: done
Assigned To: eng-01
Created: 2026-02-09 10:00:00 UTC
Started: 2026-02-09 10:15:00 UTC
Completed: 2026-02-09 14:30:00 UTC
Duration: 4h 15m
Priority: high
Branch: feature/user-auth
Commits: 5
Merge Status: merged
Merge Conflicts: 0
```

---

### bc queue assign
Assign a task to an agent.

```bash
bc queue assign work-0001 eng-01
```

**Output:**
```
✓ Assigned work-0001 to eng-01
✓ Status: assigned
✓ Ready for eng-01 to start
```

**Validation:**
- Task must exist
- Agent must exist and have role capability
- Agent must not already be assigned max tasks

---

### bc queue unassign
Remove assignment of a task.

```bash
bc queue unassign work-0001
```

**Output:**
```
✓ Unassigned work-0001 from eng-01
✓ Status: pending
✓ Ready for new assignment
```

---

### bc report
Report task status update.

```bash
bc report working "Implementing JWT validation"
```

**Statuses:**
- `working` - Task is in progress
- `done` - Task is completed
- `stuck` - Task is blocked
- `failed` - Task failed

**Examples:**
```bash
bc report working "Building API endpoints"
bc report done "Authentication complete - ready for testing"
bc report stuck "Waiting on database schema from team"
bc report failed "Dependency not compatible"
```

---

## Git & Merging

Commands for managing git integration and merging work.

### bc merge list
Show tasks ready to merge.

```bash
bc merge list
```

**Output:**
```
READY TO MERGE (3)
┌────────────────────────────────────────────────────────────────┐
│ ID    │ Agent  │ Branch             │ Changes │ Conflicts │ Time │
├────────────────────────────────────────────────────────────────┤
│ 0001  │ eng-01 │ feature/user-auth  │ 12f     │ 0         │ 4h   │
│ 0002  │ eng-02 │ feature/payments   │ 8f      │ 0         │ 6h   │
│ 0003  │ eng-03 │ feature/emails     │ 5f      │ 0         │ 3h   │
└────────────────────────────────────────────────────────────────┘
```

**Options:**
```bash
bc merge list --conflicts    # Show only conflicting
bc merge list --agent eng-01 # Agent's work
```

---

### bc merge process
Merge all ready tasks to main branch.

```bash
bc merge process
```

**Output:**
```
Merging work-0001 from eng-01...
✓ feature/user-auth merged to main
✓ Branch deleted

Merging work-0002 from eng-02...
✓ feature/payments merged to main
✓ Branch deleted

Merging work-0003 from eng-03...
✓ feature/emails merged to main
✓ Branch deleted

✓ All 3 tasks merged
✓ Worktrees cleaned up
```

**Options:**
```bash
bc merge process --dry-run       # Preview without merging
bc merge process --no-delete     # Keep branches after merge
```

---

### bc merge abort
Cancel in-progress merge.

```bash
bc merge abort
```

**Output:**
```
✓ Merge aborted
✓ Branches restored
✓ Worktrees cleaned
```

---

## Channels & Communication

Commands for agent communication.

### bc send
Send message to agent or channel.

```bash
# Send to agent
bc send eng-01 "Check requirements in /docs/spec.md"

# Send to channel
bc send #engineering "API ready for integration"
```

**Output:**
```
✓ Message sent to eng-01
✓ Timestamp: 2026-02-09 15:35:20
```

---

### bc logs
View communication and activity logs.

```bash
# Agent logs
bc logs eng-01

# Channel logs
bc logs #engineering

# Full system logs
bc logs
```

**Options:**
```bash
bc logs eng-01 --lines 50        # Last 50 lines
bc logs eng-01 --since "2h ago"  # Logs from past 2 hours
bc logs eng-01 --json            # JSON format
```

---

### bc channels list
List all active channels.

```bash
bc channels list
```

**Output:**
```
CHANNELS (6)
├── #engineering   (5 members)
├── #design        (3 members)
├── #qa            (2 members)
├── #deployments   (4 members)
├── #general       (5 members)
└── #random        (5 members)
```

---

## Configuration

Commands for workspace configuration.

### bc config show
Display current configuration.

```bash
bc config show
```

**Output:**
```
WORKSPACE CONFIG (.bc/config.toml)

[workspace]
name = "my-project"
root = "/Users/user/my-project"

[agents]
max_concurrent = 10
timeout = "30m"
auto_restart = true

[git]
auto_merge = true
merge_strategy = "squash"
```

---

### bc config set
Update configuration value.

```bash
bc config set workspace.name "new-name"
bc config set agents.max_concurrent 20
bc config set git.auto_merge false
```

---

## Monitoring & Logs

Commands for monitoring and debugging.

### bc home
Open interactive TUI dashboard.

```bash
bc home
```

**Features:**
- Real-time agent status
- Work queue progress
- Communication history
- System metrics
- Merge queue status

**Navigation:**
- ↑/↓ - Navigate
- Enter - View details
- q - Quit
- : - Command mode

---

### bc metrics
Show performance metrics.

```bash
bc metrics
```

**Output:**
```
METRICS
├── Tasks Completed: 42
├── Avg Duration: 2h 15m
├── Success Rate: 98.8%
├── Merge Conflicts: 2
├── Team Size: 5
└── Uptime: 23h 45m
```

---

### bc health
Check workspace health.

```bash
bc health
```

**Output:**
```
HEALTH CHECK
├── Workspace: OK
├── Git Repository: OK
├── Agents (5): 5 running, 0 stuck
├── Queue: 1 working, 2 pending
├── Disk Space: 2.3 GB / 5 GB
└── Overall: HEALTHY
```

---

## Admin Commands

Advanced administrative commands.

### bc backup
Create workspace backup.

```bash
bc backup
```

**Output:**
```
✓ Backup created: .bc/backup-2026-02-09-T15-35-20.tar.gz
✓ Size: 2.3 MB
✓ Location: .bc/backups/
```

---

### bc restore
Restore from backup.

```bash
bc restore .bc/backup-2026-02-09-T15-35-20.tar.gz
```

**Output:**
```
✓ Restoring from backup...
✓ Workspace restored
✓ All agents: running
✓ Queue: 42 items
```

---

### bc clean
Clean up temporary files and optimize workspace.

```bash
bc clean
```

**Output:**
```
✓ Removed temporary files
✓ Cleaned up old logs
✓ Optimized git repository
✓ Freed 156 MB
```

---

## Quick Reference

### Most Common Commands

```bash
# Daily workflow
bc status                    # Check status
bc queue add "Task title"    # Create task
bc queue assign work-0001 eng-01  # Assign
bc attach eng-01             # Watch agent
bc merge process             # Merge when done

# Team management
bc spawn eng-01 --role engineer   # Add agent
bc send eng-01 "Message"          # Communicate
bc logs eng-01                    # View history

# Dashboard
bc home                      # Open dashboard
```

### Exit Codes

```
0   - Success
1   - General error
2   - Invalid arguments
3   - Agent not found
4   - Task not found
5   - Permission denied
10  - Merge conflict
```

---

## Environment Variables

```bash
BC_WORKSPACE         # Workspace root (auto-detected)
BC_AGENT_ID          # Current agent ID
BC_AGENT_ROLE        # Current agent role
BC_AGENT_WORKTREE    # Agent's worktree path
BC_DEBUG             # Enable debug logging (true/false)
BC_VERBOSE           # Verbose output (true/false)
```

---

## Tips & Tricks

```bash
# Watch agent in real-time
bc attach eng-01 --follow

# Export queue as JSON
bc queue list --json > queue.json

# Get agent metrics
bc status --json | jq '.agents[] | {id, role, duration}'

# Find stuck agents
bc status | grep "stuck"

# Merge specific task
bc merge process --only work-0001
```

---

**For more help:** `bc --help` or `bc <command> --help`
