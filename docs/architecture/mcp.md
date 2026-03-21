# MCP Server Architecture

## Overview

bc exposes a Model Context Protocol (MCP) server so AI agents (Claude Code, etc.) can read workspace state and take actions programmatically. Protocol version: `2024-11-05`.

## Transports

| Transport | How | Use case |
|-----------|-----|----------|
| **stdio** | `bc mcp serve` — newline-delimited JSON on stdin/stdout | Claude Code connects via `.mcp.json` config |
| **SSE** | `GET /mcp/sse` + `POST /mcp/message` on bcd :9374 | Browser-based MCP clients, remote access |

Both transports have a 4MB message size limit.

## Resources (read-only)

| URI | Returns |
|-----|---------|
| `bc://workspace/status` | Name, path, state dir, version |
| `bc://agents` | All agents: name, role, state, tool, worktree, session |
| `bc://channels` | All channels: name, description, members, message count |
| `bc://costs` | Workspace total + per-agent cost breakdown |
| `bc://roles` | Role definitions: name, description, MCP servers, secrets |
| `bc://tools` | Available AI tools with PATH availability check |

## Tools (actions)

| Tool | Required Args | Description |
|------|--------------|-------------|
| `create_agent` | name, role | Creates and starts a new agent. Shells out to `bc agent create`. |
| `send_message` | channel, message | Posts to a channel. Uses ChannelService when available (triggers delivery + SSE). |
| `report_status` | agent, task | Updates agent's current task description. |
| `query_costs` | (optional: agent) | Returns cost summary — workspace or per-agent. |

## Notifications (server-pushed)

| Method | Trigger |
|--------|---------|
| `notifications/message` | New channel message (polled every 2s) |

**Known bug:** Polling uses `len(history)` which caps at 100. After 100 messages per channel, no new messages are detected (#2164).

## MCP Server Config Store

bc also manages external MCP servers that agents connect to, stored in `mcp_servers` table:
- Transport: stdio (command + args) or SSE (URL)
- Env vars support `${secret:NAME}` references
- Per-server enable/disable
- Role-scoped: roles declare which MCP servers their agents get

## Code

- `server/mcp/server.go` — Server struct, Handle() dispatcher, channel polling
- `server/mcp/protocol.go` — JSON-RPC 2.0 types
- `server/mcp/tools.go` — Tool implementations
- `server/mcp/resources.go` — Resource readers
- `server/mcp/sse.go` — SSE transport + broker
- `server/mcp/stdio.go` — stdio transport
- `pkg/mcp/store.go` — MCP server config storage (SQLite)
