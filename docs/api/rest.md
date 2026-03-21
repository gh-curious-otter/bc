# REST API Reference

**Base URL:** `http://127.0.0.1:9374`
**Content-Type:** `application/json`
**Authentication:** None (localhost-only)

---

## Health

### `GET /health`
Liveness probe. Does not check downstream dependencies.

**Response:** `200 OK`
```json
{"status": "ok", "addr": "127.0.0.1:9374"}
```

---

## Agents

### `GET /api/agents`
List all agents. Calls RefreshState() to reconcile with live sessions.

**Response:** `200 OK` — `AgentDTO[]`
```json
[{
  "name": "eng-01",
  "role": "engineer",
  "state": "working",
  "task": "Implementing auth middleware",
  "tool": "claude",
  "session": "eng-01",
  "session_id": "a1b2c3d4-...",
  "team": "backend",
  "parent_id": "root",
  "children": [],
  "created_at": "2026-03-20T10:00:00Z",
  "started_at": "2026-03-20T10:00:05Z",
  "updated_at": "2026-03-20T10:30:00Z"
}]
```

### `POST /api/agents`
Create and start a new agent.

**Body:**
```json
{
  "name": "eng-01",       // required, alphanumeric + hyphens/underscores
  "role": "engineer",     // required, must match a .bc/roles/*.md file
  "tool": "claude",       // optional, default from config
  "runtime": "docker",    // optional, "tmux" or "docker"
  "parent": "root"        // optional, parent agent name
}
```
**Response:** `201 Created` — `AgentDTO`

### `GET /api/agents/{name}`
Get single agent by name.

**Response:** `200 OK` — `AgentDTO` | `404`

### `DELETE /api/agents/{name}?force=true`
Delete agent. Must be stopped unless `force=true`.

**Query params:** `force` (boolean) — stop before deleting

**Response:** `204 No Content` | `400`

### `POST /api/agents/{name}/start`
Start a stopped agent. Resumes Claude session if valid UUID exists.

**Response:** `200 OK` — `AgentDTO`

### `POST /api/agents/{name}/stop`
Stop a running agent.

**Response:** `200 OK` — `{"status": "stopped"}`

### `POST /api/agents/{name}/send`
Send text to agent's tmux/Docker session.

**Body:** `{"message": "string"}`

**Response:** `200 OK` — `{"status": "sent"}`

### `POST /api/agents/{name}/hook`
Receive Claude Code hook event. Updates agent state (idle/working/etc).

**Body:** `{"event": "tool_use_start"}`

**Response:** `200 OK` — `{"ok": true}`

### `GET /api/agents/{name}/stats?limit=20`
Docker resource stats for agent.

**Query params:** `limit` (int, default 20)

**Response:** `200 OK` — `AgentStatsRecord[]`

### `POST /api/agents/{name}/rename`
**Body:** `{"new_name": "string"}`

**Response:** `200 OK` — `{"status": "renamed", "name": "new-name"}`

### `GET /api/agents/{name}/peek?lines=500`
Read recent terminal output.

**Query params:** `lines` (int, default 500)

**Response:** `200 OK` — `{"output": "string"}`

### `GET /api/agents/{name}/sessions`
List session history (current + archived).

**Response:** `200 OK` — `SessionEntry[]`

### `POST /api/agents/generate-name`
Generate a unique agent name.

**Response:** `200 OK` — `{"name": "witty-parrot"}`

### `POST /api/agents/broadcast`
Send message to all running agents.

**Body:** `{"message": "string"}`

**Response:** `200 OK` — `{"sent": 3}`

### `POST /api/agents/send-role`
Send to all agents with a specific role.

**Body:** `{"role": "engineer", "message": "string"}`

**Response:** `200 OK` — `SendResult`

### `POST /api/agents/send-pattern`
Send to agents matching glob pattern.

**Body:** `{"pattern": "eng-*", "message": "string"}`

**Response:** `200 OK` — `SendResult`

### `POST /api/agents/stop-all`
Stop all running agents.

**Response:** `200 OK` — `{"stopped": 5}`

---

## Channels

### `GET /api/channels`
List all channels.

**Response:** `200 OK` — `ChannelDTO[]`

### `POST /api/channels`
Create channel.

**Body:** `{"name": "reviews", "description": "Code review channel"}`

**Response:** `201 Created` — `ChannelDTO`

### `GET /api/channels/{name}`
**Response:** `200 OK` — `ChannelDTO`

### `PATCH /api/channels/{name}`
Update channel description.

**Body:** `{"description": "Updated description"}`

**Response:** `200 OK` — `ChannelDTO`

### `DELETE /api/channels/{name}`
**Response:** `204 No Content`

### `GET /api/channels/{name}/history?limit=50&offset=0`
Message history for a channel.

**Query params:** `limit` (int, default 50), `offset` (int, default 0)

**Response:** `200 OK` — `MessageDTO[]`

### `POST /api/channels/{name}/messages`
Post a message.

**Body:** `{"sender": "eng-01", "content": "PR ready for review"}`

**Response:** `201 Created` — `MessageDTO`

### `POST /api/channels/{name}/members`
Add member to channel.

**Body:** `{"agent_id": "eng-01"}`

**Response:** `204 No Content`

### `DELETE /api/channels/{name}/members?agent_id=eng-01`
Remove member.

**Response:** `204 No Content`

---

## Costs

### `GET /api/costs`
Workspace cost summary.

**Response:** `200 OK` — `CostSummary`

### `GET /api/costs/agents`
Cost breakdown by agent.

### `GET /api/costs/teams`
Cost breakdown by team.

### `GET /api/costs/models`
Cost breakdown by model.

### `GET /api/costs/daily?days=30`
Daily cost totals.

**Query params:** `days` (int, default 30)

### `POST /api/costs/sync`
Trigger JSONL cost import.

**Response:** `200 OK` — `{"imported": 42}`

---

## Secrets

Values are AES-256-GCM encrypted. API never returns secret values — only metadata.

### `GET /api/secrets`
List secret metadata.

### `POST /api/secrets`
**Body:** `{"name": "GITHUB_TOKEN", "value": "ghp_...", "description": "GitHub PAT"}`

**Response:** `201 Created` — `SecretMeta`

### `GET /api/secrets/{name}`
Get secret metadata (no value).

### `PUT /api/secrets/{name}`
Update secret.

**Body:** `{"value": "new-value", "description": "updated"}`

### `DELETE /api/secrets/{name}`
**Response:** `204 No Content`

---

## Cron

### `GET /api/cron`
List cron jobs.

### `POST /api/cron`
Create cron job.

**Body:**
```json
{
  "name": "nightly-lint",
  "schedule": "0 2 * * *",
  "agent": "eng-01",
  "prompt": "Run lint and fix issues",
  "enabled": true
}
```

### `GET /api/cron/{name}`
### `DELETE /api/cron/{name}`
### `POST /api/cron/{name}/enable`
### `POST /api/cron/{name}/disable`
### `POST /api/cron/{name}/run`
Manual trigger. Job must be enabled.

### `GET /api/cron/{name}/logs?last=20`
Execution logs.

---

## Daemons

Long-running processes managed by bcd.

### `GET /api/daemons`
### `POST /api/daemons`
**Body:** `{"name": "db", "cmd": "postgres", "runtime": "docker", "image": "postgres:17", "ports": ["5432:5432"]}`

### `GET /api/daemons/{name}`
### `POST /api/daemons/{name}/stop`
### `POST /api/daemons/{name}/restart`
### `DELETE /api/daemons/{name}`

---

## Tools

AI tool provider configurations.

### `GET /api/tools`
### `GET /api/tools/{name}`
### `PUT /api/tools/{name}`
### `DELETE /api/tools/{name}`
### `POST /api/tools/{name}/enable`
### `POST /api/tools/{name}/disable`

---

## MCP Servers

External MCP server configurations.

### `GET /api/mcp`
### `POST /api/mcp`
**Body:** `{"name": "playwright", "transport": "sse", "url": "http://localhost:3100/sse"}`

### `GET /api/mcp/{name}`
### `DELETE /api/mcp/{name}`
### `POST /api/mcp/{name}/enable`
### `POST /api/mcp/{name}/disable`

---

## Event Log

### `GET /api/logs?tail=100`
All events. **Warning:** unbounded without `tail` param.

### `GET /api/logs/{agent}`
Events for specific agent. **Warning:** unbounded.

---

## Workspace

### `GET /api/workspace`
Workspace status.

**Response:** `200 OK`
```json
{
  "name": "my-project",
  "root_dir": "/home/user/project",
  "agent_count": 5,
  "running_count": 3,
  "is_healthy": true
}
```

### `GET /api/workspace/status`
Alias for `GET /api/workspace`.

### `GET /api/workspace/roles`
All resolved roles (with BFS inheritance applied).

### `POST /api/workspace/up`
Start root agent. Optional body: `{"tool": "claude", "runtime": "docker"}`

### `POST /api/workspace/down`
Stop all agents.

---

## Doctor

### `GET /api/doctor`
Run all health checks.

### `GET /api/doctor/{category}`
Run specific category (workspace, agents, database, tools, etc.).

---

## SSE Events

### `GET /api/events`
Server-Sent Events stream. Sends JSON payloads:

```
data: {"type": "agent.created", "payload": {"name": "eng-01", "role": "engineer"}}
data: {"type": "agent.stopped", "payload": {"name": "eng-01", "reason": "user_request"}}
data: {"type": "channel.message", "payload": {"channel": "general", "message": {...}}}
```

Event types: `connected`, `agent.created`, `agent.started`, `agent.stopped`, `agent.deleted`, `agent.renamed`, `agents.stopped_all`, `channel.message`

---

## MCP Protocol

### `GET /mcp/sse`
MCP SSE transport — server-to-client events.

### `POST /mcp/message`
MCP JSON-RPC 2.0 — client-to-server requests. 4MB body limit.

See [docs/architecture/mcp.md](../architecture/mcp.md) for protocol details.
