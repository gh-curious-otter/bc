# bc Workflows

Common operational workflows for the bc multi-agent orchestration system.

---

## 1. Basic Workflow

The simplest bc workflow: spawn an agent, assign work, let it complete.

### Step-by-Step

```bash
# 1. Initialize workspace (first time only)
bc init

# 2. Start the system
bc up

# 3. Add work to the queue
bc queue add "Fix login button not responding"

# 4. Spawn an engineer agent
bc spawn eng-01 --role engineer

# 5. Assign work to the agent
bc queue assign work-001 eng-01

# 6. Monitor progress
bc status

# 7. When done, process merge queue
bc merge process

# 8. Shut down
bc down
```

### Flow Diagram

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   bc init    │────▶│    bc up     │────▶│ bc queue add │
└──────────────┘     └──────────────┘     └───────┬──────┘
                                                   │
                                                   ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   bc down    │◀────│ bc merge     │◀────│  bc spawn    │
└──────────────┘     │   process    │     └───────┬──────┘
                     └──────────────┘             │
                           ▲                      ▼
                           │              ┌──────────────┐
                           │              │ bc queue     │
                           │              │   assign     │
                           │              └───────┬──────┘
                           │                      │
                     ┌─────┴──────┐               ▼
                     │   work     │       ┌──────────────┐
                     │ completed  │◀──────│ agent works  │
                     └────────────┘       └──────────────┘
```

---

## 2. Hierarchical Team Workflow

Using the full PM → Manager → Engineer hierarchy for complex work.

### Overview

```
ProductManager (pm-01)
    │
    │ creates epic, spawns manager
    ▼
Manager (mgr-01)
    │
    │ breaks down epic, spawns engineers
    ├──────────────────┐
    ▼                  ▼
Engineer (eng-01)   Engineer (eng-02)
    │                  │
    │ implements       │ implements
    ▼                  ▼
work-001            work-002
```

### Commands

```bash
# 1. Start coordinator
bc up

# 2. Spawn PM (optional - can use bc up for this)
bc spawn pm-01 --role product-manager

# 3. PM creates an epic (via bc queue add or PM's own commands)
bc queue add "User authentication system" --description "Complete OAuth2 login flow"

# 4. PM spawns a manager
bc spawn mgr-01 --role manager --parent pm-01

# 5. Manager breaks down work
bc queue add "Implement login form" --description "Create login UI component"
bc queue add "Implement OAuth2 flow" --description "Add OAuth2 provider integration"

# 6. Manager spawns engineers
bc spawn eng-01 --role engineer --parent mgr-01
bc spawn eng-02 --role engineer --parent mgr-01

# 7. Manager assigns work
bc queue assign work-001 eng-01
bc queue assign work-002 eng-02

# 8. Engineers work, report progress
# (In eng-01's session)
bc report working "Building login form"
# ... work ...
bc report done "Login form complete"

# 9. Process merges
bc merge process

# 10. Clean up
bc down
```

---

## 3. Agent Lifecycle

Understanding how agents move through states.

### State Machine

```
┌──────────────────────────────────────────────────────────────────┐
│                        AGENT LIFECYCLE                            │
└──────────────────────────────────────────────────────────────────┘

                              bc spawn
                                 │
                                 ▼
                          ┌───────────┐
                          │ starting  │
                          └─────┬─────┘
                                │ session ready
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        ACTIVE STATES                             │
│                                                                  │
│    ┌──────┐  work assigned  ┌─────────┐  task done  ┌──────┐   │
│    │ idle │────────────────▶│ working │────────────▶│ done │   │
│    └──┬───┘                 └────┬────┘             └──┬───┘   │
│       │                          │                     │        │
│       │                          ▼                     │        │
│       │                    ┌─────────┐                 │        │
│       └───────────────────▶│  stuck  │◀────────────────┘        │
│                            └────┬────┘                          │
│                                 │                               │
└─────────────────────────────────┼───────────────────────────────┘
                                  │
                                  ▼ unrecoverable
                           ┌───────────┐
                           │   error   │
                           └─────┬─────┘
                                 │
                                 ▼ bc down / cleanup
                           ┌───────────┐
                           │  stopped  │
                           └───────────┘
```

### State Transitions

| Current | Trigger | Next |
|---------|---------|------|
| starting | Session initialized | idle |
| idle | Work assigned | working |
| working | Task completed | done |
| working | Needs help | stuck |
| done | New work | working |
| stuck | Resolved | working |
| any | Error | error |
| any | bc down | stopped |

### Reporting State Changes

Agents report their state changes:

```bash
# Agent starts working
bc report working "Implementing feature X"

# Agent completes task
bc report done "Feature X implemented"

# Agent gets stuck
bc report stuck "Need clarification on requirements"

# Agent encounters error
bc report error "Build failed: missing dependency"
```

---

## 4. Work Queue Lifecycle

How work items flow through the system.

### States

```
┌────────────────────────────────────────────────────────────────┐
│                    WORK ITEM LIFECYCLE                          │
└────────────────────────────────────────────────────────────────┘

                          bc queue add
                               │
                               ▼
                         ┌──────────┐
                         │ pending  │
                         └────┬─────┘
                              │ bc queue assign
                              ▼
                         ┌──────────┐
                         │ assigned │
                         └────┬─────┘
                              │ agent starts work
                              ▼
                         ┌──────────┐
                         │ working  │
                         └────┬─────┘
                              │
                 ┌────────────┴────────────┐
                 │                         │
                 ▼                         ▼
            ┌──────────┐             ┌──────────┐
            │   done   │             │  failed  │
            └────┬─────┘             └──────────┘
                 │
                 │ bc merge process
                 ▼
            ┌──────────┐
            │  merged  │
            └──────────┘
```

### Queue Commands

```bash
# Add work item
bc queue add "Fix bug #123" --description "Users can't login"

# List all work
bc queue list

# Filter by status
bc queue list --status pending
bc queue list --status working

# Assign to agent
bc queue assign work-001 eng-01

# Update status (usually done by agent via bc report)
bc queue status work-001 working
bc queue status work-001 done
```

---

## 5. Merge Queue Processing

Handling completed work through to merge.

### Merge States

| Status | Description |
|--------|-------------|
| (empty) | Work not completed yet |
| unmerged | Ready for merge |
| merging | Currently being merged |
| merged | Successfully merged |
| conflict | Has merge conflicts |

### Merge Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                      1. WORK COMPLETION                          │
│  Agent: bc report done "Feature complete"                       │
│  → Sets work item: status=done, merge=unmerged                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      2. LIST MERGEABLE                           │
│  bc merge list                                                  │
│  → Shows all items with merge=unmerged                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      3. PROCESS MERGE                            │
│  bc merge process                                               │
│  → Sets: merge=merging                                          │
│  → Fetches agent's branch                                       │
│  → Rebases onto main                                            │
│  → Runs tests                                                   │
│  → Merges to main                                               │
│  → Sets: merge=merged                                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
        ┌──────────┐                  ┌──────────┐
        │  merged  │                  │ conflict │
        └──────────┘                  └────┬─────┘
                                           │
                                           ▼
                                     Needs manual
                                     resolution
```

### Handling Conflicts

When merge conflicts occur:

```bash
# Check merge status
bc merge status

# Output shows:
# work-001: conflict (eng-01)
#   Branch: eng-01/work-001/feature
#   Conflicts: src/auth.go, src/login.go

# Option 1: Send agent to resolve
bc send eng-01 "Your work has merge conflicts. Please resolve and re-submit."

# Option 2: Resolve manually
cd .bc/worktrees/eng-01
git rebase main
# ... resolve conflicts ...
git add .
git rebase --continue

# Retry merge
bc merge process
```

---

## 6. Communication Patterns

Inter-agent and human-to-agent communication.

### Direct Messages

```bash
# Send message to agent
bc send eng-01 "Please prioritize work-001"

# Message is delivered to agent's tmux session
# Agent sees: "Please prioritize work-001"
```

### Channels

Broadcast to multiple agents via channels:

```bash
# Create a channel
bc channel create announcements

# Send to channel
bc channel send announcements "All agents: Stand down for system update"

# List channels
bc channel list
```

### Attach for Interaction

For direct interaction with an agent:

```bash
# Attach to agent's session
bc attach eng-01

# You're now in the agent's tmux session
# Interact directly with Claude Code

# Detach (doesn't stop agent)
# Press: Ctrl+b d
```

---

## 7. Parallel Work Pattern

Multiple agents working simultaneously on different tasks.

### Setup

```bash
# Spawn multiple engineers
bc spawn eng-01 --role engineer
bc spawn eng-02 --role engineer
bc spawn eng-03 --role engineer

# Add multiple work items
bc queue add "Implement login"
bc queue add "Implement logout"
bc queue add "Add password reset"

# Assign in parallel
bc queue assign work-001 eng-01
bc queue assign work-002 eng-02
bc queue assign work-003 eng-03
```

### Monitoring

```bash
# Watch all agents
bc status --watch

# Output updates in real-time:
# AGENT      ROLE       STATE     TASK
# eng-01     engineer   working   Implementing login form
# eng-02     engineer   working   Adding logout button
# eng-03     engineer   idle      (waiting for context)
```

### Worktree Isolation

Each agent works in their own worktree:

```
.bc/worktrees/
├── eng-01/          # Working on login
│   └── (full repo)
├── eng-02/          # Working on logout
│   └── (full repo)
└── eng-03/          # Working on password reset
    └── (full repo)
```

No merge conflicts during development - conflicts only occur at merge time.

---

## 8. Recovery Workflows

Handling stuck agents and failed work.

### Stuck Agent Recovery

```bash
# Check status - agent is stuck
bc status
# AGENT      ROLE       STATE     TASK
# eng-01     engineer   stuck     Need help with auth flow

# Option 1: Send guidance
bc send eng-01 "Use the OAuth2 library in pkg/auth. Example in docs/oauth.md"

# Option 2: Attach and help directly
bc attach eng-01
# Interact with agent, then detach

# Option 3: Restart agent (loses context)
bc down
bc spawn eng-01 --role engineer
```

### Failed Work Recovery

```bash
# Check queue - work failed
bc queue list
# ID         STATUS    TITLE                    ASSIGNED TO
# work-001   failed    Implement auth           eng-01

# Reassign to different agent
bc spawn eng-02 --role engineer
bc queue assign work-001 eng-02

# Or reset and retry with same agent
bc queue status work-001 pending
bc queue assign work-001 eng-01
```

### Full System Restart

```bash
# Stop everything
bc down

# Clean worktrees (optional - removes uncommitted work)
bc worktree clean

# Start fresh
bc up
```

---

## 9. Monitoring Workflows

Keeping track of system state.

### Status Dashboard

```bash
# One-time status
bc status

# Continuous monitoring
bc status --watch

# JSON output for scripting
bc status --json | jq '.agents[] | select(.state == "working")'
```

### Event Log

```bash
# Recent events
bc logs

# Tail events live
bc logs --follow

# Filter by type (via grep)
bc logs --json | grep agent_spawn
```

### Statistics

```bash
# Summary stats
bc stats

# Output:
# Agents:     5 total (3 running, 2 stopped)
# Work Queue: 12 items (3 pending, 5 working, 4 done)
# Worktrees:  3 active
# Uptime:     2h 15m
```

---

## 10. Cleanup Workflows

Proper shutdown and cleanup procedures.

### Graceful Shutdown

```bash
# Stop all agents gracefully
bc down

# This:
# 1. Signals agents to wrap up
# 2. Waits for graceful shutdown
# 3. Kills tmux sessions
# 4. Preserves worktrees
```

### Force Shutdown

```bash
# Force stop (skips graceful shutdown)
bc down --force

# Use when agents are unresponsive
```

### Full Cleanup

```bash
# Stop agents
bc down

# Remove orphaned worktrees
bc worktree clean

# This removes worktrees for stopped agents
# Preserves uncommitted work if any

# Dry run to see what would be removed
bc worktree clean --dry-run
```

### Reset Workspace

To completely reset (loses all state):

```bash
# Stop everything
bc down

# Remove bc state
rm -rf .bc

# Reinitialize
bc init
```

---

## Quick Reference

### Common Workflows

| Workflow | Commands |
|----------|----------|
| **Start session** | `bc init` → `bc up` |
| **Spawn agent** | `bc spawn <name> --role <role>` |
| **Assign work** | `bc queue add "..."` → `bc queue assign <id> <agent>` |
| **Monitor** | `bc status --watch` |
| **Merge work** | `bc merge list` → `bc merge process` |
| **End session** | `bc down` |

### State Reporting

| Agent State | Report Command |
|-------------|----------------|
| Starting work | `bc report working "Task description"` |
| Completed | `bc report done "Completion message"` |
| Stuck | `bc report stuck "What's blocking"` |
| Error | `bc report error "Error description"` |

### Recovery Actions

| Problem | Solution |
|---------|----------|
| Agent stuck | `bc send <agent> "guidance"` |
| Work failed | `bc queue status <id> pending` → reassign |
| Merge conflict | Manual resolve in worktree, then `bc merge process` |
| Unresponsive | `bc down --force` → `bc up` |
