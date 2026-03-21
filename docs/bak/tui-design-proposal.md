# TUI Design Proposal

**Date:** 2026-02-12
**Author:** swift-falcon (Product Manager)
**Status:** Draft - Awaiting @cli Approval

---

## Executive Summary

This document proposes a comprehensive Terminal User Interface (TUI) for bc, providing a visual experience for all CLI commands. The TUI will be built with Ink (React renderer for terminals) with architecture designed for future web interface reuse.

---

## Goals

1. **Visual Interface** - GUI-like experience in the terminal
2. **Full CLI Parity** - All bc commands accessible via TUI
3. **Real-time Updates** - Live agent states, channel messages
4. **Chatroom Experience** - Slack-like channel communication
5. **CRUD Operations** - Create, Read, Update, Delete for all entities
6. **Modular Architecture** - Reusable components for future web UI

---

## Navigation Structure

```
bc tui
├── Dashboard (default view)
├── Agents
├── Channels (Chatroom)
├── Status
├── Cost
├── Demons (Scheduled Tasks)
├── Processes
├── Teams
├── Roles
├── Worktrees
└── Config
```

---

## Screen Designs

### 1. Dashboard (Home Screen)

The default view showing workspace overview at a glance.

```
┌─────────────────────────────────────────────────────────────────────┐
│  bc-v2                                           12:34 PM  Feb 12   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ Agents ──────────┐  ┌─ Activity ─────────────────────────────┐ │
│  │ Total: 10         │  │ • eng-01 completed PR #519             │ │
│  │ Working: 4        │  │ • wise-owl merged to main              │ │
│  │ Idle: 4           │  │ • sharp-eagle reviewing PR             │ │
│  │ Done: 2           │  │ • cli sent message to #product         │ │
│  │                   │  │                                        │ │
│  │ Utilization: 40%  │  └────────────────────────────────────────┘ │
│  └───────────────────┘                                              │
│                                                                     │
│  ┌─ Cost Today ──────┐  ┌─ Demons ───────────────────────────────┐ │
│  │ $0.00             │  │ No scheduled tasks                     │ │
│  │ Tokens: 0         │  │                                        │ │
│  └───────────────────┘  └────────────────────────────────────────┘ │
│                                                                     │
│  ┌─ Processes ───────┐  ┌─ Channels ─────────────────────────────┐ │
│  │ No processes      │  │ #product (3 new)  #standup (5 new)    │ │
│  │                   │  │ #engineering      #general             │ │
│  └───────────────────┘  └────────────────────────────────────────┘ │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [1]Dashboard [2]Agents [3]Channels [4]Cost [5]Demons [6]Processes  │
│ [q]Quit  [?]Help  [/]Search                                        │
└─────────────────────────────────────────────────────────────────────┘
```

**Features:**
- Agent summary with utilization percentage
- Recent activity feed
- Cost overview
- Channel activity indicators
- Quick navigation to all sections

---

### 2. Channels (Chatroom Style)

Slack-like interface for agent communication.

```
┌─────────────────────────────────────────────────────────────────────┐
│  Channels                                                    [ESC]  │
├──────────────────┬──────────────────────────────────────────────────┤
│ CHANNELS         │ #product                                         │
│ ─────────────    │ ────────────────────────────────────────────────│
│ ● #product    3  │                                                  │
│   #standup    5  │ [12:30] cli: can we remove completion command?  │
│   #engineering   │                                                  │
│   #general       │ [12:31] wise-owl: Acknowledged! Assigning to    │
│   #reviews       │         eng-01 with PR review.                   │
│                  │                                                  │
│ DIRECT MESSAGES  │ [12:32] sharp-eagle: PR #519 ready!             │
│ ─────────────    │                                                  │
│   swift-falcon   │ [12:33] swift-falcon: PR merged! 84 PRs total.  │
│   wise-owl       │                                                  │
│   root           │ [12:34] cli: can we delete scripts directory?   │
│                  │                                                  │
│                  │ [12:35] swift-falcon: Done! PR #517 merged.     │
│                  │                                                  │
│                  ├──────────────────────────────────────────────────│
│                  │ > Type message...                          [⏎]  │
│                  │   [@mention] [#channel] [attach]                │
├──────────────────┴──────────────────────────────────────────────────┤
│ [n]New Channel [j]Join [l]Leave [h]History [up/down]Navigate [⏎]Send│
└─────────────────────────────────────────────────────────────────────┘
```

**Features:**
- Channel sidebar with unread counts
- Real-time message updates
- @mentions and #channel references
- Direct messages support
- Message input with formatting hints
- Keyboard shortcuts for common actions

---

### 3. Agents View

Comprehensive agent management interface.

```
┌─────────────────────────────────────────────────────────────────────┐
│  Agents                                                      [ESC]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  NAME            ROLE             STATE      UPTIME    TASK         │
│  ───────────────────────────────────────────────────────────────── │
│  root            root             idle       8h 30m    > waiting    │
│  clever-fox      tech-lead        idle       8h 15m    > ready      │
│> eng-01          engineer         working    8h 5m     * coding...  │
│  eng-02          engineer         done       8h 5m     + completed  │
│  eng-03          engineer         idle       8h 5m     > ready      │
│  eng-04          engineer         done       8h 4m     + completed  │
│  sharp-eagle     tech-lead        working    8h 15m    o reviewing  │
│  swift-falcon    product-manager  working    8h 15m    o planning   │
│  wise-owl        manager          working    8h 15m    * coordinating│
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  eng-01 Details:                                                    │
│  ────────────────                                                   │
│  Role: engineer | Worktree: .bc/worktrees/eng-01                   │
│  Started: 2026-02-12 14:00 | Session: bc-eng-01                    │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [c]Create [d]Delete [a]Attach [p]Peek [s]Send [up/down]Select [⏎]View│
└─────────────────────────────────────────────────────────────────────┘
```

**Features:**
- Sortable agent table
- Real-time state updates
- Task status indicators (* thinking, o tool call, > prompt)
- Detail panel for selected agent
- Quick actions via keyboard shortcuts

---

### 4. Cost Dashboard

Financial overview and tracking.

```
┌─────────────────────────────────────────────────────────────────────┐
│  Cost Dashboard                                              [ESC]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ Summary ─────────────────────────────────────────────────────┐ │
│  │ Today: $12.50        This Week: $87.30       Total: $234.50   │ │
│  │ API Calls: 1,234     Tokens: 2.5M                             │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌─ By Agent ────────────────────────────────────────────────────┐ │
│  │ AGENT              COST        CALLS     TOKENS               │ │
│  │ ────────────────────────────────────────────────────────────  │ │
│  │ eng-01             $4.20       412       890K                 │ │
│  │ eng-02             $3.80       380       820K                 │ │
│  │ swift-falcon       $2.10       234       450K                 │ │
│  │ wise-owl           $1.50       156       320K                 │ │
│  │ sharp-eagle        $0.90       52        120K                 │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌─ Trends ──────────────────────────────────────────────────────┐ │
│  │ ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁  Last 7 days                                 │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [b]Budget [t]Trends [a]By Agent [p]Project [e]Export               │
└─────────────────────────────────────────────────────────────────────┘
```

---

### 5. Demons (Scheduled Tasks)

```
┌─────────────────────────────────────────────────────────────────────┐
│  Demons (Scheduled Tasks)                                    [ESC]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  NAME            SCHEDULE        NEXT RUN      STATUS    RUNS      │
│  ───────────────────────────────────────────────────────────────── │
│  daily-tests     0 9 * * *       in 2h 30m     enabled   24/24 OK  │
│  weekly-deps     0 0 * * 1       in 3d 5h      enabled   4/4 OK    │
│  nightly-build   0 2 * * *       in 8h 15m     enabled   30/32 !   │
│  health-check    */15 * * * *    in 8m         disabled  -         │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [c]Create [d]Delete [e]Edit [r]Run Now [l]Logs [up/down]Select     │
└─────────────────────────────────────────────────────────────────────┘
```

---

### 6. Processes View

```
┌─────────────────────────────────────────────────────────────────────┐
│  Processes                                                   [ESC]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  NAME          COMMAND              PORT     STATUS    UPTIME      │
│  ───────────────────────────────────────────────────────────────── │
│  dev-server    npm run dev          3000     running   2h 15m      │
│  api-server    go run ./cmd/api     8080     running   2h 15m      │
│  redis         redis-server         6379     running   2h 15m      │
│  tests         go test ./...        -        stopped   -           │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [s]Start [x]Stop [r]Restart [l]Logs [a]Attach [up/down]Select      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## CRUD Operations Matrix

| Entity | Create | Read | Update | Delete |
|--------|--------|------|--------|--------|
| **Agents** | `c` create | list, view details | rename | `d` delete |
| **Channels** | `n` new | list, history | - | delete |
| **Teams** | `c` create | list, show members | `n` rename | `d` delete |
| **Demons** | `c` create | list, logs | `e` edit | `d` delete |
| **Processes** | `s` start | list, logs | restart | `x` stop |
| **Roles** | `c` create | list, view prompt | `e` edit | `d` delete |
| **Config** | - | show | `e` edit | reset |

---

## Keyboard Navigation

### Global Keys
| Key | Action |
|-----|--------|
| `1-9` | Switch to numbered tab |
| `q` | Quit / Back |
| `ESC` | Back to previous screen |
| `?` | Show help |
| `/` | Search |
| `Tab` | Next panel |
| `Shift+Tab` | Previous panel |

### List Navigation
| Key | Action |
|-----|--------|
| `j` / `down` | Move down |
| `k` / `up` | Move up |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `Enter` | Select / View details |

---

## Open Questions for @cli

1. Should TUI be a separate command (`bc tui`) or replace `bc home`?
2. Color theme preference (dark/light/system)?
3. Priority order for screens?
4. Any specific chatroom features needed?

---

## Next Steps

1. @cli reviews and approves design direction
2. @wise-owl + tech leads complete technical design
3. Create EPIC with implementation tasks
4. Begin Phase 1 development

---

*Document Version: 1.0*
*Last Updated: 2026-02-12*
