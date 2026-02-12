# TUI Technical Design

> **Status**: Draft - Pending cli approval
> **Date**: 2026-02-12
> **Authors**: wise-owl (Manager), tech-lead team

## Executive Summary

This document outlines the technical design for building a terminal user interface (TUI) for bc using Ink (React for terminals). The TUI will provide a visual experience for all bc commands including channels (chatroom), status, dashboard, costs, demons, and processes with full CRUD operations.

---

## Table of Contents

1. [Goals & Requirements](#goals--requirements)
2. [Current State](#current-state)
3. [Architecture Options](#architecture-options)
4. [Recommended Architecture](#recommended-architecture)
5. [Repository Structure](#repository-structure)
6. [Web Reuse Strategy](#web-reuse-strategy)
7. [Implementation Phases](#implementation-phases)
8. [Technical Decisions](#technical-decisions)
9. [Alternatives Considered](#alternatives-considered)
10. [Open Questions](#open-questions)

---

## Goals & Requirements

### Primary Goals

1. **Visual TUI** - Terminal interface for all bc commands
2. **Chatroom Experience** - Channels with real-time messaging like Slack
3. **CRUD Operations** - Create, read, update, delete for all entities
4. **Modular Code** - Enable future web interface with minimal rework
5. **Production Quality** - Professional, responsive, and reliable

### Features (from Vision doc)

| View | Description | Operations |
|------|-------------|------------|
| Dashboard | Summary stats, health, activity | View |
| Agents | List all agents with status | Create, peek, attach, stop, send |
| Channels | Slack-like chatroom | Send, history, join, leave |
| Costs | Cost tracking and limits | View, set limits |
| Demons | Scheduled tasks | Create, run, stop, logs |
| Processes | Running servers/builds | Start, stop, logs, attach |
| Teams | Organizational units | Create, add/remove members |
| Memory | Agent experiences | View, search, clear |

---

## Current State

### Existing Protocol

We have a JSON streaming protocol in `pkg/tui/runtime/protocol.go`:

```
AI → TUI (specs):
  {"type": "view", "view": "table", "id": "agents", "title": "Agents"}
  {"type": "set", "path": "columns", "value": [...]}
  {"type": "append", "path": "rows", "value": {...}}
  {"type": "done"}

TUI → AI (events):
  {"type": "key", "key": "enter", "view": "agents", "selected": {...}}
  {"type": "ready"}
```

### Existing Implementation

- `pkg/tui/runtime/driver.go` - BubbleTea (Go) TUI driver
- `pkg/tui/runtime/renderer.go` - Go-based rendering
- `internal/cmd/ui.go` - `bc ui` command with demo mode

This foundation supports our streaming architecture.

---

## Architecture Options

### Option A: Go Spawns Ink (Recommended)

```
┌─────────────────────────────────────────────────────────────┐
│                         bc home                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌──────────────┐         JSON          ┌──────────────┐  │
│   │              │  ──────────────────▶  │              │  │
│   │   Go CLI     │  stdin (specs)        │   Ink TUI    │  │
│   │   (bc)       │                       │   (Node.js)  │  │
│   │              │  ◀──────────────────  │              │  │
│   └──────────────┘  stdout (events)      └──────────────┘  │
│         │                                       │          │
│         ▼                                       ▼          │
│   ┌──────────────┐                       ┌──────────────┐  │
│   │  .bc/        │                       │   Terminal   │  │
│   │  state files │                       │   (TTY)      │  │
│   └──────────────┘                       └──────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**How it works:**
1. User runs `bc home`
2. Go spawns Node.js process running Ink TUI
3. Go sends JSON specs via stdin pipe
4. Ink renders to terminal
5. User events sent back via stdout pipe
6. Go processes events, updates state, sends new specs

**Pros:**
- Matches existing protocol
- Go manages all data/state
- Ink focuses purely on rendering
- Clean separation of concerns

**Cons:**
- Requires Node.js runtime
- Process coordination complexity

### Option B: Ink Calls Go CLI

```
┌──────────────┐
│   Ink TUI    │ ──▶ exec("bc agent list --json")
│   (Node.js)  │ ◀── JSON response
└──────────────┘
```

**How it works:**
- Ink TUI is primary process
- Shells out to `bc` CLI for data operations
- Parses JSON output

**Pros:**
- Simpler architecture
- No custom protocol needed

**Cons:**
- Higher latency (process spawn per operation)
- Real-time updates require polling
- Doesn't leverage existing protocol

### Option C: HTTP API

```
┌──────────────┐      HTTP       ┌──────────────┐
│   Ink TUI    │ ◀────────────▶  │  Go Server   │
│   (Node.js)  │   REST/WS       │  (bc serve)  │
└──────────────┘                 └──────────────┘
```

**How it works:**
- Go runs HTTP/WebSocket server
- Ink fetches data via HTTP
- Real-time via WebSocket

**Pros:**
- Standard web patterns
- Easy web migration
- Multiple clients possible

**Cons:**
- More infrastructure
- Overkill for single-user TUI
- Port management complexity

---

## Recommended Architecture

**Option A: Go Spawns Ink** with the existing JSON protocol.

### Rationale

1. **Existing foundation** - Protocol already designed and tested
2. **Real-time streaming** - Natural fit for pipe-based communication
3. **Single entry point** - User runs `bc home`, everything handled
4. **State management** - Go owns data, Ink owns rendering
5. **Future-proof** - Easy to add HTTP layer for web later

### Server Requirement

**Does Ink need a server? No.**

Ink is a pure terminal renderer. It reads from stdin and writes to stdout. No HTTP server is required for the TUI itself.

For bc data access, we use stdin/stdout pipes - no server needed.

---

## Repository Structure

```
bc-v2/
├── cmd/bc/                    # Go CLI entry
├── pkg/
│   ├── agent/                 # Agent management
│   ├── channel/               # Channel system
│   ├── cost/                  # Cost tracking
│   └── tui/
│       └── runtime/
│           ├── protocol.go    # JSON protocol (KEEP)
│           ├── driver.go      # BubbleTea fallback
│           └── bridge.go      # NEW: Spawns Ink process
│
├── tui/                       # NEW: Ink TUI package
│   ├── package.json
│   ├── tsconfig.json
│   ├── src/
│   │   ├── index.tsx          # Entry point
│   │   ├── App.tsx            # Root component
│   │   │
│   │   ├── core/              # SHARED (terminal + web)
│   │   │   ├── protocol/
│   │   │   │   └── types.ts   # Generated from Go
│   │   │   ├── hooks/
│   │   │   │   ├── useProtocol.ts
│   │   │   │   ├── useAgents.ts
│   │   │   │   └── useChannels.ts
│   │   │   └── state/
│   │   │       └── store.ts
│   │   │
│   │   ├── components/        # Terminal components (Ink)
│   │   │   ├── Table.tsx
│   │   │   ├── Detail.tsx
│   │   │   ├── Modal.tsx
│   │   │   ├── StatusBar.tsx
│   │   │   ├── Spinner.tsx
│   │   │   └── Channel/
│   │   │       ├── MessageList.tsx
│   │   │       ├── MessageInput.tsx
│   │   │       └── MemberList.tsx
│   │   │
│   │   └── views/             # Screen views
│   │       ├── Dashboard.tsx
│   │       ├── Agents.tsx
│   │       ├── AgentDetail.tsx
│   │       ├── Channels.tsx
│   │       ├── ChannelChat.tsx
│   │       ├── Costs.tsx
│   │       ├── Demons.tsx
│   │       ├── Processes.tsx
│   │       └── Teams.tsx
│   │
│   └── bin/
│       └── bc-tui             # Compiled/bundled entry
│
└── docs/
    └── tui-technical-design.md  # This document
```

---

## Web Reuse Strategy

### The Challenge

Ink components render to terminal, not DOM. They use terminal-specific primitives (ANSI codes, cursor positioning) that don't work in browsers.

### The Solution: Shared Core

```
┌─────────────────────────────────────────────────────────────┐
│                        tui/src/core/                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  protocol/  │  │   hooks/    │  │   state/    │         │
│  │  types.ts   │  │ useAgents   │  │  store.ts   │         │
│  │             │  │ useChannels │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                          │                                  │
│            ┌─────────────┴─────────────┐                   │
│            ▼                           ▼                   │
│   ┌─────────────────┐        ┌─────────────────┐          │
│   │ tui/components/ │        │ web/components/ │          │
│   │   (Ink/JSX)     │        │  (React DOM)    │          │
│   └─────────────────┘        └─────────────────┘          │
│            │                           │                   │
│            ▼                           ▼                   │
│   ┌─────────────────┐        ┌─────────────────┐          │
│   │    Terminal     │        │    Browser      │          │
│   └─────────────────┘        └─────────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### What Can Be Shared

| Layer | Reusable? | Notes |
|-------|-----------|-------|
| Protocol types | 100% | TypeScript interfaces |
| Data hooks | 100% | useAgents, useChannels, etc. |
| State management | 100% | Zustand/Jotai store |
| Business logic | 100% | Validation, formatting |
| Component structure | 70% | Same patterns, different JSX |
| Styling | 0% | Terminal vs CSS |

**Estimated code reuse: 60-70%**

### Future Web Architecture

```
bc-v2/
├── tui/
│   └── src/
│       ├── core/           # Shared
│       ├── components/     # Ink (terminal)
│       └── views/          # Ink (terminal)
│
└── web/                    # FUTURE
    └── src/
        ├── components/     # React DOM (web)
        └── views/          # React DOM (web)
        # imports from tui/src/core/
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

1. **Setup Ink project**
   - Create `/tui` directory
   - Configure TypeScript, esbuild
   - Setup Ink with @inkjs/ui

2. **Protocol bridge**
   - Generate TypeScript types from Go protocol
   - Implement `useProtocol` hook
   - Test stdin/stdout communication

3. **Go integration**
   - Add `pkg/tui/runtime/bridge.go`
   - Spawn Node process from `bc home`
   - Pipe JSON specs and events

4. **Basic table view**
   - Render agent list
   - Handle navigation (j/k, up/down)
   - Send key events back to Go

### Phase 2: Core Views (Week 3-4)

1. **Dashboard**
   - Summary stats (agents, costs, activity)
   - Quick actions

2. **Agents view**
   - Table with status indicators
   - Peek (live view)
   - Create/stop actions

3. **Agent detail**
   - Full agent info
   - Recent activity
   - Actions (attach, nudge, stop)

### Phase 3: Channels/Chatroom (Week 5-6)

1. **Channel list**
   - Show all channels
   - Unread indicators
   - Create channel

2. **Channel chat view**
   - Message history (scrollable)
   - Real-time updates
   - Message input
   - @mentions, formatting

3. **Member list**
   - Show channel members
   - Add/remove members

### Phase 4: Additional Views (Week 7-8)

1. **Costs**
   - Cost breakdown by agent
   - Limits and warnings
   - Historical chart (sparkline)

2. **Demons**
   - Scheduled task list
   - Run history
   - Create/edit schedules

3. **Processes**
   - Running processes
   - Logs view
   - Start/stop controls

### Phase 5: Polish (Week 9-10)

1. **Keyboard navigation**
   - Global shortcuts
   - Vim-style bindings
   - Help overlay

2. **Theming**
   - Color scheme
   - Terminal compatibility

3. **Performance**
   - Optimize re-renders
   - Handle large lists

4. **Documentation**
   - User guide
   - Keyboard shortcuts reference

---

## Technical Decisions

### Decided

| Decision | Choice | Rationale |
|----------|--------|-----------|
| TUI framework | Ink | React patterns, cli preference, web reuse |
| Language | TypeScript | Type safety, better DX |
| Architecture | Go spawns Ink | Existing protocol, clean separation |
| Transport | stdin/stdout pipes | Simple, no server needed |

### Needs Decision

| Decision | Options | Recommendation |
|----------|---------|----------------|
| State management | Context, Zustand, Jotai | **Zustand** - simple, works outside React |
| Build tooling | esbuild, tsup, Bun | **tsup** - esbuild wrapper, simpler config |
| BubbleTea fallback | Keep, Remove | **Keep** - for no-Node environments |
| Protocol sync | Manual, auto-generate | **Auto-generate** - TypeScript from Go |

---

## Alternatives Considered

### BubbleTea (Go Native)

| Aspect | BubbleTea | Ink |
|--------|-----------|-----|
| Language | Go | TypeScript/React |
| Distribution | Single binary | Requires Node.js |
| Pattern | Elm MVU | React components |
| Performance | Excellent | Good (VDOM overhead) |
| Web reuse | None | 60-70% |
| Team familiarity | Go codebase | React patterns |

**Why not chosen:** cli wants React patterns for web reuse.

### Blessed/neo-blessed

- Widget-based, not component-based
- Original blessed unmaintained
- No React mental model

**Why not chosen:** Different paradigm, less maintained.

### terminal-kit

- Full-featured but less popular
- Steeper learning curve
- No React patterns

**Why not chosen:** Smaller ecosystem, no React benefit.

---

## Open Questions

1. **Node.js version requirement?**
   - Ink supports Node 18+
   - Recommend Node 20 LTS or 22

2. **Bundle strategy?**
   - Ship bundled JS with bc binary?
   - Require user to have Node installed?
   - Use pkg/nexe to create standalone?

3. **Fallback behavior?**
   - If Node not available, use BubbleTea Go TUI?
   - Or require Node and fail fast?

4. **Testing strategy?**
   - ink-testing-library for component tests
   - Integration tests with Go bridge

---

## References

- [Ink GitHub](https://github.com/vadimdemedes/ink) - React for CLIs
- [Ink UI](https://github.com/vadimdemedes/ink-ui) - Component library
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - Go TUI (fallback)
- [bc Vision Issue #2](https://github.com/rpuneet/bc/issues/2) - Original vision
- [TUI Development: Ink + React](https://combray.prose.sh/2025-12-01-tui-development)

---

## Appendix: Protocol Types

```typescript
// Generated from pkg/tui/runtime/protocol.go

type MessageType =
  | 'view' | 'set' | 'append' | 'delete' | 'done' | 'error'  // AI → TUI
  | 'key' | 'select' | 'input' | 'ready' | 'init';           // TUI → AI

type ViewType = 'table' | 'detail' | 'form' | 'modal' | 'list';

interface ViewMessage {
  type: 'view';
  view: ViewType;
  id: string;
  title?: string;
  loading?: boolean;
}

interface SetMessage {
  type: 'set';
  path: string;
  value: unknown;
}

interface AppendMessage {
  type: 'append';
  path: string;
  value: unknown;
}

interface KeyEvent {
  type: 'key';
  key: string;
  view: string;
  selected?: RowRef;
}

interface RowRef {
  id: string;
  index: number;
  values: string[];
  data?: unknown;
}
```

---

*Document version: 1.0*
*Last updated: 2026-02-12*
