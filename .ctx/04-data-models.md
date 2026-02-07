# bc Data Models

This document describes the data structures and file formats used by bc for agent orchestration, work queue management, and state persistence.

---

## Directory Structure

### .bc/ Workspace Directory

```
.bc/                           # bc workspace root
├── agents/                    # Agent state directory
│   └── agents.json            # All agent states
├── bin/                       # Wrapper scripts
│   └── git                    # Git wrapper for worktree enforcement
├── logs/                      # Agent logs
├── worktrees/                 # Per-agent git worktrees
│   ├── pm-01/                 # ProductManager worktree
│   ├── mgr-01/                # Manager worktree
│   ├── eng-01/                # Engineer worktree
│   └── qa-01/                 # QA worktree
├── config.json                # Workspace configuration
├── queue.json                 # Work queue items
├── channels.json              # Communication channels
└── events.jsonl               # Append-only event log
```

---

## 1. Agent State

### agents.json

Located at: `.bc/agents/agents.json`

Stores the current state of all agents in the workspace.

```json
{
  "pm-01": {
    "id": "pm-01",
    "name": "pm-01",
    "role": "product-manager",
    "state": "idle",
    "workspace": "/path/to/project",
    "session": "pm-01",
    "parent_id": "",
    "children": ["mgr-01"],
    "hooked_work": "",
    "worktree_dir": "/path/to/project/.bc/worktrees/pm-01",
    "tool": "claude",
    "task": "",
    "started_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T10:05:00Z"
  },
  "eng-01": {
    "id": "eng-01",
    "name": "eng-01",
    "role": "engineer",
    "state": "working",
    "workspace": "/path/to/project",
    "session": "eng-01",
    "parent_id": "mgr-01",
    "children": [],
    "hooked_work": "work-001",
    "worktree_dir": "/path/to/project/.bc/worktrees/eng-01",
    "tool": "claude",
    "task": "Implementing login feature",
    "started_at": "2026-01-15T10:10:00Z",
    "updated_at": "2026-01-15T11:30:00Z"
  }
}
```

### Agent Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique agent identifier |
| `name` | string | Display name (usually same as id) |
| `role` | string | Agent role: product-manager, manager, engineer, qa, coordinator, worker |
| `state` | string | Current state (see Agent States) |
| `workspace` | string | Workspace root path |
| `session` | string | Tmux session name |
| `parent_id` | string | Parent agent ID (empty if top-level) |
| `children` | []string | Child agent IDs |
| `hooked_work` | string | Currently assigned work item ID |
| `worktree_dir` | string | Path to agent's git worktree |
| `tool` | string | AI tool name (claude, cursor-agent) |
| `task` | string | Current task description |
| `memory` | object | Role-specific prompt content (optional) |
| `started_at` | string | ISO 8601 spawn timestamp |
| `updated_at` | string | ISO 8601 last update timestamp |

### Agent States

| State | Description |
|-------|-------------|
| `idle` | Ready for work, no active task |
| `starting` | Session initializing |
| `working` | Actively executing task |
| `done` | Task completed |
| `stuck` | Needs assistance |
| `error` | Error occurred |
| `stopped` | Session terminated |

### Agent Roles

| Role | Level | Description |
|------|-------|-------------|
| `product-manager` | 0 | Creates epics, spawns managers |
| `manager` | 1 | Breaks down work, spawns engineers/QA |
| `engineer` | 2 | Implements code |
| `qa` | 2 | Tests and validates |
| `coordinator` | 0 | Legacy: like product-manager |
| `worker` | 2 | Legacy: like engineer |

---

## 2. Work Queue

### queue.json

Located at: `.bc/queue.json`

Stores all work items in the queue.

```json
[
  {
    "id": "work-001",
    "beads_id": "gt-abc123",
    "title": "Implement user authentication",
    "description": "Add login/logout functionality with OAuth2",
    "status": "working",
    "assigned_to": "eng-01",
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-01-15T11:00:00Z",
    "branch": "eng-01/work-001/auth",
    "merge": "unmerged",
    "merged_at": "",
    "merge_commit": ""
  },
  {
    "id": "work-002",
    "beads_id": "",
    "title": "Fix login button styling",
    "description": "",
    "status": "pending",
    "assigned_to": "",
    "created_at": "2026-01-15T11:00:00Z",
    "updated_at": "2026-01-15T11:00:00Z",
    "branch": "",
    "merge": "",
    "merged_at": "",
    "merge_commit": ""
  }
]
```

### Work Item Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique work item ID (auto-generated: work-001, work-002, ...) |
| `beads_id` | string | Optional link to beads issue |
| `title` | string | Brief title |
| `description` | string | Detailed description |
| `status` | string | Item status (see Work Item Statuses) |
| `assigned_to` | string | Agent ID (empty if unassigned) |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last update timestamp |
| `branch` | string | Git branch name (set when work starts) |
| `merge` | string | Merge status (see Merge Statuses) |
| `merged_at` | string | ISO 8601 merge timestamp |
| `merge_commit` | string | Merge commit SHA |

### Work Item Statuses

| Status | Description |
|--------|-------------|
| `pending` | Available for assignment |
| `assigned` | Claimed by agent but not started |
| `working` | Being actively executed |
| `done` | Completed successfully |
| `failed` | Execution failed |

### Merge Statuses

| Status | Description |
|--------|-------------|
| (empty) | Not applicable (item not done) |
| `unmerged` | Ready for merge |
| `merging` | Currently being merged |
| `merged` | Successfully merged |
| `conflict` | Has merge conflicts |

### Queue Statistics

```go
type Stats struct {
    Total    int  // Total items
    Pending  int  // Status: pending
    Assigned int  // Status: assigned
    Working  int  // Status: working
    Done     int  // Status: done
    Failed   int  // Status: failed
    Merged   int  // Merge: merged
    Unmerged int  // Merge: unmerged/merging/conflict
}
```

---

## 3. Communication Channels

### channels.json

Located at: `.bc/channels.json`

Stores communication channel data.

```json
{
  "version": 1,
  "channels": {
    "announcements": {
      "name": "announcements",
      "created_at": "2026-01-15T10:00:00Z",
      "messages": [
        {
          "id": "msg-001",
          "from": "pm-01",
          "content": "Sprint planning at 2pm",
          "timestamp": "2026-01-15T11:00:00Z"
        }
      ]
    },
    "engineering": {
      "name": "engineering",
      "created_at": "2026-01-15T10:00:00Z",
      "messages": []
    }
  }
}
```

### Channel Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Channel name |
| `created_at` | string | ISO 8601 creation timestamp |
| `messages` | []Message | Channel messages |

### Message Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Message ID |
| `from` | string | Sender agent ID |
| `content` | string | Message content |
| `timestamp` | string | ISO 8601 send timestamp |

---

## 4. Event Log

### events.jsonl

Located at: `.bc/events.jsonl`

Append-only log of all events. One JSON object per line.

```jsonl
{"ts":"2026-01-15T10:00:00Z","type":"workspace_init","actor":"user","payload":{"workspace":"/path/to/project"}}
{"ts":"2026-01-15T10:01:00Z","type":"agent_spawn","actor":"bc","payload":{"agent":"pm-01","role":"product-manager"}}
{"ts":"2026-01-15T10:05:00Z","type":"agent_spawn","actor":"pm-01","payload":{"agent":"mgr-01","role":"manager"}}
{"ts":"2026-01-15T10:10:00Z","type":"work_added","actor":"pm-01","payload":{"id":"work-001","title":"Implement auth"}}
{"ts":"2026-01-15T10:15:00Z","type":"work_assigned","actor":"mgr-01","payload":{"id":"work-001","agent":"eng-01"}}
{"ts":"2026-01-15T10:16:00Z","type":"state_change","actor":"eng-01","payload":{"from":"idle","to":"working","task":"Implementing auth"}}
{"ts":"2026-01-15T11:30:00Z","type":"state_change","actor":"eng-01","payload":{"from":"working","to":"done","task":"Completed auth"}}
{"ts":"2026-01-15T11:35:00Z","type":"work_merged","actor":"bc","payload":{"id":"work-001","commit":"abc123"}}
```

### Event Base Structure

```json
{
  "ts": "2026-01-15T10:00:00Z",
  "type": "event_type",
  "actor": "who_triggered",
  "payload": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ts` | string | ISO 8601 timestamp (UTC) |
| `type` | string | Event type (see Event Types) |
| `actor` | string | Who triggered the event |
| `payload` | object | Event-specific data |

### Event Types

| Type | Payload | Description |
|------|---------|-------------|
| `workspace_init` | `{workspace}` | Workspace initialized |
| `agent_spawn` | `{agent, role, parent}` | Agent spawned |
| `agent_stop` | `{agent}` | Agent stopped |
| `state_change` | `{from, to, task}` | Agent state changed |
| `work_added` | `{id, title}` | Work item added to queue |
| `work_assigned` | `{id, agent}` | Work item assigned |
| `work_completed` | `{id, status}` | Work item completed |
| `work_merged` | `{id, commit}` | Work item merged |
| `message_sent` | `{from, to, content}` | Direct message sent |
| `channel_message` | `{channel, from, content}` | Channel message sent |

---

## 5. Workspace Configuration

### config.json

Located at: `.bc/config.json`

Workspace-specific configuration.

```json
{
  "workspace": "/path/to/project",
  "agent_command": "claude --dangerously-skip-permissions",
  "created_at": "2026-01-15T10:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `workspace` | string | Workspace root path |
| `agent_command` | string | Default agent command override |
| `created_at` | string | ISO 8601 creation timestamp |

---

## 6. Global Configuration

### config.toml

Located at: `~/.config/bc/config.toml`

Global bc configuration.

```toml
# Application metadata
name = "bc"
version = "0.1.0"

[agent]
command = "claude --dangerously-skip-permissions"
coordinator_name = "coordinator"
worker_prefix = "worker"

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

[costs]
enabled = false
limit = 100
warn_threshold = 0.8

[[roles]]
name = "product-manager"
prompt_file = "prompts/product_manager.md"
permissions = ["create_agents", "assign_work", "create_epics", "review_work"]

[[roles]]
name = "manager"
prompt_file = "prompts/manager.md"
permissions = ["create_agents", "assign_work", "review_work"]

[[roles]]
name = "engineer"
prompt_file = "prompts/engineer.md"
permissions = ["implement_tasks"]

[[roles]]
name = "qa"
prompt_file = "prompts/qa.md"
permissions = ["test_work", "review_work"]
```

### Configuration Sections

| Section | Purpose |
|---------|---------|
| `agent` | Default agent settings |
| `agents` | Available AI tools |
| `tmux` | Tmux session settings |
| `tui` | Terminal UI settings |
| `costs` | Cost tracking (future) |
| `roles` | Role definitions |

---

## 7. Git Wrapper

### .bc/bin/git

A shell script that shadows `/usr/bin/git` to warn on write operations outside the agent's worktree.

```bash
#!/bin/bash
# bc worktree enforcement — warns on git write ops outside agent worktree
REAL_GIT="/usr/bin/git"

# No-op when BC_AGENT_WORKTREE is unset (tests, human usage)
if [ -z "$BC_AGENT_WORKTREE" ]; then
    exec "$REAL_GIT" "$@"
fi

# Check if CWD is inside the agent's worktree
case "$PWD" in
    "$BC_AGENT_WORKTREE"*) ;; # Inside worktree — OK
    *)
        # Warn only on write operations, not reads
        case "$1" in
            checkout|commit|push|reset|clean|merge|rebase|stash|add|rm|mv|init)
                echo "WARNING: git $1 outside worktree ($PWD != $BC_AGENT_WORKTREE)" >&2
                ;;
        esac
        ;;
esac

exec "$REAL_GIT" "$@"
```

---

## 8. Agent Memory

### AgentMemory Structure

Optional role-specific content loaded from prompt files.

```go
type AgentMemory struct {
    LoadedAt   time.Time `json:"loaded_at"`
    RolePrompt string    `json:"role_prompt"`
}
```

### Prompt Files

Located at: `prompts/<role>.md`

```
prompts/
├── product_manager.md    # PM instructions
├── manager.md            # Manager instructions
├── engineer.md           # Engineer instructions
└── qa.md                 # QA instructions
```

Role names are normalized: `product-manager` → `product_manager.md`

---

## Type Reference

### Go Type Definitions

From `pkg/agent/agent.go`:

```go
type Role string

const (
    RoleCoordinator    Role = "coordinator"      // Legacy
    RoleWorker         Role = "worker"           // Legacy
    RoleProductManager Role = "product-manager"
    RoleManager        Role = "manager"
    RoleEngineer       Role = "engineer"
    RoleQA             Role = "qa"
)

type State string

const (
    StateIdle     State = "idle"
    StateStarting State = "starting"
    StateWorking  State = "working"
    StateDone     State = "done"
    StateStuck    State = "stuck"
    StateError    State = "error"
    StateStopped  State = "stopped"
)

type Capability string

const (
    CapCreateAgents   Capability = "create_agents"
    CapAssignWork     Capability = "assign_work"
    CapCreateEpics    Capability = "create_epics"
    CapImplementTasks Capability = "implement_tasks"
    CapReviewWork     Capability = "review_work"
    CapTestWork       Capability = "test_work"
)
```

From `pkg/queue/queue.go`:

```go
type ItemStatus string

const (
    StatusPending  ItemStatus = "pending"
    StatusAssigned ItemStatus = "assigned"
    StatusWorking  ItemStatus = "working"
    StatusDone     ItemStatus = "done"
    StatusFailed   ItemStatus = "failed"
)

type MergeStatus string

const (
    MergeNone     MergeStatus = ""
    MergeUnmerged MergeStatus = "unmerged"
    MergeMerging  MergeStatus = "merging"
    MergeMerged   MergeStatus = "merged"
    MergeConflict MergeStatus = "conflict"
)
```

---

## File Locations Summary

| File | Location | Purpose |
|------|----------|---------|
| `agents.json` | `.bc/agents/agents.json` | Agent states |
| `queue.json` | `.bc/queue.json` | Work queue |
| `channels.json` | `.bc/channels.json` | Communication channels |
| `events.jsonl` | `.bc/events.jsonl` | Event log |
| `config.json` | `.bc/config.json` | Workspace config |
| `config.toml` | `~/.config/bc/config.toml` | Global config |
| `git` | `.bc/bin/git` | Git wrapper |
| Worktrees | `.bc/worktrees/<agent>/` | Per-agent worktrees |
| Prompts | `prompts/<role>.md` | Role prompts |
