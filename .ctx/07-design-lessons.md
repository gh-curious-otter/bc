# Design Lessons from Gas Town

Key considerations for building bc based on Gas Town analysis.

---

## 1. What Works Well (Keep These)

### Git-Backed Persistence
- **Why it works**: Crash recovery is automatic. State survives restarts.
- **Implementation**: All state lives in git-tracked files. No database needed.
- **Lesson**: Build on git from day one. It's the source of truth.

### Tmux Session Isolation
- **Why it works**: Each agent gets a clean environment. No cross-contamination.
- **Implementation**: Named sessions with predictable naming scheme.
- **Lesson**: Process isolation prevents cascading failures.

### Event Sourcing with JSONL
- **Why it works**: Full audit trail. Can replay history. Easy to debug.
- **Implementation**: Append-only logs capture every state change.
- **Lesson**: Append-only is simpler than update-in-place.

### Hierarchical Naming (rig/agent/worker)
- **Why it works**: Clear ownership. Easy to find related resources.
- **Implementation**: `gtn.rig-name.agent-type.worker-id` pattern.
- **Lesson**: Consistent naming reduces cognitive load.

### Hook-Based Work Assignment
- **Why it works**: Decouples work discovery from work execution.
- **Implementation**: Hooks check for available work, claim it, execute.
- **Lesson**: Pull-based beats push-based for agents.

### Beads for Issue Tracking
- **Why it works**: Links work to business value. Traceable outcomes.
- **Implementation**: Each unit of work tied to a bead/issue.
- **Lesson**: Always track what prompted the work.

---

## 2. Pain Points (Avoid These)

### Too Many Agent Types
- **Problem**: Witness, Refinery, Deacon, Foreman, Prospector all have overlapping concerns.
- **Symptom**: Hard to know which agent handles what.
- **Cost**: Mental overhead, debugging complexity.

### Complex State Management
- **Problem**: State scattered across daemon, witness, refinery processes.
- **Symptom**: Race conditions, inconsistent views of world state.
- **Cost**: Bugs that only appear under load or timing conditions.

### High Burn Rate ($100/hr+)
- **Problem**: Agents run continuously, even when idle.
- **Symptom**: Costs accumulate regardless of productive output.
- **Cost**: Unsustainable for long-running projects.

### "Murderous Rampaging Deacon" Chaos
- **Problem**: Agents make destructive decisions without guardrails.
- **Symptom**: Force pushes, deleted branches, broken builds.
- **Cost**: Human cleanup time, lost work, trust erosion.

### Constant Steering Required
- **Problem**: Agents drift without frequent human correction.
- **Symptom**: Work goes off-track, wasted cycles.
- **Cost**: Human attention is the bottleneck.

---

## 3. Simplification Opportunities

### Combine Witness + Refinery
- **Current**: Two separate processes watching and processing.
- **Proposed**: Single observer that both watches and acts.
- **Benefit**: Fewer moving parts, clearer data flow.

### Simpler Work Assignment
- **Current**: Complex negotiation between multiple agents.
- **Proposed**: Single queue, workers pull from it.
- **Benefit**: Predictable, debuggable, no coordination overhead.

### Built-in Cost Controls
- **Current**: Cost tracking is after-the-fact reporting.
- **Proposed**: Hard limits that pause work when reached.
- **Benefit**: No bill shock, predictable spend.

### Deterministic Agent Behavior
- **Current**: Agents have broad autonomy, unpredictable actions.
- **Proposed**: Narrow, well-defined action space per agent type.
- **Benefit**: Predictable outcomes, easier testing.

---

## 4. Key Design Decisions for bc

### Keep From Gas Town

| Feature | Reason |
|---------|--------|
| Git persistence | Proven reliability, zero infrastructure |
| Tmux isolation | Clean separation, easy debugging |
| JSONL event logs | Audit trail, replayability |
| Hierarchical names | Clarity, organization |
| Hook-based execution | Decoupled, testable |
| Issue/bead tracking | Traceability |

### Simplify or Remove

| Feature | Action | Reason |
|---------|--------|--------|
| Multiple agent types | Reduce to 2-3 | Overlapping responsibilities |
| Daemon + witness + refinery | Single coordinator | State complexity |
| Unrestricted agent autonomy | Constrain actions | Prevent chaos |
| Always-on agents | On-demand execution | Cost control |

### Alternative Approaches to Consider

**1. Single Coordinator Model**
- One process manages all state
- Workers are stateless executors
- Coordinator assigns work, tracks progress
- Simpler than distributed agent model

**2. Budget-First Scheduling**
- Work items have estimated costs
- Scheduler respects budget constraints
- Work pauses when budget exhausted
- No surprises

**3. Checkpoint-Based Recovery**
- Save state at defined points
- Can resume from any checkpoint
- Enables "rewind and replay" debugging
- Git commits as natural checkpoints

**4. Explicit Action Allowlists**
- Each agent type has permitted actions
- Anything not on the list is blocked
- Prevents destructive operations
- Easier to audit and trust

---

## 5. Proposed bc Architecture

```
bc/
├── coordinator/     # Single source of truth
│   ├── state.jsonl  # Event log
│   └── queue/       # Work items
├── workers/         # Stateless executors
│   └── [tmux sessions]
└── hooks/           # Work discovery
```

### Core Principles

1. **One coordinator, many workers** - No peer-to-peer complexity
2. **State in git, always** - Crash recovery built-in
3. **Budget is a first-class constraint** - Not an afterthought
4. **Actions are explicit** - No implicit side effects
5. **Human approval for destructive ops** - Force push, delete, etc.

### Minimal Agent Types

| Type | Responsibility |
|------|----------------|
| Coordinator | State management, work assignment, cost tracking |
| Worker | Execute assigned tasks, report results |

That's it. Two types. Clear boundaries.

---

## 6. Success Metrics

How to know if bc improves on Gas Town:

- [ ] Can explain system in 5 minutes
- [ ] Cost predictable within 20%
- [ ] No "runaway agent" incidents
- [ ] Recovery from crash in <1 minute
- [ ] Human intervention <1x per hour of agent work
