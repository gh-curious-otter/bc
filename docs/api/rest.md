# REST API Reference

**Base URL:** `http://127.0.0.1:9374`
**Content-Type:** `application/json`
**Authentication:** None (localhost-only)

---

## Health

### `GET /health`
Liveness probe.

**Response:** `200 OK`
```json
{
  "status": "ok",
  "addr": "127.0.0.1:9374"
}
```

### `GET /health/ready`
Readiness probe. Checks DB connectivity and agent runtime.

**Response:** `200 OK`
```json
{
  "status": "ok",
  "checks": {
    "db": "ok",
    "agents": "23 total"
  }
}
```

> Returns `503 Service Unavailable` if status is `"degraded"`.

---

## Agents

### `GET /api/agents`
List all agents. Reconciles with live sessions before returning.

**Query params:**
| Param | Type | Description |
|-------|------|-------------|
| role | string | Filter by role |
| state | string | Filter: running, stopped, error, starting |
| team | string | Filter by team ID |

**Response:** `200 OK` — `AgentDTO[]`

### `POST /api/agents`
Create and start a new agent.

**Body:**
```json
{
  "name": "eng-01",
  "role": "engineer",
  "workspace": "~/repos/my-project",
  "tool": "claude",
  "runtime": "docker",
  "team": "backend-team"
}
```
- `name` — required, alphanumeric + hyphens/underscores
- `role` — required, must exist in `roles` table
- `workspace` — required if no team (inherits team workspace if omitted)
- `tool` — optional, default from settings
- `runtime` — optional, "tmux" or "docker"
- `team` — optional, adds agent to team

**Response:** `201 Created` — `AgentDTO`

### `GET /api/agents/{name}`
**Response:** `200 OK` — `AgentDTO` | `404`

### `DELETE /api/agents/{name}?force=true`
Delete agent. Cleans up Docker container, git worktree, and branch.

**Response:** `204 No Content`

### `POST /api/agents/{name}/start`
Start a stopped agent. Resumes Claude session if valid UUID exists.

**Response:** `200 OK` — `AgentDTO`

### `POST /api/agents/{name}/stop`
**Response:** `200 OK` — `{"status": "stopped"}`

### `POST /api/agents/{name}/send`
Send text to agent's tmux/Docker session.

**Body:** `{"message": "string"}`
**Response:** `200 OK` — `{"status": "sent"}`

### `POST /api/agents/{name}/hook`
Receive Claude Code hook event. Updates agent state.

**Body:** `{"event": "tool_use_start | tool_use_end | user_input_required | stop"}`
**Response:** `200 OK` — `{"ok": true}`

### `GET /api/agents/{name}/peek?lines=500`
Read recent terminal output via `tmux capture-pane`. Returns readable formatted output.

**Query params:** `lines` (int, default 500, max 10000)
**Response:** `200 OK` — `{"output": "string"}`

### `GET /api/agents/{name}/stats?limit=20`
Docker resource stats (CPU, memory, network).

**Response:** `200 OK` — `AgentStatsRecord[]`

### `POST /api/agents/{name}/rename`
**Body:** `{"new_name": "string"}`
**Response:** `200 OK`

### `GET /api/agents/{name}/sessions`
Session history (current + archived UUIDs with timestamps).

**Response:** `200 OK` — `SessionEntry[]`

### `GET /api/agents/generate-name`
**Response:** `200 OK` — `{"name": "witty-parrot"}`

### `POST /api/agents/broadcast`
Send to all running agents.

**Body:** `{"message": "string", "team": "optional-team-id"}`
- If `team` specified, sends only to agents in that team
**Response:** `200 OK` — `{"sent": 3}`

### `POST /api/agents/send-role`
Send to all agents with a specific role.

**Body:** `{"role": "engineer", "message": "string"}`
**Response:** `200 OK` — `SendResult`

### `POST /api/agents/stop-all`
**Response:** `200 OK` — `{"stopped": 5}`

---

## Teams

### `GET /api/teams`
List all teams as a flat list. Use `parent_id` to build tree client-side.

**Response:** `200 OK` — `TeamDTO[]`
```json
[{
  "id": "backend-team",
  "name": "Backend",
  "parent_id": "root-team",
  "workspace": "~/repos/api",
  "agents": ["eng-01", "eng-02"],
  "children": ["db-team"],
  "created_at": 1711000000000
}]
```

### `POST /api/teams`
**Body:** `{"id": "backend", "name": "Backend Team", "parent_id": "root", "workspace": "~/repos/api"}`
**Response:** `201 Created` — `TeamDTO`

### `GET /api/teams/{id}`
### `PUT /api/teams/{id}`
### `DELETE /api/teams/{id}`
Deleting a team does NOT delete its agents.

### `POST /api/teams/{id}/members`
Add agent to team.

**Body:** `{"agent_id": "eng-01"}`
**Response:** `204 No Content`

### `DELETE /api/teams/{id}/members?agent_id=eng-01`
Remove agent from team.

---

## Roles

Roles are stored in the database. No markdown files on disk.

### `GET /api/roles`
List all roles (metadata only, no prompt bodies).

### `POST /api/roles`
Create role.

**Body:**
```json
{
  "name": "engineer",
  "description": "Implements features and fixes bugs",
  "prompt": "You are a senior engineer...",
  "settings": {"model": "opus"},
  "commands": {"lint": "Run linting on the codebase"},
  "mcp_servers": ["playwright", "github"],
  "secrets": ["GITHUB_TOKEN"]
}
```
**Response:** `201 Created`

### `GET /api/roles/{id}`
Full role including prompt body and settings.

### `PUT /api/roles/{id}`
Update role.

### `DELETE /api/roles/{id}`
Delete role. Agents keep their current config.

---

## Channels

### `GET /api/channels`
### `POST /api/channels`
**Body:** `{"name": "reviews", "description": "Code review channel"}`

### `GET /api/channels/{name}`
### `PATCH /api/channels/{name}`
### `DELETE /api/channels/{name}`

### `GET /api/channels/{name}/history?limit=50&offset=0`
**Query params:** `limit` (max 1000), `offset`

### `POST /api/channels/{name}/messages`
**Body:** `{"sender": "eng-01", "content": "PR ready for review"}`
Triggers delivery to all channel members via `tmux send-keys`.

### `POST /api/channels/{name}/members`
**Body:** `{"agent_id": "eng-01"}`

### `DELETE /api/channels/{name}/members?agent_id=eng-01`

---

## Costs

### `GET /api/costs`
Workspace cost summary with token breakdown.

**Response:**
```json
{
  "total_cost_usd": 12.50,
  "input_tokens": 500000,
  "output_tokens": 150000,
  "cache_read_tokens": 300000,
  "cache_creation_tokens": 50000,
  "request_count": 250,
  "period": "all_time"
}
```

### `GET /api/costs/agents`
Per-agent cost breakdown with token details.

### `GET /api/costs/teams`
Per-team cost aggregation.

### `GET /api/costs/models`
Per-model cost breakdown.

### `GET /api/costs/daily?days=30`
Daily cost time series (for graphs).

**Response:** `200 OK`
```json
[
  {"date": "2026-03-20", "cost_usd": 2.50, "input_tokens": 100000, "output_tokens": 30000, "requests": 45},
  {"date": "2026-03-21", "cost_usd": 3.10, "input_tokens": 120000, "output_tokens": 35000, "requests": 52}
]
```

### `GET /api/costs/agent/{name}?days=7`
Single agent cost time series.

### `POST /api/costs/sync`
Trigger JSONL cost import from Claude session files.

---

## Secrets

Values are AES-256-GCM encrypted. API never returns values.

### `GET /api/secrets`
### `POST /api/secrets`
**Body:** `{"name": "GITHUB_TOKEN", "value": "ghp_...", "description": "GitHub PAT"}`

### `GET /api/secrets/{name}`
Metadata only (no value).

### `PUT /api/secrets/{name}`
### `DELETE /api/secrets/{name}`

---

## Cron

Scheduled bash commands that run on a timer. To prompt an agent, use a cron job that curls the agent send API.

### `GET /api/cron`
### `POST /api/cron`
**Body:**
```json
{
  "name": "nightly-lint",
  "schedule": "0 2 * * *",
  "command": "cd ~/repos/api && make lint",
  "enabled": true
}
```

### `GET /api/cron/{name}`
### `DELETE /api/cron/{name}`
### `POST /api/cron/{name}/enable`
### `POST /api/cron/{name}/disable`
### `POST /api/cron/{name}/run`
Manual trigger.

### `GET /api/cron/{name}/logs?last=20`

---

## Daemons

Long-running processes managed by bcd.

### `GET /api/daemons`
### `POST /api/daemons`
**Body:** `{"name": "db", "cmd": "postgres", "runtime": "docker", "image": "postgres:17", "ports": ["5432:5432"]}`

### `GET /api/daemons/{name}`
### `POST /api/daemons/{name}/start`
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

External MCP server configurations for agents.

### `GET /api/mcp`
### `POST /api/mcp`
**Body:**
```json
{
  "name": "playwright",
  "transport": "sse",
  "url": "http://localhost:3100/sse",
  "env": {"BROWSER": "chromium"},
  "enabled": true
}
```

### `GET /api/mcp/{name}`
### `DELETE /api/mcp/{name}`
### `POST /api/mcp/{name}/enable`
### `POST /api/mcp/{name}/disable`

---

## Event Log

### `GET /api/logs?tail=100`
Recent events. Default: last 100.

**Query params:** `tail` (int, default 100, max 10000), `type` (filter by event type)

### `GET /api/logs/{agent}`
Events for specific agent. Same params.

---

## Doctor

### `GET /api/doctor`
Run all health checks.

### `GET /api/doctor/{category}`

---

## SSE Events

### `GET /api/events`
Server-Sent Events stream.

**Event types:**

| Type | Payload | When |
|------|---------|------|
| `connected` | `{}` | Client connects |
| `agent.created` | `{name, role, tool}` | Agent created |
| `agent.started` | `{name, session_id}` | Agent started/restarted |
| `agent.stopped` | `{name, reason}` | Agent stopped |
| `agent.deleted` | `{name}` | Agent deleted |
| `agent.state_changed` | `{name, state, task}` | State transition (idle/working/stuck) |
| `channel.message` | `{channel, sender, content, type}` | New message |
| `cost.updated` | `{agent, cost_usd, tokens}` | Cost import completed |
| `team.updated` | `{team_id, action}` | Team membership changed |

---

## MCP Protocol

### `GET /mcp/sse`
MCP SSE transport — server-to-client events.

### `POST /mcp/message`
MCP JSON-RPC 2.0 — client-to-server. 4MB body limit.

See [architecture/mcp.md](../mcp.md) for resources, tools, and notifications.
