# Gas Town Architecture Overview

## Table of Contents

1. [System Overview](#system-overview)
2. [Core Concepts](#core-concepts)
3. [Hierarchical Organization](#hierarchical-organization)
4. [Data Flow](#data-flow)
5. [Key Design Principles](#key-design-principles)
6. [Technology Stack](#technology-stack)
7. [Component Deep Dives](#component-deep-dives)

---

## System Overview

**Gas Town** is a multi-agent orchestration system for Claude Code (and other AI coding assistants) with persistent work tracking. It solves the fundamental challenge of coordinating multiple AI agents working on different tasks without losing context when agents restart.

### The Problem Gas Town Solves

| Challenge | Gas Town Solution |
|-----------|-------------------|
| Agents lose context on restart | Work persists in git-backed hooks |
| Manual agent coordination | Built-in mailboxes, identities, and handoffs |
| 4-10 agents become chaotic | Scale comfortably to 20-30 agents |
| Work state lost in agent memory | Work state stored in Beads ledger |

### What Gas Town Is

Gas Town is a **workspace manager** that:
- Coordinates multiple Claude Code agents working on different tasks
- Persists work state in git-backed hooks, enabling reliable multi-agent workflows
- Provides a structured hierarchy for organizing projects, agents, and work items
- Uses tmux sessions as the source of truth for agent liveness (ZFC-compliant)

---

## Core Concepts

### Town

The **Town** is your top-level workspace directory (e.g., `~/gt/`). It contains:
- All projects (Rigs)
- Town-level agents (Mayor, Deacon)
- Shared configuration
- The town-level beads database

```
~/gt/                          # Town root
├── mayor/                     # Town-level Mayor
│   └── town.json              # Town configuration (workspace marker)
├── deacon/                    # Town-level Deacon
├── plugins/                   # Town-level plugins
├── .beads/                    # Town-level issue tracking
└── <rig-name>/                # Project containers (Rigs)
```

### Rig

A **Rig** is a project container that wraps a git repository and manages its associated agents. Each rig is NOT a git clone itself but a container holding multiple clones.

```
<rig>/                         # Container (NOT a git clone)
├── config.json                # Rig configuration
├── .beads/                    # Rig-level issue tracking (or redirect)
├── .repo.git/                 # Shared bare repository
├── mayor/rig/                 # Mayor's working clone
├── refinery/rig/              # Refinery's worktree (merge queue)
├── witness/                   # Witness agent (no clone needed)
├── polecats/                  # Worker agent directories
│   └── <name>/<rig>/          # Polecat worktree
└── crew/                      # Human workspace directories
    └── <name>/                # Crew member clone
```

### Mayor

The **Mayor** is your primary AI coordinator - a Claude Code instance with full context about your workspace, projects, and agents. Start here by telling the Mayor what you want to accomplish.

**Key characteristics:**
- Town-level agent (one per Town)
- Human-facing coordinator
- Creates convoys and orchestrates work
- Spawns Polecats for batch work
- Session name: `gt-mayor`

### Deacon

The **Deacon** is the Mayor's daemon - a background agent that handles:
- Periodic health checks (heartbeat)
- Callback processing
- Cleanup operations
- Session lifecycle management

**Key characteristics:**
- Town-level agent (one per Town)
- Autonomous patrol loop
- Restarted by the daemon process on failure
- Session name: `gt-deacon`

### Witness

The **Witness** is a per-rig monitoring agent that:
- Monitors polecat health and progress
- Detects stalled or stuck workers
- Handles progressive nudging
- Cleans up zombie sessions

**Key characteristics:**
- Rig-level agent (one per Rig)
- Autonomous patrol loop
- Uses ZFC-compliant state derivation (tmux = source of truth)
- Session name: `gt-<rig>-witness`

### Refinery

The **Refinery** is a per-rig merge queue processor that:
- Processes merge requests from polecats
- Runs tests and validates merges
- Handles conflicts and failures
- Pushes successful merges to remote

**Key characteristics:**
- Rig-level agent (one per Rig)
- Worktree from shared `.repo.git`
- Can see polecat branches (shared repository)
- Session name: `gt-<rig>-refinery`

### Crew

**Crew members** are persistent, user-managed workspaces within a rig. Unlike polecats, crew workspaces are never auto-garbage-collected.

**Key characteristics:**
- Human workspace (your personal working area)
- Full git clone (not worktree)
- Persistent across sessions
- Session name: `gt-<rig>-crew-<name>`

### Polecats

**Polecats** are ephemeral worker agents that:
- Spawn with a specific work assignment
- Complete the task
- Submit to merge queue
- Exit (cleaned up by Witness)

**Key characteristics:**
- Transient (spawn-work-die lifecycle)
- Git worktree from shared `.repo.git`
- Unique timestamped branches
- Session name: `gt-<rig>-<name>`

**Polecat States:**
- `working` - Actively working on assigned issue
- `done` - Work complete, ready for cleanup
- `stuck` - Explicitly signaled need for assistance

### Beads

**Beads** is a git-backed issue tracking system that stores work state as structured data. Issue IDs use a prefix + 5-character alphanumeric format (e.g., `gt-abc12`).

**Key characteristics:**
- CLI-first design (works with AI agents)
- Stored in `.beads/` directory
- JSONL format for git-friendly merging
- Supports custom types (agent, role, convoy, etc.)

### Convoy

A **Convoy** is a work tracking unit that bundles multiple beads/issues assigned to agents. Convoys provide visibility into parallel work progress.

---

## Hierarchical Organization

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              TOWN (~gt/)                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │     MAYOR       │  │     DEACON      │  │    .beads/ (town)       │  │
│  │  (coordinator)  │  │    (daemon)     │  │   routes.jsonl          │  │
│  └────────┬────────┘  └────────┬────────┘  └─────────────────────────┘  │
│           │                    │                                         │
│  ┌────────┴────────────────────┴─────────────────────────────────────┐  │
│  │                           RIGS                                     │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                      RIG: project-a                          │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │  │  │
│  │  │  │ .repo.git│  │ WITNESS  │  │ REFINERY │  │   .beads/   │  │  │  │
│  │  │  │ (shared) │  │(monitor) │  │  (merge) │  │ (rig-level) │  │  │  │
│  │  │  └────┬─────┘  └──────────┘  └────┬─────┘  └─────────────┘  │  │  │
│  │  │       │                           │                          │  │  │
│  │  │       │         WORKTREES         │                          │  │  │
│  │  │       ▼                           ▼                          │  │  │
│  │  │  ┌─────────────────────────────────────────────────────┐    │  │  │
│  │  │  │  POLECATS (ephemeral workers)                       │    │  │  │
│  │  │  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐    │    │  │  │
│  │  │  │  │Toast-01│  │Toast-02│  │Toast-03│  │  ...   │    │    │  │  │
│  │  │  │  └────────┘  └────────┘  └────────┘  └────────┘    │    │  │  │
│  │  │  └─────────────────────────────────────────────────────┘    │  │  │
│  │  │                                                              │  │  │
│  │  │  ┌─────────────────────────────────────────────────────┐    │  │  │
│  │  │  │  CREW (persistent human workspaces)                 │    │  │  │
│  │  │  │  ┌────────┐  ┌────────┐                             │    │  │  │
│  │  │  │  │  alice │  │   bob  │                             │    │  │  │
│  │  │  │  └────────┘  └────────┘                             │    │  │  │
│  │  │  └─────────────────────────────────────────────────────┘    │  │  │
│  │  │                                                              │  │  │
│  │  │  ┌─────────────────────────────────────────────────────┐    │  │  │
│  │  │  │  mayor/rig/ (Mayor's clone for this rig)            │    │  │  │
│  │  │  └─────────────────────────────────────────────────────┘    │  │  │
│  │  └──────────────────────────────────────────────────────────────┘  │  │
│  │                                                                     │  │
│  │  ┌──────────────────────────────────────────────────────────────┐  │  │
│  │  │                      RIG: project-b                           │  │  │
│  │  │                        (same structure)                       │  │  │
│  │  └──────────────────────────────────────────────────────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Agent Hierarchy

```
                    ┌─────────────┐
                    │   DAEMON    │  (Go process, not AI)
                    │ (scheduler) │
                    └──────┬──────┘
                           │ heartbeat
                           ▼
                    ┌─────────────┐
                    │   DEACON    │  (AI agent, town-level)
                    │  (daemon's  │
                    │   agent)    │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │   MAYOR     │ │  WITNESS    │ │  REFINERY   │
    │(coordinator)│ │ (per-rig)   │ │ (per-rig)   │
    └──────┬──────┘ └──────┬──────┘ └─────────────┘
           │               │
           │         monitors
           │               ▼
           │        ┌─────────────┐
           │        │  POLECATS   │
           │        │ (workers)   │
           └───────▶└─────────────┘
                 spawns
```

---

## Data Flow

### Work Assignment Flow

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌─────────┐
│  Human  │────▶│  Mayor  │────▶│  Convoy  │────▶│ Polecat │
│         │     │         │     │ (beads)  │     │         │
└─────────┘     └─────────┘     └──────────┘     └────┬────┘
                                                      │
                                                      ▼ work
                                                 ┌─────────┐
                                                 │  Hook   │
                                                 │ (work   │
                                                 │  item)  │
                                                 └────┬────┘
                                                      │
                                                      ▼ complete
                                                 ┌──────────┐
                                                 │ Refinery │
                                                 │  (merge  │
                                                 │  queue)  │
                                                 └────┬─────┘
                                                      │
                                                      ▼ merge
                                                 ┌──────────┐
                                                 │  Remote  │
                                                 │ (origin) │
                                                 └──────────┘
```

### Session Lifecycle Flow

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ gt sling     │───▶│ Spawn        │───▶│ gt prime     │
│ (assign work)│    │ Worktree     │    │ (context)    │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                         ┌─────────────────────┘
                         ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Work on      │───▶│ git commit   │───▶│ gt done      │
│ Issue        │    │ bd sync      │    │ (MQ submit)  │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                         ┌─────────────────────┘
                         ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Refinery     │───▶│ Tests Pass   │───▶│ Merge &      │
│ Processes    │    │ Conflicts?   │    │ Push         │
└──────────────┘    └──────────────┘    └──────────────┘
```

### Beads Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        BEADS ROUTING                             │
│                                                                  │
│   ┌──────────────┐         routes.jsonl        ┌──────────────┐ │
│   │ Town .beads/ │◀───────────────────────────▶│ Rig .beads/  │ │
│   │              │                              │              │ │
│   │ - hq-* beads │                              │ - gt-* beads │ │
│   │ - mayor      │                              │ - polecats   │ │
│   │ - deacon     │                              │ - issues     │ │
│   │ - roles      │                              │ - merge reqs │ │
│   └──────────────┘                              └──────────────┘ │
│                                                                  │
│   Polecats use redirect files to share rig's beads database      │
│                                                                  │
│   polecat/.beads/redirect ─────▶ ../../.beads/                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Design Principles

### 1. Git-Backed Persistence

All work state is stored in git-tracked files:
- **Beads database** (`.beads/issues.jsonl`) - Issue tracking
- **Hooks** - Work assignments attached to agents
- **Configuration** - Town and rig settings

Benefits:
- Survives crashes and restarts
- Provides rollback capability
- Enables multi-agent coordination through git

### 2. ZFC (Zero File-Based State Coupling)

**tmux session = source of truth for agent liveness**

The system does not maintain state files to track whether agents are running. Instead:
- `tmux has-session` determines if an agent is alive
- `tmux` session existence is the authoritative source
- No PID files or state files for liveness detection

This eliminates:
- Stale state files after crashes
- Race conditions between state and reality
- Complex state synchronization logic

### 3. Event Sourcing via Beads

Work state is derived from the beads ledger:
- Issue assignments determine who is working on what
- Status transitions are recorded as events
- State is computed, not stored

### 4. The Propulsion Principle (GUPP)

**Gas Town Universal Propulsion Principle:**

> "If you find work on your hook, YOU RUN IT."

No confirmation. No waiting. No announcements. The hook having work IS the assignment. This is physics, not politeness.

**Failure mode prevented:**
- Agent starts with work on hook
- Agent announces itself and waits for human
- Human is AFK
- Work sits idle, system stalls

### 5. Transient Worker Model

Polecats follow a strict lifecycle:
1. **Spawn** - Created with work assignment
2. **Work** - Execute the assigned task
3. **Submit** - Send to merge queue via `gt done`
4. **Exit** - Session terminates, Witness cleans up

There is NO idle pool. Polecats without work should be garbage collected.

### 6. Shared Bare Repository Architecture

```
.repo.git/                    # Shared bare repo
    │
    ├──▶ refinery/rig/        # Worktree on main
    │
    ├──▶ polecats/a/rig/      # Worktree on polecat/a-xxx
    │
    └──▶ polecats/b/rig/      # Worktree on polecat/b-xxx
```

Benefits:
- Refinery sees all polecat branches without pushing
- Fast worktree creation (no network clone)
- Shared git objects reduce disk usage

---

## Technology Stack

### Core Technologies

| Component | Technology | Purpose |
|-----------|------------|---------|
| CLI Framework | **Go + Cobra** | Command-line interface |
| TUI Components | **Bubble Tea** | Interactive terminal UI |
| Session Management | **tmux** | Agent session lifecycle |
| Issue Tracking | **Beads** | Git-backed work state |
| Configuration | **JSON/TOML** | Settings and formulas |
| Version Control | **Git** | Worktrees, persistence |

### Language: Go

Gas Town is written in Go (1.23+) for:
- Single binary distribution
- Cross-platform support
- Strong concurrency primitives
- Fast compilation

### CLI: Cobra

The `gt` CLI uses Cobra for:
- Subcommand structure
- Flag parsing
- Shell completions
- Help generation

### TUI: Bubble Tea

Interactive components use Bubble Tea:
- Dashboard views
- Convoy monitoring
- Agent status displays

### Session Management: tmux

tmux provides:
- Detached session management
- Session persistence
- Multi-pane layouts
- Programmatic control via CLI

Session naming convention:
```
gt-mayor                      # Town Mayor
gt-deacon                     # Town Deacon
gt-<rig>-witness              # Rig Witness
gt-<rig>-refinery             # Rig Refinery (if AI-powered)
gt-<rig>-<polecat>            # Polecat worker
gt-<rig>-crew-<name>          # Crew member
```

### Issue Tracking: Beads

Beads (`bd` CLI) provides:
- Git-native issue storage
- JSONL format (merge-friendly)
- Dependency tracking
- Custom issue types

Key commands:
```bash
bd create "Title"             # Create issue
bd list --status open         # List issues
bd update <id> --status done  # Update status
bd sync                       # Sync with remote
bd ready                      # Find unblocked issues
```

---

## Component Deep Dives

### Daemon Process

The daemon is a **Go process** (not an AI agent) that:
1. Runs in the background
2. Sends periodic heartbeats
3. Processes lifecycle requests
4. Restarts agents when needed

```go
type Config struct {
    HeartbeatInterval time.Duration  // Default: 5 minutes
    TownRoot          string
    LogFile           string
    PidFile           string
}
```

### Polecat Manager

Manages polecat lifecycle:

```go
type Manager struct {
    rig      *rig.Rig
    git      *git.Git
    beads    *beads.Beads
    namePool *NamePool
    tmux     *tmux.Tmux
}
```

Key operations:
- `Add()` - Create polecat worktree
- `Remove()` - Clean up worktree (with safety checks)
- `List()` - Enumerate polecats
- `DetectStalePolecats()` - Find cleanup candidates

### Name Pool

Polecats get names from a pool:
- Custom names from rig settings
- Default numbered names (polecat-01 through polecat-50)
- Overflow names when pool exhausted

### Merge Request Lifecycle

```
┌────────┐      ┌─────────────┐      ┌────────┐
│  open  │─────▶│ in_progress │─────▶│ closed │
└────────┘      └─────────────┘      └────────┘
     │                │                   │
     │                │                   │
     ▼                ▼                   ▼
 (waiting)      (Engineer       (merged, rejected,
                 claims)         conflict, superseded)
```

Close reasons:
- `merged` - Successfully merged
- `rejected` - Manually rejected
- `conflict` - Unresolvable conflicts
- `superseded` - Replaced by another MR

### Formula System

TOML-based workflow definitions:

```toml
formula = "release"
description = "Standard release process"
type = "workflow"

[vars.version]
description = "Version to release"
required = true

[[steps]]
id = "test"
title = "Run Tests"

[[steps]]
id = "build"
title = "Build"
needs = ["test"]

[[steps]]
id = "publish"
title = "Publish"
needs = ["build"]
```

Formula types:
- **workflow** - Sequential steps with dependencies
- **convoy** - Parallel legs with synthesis
- **expansion** - Template-based parameterized workflows
- **aspect** - Multi-aspect parallel analysis

---

## Quick Reference

### Key Commands

```bash
# Workspace
gt install ~/gt --git          # Initialize workspace
gt rig add <name> <url>        # Add project

# Agents
gt mayor attach                # Start Mayor
gt agents                      # List active agents
gt sling <id> <rig>            # Assign work

# Work tracking
gt convoy create "Name" <ids>  # Create convoy
gt convoy list                 # List convoys
gt hook                        # Check your hook

# Session management
gt prime                       # Recover context
gt done                        # Complete and exit
gt cycle                       # Request session restart
```

### Environment Variables

```bash
GT_TOWN_ROOT     # Town workspace root
GT_ROLE          # Current agent role
GT_RIG           # Current rig name
BD_ACTOR         # Beads actor identity
BEADS_DIR        # Beads database directory
```

### File Markers

```
mayor/town.json               # Town root marker (primary)
mayor/                        # Town root marker (secondary)
config.json                   # Rig configuration
.beads/redirect               # Beads database redirect
state.json                    # Agent/worker state
```

---

## Summary

Gas Town is an orchestration layer that turns chaotic multi-agent development into a coordinated system. Key insights:

1. **Git is the persistence layer** - All state survives restarts
2. **tmux is the liveness layer** - Session existence = agent alive
3. **Beads is the coordination layer** - Work state as structured data
4. **Polecats are disposable** - Spawn, work, submit, die
5. **GUPP drives autonomy** - Work on hook = immediate execution

The architecture enables scaling from a single developer with a few agents to teams coordinating dozens of AI workers across multiple projects.
