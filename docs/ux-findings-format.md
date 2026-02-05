# UX Findings Report Format

Standard format for reporting UX findings in the bc multi-agent system. Used by UX agents and anyone filing usability issues.

## Issue Format (bd create)

Every UX finding is filed as a beads issue with this structure:

```bash
bd create --type <bug|feature> --title "UX: <Category> — <Brief description>" --body "<body>"
```

### Title Convention

```
UX: <Category> — <Brief description>
```

Categories:
- `Bug` — Broken functionality (crash, wrong data, no response)
- `Visual` — Rendering, color, alignment, truncation issues
- `Navigation` — Can't reach something, dead ends, unexpected jumps
- `Consistency` — CLI and TUI disagree, or same action behaves differently
- `Feedback` — Missing confirmation, unclear status, silent failures
- `Polish` — Works but feels rough (ordering, defaults, labels)

Examples:
```
UX: Bug — Queue drill-down does nothing on Enter
UX: Visual — Working agents shown in red instead of green
UX: Consistency — Agent count differs between status and home
UX: Feedback — bc queue add gives no confirmation
UX: Navigation — No way to return from issue detail view
```

### Body Template

```markdown
## Category
Bug | Visual | Navigation | Consistency | Feedback | Polish

## Severity
P0 — Crash or data loss
P1 — Feature broken, no workaround
P2 — Feature broken, workaround exists
P3 — Cosmetic or minor annoyance

## Steps to Reproduce
1. [Starting state]
2. [Action taken]
3. [What to observe]

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happens]

## Impact
[Who is affected and how — e.g., "All users see done items before active work in queue"]

## Build
- Commit: [short hash]
- Branch: main
```

## Cycle Report Format (Channel Message)

After each test cycle, the UX agent posts a summary to the product-manager. This is a channel message, not a beads issue.

```
UX CYCLE REPORT — [Area]
Build: [commit hash]
Tested: [brief description of what was tested]

NEW FINDINGS:
  [P1] bc-xxx: One-line description
  [P2] bc-yyy: One-line description

VERIFIED FIXES:
  bc-zzz: Fixed in [commit] — confirmed working

CLEAN AREAS:
  - [Area that passed with no issues]

NEXT CYCLE: [What will be tested next]
```

### Rules

- One finding per beads issue — never combine multiple problems
- Always include steps to reproduce — "it looks wrong" is not enough
- Always include the git commit hash — findings against unknown builds are useless
- Severity is from the user's perspective, not the developer's
- File first, triage later — don't self-censor findings

## Severity Guide

| Level | Meaning | Example | Response |
|-------|---------|---------|----------|
| **P0** | Crash, data loss, or complete feature failure | `bc home` segfaults | Fix immediately, block other work |
| **P1** | Feature broken, user can't accomplish task | Can't view queue item details | Fix in current sprint |
| **P2** | Feature works but confusing or misleading | Wrong colors, bad sort order | Fix when capacity allows |
| **P3** | Cosmetic, minor polish | Column alignment off by 1 space | Batch with other work |

## Flow: Finding to Fix

```
UX Agent finds issue
    ↓
Files bd issue with standard format
    ↓
Posts cycle report to product-manager
    ↓
PM reviews, confirms severity, adds to queue
    ↓
Manager assigns to engineer
    ↓
Engineer fixes, commits to branch
    ↓
UX Agent verifies fix in next cycle
    ↓
bd close <issue-id>
```

## Channel Integration

- UX cycle reports go to the **product-manager** via `bc send`
- PM escalates urgent findings to `#all` or `#engineering` channels
- Verified fixes are noted in cycle reports so PM can close issues and update the queue
