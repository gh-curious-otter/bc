# Gas Town Workflows

Comprehensive documentation of key operational workflows in the Gas Town multi-agent system.

---

## 1. Work Assignment Flow

The complete lifecycle from issue creation to merged code.

### Overview

```
Issue Creation → Convoy → Sling → Polecat Spawn → Work → Done → Merge
```

### Step-by-Step Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          1. ISSUE CREATION                              │
│  bd create --title="Fix auth bug" --type=bug                           │
│  → Creates: gt-abc (issue bead in rig's .beads/)                       │
│  → Event: issue.created                                                │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          2. CONVOY CREATION                             │
│  gt convoy create "Auth fixes" gt-abc gt-def --notify overseer         │
│  → Creates: hq-cv-xyz (convoy bead in town .beads/)                    │
│  → Links: tracks relation to gt-abc, gt-def                            │
│  → Event: convoy.created                                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          3. SLING (Work Assignment)                     │
│  gt sling gt-abc gastown                                               │
│  → Allocates: polecat slot from pool (e.g., "Toast")                   │
│  → Creates: worktree at ~/gt/gastown/polecats/Toast/rig/               │
│  → Pours: molecule from mol-polecat-work formula                       │
│  → Hooks: molecule to issue (gt-abc)                                   │
│  → Spawns: tmux session gt-gastown-Toast                               │
│  → Event: polecat.spawned                                              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          4. POLECAT WORK                                │
│  Polecat finds work on hook via gt hook                                │
│  → Executes: molecule steps (design → implement → test → submit)       │
│  → Commits: changes to feature branch                                  │
│  → Events: step.started, step.completed for each step                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          5. GT DONE (Completion)                        │
│  gt done                                                               │
│  → Pushes: branch to origin                                            │
│  → Submits: MR to merge queue (creates mr-* bead)                      │
│  → Sends: POLECAT_DONE mail to Witness                                 │
│  → Requests: self-nuke (cleanup)                                       │
│  → Exits: session immediately                                          │
│  → Event: polecat.done                                                 │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          6. WITNESS VERIFICATION                        │
│  Witness receives POLECAT_DONE mail                                    │
│  → Verifies: clean git state, branch pushed                            │
│  → Sends: MERGE_READY mail to Refinery                                 │
│  → Event: merge.ready                                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          7. REFINERY MERGE                              │
│  Refinery processes merge queue                                        │
│  → Rebases: branch onto target (main)                                  │
│  → Runs: CI/tests                                                      │
│  → Merges: to main                                                     │
│  → Sends: MERGED mail to Witness                                       │
│  → Event: mr.merged                                                    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          8. CLEANUP                                     │
│  Witness receives MERGED mail                                          │
│  → Nukes: polecat worktree and branch                                  │
│  → Closes: issue (gt-abc)                                              │
│  → Releases: polecat slot back to pool                                 │
│  → Checks: convoy completion (all issues done?)                        │
│  → Event: polecat.nuked, issue.closed                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          9. CONVOY LANDS                                │
│  When all tracked issues close:                                        │
│  → Closes: convoy bead (hq-cv-xyz)                                     │
│  → Notifies: convoy subscribers (overseer)                             │
│  → Event: convoy.landed                                                │
└─────────────────────────────────────────────────────────────────────────┘
```

### Sequence Diagram

```
Human          Mayor          Daemon         Witness        Polecat        Refinery
  │              │              │              │              │              │
  │ bd create    │              │              │              │              │
  │─────────────>│              │              │              │              │
  │              │              │              │              │              │
  │ gt sling     │              │              │              │              │
  │─────────────>│              │              │              │              │
  │              │ spawn        │              │              │              │
  │              │─────────────>│              │              │              │
  │              │              │ create       │              │              │
  │              │              │ worktree     │              │              │
  │              │              │─────────────────────────────>│              │
  │              │              │              │              │              │
  │              │              │              │              │ work...      │
  │              │              │              │              │──────┐       │
  │              │              │              │              │      │       │
  │              │              │              │              │<─────┘       │
  │              │              │              │              │              │
  │              │              │              │ POLECAT_DONE │              │
  │              │              │              │<─────────────│              │
  │              │              │              │              │              │
  │              │              │              │ MERGE_READY  │              │
  │              │              │              │─────────────────────────────>│
  │              │              │              │              │              │
  │              │              │              │              │    merge     │
  │              │              │              │              │              │
  │              │              │              │ MERGED       │              │
  │              │              │              │<─────────────────────────────│
  │              │              │              │              │              │
  │              │              │              │ nuke         │              │
  │              │              │              │─────────────>X              │
  │              │              │              │              │              │
  │ notification │              │              │              │              │
  │<─────────────────────────────────────────────────────────────────────────│
  │              │              │              │              │              │
```

---

## 2. Polecat Lifecycle

Understanding the three-layer architecture of polecat workers.

### The Three Layers

| Layer | Component | Lifecycle | Persistence |
|-------|-----------|-----------|-------------|
| **Session** | Claude (tmux pane) | Ephemeral | Cycles per step/handoff |
| **Sandbox** | Git worktree | Persistent | Until nuke |
| **Slot** | Name from pool | Persistent | Until nuke |

### Operating States

Polecats have exactly three operating states. There is NO idle pool.

| State | Description | How it happens |
|-------|-------------|----------------|
| **Working** | Actively doing assigned work | Normal operation |
| **Stalled** | Session stopped mid-work | Interrupted, crashed, or timed out |
| **Zombie** | Completed work but failed to die | `gt done` failed during cleanup |

### Lifecycle Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           gt sling                                      │
│  1. Allocate slot from pool (Toast)                                    │
│  2. Create sandbox (worktree on new branch)                            │
│  3. Start session (Claude in tmux)                                     │
│  4. Hook molecule to polecat                                           │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Work Phase                                    │
│                                                                         │
│  Session cycles may occur:                                             │
│  ├─ gt handoff between steps (voluntary)                               │
│  ├─ Context compaction (automatic)                                     │
│  └─ Crash → Witness respawns (failure recovery)                        │
│                                                                         │
│  SANDBOX PERSISTS through ALL session cycles                           │
│  Session 1 → Session 2 → Session 3 = SAME POLECAT                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           gt done (self-cleaning)                       │
│  1. Push branch to origin                                              │
│  2. Submit work to merge queue (MR bead)                               │
│  3. Request self-nuke (sandbox + session cleanup)                      │
│  4. Exit immediately                                                   │
│                                                                         │
│  Work now lives in MQ, not in polecat.                                 │
│  Polecat is GONE. No idle state.                                       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Refinery merge queue                          │
│  1. Rebase and merge to main                                           │
│  2. Close the issue                                                    │
│  3. If conflict: spawn FRESH polecat to re-implement                   │
│     (never send work back to original - it's gone)                     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Session Initialization (gt prime)

When a polecat session starts (or restarts after handoff):

```bash
# Automatic at session start via hook
gt prime
```

This injects:
1. Role context (polecat instructions)
2. Current molecule state
3. Hook information
4. Rig-specific configuration

### The Propulsion Principle

**If you find work on your hook, YOU RUN IT.**

```
1. gt hook                    # What's hooked?
2. bd mol current             # Where am I in the molecule?
3. Execute current step
4. bd close <step> --continue # Close and advance
5. GOTO 2 (until done)
6. gt done                    # Signal completion
```

No confirmation. No waiting. The hook having work IS the assignment.

### Hook Retrieval

```bash
# Check what's hooked
gt hook

# Example output:
#   Hooked: gt-abc (Fix authentication bug)
#   Molecule: gt-abc-mol (step 3/6: implement)
#   Branch: polecat/Toast-20260102

# Navigate molecule
bd mol current               # Where am I?
bd ready                     # What's the next step?
bd show <step-id>            # Step details
```

### Work Execution Pattern

```bash
# 1. Find current step
bd mol current
# Output: → gt-abc.3: Implement [in_progress]

# 2. Do the work
# ... write code, make changes ...

# 3. Complete step and advance
bd close gt-abc.3 --continue
# Output: ✓ Closed gt-abc.3
#         → Marked gt-abc.4 in_progress

# 4. Repeat until final step
gt done
```

### Completion and Cleanup

```bash
# Polecat runs gt done:
gt done

# This:
# 1. Commits any uncommitted work
# 2. Pushes branch to origin
# 3. Creates MR bead in merge queue
# 4. Sends POLECAT_DONE mail to Witness
# 5. Exits session
# 6. Witness nukes worktree after merge
```

---

## 3. Merge Queue Processing

The Refinery handles all merges to protected branches.

### MQ Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                         MERGE QUEUE                                    │
│                                                                        │
│  Priority Queue (FIFO with priority override)                         │
│                                                                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ mr-abc   │  │ mr-def   │  │ mr-ghi   │  │ mr-jkl   │              │
│  │ P:high   │  │ P:normal │  │ P:normal │  │ P:low    │              │
│  │ queued   │  │ queued   │  │ merging  │  │ queued   │              │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘              │
│       ▲                           │                                   │
│       │                           ▼                                   │
│   SUBMIT                      PROCESS                                 │
└────────────────────────────────────────────────────────────────────────┘
```

### MR Lifecycle States

```
QUEUED → PROCESSING → MERGED
                   ↘ FAILED
                   ↘ CONFLICT (needs rework)
```

### Merge Queue Commands

```bash
# View queue
gt mq list [rig]              # Show merge queue
gt mq next [rig]              # Show highest-priority MR
gt mq status <id>             # Detailed MR status

# Submit work
gt mq submit                  # Submit current branch to MQ

# Handle failures
gt mq retry <id>              # Retry failed merge
gt mq reject <id>             # Reject merge request
```

### Processing Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     1. MR SUBMISSION                                    │
│  Polecat: gt done → gt mq submit                                       │
│  → Creates: mr-xyz bead with branch, issue, polecat info               │
│  → Status: QUEUED                                                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     2. REFINERY PATROL                                  │
│  Refinery runs continuous patrol loop                                  │
│  → Checks: gt mq next for highest-priority item                        │
│  → Claims: MR for processing                                           │
│  → Status: PROCESSING                                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     3. REBASE ATTEMPT                                   │
│  git fetch origin                                                      │
│  git checkout <branch>                                                 │
│  git rebase origin/main                                                │
│                                                                         │
│  ┌─ SUCCESS: continue to tests                                         │
│  └─ CONFLICT: send REWORK_REQUEST to Witness                          │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     4. CI/TESTS                                         │
│  Run verification (tests, build, lint)                                 │
│                                                                         │
│  ┌─ SUCCESS: continue to merge                                         │
│  └─ FAILURE: send MERGE_FAILED to Witness                             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     5. MERGE                                            │
│  git checkout main                                                     │
│  git merge --ff-only <branch>                                          │
│  git push origin main                                                  │
│  → Sends: MERGED mail to Witness                                       │
│  → Status: MERGED                                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

### Conflict Handling

```
Refinery                    Witness                    Polecat (new)
    │                          │                          │
    │ (conflict detected)      │                          │
    │                          │                          │
    │ REWORK_REQUEST           │                          │
    │─────────────────────────>│                          │
    │                          │                          │
    │                          │ spawn fresh polecat      │
    │                          │ with rebase instructions │
    │                          │─────────────────────────>│
    │                          │                          │
    │                          │                          │ rebase work
    │                          │                          │────────┐
    │                          │                          │        │
    │                          │                          │<───────┘
    │                          │                          │
    │                          │ POLECAT_DONE             │
    │                          │<─────────────────────────│
    │                          │                          │
    │ MERGE_READY              │                          │
    │<─────────────────────────│                          │
    │                          │                          │
    │ (retry merge)            │                          │
```

**Key principle**: Original polecat is gone. Conflicts spawn a FRESH polecat.

### Merge Failure Handling

When merge fails (tests, build, not conflict):

1. Refinery sends `MERGE_FAILED` mail to Witness
2. Witness notifies relevant parties
3. Issue may be reassigned for rework
4. MR status set to FAILED

---

## 4. Communication Patterns

Inter-agent communication via the mail system.

### Mail System Overview

```
┌────────────────────────────────────────────────────────────────────────┐
│                         MAIL ROUTING                                   │
│                                                                        │
│  Addresses: <rig>/<role> or <rig>/<type>/<name>                       │
│                                                                        │
│  Examples:                                                             │
│    gastown/witness        → Witness for gastown rig                   │
│    beads/refinery         → Refinery for beads rig                    │
│    gastown/polecats/Toast → Specific polecat                          │
│    mayor/                 → Town-level Mayor                          │
│    deacon/                → Town-level Deacon                         │
└────────────────────────────────────────────────────────────────────────┘
```

### Core Message Types

| Message | Route | Purpose |
|---------|-------|---------|
| `POLECAT_DONE` | Polecat → Witness | Signal work completion |
| `MERGE_READY` | Witness → Refinery | Branch ready for merge |
| `MERGED` | Refinery → Witness | Merge successful |
| `MERGE_FAILED` | Refinery → Witness | Merge failed (tests/build) |
| `REWORK_REQUEST` | Refinery → Witness | Conflicts need resolution |
| `WITNESS_PING` | Witness → Deacon | Second-order monitoring |
| `HELP` | Any → Mayor | Request intervention |
| `HANDOFF` | Agent → self | Session continuity |

### Polecat Completion Flow

```
Polecat                    Witness                    Refinery
   │                          │                          │
   │ POLECAT_DONE             │                          │
   │ Subject: POLECAT_DONE Toast                         │
   │ Body:                    │                          │
   │   Exit: MERGED           │                          │
   │   Issue: gt-abc          │                          │
   │   MR: mr-xyz             │                          │
   │   Branch: polecat/Toast  │                          │
   │─────────────────────────>│                          │
   │                          │                          │
   │                    (verify clean)                   │
   │                          │                          │
   │                          │ MERGE_READY              │
   │                          │ Subject: MERGE_READY Toast
   │                          │ Body:                    │
   │                          │   Branch: polecat/Toast  │
   │                          │   Issue: gt-abc          │
   │                          │   Polecat: Toast         │
   │                          │   Verified: clean        │
   │                          │─────────────────────────>│
   │                          │                          │
   │                          │                    (merge)
   │                          │                          │
   │                          │ MERGED                   │
   │                          │<─────────────────────────│
   │                          │                          │
   │                    (nuke polecat)                   │
   │                          │                          │
```

### Nudge for Status Checks

The `gt nudge` command sends messages to Claude sessions:

```bash
# Check on a stuck agent
gt nudge gastown/polecats/Toast "Status check - are you still working?"

# Wake a potentially stuck Deacon
gt nudge deacon "Boot wake: check your inbox"

# Nudge from Witness patrol
gt nudge <polecat> "Witness check-in: respond with status"
```

**Important**: Always use `gt nudge`, never raw `tmux send-keys`.

### Escalation Routes

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      ESCALATION HIERARCHY                               │
│                                                                         │
│  Severity-based routing:                                               │
│                                                                         │
│  LOW      → bead only (record for audit)                               │
│  MEDIUM   → bead + mail to Mayor                                       │
│  HIGH     → bead + mail to Mayor + email to human                      │
│  CRITICAL → bead + mail to Mayor + email + SMS                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

```bash
# Escalate with severity
gt escalate --severity=medium --subject="Polecat stuck" --body="Details..."

# Critical escalation
gt escalate --severity=critical --subject="Refinery down" --body="..."
```

### Handoff Messages

Session continuity across context limits:

```bash
gt handoff -s "Work in progress" -m "Completed steps 1-3, starting step 4"
```

Creates mail for successor session:
```
Subject: HANDOFF: Work in progress
Body:
  attached_molecule: gt-abc-mol
  attached_at: 2026-01-02T10:00:00Z

  ## Context
  Working on authentication fix for gt-abc

  ## Status
  Steps 1-3 complete, step 4 (tests) in progress

  ## Next
  Complete test implementation, then gt done
```

---

## 5. Manual Intervention

Handling stuck agents, blocked MRs, and escalation procedures.

### Stuck Agent Detection

The watchdog chain monitors agent health:

```
Daemon (Go process)          ← 3-min heartbeat
    │
    └─► Boot (AI agent)       ← Intelligent triage
            │
            └─► Deacon (AI agent)  ← Continuous patrol
                    │
                    └─► Witnesses    ← Per-rig monitoring
                            │
                            └─► Polecats
```

### Detection Criteria

| Condition | Indicator | Action |
|-----------|-----------|--------|
| Session dead | tmux session gone | Restart |
| Heartbeat stale (5-15 min) | No recent activity | Nudge if mail pending |
| Heartbeat very stale (>15 min) | No response | Wake/restart |
| Unresponsive to nudge | No reply after N tries | Escalate |

### Health Check Commands

```bash
# Check agent health
gt deacon health-check <agent>

# View health state
gt deacon health-state

# Manual peek at agent
gt peek <agent>

# Nudge stuck agent
gt nudge <agent> "Status check"
```

### Stuck Polecat Handling

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    STUCK POLECAT RECOVERY                               │
│                                                                         │
│  1. Witness detects stalled polecat (no progress, no heartbeat)        │
│  2. Witness nudges polecat: gt nudge Toast "Status check"              │
│  3. If no response after N attempts:                                   │
│     a. Escalate to Deacon                                              │
│     b. Kill session: gt session stop gastown/polecats/Toast            │
│     c. Respawn: gt sling gt-abc gastown (fresh polecat)                │
│  4. Original polecat's work preserved in git (uncommitted = lost)      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

Recovery sequence:

```bash
# 1. Check polecat status
gt peek gastown/polecats/Toast

# 2. Attempt nudge
gt nudge gastown/polecats/Toast "Witness check: respond with bd mol current"

# 3. If unresponsive, kill and respawn
gt session stop gastown/polecats/Toast
gt polecat nuke Toast                    # Clean up worktree
gt sling gt-abc gastown                  # Respawn fresh
```

### Blocked MR Handling

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    BLOCKED MR RECOVERY                                  │
│                                                                         │
│  Scenario: MR stuck in queue (conflicts, test failures, etc.)          │
│                                                                         │
│  1. Check MR status: gt mq status mr-xyz                               │
│  2. For conflicts:                                                     │
│     - Refinery sends REWORK_REQUEST                                    │
│     - Fresh polecat spawned to rebase                                  │
│  3. For test failures:                                                 │
│     - gt mq retry mr-xyz (after fix)                                   │
│     - Or gt mq reject mr-xyz (abandon)                                 │
│  4. For persistent issues:                                             │
│     - gt escalate --severity=high --subject="MR blocked"               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

Manual intervention commands:

```bash
# View blocked MR
gt mq status mr-xyz

# Retry after external fix
gt mq retry mr-xyz

# Reject and reassign
gt mq reject mr-xyz
bd update gt-abc --status=open           # Reopen issue
gt sling gt-abc gastown                  # Reassign
```

### Escalation Procedures

#### When to Escalate

| Situation | Severity | Action |
|-----------|----------|--------|
| Polecat stuck, nudge failed | MEDIUM | Escalate to Deacon |
| Witness unresponsive | HIGH | Escalate to Mayor + human |
| Refinery down | CRITICAL | Full escalation chain |
| Data corruption suspected | CRITICAL | Stop work, notify human |

#### Escalation Command

```bash
# Standard escalation
gt escalate \
  --severity=high \
  --subject="Witness unresponsive: gastown" \
  --body="Witness has been unresponsive for 5 cycles" \
  --source="patrol:deacon:health-scan"

# Output:
# ✓ Created escalation gt-esc-abc123 (severity: high)
# → Created bead
# → Mailed mayor/
# → Emailed human@example.com
```

#### Acknowledge and Close

```bash
# Acknowledge (stops re-escalation)
gt escalate ack gt-esc-abc123 --note="Investigating"

# Close when resolved
gt escalate close gt-esc-abc123 --reason="Restarted witness, working now"
```

### Emergency Procedures

#### Stop Everything

```bash
# Kill all sessions in town
gt stop --all

# Kill single rig
gt stop --rig gastown
```

#### Diagnostic Mode

```bash
# Full health check
gt doctor

# Auto-repair issues
gt doctor --fix

# Check specific agent
gt peek <agent>

# View daemon logs
tail -f ~/gt/daemon/daemon.log
```

### Recovery Checklist

```
□ Identify stuck component (gt doctor, gt peek)
□ Check mail queue (gt mail inbox)
□ Check merge queue (gt mq list)
□ Nudge affected agents (gt nudge)
□ If unresponsive:
  □ Kill session (gt session stop)
  □ Clean up (gt polecat nuke if applicable)
  □ Respawn (gt sling or gt <role> start)
□ Verify recovery (gt peek, gt doctor)
□ Close escalation if created (gt escalate close)
```

---

## Appendix: Quick Reference

### Key Commands by Workflow

| Workflow | Commands |
|----------|----------|
| **Issue to Merge** | `bd create` → `gt convoy create` → `gt sling` → (work) → `gt done` |
| **Polecat Work** | `gt hook` → `bd mol current` → (work) → `bd close --continue` → `gt done` |
| **Merge Queue** | `gt mq list` → `gt mq status` → `gt mq retry/reject` |
| **Communication** | `gt mail inbox` → `gt mail read` → `gt nudge` |
| **Escalation** | `gt escalate` → `gt escalate ack` → `gt escalate close` |
| **Recovery** | `gt doctor` → `gt peek` → `gt nudge` → `gt session stop` |

### Message Flow Summary

```
POLECAT_DONE      Polecat ──────► Witness
MERGE_READY       Witness ──────► Refinery
MERGED            Refinery ─────► Witness
MERGE_FAILED      Refinery ─────► Witness
REWORK_REQUEST    Refinery ─────► Witness
WITNESS_PING      Witness ──────► Deacon
HELP              Any ──────────► Mayor
HANDOFF           Agent ─────────► Self
```
