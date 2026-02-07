# bc Architecture Overview

## Table of Contents

1. [System Overview](#system-overview)
2. [Core Concepts](#core-concepts)
3. [Hierarchical Organization](#hierarchical-organization)
4. [Data Flow](#data-flow)
5. [Key Design Principles](#key-design-principles)
6. [Technology Stack](#technology-stack)

---

## System Overview

**bc** (beads coordinator) is a multi-agent orchestration system for Claude Code with predictable behavior and cost awareness. It coordinates multiple AI agents working on different tasks without losing context when agents restart.

### The Problem bc Solves

| Challenge | bc Solution |
|-----------|-------------|
| Agents lose context on restart | State persists in git-backed `.bc/` directory |
| Multiple agents clobber each other | Per-agent git worktrees provide isolation |
| Work tracking lost in agent memory | Work queue stored in `queue.json` |
| Complex agent hierarchies | Simple 3-level hierarchy: PM → Manager → Engineer/QA |
| Unpredictable agent behavior | Role-based capabilities restrict actions |

### What bc Is

bc is a **workspace orchestrator** that:
- Coordinates multiple Claude Code agents working on different tasks
- Persists work state in git-backed files, enabling reliable multi-agent workflows
- Provides a clear hierarchy (PM → Manager → Engineer/QA) for organizing work
- Uses tmux sessions for agent isolation and liveness detection
- Gives each agent its own git worktree to prevent merge conflicts

---

## Core Concepts

### Workspace

The **workspace** is your project directory containing the `.bc/` subdirectory. It contains:
- Agent state and configuration
- Work queue
- Per-agent worktrees
- Event log

```
project/                       # Workspace root
├── .bc/                       # bc state directory
│   ├── agents/                # Agent state
│   ├── worktrees/             # Per-agent worktrees
│   │   ├── pm-01/
│   │   ├── mgr-01/
│   │   └── eng-01/
│   ├── queue.json             # Work queue
│   ├── events.jsonl           # Event log
│   └── config.json            # Workspace config
├── src/                       # Your code
└── ...
```

### Agents

**Agents** are Claude Code instances running in isolated tmux sessions. Each agent:
- Has a unique identifier (e.g., `eng-01`)
- Has a specific role with defined capabilities
- Runs in its own tmux session
- Works in its own git worktree

### Roles

bc uses a **role-based hierarchy** with four primary roles:

| Role | Level | Capabilities |
|------|-------|--------------|
| **ProductManager** | 0 | Creates epics, spawns managers, assigns work, reviews |
| **Manager** | 1 | Spawns engineers/QA, assigns work, reviews |
| **Engineer** | 2 | Implements code |
| **QA** | 2 | Tests and validates |

### Work Queue

The **work queue** (`queue.json`) tracks work items through their lifecycle:

```
pending → assigned → working → done
                              ↘ failed
```

Each work item has:
- Unique ID (e.g., `work-001`)
- Title and description
- Assigned agent
- Status and merge state

### Worktrees

Each agent gets its own **git worktree** at `.bc/worktrees/<agent>/`. This provides:
- Isolation between agents (no merge conflicts)
- Independent branches for each agent's work
- Clean state for each agent

---

## Hierarchical Organization

### Agent Hierarchy

```
┌─────────────────────────────────────────────────────────────────┐
│                         WORKSPACE                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────────┐                                           │
│   │ ProductManager  │  Level 0 - Top-level coordinator          │
│   │    (pm-01)      │                                           │
│   └────────┬────────┘                                           │
│            │ creates                                             │
│            ▼                                                     │
│   ┌─────────────────┐                                           │
│   │    Manager      │  Level 1 - Work decomposition             │
│   │   (mgr-01)      │                                           │
│   └────────┬────────┘                                           │
│            │ creates                                             │
│   ┌────────┴────────┬───────────────┐                           │
│   ▼                 ▼               ▼                           │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐                       │
│ │ Engineer │  │ Engineer │  │    QA    │  Level 2 - Execution  │
│ │ (eng-01) │  │ (eng-02) │  │ (qa-01)  │                       │
│ └──────────┘  └──────────┘  └──────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Role Capabilities

```go
var RoleCapabilities = map[Role][]Capability{
    RoleProductManager: {CapCreateAgents, CapAssignWork, CapCreateEpics, CapReviewWork},
    RoleManager:        {CapCreateAgents, CapAssignWork, CapReviewWork},
    RoleEngineer:       {CapImplementTasks},
    RoleQA:             {CapTestWork, CapReviewWork},
}
```

### Creation Rules

| Parent Role | Can Create |
|-------------|------------|
| ProductManager | Manager |
| Manager | Engineer, QA |
| Engineer | (none) |
| QA | (none) |

---

## Data Flow

### Work Assignment Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    1. WORK CREATION                              │
│  bc queue add "Implement auth"                                  │
│  → Creates: work-001 in queue.json (status: pending)            │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    2. AGENT SPAWN                                │
│  bc spawn eng-01 --role engineer                                │
│  → Creates: worktree at .bc/worktrees/eng-01/                   │
│  → Starts: tmux session with environment                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    3. WORK ASSIGNMENT                            │
│  bc queue assign work-001 eng-01                                │
│  → Updates: work-001 status to "assigned"                       │
│  → Sets: assigned_to = "eng-01"                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    4. EXECUTION                                  │
│  Agent works in worktree, reports progress:                     │
│  bc report working "Implementing login"                         │
│  → Updates: agent state and task                                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    5. COMPLETION                                 │
│  bc report done "Auth implemented"                              │
│  → Updates: work-001 status to "done"                           │
│  → Sets: merge status to "unmerged"                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    6. MERGE                                      │
│  bc merge process                                               │
│  → Merges: agent's branch to main                               │
│  → Updates: merge status to "merged"                            │
│  → Cleans up: worktree                                          │
└─────────────────────────────────────────────────────────────────┘
```

### Agent State Flow

```
┌──────────┐     spawn     ┌──────────┐
│ (none)   │──────────────▶│ starting │
└──────────┘               └────┬─────┘
                                │
                                ▼
                           ┌──────────┐
                           │   idle   │◀─────────────────────────┐
                           └────┬─────┘                          │
                                │                                │
                                ▼ work assigned                  │
                           ┌──────────┐                          │
                           │ working  │──────────────────────────┤
                           └────┬─────┘                          │
                                │                                │
                ┌───────────────┼───────────────┐                │
                ▼               ▼               ▼                │
           ┌──────────┐   ┌──────────┐   ┌──────────┐           │
           │   done   │   │  stuck   │   │  error   │           │
           └────┬─────┘   └────┬─────┘   └────┬─────┘           │
                │              │              │                  │
                └──────────────┴──────────────┴──────────────────┘
                                │
                                ▼ bc down
                           ┌──────────┐
                           │ stopped  │
                           └──────────┘
```

---

## Key Design Principles

### 1. Git-Backed Persistence

All state is stored in git-tracked files:
- **Agent state** - `.bc/agents/agents.json`
- **Work queue** - `.bc/queue.json`
- **Events** - `.bc/events.jsonl`

Benefits:
- Survives crashes and restarts
- Provides rollback capability
- Enables debugging via history

### 2. Tmux Session Isolation

Each agent runs in its own tmux session:
- Session name = agent ID
- `tmux has-session` determines liveness
- No PID files or heartbeats needed

Session naming: `bc-<workspace-hash>-<agent-id>`

### 3. Per-Agent Worktrees

Each agent gets a git worktree:
```
.bc/worktrees/
├── pm-01/              # PM's isolated copy
├── mgr-01/             # Manager's isolated copy
└── eng-01/             # Engineer's isolated copy
```

Benefits:
- No merge conflicts between agents
- Each agent has clean git state
- Parallel work on same files

### 4. Role-Based Capabilities

Actions are gated by role capabilities:

```go
// Only PMs and Managers can spawn agents
if !agent.HasCapability(CapCreateAgents) {
    return errors.New("not authorized to spawn agents")
}
```

This prevents:
- Engineers spawning other agents
- QA assigning work
- Unauthorized hierarchy creation

### 5. Simple State Machine

Agent states follow a clear state machine with validated transitions:

| Current State | Valid Transitions |
|--------------|-------------------|
| starting | idle, error, stopped |
| idle | working, done, stuck, error, stopped |
| working | idle, done, stuck, error, stopped |
| done | idle, working, stopped |
| stuck | idle, working, error, stopped |
| error | idle, working, stopped |
| stopped | idle, starting |

---

## Technology Stack

### Core Technologies

| Component | Technology | Purpose |
|-----------|------------|---------|
| CLI Framework | **Go + Cobra** | Command-line interface |
| TUI Components | **Bubble Tea** | Interactive terminal UI |
| Session Management | **tmux** | Agent session lifecycle |
| State Persistence | **JSON/JSONL** | Configuration and events |
| Version Control | **Git worktrees** | Agent isolation |

### Language: Go

bc is written in Go for:
- Single binary distribution
- Cross-platform support
- Strong concurrency primitives
- Fast compilation

### CLI: Cobra

The `bc` CLI uses Cobra for:
- Subcommand structure (`bc spawn`, `bc status`, etc.)
- Flag parsing
- Shell completions
- Help generation

### Session Management: tmux

tmux provides:
- Detached session management
- Session persistence
- Programmatic control via CLI

Session operations:
```bash
# Create session
tmux new-session -d -s <name> -c <dir>

# Check if session exists
tmux has-session -t <name>

# Send keys to session
tmux send-keys -t <name> "message" Enter

# Kill session
tmux kill-session -t <name>
```

### State Storage: JSON

Simple JSON files for state:
- `agents.json` - Agent records
- `queue.json` - Work items
- `config.json` - Configuration

JSONL for append-only logs:
- `events.jsonl` - Event history

---

## Quick Reference

### Key Commands

```bash
# Workspace
bc init                        # Initialize workspace
bc up                          # Start coordinator
bc down                        # Stop all agents

# Agents
bc spawn <name> --role <role>  # Spawn agent
bc status                      # List agents
bc attach <agent>              # Attach to session

# Work queue
bc queue add "Title"           # Add work item
bc queue assign <id> <agent>   # Assign work
bc queue list                  # List items

# State reporting
bc report working "Task"       # Report working
bc report done "Complete"      # Report done

# Merging
bc merge list                  # List mergeable
bc merge process               # Process merge queue
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `BC_WORKSPACE` | Workspace root |
| `BC_AGENT_ID` | Agent identifier |
| `BC_AGENT_ROLE` | Agent role |
| `BC_AGENT_WORKTREE` | Worktree path |
| `BC_AGENT_TOOL` | AI tool name |
| `BC_PARENT_ID` | Parent agent ID |

### File Locations

| File | Location | Purpose |
|------|----------|---------|
| Agent state | `.bc/agents/agents.json` | Agent records |
| Work queue | `.bc/queue.json` | Work items |
| Events | `.bc/events.jsonl` | Event log |
| Config | `.bc/config.json` | Workspace config |
| Worktrees | `.bc/worktrees/<agent>/` | Per-agent code |
| Git wrapper | `.bc/bin/git` | Worktree enforcement |

---

## Summary

bc is a streamlined orchestration layer that enables coordinated multi-agent development. Key insights:

1. **Git is the persistence layer** - All state survives restarts
2. **Tmux is the isolation layer** - Each agent in its own session
3. **Worktrees prevent conflicts** - Parallel work without merge issues
4. **Roles constrain behavior** - Predictable, auditable actions
5. **Simple queue model** - Clear work lifecycle

The architecture prioritizes simplicity and predictability over flexibility, making it easier to understand, debug, and trust.
