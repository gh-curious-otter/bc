# Agent Roles and Responsibilities

This document clarifies the boundaries and responsibilities for each agent role in bc.

## Role Overview

| Role | Primary Function | Key Boundary |
|------|------------------|--------------|
| **Root/Coordinator** | System health monitoring | Does NOT assign work |
| **Product Manager** | Vision and strategy | Does NOT implement |
| **Manager** | Task coordination | Does NOT write code |
| **Engineer** | Implementation | Does NOT create epics |

## Root/Coordinator Role

**Primary Function:** System health monitoring and orchestration

### Responsibilities
- Monitor agent health and status
- Detect stuck or errored agents
- Escalate issues to appropriate managers
- Start/stop agent sessions
- Maintain system stability

### Boundaries (What NOT to do)
- Never assign work directly to engineers
- Never create tasks or epics
- Never make product decisions
- Never bypass managers for communication

### Example Actions
```
DO:  bc agent health --check-all
DO:  bc status --with-worktrees
DO:  Escalate "eng-01 stuck for 30 minutes" to manager
DON'T: Assign "fix bug X" directly to an engineer
DON'T: Create GitHub issues
```

## Product Manager Role

**Primary Function:** Define product vision and strategy

### Responsibilities
- Create and maintain product epics
- Define acceptance criteria
- Approve completed features
- Prioritize the backlog
- Review proposals from managers

### Boundaries (What NOT to do)
- Never assign individual tasks to engineers (delegate to managers)
- Never implement code
- Never make technical architecture decisions (consult tech leads)

### Communication Patterns
- **Receives from:** Stakeholders, users, market feedback
- **Sends to:** Managers (epics, priorities, approvals)
- **Reviews:** Completed epics from managers

### Example Actions
```
DO:  Create epic "User Authentication System"
DO:  Approve manager's implementation proposal
DO:  Prioritize backlog based on business value
DON'T: Tell engineer which function to write
DON'T: Review individual PRs (managers do this)
```

## Manager Role

**Primary Function:** Coordinate work and unblock engineers

### Responsibilities
- Break epics into implementable tasks
- Assign tasks to engineers
- Review PRs and provide feedback
- Unblock engineers facing issues
- Report progress to product manager
- Coordinate between engineers

### Boundaries (What NOT to do)
- Never implement features (engineers do this)
- Never create epics (product manager does this)
- Never bypass product manager for priorities

### Communication Patterns
- **Receives from:** Product Manager (epics, priorities)
- **Sends to:** Engineers (tasks, feedback, guidance)
- **Reports to:** Product Manager (progress, blockers, completion)

### Example Actions
```
DO:  Break epic into tasks: #101, #102, #103
DO:  Assign @eng-01 to #101
DO:  Review PR and provide feedback
DO:  Unblock engineer stuck on API issue
DON'T: Implement the feature yourself
DON'T: Create new epics without PM approval
```

## Engineer Role

**Primary Function:** Implement tasks and deliver code

### Responsibilities
- Implement assigned tasks
- Write tests for code
- Create PRs for review
- Report progress and blockers
- Follow established patterns
- Document code as needed

### Boundaries (What NOT to do)
- Never create epics (product manager does this)
- Never assign work to other engineers (manager does this)
- Never merge without review approval

### Communication Patterns
- **Receives from:** Manager (tasks, feedback)
- **Reports to:** Manager (status, blockers, completion)
- **Collaborates with:** Other engineers (code review, questions)

### Example Actions
```
DO:  Implement task #101
DO:  Create PR with tests
DO:  Report "blocked on API documentation"
DO:  Ask other engineers for code review
DON'T: Assign work to other engineers
DON'T: Merge PRs without approval
```

## Communication Flow Diagram

```
                    ┌─────────────────┐
                    │  Root/Coordinator │
                    │  (Monitors only)  │
                    └────────┬─────────┘
                             │ health alerts
                             ▼
    ┌─────────────────────────────────────────────────────┐
    │                                                      │
    ▼                                                      │
┌─────────────────┐    epics, priorities    ┌─────────────┴───┐
│ Product Manager │ ───────────────────────>│     Manager     │
│   (Strategy)    │ <─────────────────────  │ (Coordination)  │
└─────────────────┘   progress, proposals   └────────┬────────┘
                                                     │
                                            tasks, feedback
                                                     │
                                                     ▼
                                          ┌─────────────────┐
                                          │    Engineer     │
                                          │(Implementation) │
                                          └─────────────────┘
                                                     │
                                             status, blockers
                                                     │
                                                     ▼
                                               (back to Manager)
```

## Anti-Patterns to Avoid

### 1. Coordinator Assigning Work
```
BAD:  Coordinator tells engineer "fix bug #123"
GOOD: Coordinator alerts manager "eng-01 idle, may need assignment"
```

### 2. PM Micromanaging Engineers
```
BAD:  PM reviews every PR and gives implementation feedback
GOOD: PM reviews completed epics and validates requirements
```

### 3. Engineers Self-Assigning
```
BAD:  Engineer picks tasks from backlog without manager
GOOD: Engineer requests work or waits for assignment
```

### 4. Skipping the Hierarchy
```
BAD:  PM assigns task directly to engineer
GOOD: PM creates epic → Manager breaks into tasks → Manager assigns
```

## When Roles Overlap

In small teams, one person may fill multiple roles:
- **Tech Lead** = Manager + Senior Engineer (can review AND implement)
- **Solo Developer** = All roles (pragmatic exception)

Even with overlap, maintain mental separation of responsibilities.

## Related Documentation

- [Hierarchical Agent System](hierarchical-agents.md) - Technical hierarchy details
- [Channel Conventions](channel-conventions.md) - Communication channels
