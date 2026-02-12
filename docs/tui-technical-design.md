# TUI Technical Design

**Date:** 2026-02-12
**Authors:** wise-owl (Manager), sharp-eagle (Tech Lead), clever-fox (Tech Lead)
**Status:** Draft - Research in Progress

---

## Executive Summary

Technical design for implementing the bc TUI using Ink (React renderer for terminals), with architecture that enables future web interface reuse.

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
- Rich ecosystem (ink-text-input, ink-select, etc.)

**Cons:**
- Node.js runtime required
- Slightly higher memory footprint than native solutions
- Terminal rendering limitations vs native TUI libs

**Server Requirement:**
- **No server needed for basic TUI**
- Ink runs as a standalone Node.js process
- Communicates with bc CLI via child_process/exec
- For real-time updates: can poll or use file watchers

### Alternatives Considered

| Library | Language | Pros | Cons |
|---------|----------|------|------|
| **Bubble Tea** | Go | Native to bc, fast, low memory | No React, harder web reuse |
| **Blessed** | Node.js | Mature, feature-rich | Less React-like, older |
| **Textual** | Python | Beautiful UIs, async | Different language |
| **Ratatui** | Rust | Fast, modern | Different language |

**Recommendation:** Ink - Best balance of React reusability and terminal capability.

---

## Architecture

### Repository Structure

```
bc/
├── cmd/bc/              # Go CLI (existing)
├── internal/            # Go internal packages (existing)
├── pkg/                 # Go packages (existing)
├── tui/                 # NEW: Ink TUI application
│   ├── package.json
│   ├── tsconfig.json
│   ├── src/
│   │   ├── index.tsx           # Entry point
│   │   ├── app.tsx             # Main app component
│   │   ├── components/         # Reusable UI components
│   │   │   ├── Table.tsx
│   │   │   ├── Panel.tsx
│   │   │   ├── StatusBar.tsx
│   │   │   ├── Input.tsx
│   │   │   └── ...
│   │   ├── screens/            # Full screen views
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Agents.tsx
│   │   │   ├── Channels.tsx
│   │   │   ├── Cost.tsx
│   │   │   ├── Demons.tsx
│   │   │   ├── Processes.tsx
│   │   │   └── ...
│   │   ├── hooks/              # Custom React hooks
│   │   │   ├── useAgents.ts
│   │   │   ├── useChannels.ts
│   │   │   ├── useCost.ts
│   │   │   └── ...
│   │   ├── services/           # CLI interaction layer
│   │   │   ├── bc.ts           # bc CLI wrapper
│   │   │   ├── types.ts        # TypeScript types
│   │   │   └── ...
│   │   └── styles/             # Theming
│   │       └── theme.ts
│   └── dist/                   # Compiled output
└── web/                 # FUTURE: Web UI (shares logic with tui/)
```

### Communication Layer

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Ink TUI       │────▶│   bc CLI        │────▶│   bc packages   │
│   (React)       │     │   (subprocess)  │     │   (Go)          │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │ JSON output           │ --json flag
        ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Data Flow                                    │
│  TUI calls: bc agent list --json                                │
│  CLI returns: [{"name":"eng-01","state":"working",...}]         │
│  TUI parses and renders                                         │
└─────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**

1. **No Server Required**
   - TUI spawns bc CLI as child process
   - Uses `--json` flag for structured output
   - Parses JSON responses

2. **Real-time Updates**
   - Polling interval (configurable, default 2s)
   - File watchers for channel messages
   - Event-driven where possible

3. **State Management**
   - React hooks for local state
   - Context for global state (theme, config)
   - No Redux needed (simple state)

---

## Code Modularization for Web Reuse

### Shared Layer

```typescript
// shared/types.ts - Shared TypeScript interfaces
export interface Agent {
  name: string;
  role: string;
  state: 'idle' | 'working' | 'done' | 'stuck' | 'error';
  uptime: string;
  task: string;
}

export interface Channel {
  name: string;
  members: string[];
  messages: Message[];
}

// shared/services/bc.ts - CLI wrapper (works in both TUI and web)
export class BcService {
  async listAgents(): Promise<Agent[]> {
    const result = await this.exec('bc agent list --json');
    return JSON.parse(result);
  }

  async sendMessage(channel: string, message: string): Promise<void> {
    await this.exec(`bc channel send ${channel} "${message}"`);
  }

  // Platform-specific exec implementation injected
  constructor(private exec: (cmd: string) => Promise<string>) {}
}
```

### Platform-Specific Structure

```
tui/src/
├── shared/              # Shared between TUI and Web
│   ├── types.ts
│   ├── services/
│   └── hooks/
├── ink/                 # Ink-specific components
│   ├── components/
│   └── screens/
└── index.tsx            # TUI entry point

web/src/                 # FUTURE
├── shared/ -> symlink   # Reuse from tui
├── react/               # Web-specific components
└── index.tsx            # Web entry point
```

---

## Implementation Details

### Entry Point

```typescript
#!/usr/bin/env node
import React from 'react';
import { render } from 'ink';
import { App } from './app';

render(<App />);
```

### CLI Service

```typescript
import { spawn } from 'child_process';

export async function execBc(command: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const proc = spawn('bc', command.split(' '), { shell: true });
    let stdout = '';
    proc.stdout.on('data', (data) => stdout += data);
    proc.on('close', (code) => {
      if (code === 0) resolve(stdout);
      else reject(new Error(`bc failed with code ${code}`));
    });
  });
}
```

---

## Build & Distribution

### Integration with bc CLI

**Option 1: Separate binary (Recommended)**
```bash
bc-tui           # Standalone
bc tui           # Alias that spawns bc-tui
```

**Option 2: Embedded in bc**
```bash
bc tui           # bc spawns Node.js with bundled TUI
```

---

## Future Web Interface

For web, we would add:
1. Lightweight Go HTTP server (`bc serve`)
2. REST API endpoints wrapping CLI commands
3. WebSocket for real-time updates
4. Web UI using shared React components

---

## Answers to @cli Questions

### 1. Will we need a server to build a TUI using Ink?

**No.** Ink TUI runs as a standalone Node.js process. It communicates with bc CLI by spawning it as a child process and parsing JSON output. No HTTP server needed.

### 2. What are alternatives to Ink?

| Alternative | Language | Best For |
|-------------|----------|----------|
| Bubble Tea | Go | Native integration, no Node.js |
| Blessed | Node.js | Complex layouts, mature |
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
│   └── dist/
└── web/         # FUTURE: Web UI
```

The `tui/` directory is a separate Node.js project within the bc monorepo.

---

## Recommendation Summary

| Aspect | Recommendation |
|--------|----------------|
| **Framework** | Ink (React for terminals) |
| **Server** | Not required for TUI |
| **Language** | TypeScript |
| **Structure** | tui/ directory in bc repo |
| **CLI Integration** | Child process with --json |
| **Web Reuse** | Shared hooks/services layer |

---

## Next Steps

1. @cli approves technical direction
2. Set up tui/ directory structure
3. Create proof-of-concept with Dashboard
4. Establish shared code patterns
5. Create EPIC with implementation tasks

---

*Document Version: 1.0*
*Last Updated: 2026-02-12*
