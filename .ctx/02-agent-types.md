# bc Agent Roles and Hierarchy

This document describes the agent role system in bc, including capabilities, hierarchy, and state management.

---

## Overview

bc uses a hierarchical role-based agent system with a singleton Root agent at the top and specialized roles beneath:

| Role | Level | Primary Responsibility |
|------|-------|------------------------|
| Root | 0 | Singleton orchestrator, top-level merge integration |
| ProductManager | 1 | Product vision, creates epics |
| Manager | 1 | Breaks down epics, manages engineers |
| TechLead | 1 | Code review, architectural decisions |
| Engineer | 2 | Implements tasks (code) |
| QA | 2 | Tests and validates implementations |

Legacy roles (for backward compatibility):
- `coordinator` - Maps to ProductManager capabilities
- `worker` - Maps to Engineer capabilities

---

## Role Definitions

### Root (Level 0) - Singleton

The **Root** agent is the singleton orchestrator at the top of the hierarchy. Only one root agent can exist per workspace.

**Capabilities:**
- `create_agents` - Can spawn ProductManager, Manager, TechLead
- `assign_work` - Can assign work items to other agents
- `review_work` - Can review work from other agents
- `merge_to_main` - Exclusive: Only root can merge to main branch

**Can Create:** ProductManager, Manager, TechLead

**Typical Tasks:**
- Orchestrate the entire workspace
- Final merge integration to main branch
- Resolve escalated conflicts
- Top-level coordination

**Notes:**
- Created automatically by `bc init`
- Started by `bc up`
- Singleton constraint enforced in spawn

### ProductManager (Level 1)

The **ProductManager** (PM) is responsible for product vision and high-level work organization.

**Capabilities:**
- `create_agents` - Can spawn child agents (Managers)
- `assign_work` - Can assign work items to other agents
- `create_epics` - Can create high-level epics
- `review_work` - Can review work from other agents

**Can Create:** Manager

**Typical Tasks:**
- Define product requirements
- Create and prioritize epics
- Spawn Manager agents for complex features
- Review completed work

**Example Spawn:**
```bash
bc spawn pm-01 --role product-manager
```

### TechLead (Level 1)

The **TechLead** provides technical oversight, code review, and architectural guidance.

**Capabilities:**
- `review_work` - Can review implementations
- `assign_work` - Can delegate or reassign tasks

**Can Create:** (none - advisory role)

**Typical Tasks:**
- Review code from engineers
- Make architectural decisions
- Provide technical guidance
- Unblock engineers when stuck

**Example Spawn:**
```bash
bc spawn tech-lead-01 --role tech-lead
```

### Manager (Level 1)

The **Manager** breaks down epics into actionable tasks and coordinates Engineer and QA agents.

**Capabilities:**
- `create_agents` - Can spawn child agents (Engineers, QA)
- `assign_work` - Can assign tasks to engineers and QA
- `review_work` - Can review implementations

**Can Create:** Engineer, QA

**Typical Tasks:**
- Break down epics into implementation tasks
- Spawn Engineer agents for implementation
- Spawn QA agents for testing
- Coordinate work across team

**Example Spawn:**
```bash
bc spawn mgr-01 --role manager
```

### Engineer (Level 2)

The **Engineer** implements code changes and features.

**Capabilities:**
- `implement_tasks` - Can write code and implement features

**Can Create:** (none - leaf role)

**Typical Tasks:**
- Implement features
- Fix bugs
- Write unit tests
- Create pull requests

**Example Spawn:**
```bash
bc spawn eng-01 --role engineer
```

### QA (Level 2)

The **QA** agent tests and validates implementations.

**Capabilities:**
- `test_work` - Can test and validate implementations
- `review_work` - Can review implementations

**Can Create:** (none - leaf role)

**Typical Tasks:**
- Write integration tests
- Run test suites
- Validate implementations
- Report issues found

**Example Spawn:**
```bash
bc spawn qa-01 --role qa
```

---

## Role Hierarchy

```
                         ┌─────────────────┐
                         │      Root       │  Level 0 (singleton)
                         │    (root)       │
                         └────────┬────────┘
                                  │
           ┌──────────────────────┼──────────────────────┐
           │                      │                      │
           ▼                      ▼                      ▼
  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
  │ ProductManager  │    │    Manager      │    │   TechLead      │  Level 1
  │    (pm-01)      │    │   (mgr-01)      │    │ (tech-lead-01)  │
  └────────┬────────┘    └────────┬────────┘    └─────────────────┘
           │                      │
           │         ┌────────────┼────────────┐
           │         │            │            │
           │         ▼            ▼            ▼
           │   ┌───────────┐ ┌───────────┐ ┌───────────┐
           │   │ Engineer  │ │ Engineer  │ │    QA     │  Level 2
           │   │ (eng-01)  │ │ (eng-02)  │ │  (qa-01)  │
           │   └───────────┘ └───────────┘ └───────────┘
           │
           └──── Can create multi-level hierarchies
```

### Hierarchy Rules

1. **Root singleton** - Only one root agent per workspace, enforced on spawn
2. **Parent-child relationships** - Agents can only create roles allowed by `RoleHierarchy`
3. **Capability-based access** - Actions are gated by role capabilities
4. **Level-based sorting** - Agents are sorted by level (0=top) then by name

```go
// Role hierarchy from pkg/agent/agent.go
var RoleHierarchy = map[Role][]Role{
    RoleRoot:           {RoleProductManager, RoleManager, RoleTechLead},
    RoleProductManager: {RoleManager},
    RoleManager:        {RoleEngineer, RoleQA},
    RoleTechLead:       {}, // Advisory role
    RoleEngineer:       {}, // Cannot create children
    RoleQA:             {}, // Cannot create children
}
```

---

## Agent State Machine

Each agent has a lifecycle state that tracks its operational status.

### States

| State | Description |
|-------|-------------|
| `starting` | Agent session is initializing |
| `idle` | Ready for work, no active task |
| `working` | Actively executing a task |
| `done` | Task completed successfully |
| `stuck` | Agent needs assistance |
| `error` | Error occurred |
| `stopped` | Agent session terminated |

### State Transitions

```
                    ┌─────────┐
                    │starting │
                    └────┬────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────┐
│                                                       │
│     ┌──────┐        ┌─────────┐        ┌──────┐     │
│     │ idle │◀──────▶│ working │───────▶│ done │     │
│     └──┬───┘        └────┬────┘        └───┬──┘     │
│        │                 │                  │        │
│        │                 ▼                  │        │
│        │           ┌─────────┐              │        │
│        └──────────▶│  stuck  │◀─────────────┘        │
│                    └────┬────┘                       │
│                         │                            │
└─────────────────────────┼────────────────────────────┘
                          │
                          ▼
                    ┌─────────┐
                    │  error  │
                    └────┬────┘
                         │
                         ▼
                    ┌─────────┐
                    │ stopped │
                    └─────────┘
```

### Valid Transitions

From `pkg/agent/agent.go`:

| From State | Valid Transitions To |
|------------|---------------------|
| `starting` | idle, error, stopped |
| `idle` | idle, working, done, stuck, error, stopped |
| `working` | working, idle, done, stuck, error, stopped |
| `done` | idle, working, stopped |
| `stuck` | stuck, idle, working, error, stopped |
| `error` | idle, working, stopped |
| `stopped` | idle, starting |

---

## Agent Structure

Each agent has the following attributes:

```go
type Agent struct {
    ID          string       // Unique identifier (e.g., "eng-01")
    Name        string       // Display name
    Role        Role         // Agent role
    State       State        // Current state
    Workspace   string       // Workspace path
    Session     string       // Tmux session name
    ParentID    string       // Parent agent ID (if any)
    Children    []string     // Child agent IDs
    HookedWork  string       // Currently assigned work item
    WorktreeDir string       // Per-agent git worktree path
    Tool        string       // AI tool (claude, cursor-agent)
    Task        string       // Current task description
    Memory      *AgentMemory // Role-specific prompt content
    StartedAt   time.Time
    UpdatedAt   time.Time
}
```

### Per-Agent Worktrees

Each agent gets its own git worktree to prevent conflicts:

```
.bc/worktrees/
├── pm-01/              # PM's worktree
├── mgr-01/             # Manager's worktree
├── eng-01/             # Engineer's worktree
└── qa-01/              # QA's worktree
```

Worktrees are created at spawn time and cleaned up when the agent is stopped.

---

## Environment Variables

Agents receive these environment variables in their tmux session:

| Variable | Description |
|----------|-------------|
| `BC_AGENT_ID` | Agent identifier |
| `BC_AGENT_ROLE` | Agent role (e.g., "engineer") |
| `BC_WORKSPACE` | Workspace root path |
| `BC_AGENT_WORKTREE` | Agent's worktree directory |
| `BC_AGENT_TOOL` | AI tool name (if specified) |
| `BC_PARENT_ID` | Parent agent ID (if any) |

---

## Capability Checks

Use capabilities to gate actions:

```go
// Check if agent can create other agents
if agent.HasCapability(CapCreateAgents) {
    // Can spawn children
}

// Check if agent can implement code
if agent.HasCapability(CapImplementTasks) {
    // Can write code
}

// Check if parent can create specific child role
if agent.CanCreate(RoleEngineer) {
    // Can spawn an engineer
}
```

### Capability Summary

| Capability | PM | Manager | Engineer | QA |
|------------|:--:|:-------:|:--------:|:--:|
| create_agents | ✓ | ✓ | | |
| assign_work | ✓ | ✓ | | |
| create_epics | ✓ | | | |
| implement_tasks | | | ✓ | |
| review_work | ✓ | ✓ | | ✓ |
| test_work | | | | ✓ |

---

## Agent Lifecycle

### 1. Spawn

```bash
bc spawn eng-01 --role engineer
```

Creates:
1. Agent record in memory/state
2. Git worktree at `.bc/worktrees/eng-01/`
3. Tmux session with environment variables
4. Loads role prompt from `prompts/engineer.md` (if exists)

### 2. Work Execution

Agent checks for work, executes tasks, reports state:

```bash
# Agent reports state change
bc report working "Implementing login feature"
bc report done "Completed login implementation"
```

### 3. Stop

```bash
bc down           # Stop all agents
```

Cleans up:
1. Kills tmux session
2. Removes git worktree
3. Updates agent state to `stopped`

---

## Role Prompts

Role-specific prompts are loaded from `prompts/<role>.md`:

```
prompts/
├── product_manager.md    # PM instructions
├── manager.md            # Manager instructions
├── engineer.md           # Engineer instructions
└── qa.md                 # QA instructions
```

The prompt is sent to the agent's Claude session at spawn time.

---

## Legacy Roles

For backward compatibility with older configurations:

| Legacy Role | Equivalent | Capabilities |
|-------------|------------|--------------|
| `coordinator` | ProductManager | create_agents, assign_work, review_work |
| `worker` | Engineer | implement_tasks |

These roles can still be used but new code should use the hierarchical roles.
