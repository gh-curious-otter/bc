# Gas Town Agent Types

This document describes all agent types in the Gas Town system, their roles, responsibilities, lifecycles, and how they interact with each other.

---

## Overview

Gas Town operates as a distributed work coordination system with six distinct agent types:

| Agent | Type | Lifecycle | Primary Role |
|-------|------|-----------|--------------|
| Mayor | AI Coordinator | Persistent | Work orchestration and rig management |
| Deacon | Town Daemon | Persistent | System monitoring and plugin management |
| Witness | Polecat Monitor | Persistent | State tracking and issue detection |
| Refinery | Merge Queue | Persistent | MR processing and conflict resolution |
| Polecats | Ephemeral Workers | Ephemeral | Task execution |
| Crew | Human Workers | Persistent | Personal workspace and manual work |

```
┌─────────────────────────────────────────────────────────────────┐
│                        GAS TOWN                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐     coordinates      ┌──────────────────────┐    │
│   │  MAYOR  │ ─────────────────▶   │      POLECATS        │    │
│   │   (AI)  │                      │  (Ephemeral Workers) │    │
│   └────┬────┘                      └──────────────────────┘    │
│        │                                      ▲                 │
│        │ manages                              │ monitors        │
│        ▼                                      │                 │
│   ┌─────────┐                          ┌──────┴─────┐          │
│   │  RIGS   │                          │  WITNESS   │          │
│   │ (Slots) │                          │ (Monitor)  │          │
│   └─────────┘                          └────────────┘          │
│                                                                 │
│   ┌─────────┐     processes      ┌─────────────┐               │
│   │ REFINERY│ ◀───────────────── │  MR QUEUE   │               │
│   │ (Merge) │                    └─────────────┘               │
│   └─────────┘                                                  │
│                                                                 │
│   ┌─────────┐                    ┌─────────────┐               │
│   │ DEACON  │ ───────────────▶   │  PLUGINS    │               │
│   │(Daemon) │    manages         └─────────────┘               │
│   └─────────┘                                                  │
│                                                                 │
│   ┌─────────┐                                                  │
│   │  CREW   │  (Human Workers - Personal Workspaces)           │
│   └─────────┘                                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 1. Mayor - AI Coordinator

The **Mayor** is the central AI coordinator responsible for orchestrating all work within Gas Town. It manages rigs (work slots), assigns tasks to polecats, and maintains session continuity.

### Role and Responsibilities

- **Work Orchestration**: Breaks down high-level tasks into actionable work items
- **Rig Management**: Allocates and monitors work slots for polecats
- **Session Management**: Maintains context across work sessions
- **Priority Management**: Determines task ordering and resource allocation
- **Coordination**: Ensures polecats don't conflict on shared resources

### Capabilities

| Capability | Description |
|------------|-------------|
| Task Decomposition | Breaks complex work into polecat-sized chunks |
| Resource Allocation | Assigns polecats to available rigs |
| State Persistence | Maintains work context across restarts |
| Conflict Detection | Prevents overlapping work assignments |
| Progress Tracking | Monitors completion status of all active work |

### How Mayor Manages Rigs

Rigs are work slots that polecats occupy while performing tasks. The Mayor maintains a registry of available rigs and their current occupants.

```
┌────────────────────────────────────────────────────┐
│                    MAYOR                           │
│                                                    │
│   ┌──────────────────────────────────────────┐    │
│   │              RIG REGISTRY                 │    │
│   ├──────────┬──────────┬──────────┬─────────┤    │
│   │  Rig 1   │  Rig 2   │  Rig 3   │  Rig 4  │    │
│   │ (furiosa)│  (nux)   │  (empty) │ (slit)  │    │
│   │ [active] │ [active] │  [free]  │ [active]│    │
│   └──────────┴──────────┴──────────┴─────────┘    │
│                                                    │
│   Work Queue: [task-1, task-2, task-3, ...]       │
└────────────────────────────────────────────────────┘
```

### Session Management

The Mayor maintains session state to enable:
- Resumption after interruptions
- Context preservation across polecat lifecycles
- Work history and audit trails

### Configuration Example

```yaml
# mayor.yaml
mayor:
  max_concurrent_rigs: 5
  session:
    persistence: true
    state_dir: .gtn/mayor/sessions
    ttl: 24h
  coordination:
    work_queue: .gtn/mayor/queue
    assignment_strategy: round_robin
  hooks:
    on_task_complete: notify
    on_error: escalate
```

---

## 2. Deacon - Town Daemon

The **Deacon** is the persistent background daemon that keeps Gas Town running. It manages the plugin system and monitors system health.

### Purpose

- Maintain persistent system processes
- Load and manage plugins
- Provide system-level monitoring
- Handle graceful shutdown and restart

### Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    DEACON LIFECYCLE                         │
│                                                             │
│   ┌──────────┐    ┌──────────┐    ┌──────────────────────┐ │
│   │  START   │───▶│  INIT    │───▶│   LOAD PLUGINS       │ │
│   └──────────┘    │  CONFIG  │    └──────────┬───────────┘ │
│                   └──────────┘               │              │
│                                              ▼              │
│   ┌──────────┐    ┌──────────┐    ┌──────────────────────┐ │
│   │ SHUTDOWN │◀───│ SIGNAL   │◀───│   RUNNING (LOOP)     │ │
│   └──────────┘    │ RECEIVED │    │   - Health checks    │ │
│        │          └──────────┘    │   - Plugin events    │ │
│        ▼                          │   - State sync       │ │
│   ┌──────────┐                    └──────────────────────┘ │
│   │ CLEANUP  │                                             │
│   │ PLUGINS  │                                             │
│   └──────────┘                                             │
└─────────────────────────────────────────────────────────────┘
```

### Plugin System

The Deacon manages a plugin architecture that extends Gas Town functionality:

```
┌─────────────────────────────────────────────┐
│                 DEACON                      │
│                                             │
│   ┌─────────────────────────────────────┐  │
│   │          PLUGIN MANAGER             │  │
│   └─────────────────────────────────────┘  │
│        │           │            │          │
│        ▼           ▼            ▼          │
│   ┌─────────┐ ┌─────────┐ ┌─────────┐     │
│   │ Plugin  │ │ Plugin  │ │ Plugin  │     │
│   │   A     │ │   B     │ │   C     │     │
│   └─────────┘ └─────────┘ └─────────┘     │
│                                             │
│   Plugin Hooks:                            │
│   - on_polecat_spawn                       │
│   - on_task_complete                       │
│   - on_merge_ready                         │
│   - on_error                               │
└─────────────────────────────────────────────┘
```

### Monitoring Responsibilities

| Area | What Deacon Monitors |
|------|---------------------|
| System Health | CPU, memory, disk usage |
| Agent Status | Liveness of all persistent agents |
| Plugin Health | Plugin responsiveness and errors |
| Queue Depths | Work queue backlog alerts |

### Configuration Example

```yaml
# deacon.yaml
deacon:
  daemon:
    pid_file: .gtn/deacon.pid
    log_file: .gtn/logs/deacon.log
  plugins:
    directory: .gtn/plugins
    autoload: true
    enabled:
      - notifications
      - metrics
      - git-hooks
  monitoring:
    interval: 30s
    health_endpoint: /health
  signals:
    graceful_shutdown: SIGTERM
    reload_config: SIGHUP
```

---

## 3. Witness - Polecat Monitor

The **Witness** is a specialized monitor that observes polecat activity, tracks state, and detects issues requiring intervention.

### Auto-Spawn Logic

The Witness automatically spawns when:
1. A polecat enters `working` state
2. Multiple polecats are active simultaneously
3. Long-running tasks exceed threshold duration
4. Error conditions are detected

```
┌─────────────────────────────────────────────────────────────┐
│                  WITNESS AUTO-SPAWN                         │
│                                                             │
│   Trigger Conditions:                                       │
│                                                             │
│   ┌─────────────────┐                                      │
│   │ Polecat Active? │───Yes───▶ Spawn Witness              │
│   └─────────────────┘                                      │
│            │                                                │
│           No                                                │
│            ▼                                                │
│   ┌─────────────────┐                                      │
│   │ Tasks Pending?  │───Yes───▶ Spawn Witness (standby)    │
│   └─────────────────┘                                      │
│            │                                                │
│           No                                                │
│            ▼                                                │
│   [Witness remains dormant]                                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### State Tracking

The Witness maintains a real-time view of all polecat states:

```
┌────────────────────────────────────────────────────────────┐
│                 WITNESS STATE TRACKER                      │
│                                                            │
│   Polecat        State       Duration    Issues           │
│   ─────────────────────────────────────────────────────   │
│   furiosa        working     12m 34s     none             │
│   nux            working     3m 12s      none             │
│   slit           done        -           cleanup pending  │
│   rictus         idle        45m 00s     stale?           │
│   dementus       spawning    0m 15s      none             │
│                                                            │
│   Active: 2    Idle: 1    Done: 1    Spawning: 1          │
└────────────────────────────────────────────────────────────┘
```

### Issue Monitoring

| Issue Type | Detection | Response |
|------------|-----------|----------|
| Stuck Polecat | No progress for N minutes | Alert Mayor |
| Error State | Exception or failure logged | Log and notify |
| Resource Contention | Multiple polecats on same resource | Flag conflict |
| Orphaned Work | Polecat died with incomplete task | Mark for reassignment |
| Timeout | Task exceeds max duration | Terminate and report |

### Configuration Example

```yaml
# witness.yaml
witness:
  auto_spawn:
    on_polecat_active: true
    on_pending_tasks: true
    spawn_delay: 5s
  monitoring:
    poll_interval: 10s
    stuck_threshold: 30m
    timeout_threshold: 2h
  alerts:
    channels:
      - log
      - webhook
    escalation_after: 3
  state:
    persistence: .gtn/witness/state.json
    history_retention: 7d
```

---

## 4. Refinery - Merge Queue Processor

The **Refinery** handles the merge queue, processing merge requests, resolving conflicts, and managing git integration.

### MR Handling Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                    REFINERY MR WORKFLOW                         │
│                                                                 │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│   │   MR     │───▶│  QUEUE   │───▶│ VALIDATE │───▶│  MERGE   │ │
│   │ CREATED  │    │  ENTRY   │    │          │    │          │ │
│   └──────────┘    └──────────┘    └────┬─────┘    └────┬─────┘ │
│                                        │               │        │
│                                        ▼               ▼        │
│                                   ┌──────────┐   ┌──────────┐  │
│                                   │ CONFLICT │   │ COMPLETE │  │
│                                   │ DETECTED │   │          │  │
│                                   └────┬─────┘   └──────────┘  │
│                                        │                        │
│                                        ▼                        │
│                                   ┌──────────┐                  │
│                                   │  RESOLVE │                  │
│                                   │   AUTO   │                  │
│                                   └────┬─────┘                  │
│                                        │                        │
│                            ┌───────────┴───────────┐            │
│                            ▼                       ▼            │
│                       ┌──────────┐           ┌──────────┐       │
│                       │ SUCCESS  │           │  MANUAL  │       │
│                       │  MERGE   │           │ REQUIRED │       │
│                       └──────────┘           └──────────┘       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Conflict Resolution

The Refinery employs a multi-stage conflict resolution strategy:

```
┌────────────────────────────────────────────────────────────┐
│              CONFLICT RESOLUTION STAGES                    │
│                                                            │
│   Stage 1: Auto-Resolution                                 │
│   ─────────────────────────────                           │
│   - Whitespace conflicts                                   │
│   - Import ordering                                        │
│   - Non-overlapping changes                               │
│                                                            │
│   Stage 2: Semantic Analysis                               │
│   ──────────────────────────                              │
│   - Function signature changes                             │
│   - Dependency updates                                     │
│   - Configuration merges                                   │
│                                                            │
│   Stage 3: Human Escalation                                │
│   ─────────────────────────                               │
│   - Logic conflicts                                        │
│   - Semantic incompatibilities                             │
│   - High-risk file changes                                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Integration with Git

| Operation | Refinery Action |
|-----------|-----------------|
| Branch Creation | Creates merge staging branches |
| Rebase | Keeps MRs up-to-date with target |
| Merge | Fast-forward or merge commit based on config |
| Tag | Optionally tags merged commits |
| Push | Pushes merged result to remote |

### Configuration Example

```yaml
# refinery.yaml
refinery:
  queue:
    directory: .gtn/refinery/queue
    max_concurrent: 3
    priority_order: fifo  # or priority-based
  merge:
    strategy: rebase  # or merge, squash
    require_clean: true
    run_checks: true
    auto_resolve:
      enabled: true
      strategies:
        - whitespace
        - imports
        - lockfiles
  conflict:
    auto_resolve_threshold: low  # low, medium, high
    escalation_timeout: 1h
  git:
    remote: origin
    protected_branches:
      - main
      - release/*
    commit_message_template: |
      Merge: {title}

      {description}

      Closes: {issue_ref}
```

---

## 5. Polecats - Ephemeral Workers

**Polecats** are ephemeral worker agents that perform actual tasks. They spawn, do work, and are cleaned up when done.

### Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                     POLECAT LIFECYCLE                           │
│                                                                 │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────────┐ │
│   │  SPAWN  │───▶│  INIT   │───▶│  WORK   │───▶│    DONE     │ │
│   └─────────┘    └─────────┘    └─────────┘    └──────┬──────┘ │
│       │              │              │                  │        │
│       │              │              │                  ▼        │
│       │              │              │           ┌─────────────┐ │
│       │              │              │           │   CLEANUP   │ │
│       │              │              │           └─────────────┘ │
│       │              │              │                           │
│       ▼              ▼              ▼                           │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    STATE DETAILS                         │  │
│   ├─────────────────────────────────────────────────────────┤  │
│   │ SPAWN:   Mayor requests worker, rig allocated           │  │
│   │ INIT:    Clone/setup workspace, load context            │  │
│   │ WORK:    Execute assigned task via hooks                │  │
│   │ DONE:    Task complete, results reported                │  │
│   │ CLEANUP: Workspace removed, rig released                │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Naming Convention

Polecats are named after War Boys from Mad Max: Fury Road:

| Name | Origin | Typical Assignment |
|------|--------|-------------------|
| **furiosa** | Imperator Furiosa | Primary/lead tasks |
| **nux** | War Boy Nux | General development |
| **slit** | War Boy Slit | Testing/validation |
| **rictus** | Rictus Erectus | Heavy processing |
| **dementus** | Dementus | Cleanup/maintenance |

Names are assigned round-robin from the pool as polecats spawn.

### Session and Slot Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  SESSION & SLOT ARCHITECTURE                    │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                      SESSION                             │  │
│   │   ID: session-2024-01-15-abc123                         │  │
│   │   Created: 2024-01-15T10:00:00Z                         │  │
│   │   Context: Feature implementation                        │  │
│   └─────────────────────────────────────────────────────────┘  │
│                              │                                  │
│              ┌───────────────┼───────────────┐                  │
│              ▼               ▼               ▼                  │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐           │
│   │    SLOT 1    │ │    SLOT 2    │ │    SLOT 3    │           │
│   │   (Rig A)    │ │   (Rig B)    │ │   (Rig C)    │           │
│   ├──────────────┤ ├──────────────┤ ├──────────────┤           │
│   │ Polecat:     │ │ Polecat:     │ │ Polecat:     │           │
│   │   furiosa    │ │   nux        │ │   (empty)    │           │
│   │ Task:        │ │ Task:        │ │ Task:        │           │
│   │   impl-auth  │ │   write-test │ │   available  │           │
│   │ State:       │ │ State:       │ │ State:       │           │
│   │   working    │ │   working    │ │   free       │           │
│   └──────────────┘ └──────────────┘ └──────────────┘           │
│                                                                 │
│   Slot Properties:                                              │
│   - Isolated workspace (worktree or container)                  │
│   - Dedicated git branch                                        │
│   - Resource limits (CPU, memory)                               │
│   - Independent environment                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Hook-Based Work Assignment

Polecats receive work through a hook system:

```
┌─────────────────────────────────────────────────────────────────┐
│                    HOOK-BASED ASSIGNMENT                        │
│                                                                 │
│   MAYOR                          POLECAT (furiosa)              │
│   ──────                         ─────────────────              │
│      │                                  │                       │
│      │  1. Assign task via hook         │                       │
│      │ ─────────────────────────────▶   │                       │
│      │    .gtn/hooks/polecat-start      │                       │
│      │                                  │                       │
│      │                                  │  2. Read hook file    │
│      │                                  │     Parse task spec   │
│      │                                  │                       │
│      │                                  │  3. Execute work      │
│      │                                  │     ...               │
│      │                                  │                       │
│      │  4. Progress hooks (optional)    │                       │
│      │ ◀─────────────────────────────   │                       │
│      │    .gtn/hooks/polecat-progress   │                       │
│      │                                  │                       │
│      │  5. Completion hook              │                       │
│      │ ◀─────────────────────────────   │                       │
│      │    .gtn/hooks/polecat-done       │                       │
│      │                                  │                       │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration Example

```yaml
# polecat.yaml
polecat:
  naming:
    pool:
      - furiosa
      - nux
      - slit
      - rictus
      - dementus
    assignment: round_robin
  workspace:
    type: worktree  # or container, directory
    base_path: .gtn/polecats
    cleanup_on_done: true
  resources:
    max_memory: 4G
    max_cpu: 2
    timeout: 2h
  hooks:
    start: .gtn/hooks/polecat-start
    progress: .gtn/hooks/polecat-progress
    done: .gtn/hooks/polecat-done
    error: .gtn/hooks/polecat-error
  state:
    file: .gtn/polecats/{name}/state.json
    report_interval: 30s
```

---

## 6. Crew - Human Workers

**Crew** members are human workers who interact with Gas Town through personal, persistent workspaces.

### Personal Workspace Model

Unlike ephemeral polecats, crew members have persistent workspaces that survive across sessions:

```
┌─────────────────────────────────────────────────────────────────┐
│                   CREW WORKSPACE MODEL                          │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    CREW MEMBER: alice                    │  │
│   ├─────────────────────────────────────────────────────────┤  │
│   │                                                          │  │
│   │   Personal Workspace: ~/projects/gastown                 │  │
│   │   ──────────────────────────────────────                │  │
│   │   - Persistent across sessions                          │  │
│   │   - Personal git configuration                          │  │
│   │   - Custom tool preferences                             │  │
│   │   - Local environment settings                          │  │
│   │                                                          │  │
│   │   Active Branches:                                       │  │
│   │   ──────────────────                                    │  │
│   │   - feature/alice-auth-work                             │  │
│   │   - bugfix/alice-login-fix                              │  │
│   │                                                          │  │
│   │   Session History:                                       │  │
│   │   ────────────────                                      │  │
│   │   - 2024-01-15: Auth implementation (8h)                │  │
│   │   - 2024-01-14: Code review (2h)                        │  │
│   │   - 2024-01-13: Bug investigation (4h)                  │  │
│   │                                                          │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Persistence vs Ephemeral Polecats

| Aspect | Crew (Human) | Polecats (AI) |
|--------|--------------|---------------|
| **Workspace** | Persistent, personal | Ephemeral, disposable |
| **State** | Survives indefinitely | Destroyed on completion |
| **Context** | Accumulated knowledge | Fresh each spawn |
| **Branches** | Personal feature branches | Task-specific branches |
| **Tools** | Personal preferences | Standardized tooling |
| **Concurrent Work** | Single focus typical | Parallel execution |
| **Cleanup** | Manual | Automatic |

### Crew Interaction with Gas Town

```
┌─────────────────────────────────────────────────────────────────┐
│              CREW INTERACTION PATTERNS                          │
│                                                                 │
│   ┌──────────┐                                                 │
│   │   CREW   │                                                 │
│   │  (alice) │                                                 │
│   └────┬─────┘                                                 │
│        │                                                        │
│        ├───▶ Request work from Mayor                           │
│        │     - "I'll take the auth feature"                    │
│        │                                                        │
│        ├───▶ Collaborate with Polecats                         │
│        │     - "Polecat: write tests for my changes"           │
│        │                                                        │
│        ├───▶ Submit to Refinery                                │
│        │     - "Ready for merge review"                        │
│        │                                                        │
│        ├───▶ Monitor via Witness                               │
│        │     - "Show me polecat progress"                      │
│        │                                                        │
│        └───▶ Use Deacon plugins                                │
│              - Notifications, metrics, integrations            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration Example

```yaml
# crew.yaml
crew:
  members:
    alice:
      workspace: ~/projects/gastown
      email: alice@example.com
      preferences:
        editor: vscode
        shell: zsh
      permissions:
        - submit_mr
        - spawn_polecat
        - view_all
    bob:
      workspace: ~/dev/gastown
      email: bob@example.com
      preferences:
        editor: vim
        shell: bash
      permissions:
        - submit_mr
        - view_own
  defaults:
    branch_prefix: "{username}/"
    commit_template: .gtn/templates/commit.txt
  notifications:
    on_polecat_done: true
    on_mr_merged: true
    on_conflict: true
```

---

## State Diagram Summary

```
┌─────────────────────────────────────────────────────────────────┐
│              GAS TOWN COMPLETE STATE FLOW                       │
│                                                                 │
│                        ┌─────────┐                              │
│                        │  CREW   │                              │
│                        │ Request │                              │
│                        └────┬────┘                              │
│                             │                                   │
│                             ▼                                   │
│   ┌──────────────────────────────────────────────────────────┐ │
│   │                       MAYOR                               │ │
│   │                                                           │ │
│   │   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │ │
│   │   │ Decompose   │───▶│   Assign    │───▶│   Monitor   │  │ │
│   │   │   Task      │    │  to Rig     │    │  Progress   │  │ │
│   │   └─────────────┘    └─────────────┘    └─────────────┘  │ │
│   │                             │                             │ │
│   └─────────────────────────────┼─────────────────────────────┘ │
│                                 │                               │
│                                 ▼                               │
│   ┌──────────────────────────────────────────────────────────┐ │
│   │                     POLECAT                               │ │
│   │   spawn ──▶ init ──▶ work ──▶ done ──▶ cleanup           │ │
│   └──────────────────────────────┬───────────────────────────┘ │
│                                  │                              │
│            ┌─────────────────────┼─────────────────────┐        │
│            │                     │                     │        │
│            ▼                     ▼                     ▼        │
│   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────┐  │
│   │    WITNESS      │   │    REFINERY     │   │   DEACON    │  │
│   │   (monitors)    │   │   (merges)      │   │  (plugins)  │  │
│   └─────────────────┘   └─────────────────┘   └─────────────┘  │
│                                  │                              │
│                                  ▼                              │
│                         ┌─────────────────┐                     │
│                         │    MERGED       │                     │
│                         │    TO MAIN      │                     │
│                         └─────────────────┘                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Reference

### Agent Communication Channels

| From | To | Channel |
|------|-----|---------|
| Mayor | Polecat | Hooks (.gtn/hooks/) |
| Polecat | Mayor | State files, hooks |
| Witness | Mayor | Alerts, state reports |
| Refinery | Mayor | MR status updates |
| Crew | Mayor | CLI commands, requests |
| Deacon | All | Plugin events, health |

### File System Layout

```
.gtn/
├── mayor/
│   ├── sessions/
│   ├── queue/
│   └── state.json
├── deacon/
│   ├── deacon.pid
│   └── plugins/
├── witness/
│   └── state.json
├── refinery/
│   └── queue/
├── polecats/
│   ├── furiosa/
│   ├── nux/
│   ├── slit/
│   ├── rictus/
│   └── dementus/
├── crew/
│   └── {username}/
├── hooks/
│   ├── polecat-start
│   ├── polecat-progress
│   ├── polecat-done
│   └── polecat-error
└── logs/
    └── *.log
```
