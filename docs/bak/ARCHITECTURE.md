# Architecture Guide

System design and component overview for bc.

## Overview

bc is a CLI-first orchestration system built in Go with a React/TypeScript TUI. It coordinates AI agents running in isolated tmux sessions, each with their own git worktree.

```
┌─────────────────────────────────────────────────────────┐
│                      bc CLI                              │
├─────────────────────────────────────────────────────────┤
│                   internal/cmd                           │
│    (Cobra commands: agent, channel, status, etc.)       │
├─────────────────────────────────────────────────────────┤
│                    pkg/ layer                            │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │ agent   │ │ channel │ │ memory  │ │workspace│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │  tmux   │ │   git   │ │  cost   │ │ events  │       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
├─────────────────────────────────────────────────────────┤
│                   Storage Layer                          │
│        SQLite (channels.db) + File-based state          │
└─────────────────────────────────────────────────────────┘
```

## Directory Structure

```
bc/
├── cmd/bc/main.go           # Entry point
├── internal/cmd/            # CLI commands (single package)
│   ├── root.go              # Root command
│   ├── agent.go             # Agent commands
│   ├── channel.go           # Channel commands
│   ├── status.go            # Status commands
│   └── ...
├── pkg/                     # Reusable packages
│   ├── agent/               # Agent lifecycle
│   ├── workspace/           # Workspace management
│   ├── channel/             # Communication
│   ├── memory/              # Persistent memory
│   ├── cost/                # Cost tracking
│   ├── events/              # Event logging
│   ├── tmux/                # Session management
│   ├── git/                 # Worktree operations
│   └── ...
├── tui/                     # React/TypeScript TUI
│   └── src/
│       ├── app.tsx          # Main application
│       ├── views/           # Screen views
│       ├── components/      # UI components
│       └── hooks/           # React hooks
├── config/                  # Generated config code
└── prompts/                 # Role prompt templates
```

## Workspace Structure

When you run `bc init`, the following structure is created:

```
your-project/
├── .bc/                     # BC state directory
│   ├── config.toml          # Workspace configuration
│   ├── agents/              # Agent state files
│   │   ├── eng-01.json
│   │   └── eng-02.json
│   ├── channels.db          # SQLite message database
│   ├── memory/              # Agent memory storage
│   │   ├── eng-01/
│   │   └── eng-02/
│   ├── roles/               # Role definitions
│   │   ├── engineer.md
│   │   └── manager.md
│   ├── worktrees/           # Agent git worktrees
│   │   ├── eng-01/
│   │   └── eng-02/
│   └── plugins/             # Installed plugins
└── ... (your project files)
```

## Core Components

### Agent Lifecycle

Agents are managed through `pkg/agent`:

1. **Create**: `agent.Create()` - Registers agent in state
2. **Start**: `agent.Start()` - Spawns tmux session with worktree
3. **Send**: `agent.Send()` - Delivers message to session
4. **Stop**: `agent.Stop()` - Terminates tmux session
5. **Delete**: `agent.Delete()` - Removes state and worktree

```go
// Agent state structure
type Agent struct {
    Name     string    `json:"name"`
    Role     Role      `json:"role"`
    State    State     `json:"state"`
    Team     string    `json:"team,omitempty"`
    Created  time.Time `json:"created"`
    Worktree string    `json:"worktree"`
}
```

### Workspace Management

`pkg/workspace` handles workspace detection and configuration:

- **v2 format**: `config.toml` (TOML, recommended)
- **v1 format**: `config.json` (JSON, legacy)

```go
// Find workspace by walking up directory tree
ws, err := workspace.Find(cwd)

// Load workspace from specific path
ws, err := workspace.Load(path)

// Initialize new workspace
ws, err := workspace.Init(path)
```

### Channel Communication

`pkg/channel` provides Slack-like communication:

- SQLite-backed message storage
- Real-time delivery to tmux sessions
- Message history and search
- Reactions support

```go
// Send message
ch.Send(agentName, message)

// Get history
messages, err := ch.History(limit)
```

### Memory System

`pkg/memory` provides persistent knowledge:

- **Experiences**: Task outcomes and context
- **Learnings**: Categorized insights
- Searchable across sessions

```go
// Record experience
mem.RecordExperience(task, outcome, context)

// Add learning
mem.Learn(category, content)

// Search
results := mem.Search(query)
```

### TMux Integration

`pkg/tmux` manages agent sessions:

- Each agent runs in isolated tmux session
- Session naming: `bc-<workspace>-<agent>`
- Supports attach, peek, send-keys

```go
// Create session
tmux.NewSession(name, workdir)

// Send text
tmux.SendKeys(session, text)

// Capture output
output := tmux.Capture(session, lines)
```

### Git Worktrees

`pkg/git` provides worktree isolation:

- Each agent gets dedicated worktree
- Prevents merge conflicts
- Enables parallel development

```go
// Create worktree
git.WorktreeAdd(path, branch)

// Remove worktree
git.WorktreeRemove(path)
```

## Data Flow

### Command Execution

```
User Input → Cobra Command → pkg Handler → Storage/External
     ↑                              │
     └──────── Output ──────────────┘
```

### Agent Communication

```
bc agent send eng-01 "message"
         │
         ▼
  ┌─────────────┐
  │ cmd/agent.go│
  └─────────────┘
         │
         ▼
  ┌─────────────┐
  │ pkg/agent   │
  └─────────────┘
         │
         ▼
  ┌─────────────┐
  │ pkg/tmux    │──── tmux send-keys ────▶ Agent Session
  └─────────────┘
```

### Channel Flow

```
bc channel send eng "message"
         │
         ▼
  ┌─────────────┐
  │ cmd/channel │
  └─────────────┘
         │
         ▼
  ┌─────────────┐
  │ pkg/channel │
  └─────────────┘
         │
    ┌────┴────┐
    ▼         ▼
SQLite DB   tmux sessions
(persist)   (deliver)
```

## TUI Architecture

The TUI is built with React and Ink:

```
tui/src/
├── app.tsx              # Root component
├── views/
│   ├── DashboardView    # Main dashboard
│   ├── AgentsView       # Agent management
│   ├── ChannelsView     # Channel messages
│   ├── MemoryView       # Agent memory
│   └── ...
├── components/
│   ├── AgentList        # Agent list widget
│   ├── StatusBar        # Status bar
│   └── ...
└── hooks/
    ├── useAgents        # Agent data hook
    ├── useChannels      # Channel data hook
    └── usePolling       # Auto-refresh hook
```

### View Navigation

```
Dashboard (d) ─┬─▶ Agents (a)
               ├─▶ Channels (c)
               ├─▶ Memory (m)
               ├─▶ Logs (l)
               └─▶ Settings (s)
```

## Configuration

### config.toml Structure

```toml
[workspace]
name = "my-project"
version = 2

[tools]
default = "claude"

[tools.claude]
command = "claude"
enabled = true

[memory]
backend = "file"
path = ".bc/memory"

[roster]
engineers = 4
tech_leads = 2
qa = 2
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `BC_WORKSPACE` | Workspace root path |
| `BC_AGENT_ID` | Current agent name |
| `BC_AGENT_ROLE` | Current agent role |
| `BC_AGENT_WORKTREE` | Agent's worktree path |
| `BC_BIN` | Path to bc binary |
| `NO_COLOR` | Disable colored output |

## Extension Points

### Plugins

Plugins extend bc functionality:

```
.bc/plugins/
├── my-plugin/
│   ├── plugin.toml      # Plugin manifest
│   └── ...
```

See [Plugins Guide](PLUGINS.md).

### MCP Integration

Model Context Protocol servers provide tools:

```toml
[[mcp.servers]]
name = "filesystem"
command = "mcp-server-filesystem"
```

See [MCP Guide](MCP.md).

## Database Schema

### channels.db

```sql
-- Messages table
CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    channel TEXT NOT NULL,
    sender TEXT NOT NULL,
    content TEXT NOT NULL,
    timestamp INTEGER NOT NULL
);

-- Full-text search
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    content='messages'
);
```

## Testing

```bash
# Run all tests with race detector
make test

# Run specific package tests
go test ./pkg/agent/...

# Run with coverage
make coverage

# Run TUI tests
cd tui && bun test
```

## Performance

- **Startup**: < 100ms for CLI commands
- **TUI Refresh**: 5s default polling
- **SQLite**: WAL mode for concurrent access
- **Memory**: ~50MB typical for TUI

## Security

- **Isolation**: Each agent in separate tmux/worktree
- **Credentials**: Never stored in state files
- **Audit**: All actions logged to events table
- **Permissions**: Role-based capabilities
