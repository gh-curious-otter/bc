# Architecture Overview

## System Design

bc is a CLI-first orchestration system for coordinating teams of AI coding agents. It runs locally as a daemon (`bcd`) managing agents across multiple git repositories from a single global installation.

### Global Installation

All bc state lives in `~/.bc/`:

```
~/.bc/
  bc.db                  # All data (agents, teams, channels, costs, etc.)
  settings.toml          # Global config (providers, runtime, defaults)
  secret-key             # Auto-generated AES-256 encryption key (0600)
  agents/
      .claude/            # Claude config (mounted into containers)
        CLAUDE.md      # Role prompt
        settings.json  # Claude Code settings + hooks
        .mcp.json      # MCP server configs
          CLAUDE.md      # Role prompt
          settings.json  # Claude Code settings + hooks
          .mcp.json      # MCP server configs
      worktree/          # Git worktree checkout
```

There is no per-project config. `bc init` initializes `~/.bc/` and starts bcd.

## Architecture Layers

```mermaid
graph TB
    subgraph Clients
        CLI[bc CLI<br/>thin HTTP client]
        WebUI[Web UI<br/>React dashboard]
        AI[AI Agents<br/>Claude, Gemini, etc.]
    end

    subgraph "bcd Daemon :9374"
        REST[REST API]
        SSE[SSE Hub<br/>real-time events]
        MCP[MCP Server<br/>JSON-RPC 2.0]
    end

    subgraph Services
        AgentSvc[Agent Service]
        ChannelSvc[Channel Service]
        TeamSvc[Team Service]
        CostSvc[Cost Service]
        SecretSvc[Secret Service]
        CronSvc[Cron Service]
        EventSvc[Event Log]
    end

    subgraph "Runtime Backends"
        Tmux[Tmux Runtime<br/>local sessions]
        Docker[Docker Runtime<br/>isolated containers]
    end

    subgraph Storage
        DB[(~/.bc/bc.db<br/>SQLite WAL)]
    end

    CLI -->|HTTP/JSON| REST
    WebUI -->|HTTP + SSE| REST
    AI -->|stdio / SSE| MCP

    REST --> AgentSvc & ChannelSvc & TeamSvc & CostSvc & SecretSvc & CronSvc & EventSvc
    MCP --> AgentSvc & ChannelSvc & CostSvc

    AgentSvc --> Tmux & Docker
    AgentSvc & ChannelSvc & TeamSvc & CostSvc & SecretSvc & CronSvc & EventSvc --> DB

    Tmux & Docker --> AI
```

## Components

### bc CLI (`cmd/bc/`)

Thin HTTP client. All commands are HTTP requests to bcd — no direct DB/filesystem access.

### bcd Daemon (`cmd/bcd/`, `server/`)

Long-running HTTP server on `127.0.0.1:9374`. Single process managing all state.

| Component | Path | Purpose |
|-----------|------|---------|
| REST API | `/api/*` | CRUD for all resources |
| SSE Hub | `/api/events` | Real-time event stream |
| MCP Server | `/mcp/*` | AI agent integration (JSON-RPC 2.0) |
| Web UI | `/` | Embedded React dashboard |
| Health | `/health` | Liveness + readiness probe |

### Agents

AI coding assistants running in isolated sessions. Each agent has:
- A tmux session or Docker container
- A git worktree (created and managed by bc)
- A role defining its prompt, MCP servers, and secrets
- An associated workspace (git repo path)
- Optional team membership for organizational grouping

See [agents.md](agents.md) for lifecycle, state machine, and runtime details.

### Teams

Hierarchical organizational groups for visualizing agents. Decoupled from agent lifecycle:

```mermaid
graph TD
    Root[root-team<br/>workspace: ~/repos/main] --> Backend[backend-team<br/>workspace: ~/repos/api]
    Root --> Frontend[frontend-team<br/>workspace: ~/repos/web]
    Backend --> E1[eng-01]
    Backend --> E2[eng-02]
    Frontend --> E3[eng-03]
    E5[devops-01<br/>workspace: ~/repos/infra] -.->|member of| Root
    E5 -.->|member of| Backend
```

- Teams are **views**, not ownership — agents exist independently
- Agents can appear in **multiple teams** (many-to-many via `team_members`)
- Teams form a tree via `parent_id`
- Teams can have a default workspace; agents inherit it but can override
- Deleting a team does NOT delete its agents

### Channels

SQLite-backed messaging for agent coordination:
- Group and direct channels with member management
- Message types: text, task, review, approval, merge, status
- @mentions, reactions, FTS5 search
- Delivery to agents via `tmux send-keys` with formatted context: `[#channel @sender] message`
- Auto-enrollment: agents join team channels on creation
- Retry queue for failed deliveries

### Secrets

AES-256-GCM encrypted secret store. Referenced in agent env vars as `${secret:NAME}`, resolved at runtime. Key derived via PBKDF2-SHA256 (600k iterations).

### Cost Tracking

Automatic import from Claude Code JSONL session files every 5 minutes. Per-agent, per-team, per-model breakdown with budget enforcement.

## Data Flow

### Agent Creation

```mermaid
sequenceDiagram
    participant CLI as bc CLI
    participant API as bcd API
    participant Svc as Agent Service
    participant RT as Runtime
    participant DB as SQLite

    CLI->>API: POST /api/agents
    API->>Svc: Create(name, role, workspace, team)
    Svc->>DB: INSERT INTO agents
    Svc->>RT: git worktree add
    Svc->>RT: Write role files (CLAUDE.md, .mcp.json)
    Svc->>RT: Create tmux session / Docker container
    Svc->>RT: cd worktree && provider-command
    RT-->>Svc: Session alive
    Svc->>DB: state = idle
    Svc-->>CLI: 201 Created
```

### Channel Message Delivery

```mermaid
sequenceDiagram
    participant Sender as Sender
    participant API as bcd API
    participant DB as SQLite
    participant Hub as SSE Hub
    participant Agent as Target Agent

    Sender->>API: POST /api/channels/{ch}/messages
    API->>DB: INSERT INTO messages
    API->>DB: SELECT members WHERE channel = ch
    loop Each member (except sender)
        API->>Agent: tmux send-keys "[#ch @sender] message"
        alt Delivery failed
            API->>DB: Queue for retry
        end
    end
    API->>Hub: Publish channel.message SSE event
```

### Agent State via Hooks

```mermaid
sequenceDiagram
    participant Claude as Claude Code
    participant API as bcd API
    participant DB as SQLite
    participant Hub as SSE Hub

    Claude->>API: POST /api/agents/{name}/hook (tool_use_start)
    API->>DB: state = working
    API->>Hub: agent.state_changed
    Claude->>API: POST /api/agents/{name}/hook (tool_use_end)
    API->>DB: state = idle
    API->>Hub: agent.state_changed
```

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Global `~/.bc/` | Not per-project | One daemon manages agents across all repos |
| SQLite WAL | Only DB for now | Zero-config, local-first. Server DB deferred |
| Teams as views | Decoupled, many-to-many | No lifecycle coupling; pure organization |
| bc owns worktrees | All providers, uniform | Avoids nesting; consistent across Claude/Gemini/etc. |
| tmux send-keys | Only delivery mechanism | Hooks are one-way; no other way into agent session |
| No RBAC | Deleted | Capabilities via secrets + MCP scoping |
| No auth | Localhost only | Local dev tool; auth when remote access needed |
| MCP curated tools | Subset of API | Agents get key operations, not full admin |
| INTEGER timestamps | Unix millis | Faster range queries, smaller storage than TEXT ISO8601 |
| goose migrations | Not CREATE TABLE IF NOT EXISTS | Proper versioning, rollback support |
