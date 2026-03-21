# Architecture Overview

## System Layers

```
+-----------------------------------------------------------------+
|                          CLIENTS                                 |
|                                                                  |
|  bc CLI          Web UI (React)     AI Agents (Claude, etc.)    |
|  (thin HTTP)     (localhost:9374)   (MCP stdio/SSE)             |
+------+----------------+-------------------+---------------------+
       |                |                   |
       | HTTP/JSON      | HTTP/SSE          | JSON-RPC 2.0
       v                v                   v
+-----------------------------------------------------------------+
|                    bcd (daemon)  :9374                           |
|                                                                  |
|  REST API (/api/*)    SSE (/api/events)    MCP (/mcp/*)         |
|  68 endpoints         Real-time events     4 tools, 6 resources |
+-----------------------------------------------------------------+
       |
       v
+-----------------------------------------------------------------+
|                    Service Layer                                 |
|                                                                  |
|  AgentService    ChannelService    CostStore    SecretStore      |
|  CronStore       DaemonManager    EventLog     MCPStore         |
|  ToolStore                                                       |
+-----------------------------------------------------------------+
       |
       v
+-----------------------------------------------------------------+
|                    Runtime Backends                               |
|                                                                  |
|  Tmux Runtime              Docker Runtime                        |
|  (local tmux sessions)     (isolated containers)                |
+-----------------------------------------------------------------+
       |                            |
       v                            v
  AI coding tools:  Claude Code, Gemini, Cursor, Aider, Codex
```

## Components

### bc CLI (`cmd/bc/`)
Thin HTTP client — sends requests to bcd daemon. Some commands still use direct pkg/ access (migration incomplete, see #2023).

### bcd Daemon (`cmd/bcd/`, `server/`)
HTTP server on 127.0.0.1:9374. Manages all workspace state. Entry point: `cmd/bcd/main.go`.

- **REST API** (`server/handlers/`) — CRUD for agents, channels, costs, cron, daemons, secrets, tools, MCP configs, workspace, doctor
- **SSE Hub** (`server/ws/`) — real-time event broadcast to web UI and TUI
- **MCP Server** (`server/mcp/`) — JSON-RPC 2.0 for AI agent integration (stdio + SSE transports)
- **Static Files** — embedded React web UI served at `/`

### Agent Manager (`pkg/agent/`)
Core orchestration engine. Creates agents in isolated tmux sessions or Docker containers, each with their own git worktree. Manages state transitions (idle -> working -> done/error), session resume, and inter-agent communication.

### Channel System (`pkg/channel/`)
SQLite-backed messaging. Supports group and direct channels, @mentions, reactions, FTS search. Message types: text, task, review, approval, merge, status. Delivery to agent sessions via OnMessage callback.

### Storage (`pkg/db/`)
SQLite (default) or PostgreSQL. All stores target `.bc/bc.db`. WAL mode, 30s busy timeout, foreign keys enabled.

## Data Flow

### Agent Creation
```
POST /api/agents → AgentService.Create → Manager.SpawnAgentWithOptions
  → git worktree create
  → tmux/Docker session create
  → role setup (CLAUDE.md, settings.json, .mcp.json)
  → provider command start (e.g., claude --dangerously-skip-permissions)
```

### Message Delivery
```
POST /api/channels/{name}/messages → ChannelService.Send
  → SQLite insert
  → OnMessage callback:
      → AgentService.Send (tmux send-keys / docker exec) for each member
      → Hub.Publish SSE event for web UI
```

### Agent State Updates
```
Claude Code hook fires → POST /api/agents/{name}/hook
  → Manager.UpdateAgentState (idle/working/done)
  → SSE event published
```

### Cost Tracking
```
Claude Code writes JSONL session files
  → CostImporter scans every 5 minutes
  → Parses token usage + model pricing
  → Inserts into cost_records table
```

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| SQLite default, Postgres optional | Zero-config for local use, Postgres for multi-user/production |
| Tmux for agent sessions | Interactive terminal access, session persistence, pipe-pane logging |
| Docker for isolation | Resource limits, network isolation, reproducible environments |
| MCP for AI integration | Native protocol for Claude Code; agents read resources + call tools |
| SSE not WebSocket | Simpler, unidirectional server-to-client events sufficient for UI updates |
| Localhost-only binding | Local dev tool, auth not needed. CORS * acceptable on loopback |
