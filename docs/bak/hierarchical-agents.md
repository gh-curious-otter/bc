# Hierarchical Agent System

bc supports a hierarchical agent system where agents are organized into a tree structure with defined roles, capabilities, and parent-child relationships.

## Agent Roles

The system defines three hierarchical roles:

| Role | Level | Description |
|------|-------|-------------|
| **Product Manager** | 0 (top) | Owns product vision, creates epics, coordinates managers |
| **Manager** | 1 | Breaks down epics into tasks, manages engineers |
| **Engineer** | 2 (leaf) | Implements tasks, writes code |

### Legacy Roles (backward compatibility)

| Role | Level | Maps to |
|------|-------|---------|
| Coordinator | 0 | Similar to Product Manager |
| Worker | 2 | Similar to Engineer |

## Role Hierarchy

```
                    +-----------------+
                    | Product Manager |
                    |   (Level 0)     |
                    +--------+--------+
                             |
              +--------------+--------------+
              |              |              |
        +-----v-----+  +-----v-----+  +-----v-----+
        |  Manager  |  |  Manager  |  |  Manager  |
        | (Level 1) |  | (Level 1) |  | (Level 1) |
        +-----+-----+  +-----+-----+  +-----+-----+
              |              |              |
         +----+----+    +----+----+    +----+----+
         |    |    |    |    |    |    |    |    |
        +v+  +v+  +v+  +v+  +v+  +v+  +v+  +v+  +v+
        |E|  |E|  |E|  |E|  |E|  |E|  |E|  |E|  |E|
        +-+  +-+  +-+  +-+  +-+  +-+  +-+  +-+  +-+

        E = Engineer (Level 2)
```

## Capabilities

Each role has specific capabilities that determine what actions it can perform:

| Capability | PM | Manager | Engineer |
|------------|:--:|:-------:|:--------:|
| Create Agents | X | X | - |
| Assign Work | X | X | - |
| Create Epics | X | - | - |
| Review Work | X | X | - |
| Implement Tasks | - | - | X |

### Capability Details

- **Create Agents**: Can spawn child agents within the hierarchy
- **Assign Work**: Can assign work items to other agents
- **Create Epics**: Can create high-level epics that define product goals
- **Review Work**: Can review and approve work done by others
- **Implement Tasks**: Can write code and complete implementation work

## Parent-Child Relationships

The hierarchy enforces strict spawning rules:

| Parent Role | Can Spawn |
|-------------|-----------|
| Product Manager | Manager |
| Manager | Engineer |
| Engineer | (none - leaf role) |

### Hierarchy Validation

When an agent tries to spawn a child:
1. The system validates the parent role can create the requested child role
2. Parent-child relationship is tracked via `parent_id` and `children` fields
3. Stopping a parent can optionally stop all descendants

## Agent States

All agents can be in one of these states:

| State | Description |
|-------|-------------|
| `idle` | Ready for work, no active task |
| `starting` | Agent session is initializing |
| `working` | Actively processing a task |
| `done` | Completed current task |
| `stuck` | Blocked or needs intervention |
| `error` | Encountered an error |
| `stopped` | Session terminated |

## Environment Variables

When an agent is spawned, these environment variables are set:

| Variable | Description |
|----------|-------------|
| `BC_AGENT_ID` | The agent's unique identifier |
| `BC_AGENT_ROLE` | The agent's role (product-manager, manager, engineer) |
| `BC_WORKSPACE` | Path to the workspace |
| `BC_PARENT_ID` | Parent agent's ID (if spawned by another agent) |

## Usage

### Spawning Agents

```go
// Create a top-level Product Manager
pm, err := manager.SpawnAgent("pm-main", agent.RoleProductManager, workspacePath)

// PM creates a Manager
mgr, err := manager.SpawnChildAgent(pm.ID, "mgr-frontend", agent.RoleManager, workspacePath)

// Manager creates Engineers
eng1, err := manager.SpawnChildAgent(mgr.ID, "eng-1", agent.RoleEngineer, workspacePath)
eng2, err := manager.SpawnChildAgent(mgr.ID, "eng-2", agent.RoleEngineer, workspacePath)
```

### Querying the Hierarchy

```go
// List all children of an agent
children := manager.ListChildren(parentID)

// List all descendants (children, grandchildren, etc.)
descendants := manager.ListDescendants(parentID)

// Get parent of an agent
parent := manager.GetParent(agentID)

// List all agents by role
engineers := manager.ListByRole(agent.RoleEngineer)
```

### Stopping Agent Trees

```go
// Stop an agent and all its descendants
manager.StopAgentTree(agentID)
```

## Workflow Example

1. **Product Manager** starts and reviews the product backlog
2. PM creates **Epics** for major features
3. PM spawns **Managers** to handle different feature areas
4. Each Manager breaks epics into smaller **Tasks**
5. Managers spawn **Engineers** to implement tasks
6. Engineers report completion back to their Manager
7. Managers aggregate progress and report to PM
8. PM reviews and closes epics

## Design Rationale

This hierarchy enables:
- **Separation of concerns**: Strategic planning vs tactical execution
- **Scalability**: Add managers/engineers without overloading the PM
- **Accountability**: Clear ownership via parent-child relationships
- **Controlled delegation**: Strict rules on who can spawn whom
- **Graceful shutdown**: Stop entire subtrees when needed
