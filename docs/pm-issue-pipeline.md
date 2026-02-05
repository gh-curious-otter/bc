# PM-to-Manager Issue Pipeline

How product findings flow from the Product Manager to the Manager for task breakdown and engineering assignment.

## Pipeline Overview

```
Source (UX agent, user feedback, PM review)
    ↓
PM Triage — confirm, prioritize, add context
    ↓
Queue Item — bc queue add with priority and description
    ↓
Manager Receives — PM sends assignment message
    ↓
Manager Breaks Down — splits into engineering tasks if needed
    ↓
Engineer Assignment — manager assigns to available agents
    ↓
Implementation + Merge
    ↓
Verification — QA/UX confirms fix
    ↓
Close — bd close + queue item marked done
```

## Step 1: Intake

Issues come from three sources:

### A. UX Agent Findings
The UX agent files beads issues and sends cycle reports to the PM. PM reviews each finding.

```bash
# UX agent files issue
bd create --type bug --title "UX: Bug — Queue shows done items first"

# UX agent notifies PM
bc send product-manager "UX CYCLE REPORT — Queue tab. NEW: [P1] bc-xxx: done items bury active work"
```

### B. PM's Own Review
PM runs `bc home` and reviews the product directly, or reviews screenshots provided by the user.

```bash
# PM identifies issues and creates them directly
bc queue add "P1: Sort queue — active items first, done items last" -d "..."
```

### C. User Feedback
User reports issues in conversation. PM translates into actionable items.

## Step 2: PM Triage

For each incoming finding, PM decides:

| Decision | Action |
|----------|--------|
| Valid, actionable | Assign priority, create queue item |
| Duplicate | Link to existing item, close new issue |
| Not a problem | Close with explanation |
| Needs investigation | Create queue item tagged for engineer research |

### Priority Assignment

```
P0 — Blocks users or causes data loss. Fix now, interrupt current work.
P1 — Feature broken. Fix in current sprint, assign in next wave.
P2 — Works but confusing. Schedule when capacity allows.
P3 — Polish. Batch with related work.
```

## Step 3: Queue Item Creation

PM creates queue items with enough context for the manager to assign without further clarification.

### Required Fields

```bash
bc queue add "<Priority>: <Brief title>" -d "$(cat <<'EOF'
## Problem
[What is wrong, from the user's perspective]

## Steps to Reproduce
[If applicable — how to see the problem]

## Expected Behavior
[What should happen instead]

## Scope
[What files/areas are likely affected]

## Bead
[Link to beads issue ID if one exists, e.g., bc-xxx]

## Parent Epic
[Link to parent epic work item if applicable]
EOF
)"
```

### Example

```bash
bc queue add "P1: Sort queue — active items first, done items last" -d "$(cat <<'EOF'
## Problem
Queue tab shows all 136 items in insertion order. 126 done items bury the 10 active items.

## Steps to Reproduce
1. Run bc home
2. Switch to Queue tab
3. Observe: must scroll past 100+ done items to find active work

## Expected Behavior
Pending/assigned/working items appear at the top. Done items at the bottom.

## Scope
Display-only sort in internal/tui/workspace.go renderQueue function.

## Parent Epic
work-137
EOF
)"
```

## Step 4: Assignment to Manager

After creating queue items, PM sends a structured message to the manager with clear priorities.

### Assignment Message Format

```bash
bc send manager "$(cat <<'EOF'
PRIORITY ASSIGNMENTS — [Context]

## P0 — Fix Immediately
1. **work-NNN**: Brief title. [One sentence of context if needed.]

## P1 — This Sprint
2. **work-NNN**: Brief title.
3. **work-NNN**: Brief title.

## P2 — When Capacity Allows
4. **work-NNN**: Brief title.

NOTES:
- [Any sequencing constraints: "work-X must land before work-Y"]
- [Any agent recommendations: "engineer-02 built this area, assign to them"]
- [Any scope warnings: "keep this narrow, don't refactor adjacent code"]
EOF
)"
```

### Rules for PM-to-Manager Communication

1. **Be specific about priority** — P0/P1/P2/P3 with every item
2. **Batch related items** — group items that belong to the same epic
3. **Note dependencies** — if work-X blocks work-Y, say so
4. **Recommend agents** — if someone has context on the area, suggest them
5. **Don't micromanage** — give the "what", let manager handle the "how" and "who"

## Step 5: Manager Breakdown

Manager receives PM's assignments and:

1. Reviews each queue item's description
2. Breaks complex items into sub-tasks if needed (creates new queue items)
3. Assigns to available engineers based on skills and current load
4. Reports back to PM with the plan

PM should NOT re-assign or override manager's engineer choices unless there's a clear reason.

## Step 6: Verification Loop

After engineers complete work:

1. Manager merges branches to main
2. QA/UX agent runs verification cycle
3. If fix confirmed: `bd close <issue-id>`, queue item marked done
4. If regression found: new issue filed, PM re-triages
5. PM reviews closed items periodically to ensure nothing fell through

## Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| PM assigns directly to engineers | PM creates queue items, manager assigns |
| File vague items ("fix the TUI") | File specific items with repro steps |
| Skip priority on items | Always include P0/P1/P2/P3 |
| Create items without checking for dupes | Search queue and beads first |
| Batch 5 problems into 1 queue item | One problem per queue item |
| Override manager's agent assignments | Trust manager's judgment on who |
| Forget to close beads after fix ships | Close issues as part of verification |

## Metrics

Track pipeline health:

- **Intake rate**: How many findings per test cycle?
- **Triage latency**: How long from finding to queue item?
- **Fix latency**: How long from queue item to merged fix?
- **Verification rate**: What % of fixes are verified by UX/QA?
- **Reopen rate**: What % of closed issues regress?
