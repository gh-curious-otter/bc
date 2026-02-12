# TUI Technical Design

**Date:** 2026-02-12
**Authors:** wise-owl (Manager), sharp-eagle (Tech Lead), clever-fox (Tech Lead), swift-falcon (Product Manager)
**Status:** Draft - Pending @cli Approval

---

## Executive Summary

Technical design for implementing the bc TUI using Ink (React renderer for terminals), with architecture that enables future web interface reuse. This document covers technology evaluation, architecture options, implementation phases, and web reuse strategy.

---

## Table of Contents

1. [Technology Evaluation](#technology-evaluation)
2. [Architecture Options](#architecture-options)
3. [Recommended Architecture](#recommended-architecture)
4. [Repository Structure](#repository-structure)
5. [Web Reuse Strategy](#web-reuse-strategy)
6. [Implementation Phases](#implementation-phases)
7. [Technical Decisions](#technical-decisions)
8. [Answers to @cli Questions](#answers-to-cli-questions)
9. [Appendix: Protocol Types](#appendix-protocol-types)

---

## Technology Evaluation

### Ink (Recommended)

**What is Ink?**
- React renderer for CLI applications
- Uses React components to build terminal UIs
- Supports hooks, state management, effects
- NPM package: `ink`

**Pros:**
- React paradigm (familiar to web developers)
- Component-based architecture
- Easy to share logic with web UI later
- Active community and maintenance
- Rich ecosystem (ink-text-input, ink-select, @inkjs/ui)

**Cons:**
- Node.js runtime required
- Slightly higher memory footprint than native solutions
- Terminal rendering limitations vs native TUI libs

**Server Requirement:**
- **No server needed for basic TUI**
- Ink runs as a standalone Node.js process
- Communicates with bc CLI via stdin/stdout pipes
- For real-time updates: can poll or use file watchers

### Alternatives Considered

| Library | Language | Pros | Cons |
|---------|----------|------|------|
| **Bubble Tea** | Go | Native to bc, fast, low memory | No React, harder web reuse |
| **Blessed** | Node.js | Mature, feature-rich | Less React-like, older, unmaintained |
| **Textual** | Python | Beautiful UIs, async | Different language |
| **Ratatui** | Rust | Fast, modern | Different language |

**Recommendation:** Ink - Best balance of React reusability and terminal capability.

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
- Matches existing protocol patterns
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

**Pros:**
- Simpler architecture
- No custom protocol needed

**Cons:**
- Higher latency (process spawn per operation)
- Real-time updates require polling
- Less integrated experience

### Option C: HTTP API

```
┌──────────────┐      HTTP       ┌──────────────┐
│   Ink TUI    │ ◀────────────▶  │  Go Server   │
│   (Node.js)  │   REST/WS       │  (bc serve)  │
└──────────────┘                 └──────────────┘
```

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

**Option B: Ink Calls bc CLI** - Simple, efficient, proven pattern.

### Rationale

1. **Simplicity** - No custom protocol, uses existing `--json` flags
2. **Efficiency** - Avoid lag issues from streaming protocol (pkg/tui/runtime had performance problems)
3. **Independence** - Ink manages its own state and rendering
4. **Maintainability** - Clear separation, easier debugging

### Why Not Option A?

The existing `pkg/tui/runtime` streaming approach had lag/efficiency issues. Option B is cleaner and leverages the already-working `--json` CLI output.

### Server Requirement

**Does Ink need a server? No.**

Ink is a pure terminal renderer. It spawns `bc` CLI commands and parses JSON output. No HTTP server required.

### Efficiency Patterns

| Pattern | Description |
|---------|-------------|
| Batch fetching | Combine related CLI calls |
| Smart polling | 2s default, configurable interval |
| Virtualized lists | Render only visible rows |
| Response caching | Avoid redundant CLI calls |

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
│           ├── protocol.go    # JSON protocol
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
│   │   │   │   └── types.ts   # TypeScript types
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
├── web/                       # FUTURE: Web UI
│   └── src/
│       ├── components/        # React DOM (web)
│       └── views/             # React DOM (web)
│       # imports from tui/src/core/
│
└── docs/
    ├── tui-design-proposal.md
    └── tui-technical-design.md
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
| State management | 100% | Zustand store |
| Business logic | 100% | Validation, formatting |
| Component structure | 70% | Same patterns, different JSX |
| Styling | 0% | Terminal vs CSS |

**Estimated code reuse: 60-70%**

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

1. **Setup Ink project**
   - Create `/tui` directory
   - Configure TypeScript, tsup (build)
   - Setup Ink with @inkjs/ui

2. **Protocol bridge**
   - Define TypeScript types for protocol
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
| Architecture | Go spawns Ink | Clean separation, real-time capable |
| Transport | stdin/stdout pipes | Simple, no server needed |

### Needs Decision (@cli)

| Decision | Options | Recommendation |
|----------|---------|----------------|
| State management | Context, Zustand, Jotai | **Zustand** - simple, works outside React |
| Build tooling | esbuild, tsup, Bun | **tsup** - esbuild wrapper, simpler config |
| Node.js requirement | Required, Optional (Go fallback) | **Required** - simpler distribution |
| Protocol sync | Manual, auto-generate | **Auto-generate** - TypeScript from Go |

---

## Answers to @cli Questions

### 1. Will we need a server to build a TUI using Ink?

**No.** Ink TUI runs as a standalone Node.js process. It communicates with bc CLI via stdin/stdout pipes. No HTTP server needed.

### 2. What are alternatives to Ink?

| Alternative | Language | Best For |
|-------------|----------|----------|
| Bubble Tea | Go | Native integration, no Node.js |
| Blessed | Node.js | Complex layouts, mature (but unmaintained) |
| Textual | Python | Rich visuals |
| Ratatui | Rust | Performance critical |

**Ink is recommended** for React reusability with future web UI.

### 3. How would the TUI code live in this repository?

```
bc/
├── cmd/bc/      # Existing Go CLI
├── pkg/         # Existing Go packages
├── tui/         # NEW: Node.js/TypeScript TUI
│   ├── package.json
│   ├── src/
│   └── bin/
└── web/         # FUTURE: Web UI
```

The `tui/` directory is a separate Node.js project within the bc monorepo.

---

## Open Questions

1. **Node.js version requirement?**
   - Ink supports Node 18+
   - Recommend Node 20 LTS or 22

2. **Bundle strategy?**
   - Ship bundled JS with bc binary?
   - Require user to have Node installed?

3. **Testing strategy?**
   - ink-testing-library for component tests
   - Integration tests with Go bridge

---

## Appendix: Protocol Types

```typescript
// TypeScript types for Go ↔ Ink communication

type MessageType =
  | 'view' | 'set' | 'append' | 'delete' | 'done' | 'error'  // Go → Ink
  | 'key' | 'select' | 'input' | 'ready' | 'init';           // Ink → Go

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

// Example data types
interface Agent {
  name: string;
  role: string;
  state: 'idle' | 'working' | 'done' | 'stuck' | 'error';
  uptime: string;
  task: string;
}

interface Channel {
  name: string;
  members: string[];
  unread: number;
}

interface Message {
  id: string;
  sender: string;
  content: string;
  timestamp: string;
}
```

---

## References

- [Ink GitHub](https://github.com/vadimdemedes/ink) - React for CLIs
- [Ink UI](https://github.com/vadimdemedes/ink-ui) - Component library
- [Zustand](https://github.com/pmndrs/zustand) - State management
- [tsup](https://github.com/egoist/tsup) - Build tool

---

*Document Version: 2.0 (Consolidated)*
*Last Updated: 2026-02-12*
