# Backend Engineering Review

**Date:** 2026-03-20
**Repo:** gh-curious-otter/bc
**Stack:** Go 1.25.1, SQLite (mattn/go-sqlite3), PostgreSQL (pgx/v5), Cobra CLI, net/http stdlib, SSE, Docker, tmux
**Backend Maturity Score:** 5/10

## Executive Summary

bc is a well-structured Go CLI and daemon for AI agent orchestration. The codebase demonstrates strong Go conventions (proper error handling, clean package layout, comprehensive test suite at the pkg level). However, the HTTP daemon (`bcd`) has **zero authentication**, **no rate limiting**, **no request body size limits**, and **no handler-level tests** — making it unsuitable for any deployment beyond a trusted localhost. The data layer uses multiple separate SQLite databases with duplicated connection setup, missing indexes on hot query paths, and an unbounded `Read()` call that could return the entire event log. The RBAC permission system exists in code but is **never enforced** at the API layer.

## API Surface Map

| Method | Path | Auth | Validation | Rate Limited | Issues |
|--------|------|------|------------|--------------|--------|
| GET | /health | None | N/A | No | Shallow - no dependency checks |
| GET | /api/events | None | N/A | No | SSE - no auth on event stream |
| GET | /api/agents | None | Minimal | No | No pagination |
| POST | /api/agents | None | Minimal | No | No body size limit, weak validation |
| GET | /api/agents/{name} | None | Name only | No | - |
| POST | /api/agents/{name}/start | None | None | No | Can start any agent |
| POST | /api/agents/{name}/stop | None | None | No | Can stop any agent |
| POST | /api/agents/{name}/send | None | None | No | Can inject commands into agents |
| DELETE | /api/agents/{name} | None | None | No | Can delete any agent |
| POST | /api/agents/{name}/hook | None | Minimal | No | Can manipulate agent state |
| GET | /api/agents/{name}/stats | None | limit param | No | Unbounded if no limit |
| POST | /api/agents/{name}/rename | None | None | No | - |
| GET | /api/agents/{name}/peek | None | lines param | No | Can read agent output |
| GET | /api/agents/{name}/sessions | None | None | No | - |
| POST | /api/agents/generate-name | None | N/A | No | - |
| POST | /api/agents/broadcast | None | None | No | Can broadcast to all agents |
| POST | /api/agents/send-role | None | None | No | - |
| POST | /api/agents/send-pattern | None | None | No | Pattern not sanitized |
| POST | /api/agents/stop-all | None | None | No | Can stop all agents |
| GET | /api/channels | None | None | No | No pagination |
| POST | /api/channels | None | Minimal | No | No body size limit |
| GET | /api/channels/{name} | None | None | No | - |
| PATCH | /api/channels/{name} | None | None | No | - |
| DELETE | /api/channels/{name} | None | None | No | - |
| GET | /api/channels/{name}/history | None | limit/offset | No | limit not capped |
| POST | /api/channels/{name}/messages | None | Minimal | No | No body size limit |
| POST/DELETE | /api/channels/{name}/members | None | Minimal | No | - |
| GET | /api/costs | None | None | No | - |
| GET | /api/costs/agents | None | None | No | No pagination |
| GET | /api/costs/teams | None | None | No | No pagination |
| GET | /api/costs/models | None | None | No | No pagination |
| GET | /api/costs/daily | None | days param | No | days not capped |
| POST | /api/costs/sync | None | None | No | Can trigger import |
| GET | /api/secrets | None | None | No | **Lists all secret metadata** |
| POST | /api/secrets | None | Minimal | No | **Can create secrets** |
| GET | /api/secrets/{name} | None | None | No | - |
| PUT | /api/secrets/{name} | None | Minimal | No | **Can update secrets** |
| DELETE | /api/secrets/{name} | None | None | No | **Can delete secrets** |
| GET | /api/cron | None | None | No | No pagination |
| POST | /api/cron | None | Minimal | No | - |
| GET/DELETE | /api/cron/{name} | None | None | No | - |
| POST | /api/cron/{name}/enable | None | None | No | - |
| POST | /api/cron/{name}/disable | None | None | No | - |
| POST | /api/cron/{name}/run | None | None | No | Can trigger jobs |
| GET | /api/cron/{name}/logs | None | last param | No | last not capped |
| GET | /api/daemons | None | None | No | No pagination |
| POST | /api/daemons | None | Minimal | No | **Can run arbitrary commands** |
| GET | /api/daemons/{name} | None | None | No | - |
| POST | /api/daemons/{name}/stop | None | None | No | - |
| POST | /api/daemons/{name}/restart | None | None | No | - |
| DELETE | /api/daemons/{name} | None | None | No | - |
| GET | /api/tools | None | None | No | No pagination |
| GET/PUT/DELETE | /api/tools/{name} | None | Minimal | No | - |
| POST | /api/tools/{name}/enable | None | None | No | - |
| GET | /api/mcp | None | None | No | - |
| POST | /api/mcp | None | Minimal | No | - |
| GET/DELETE | /api/mcp/{name} | None | None | No | - |
| GET | /api/logs | None | tail param | No | **Unbounded without tail** |
| GET | /api/logs/{agent} | None | None | No | **Unbounded** |
| GET | /api/workspace | None | None | No | Leaks root_dir path |
| GET | /api/workspace/roles | None | None | No | - |
| POST | /api/workspace/up | None | None | No | Can start workspace |
| POST | /api/workspace/down | None | None | No | Can stop all agents |
| GET | /api/doctor | None | None | No | - |

## Data Model Overview

```mermaid
erDiagram
    channels ||--o{ channel_members : has
    channels ||--o{ messages : contains
    messages ||--o{ mentions : has
    messages ||--o{ reactions : has
    cost_records }o--|| cost_budgets : "tracked by"
    events ||--o{ agents : "logs for"
    secrets ||--o{ secret_meta : "encrypted with"
    cron_jobs ||--o{ cron_logs : "execution of"

    channels {
        int id PK
        text name UK
        text type
        text description
        text created_at
        text updated_at
    }
    messages {
        int id PK
        int channel_id FK
        text sender
        text content
        text type
        text metadata
        text created_at
    }
    cost_records {
        int id PK
        text agent_id
        text team_id
        text model
        int input_tokens
        int output_tokens
        real cost_usd
        text timestamp
    }
    events {
        int id PK
        text type
        text agent
        text message
        text data
        text timestamp
    }
    secrets {
        text name PK
        text value
        text description
        text created_at
        text updated_at
    }
```

## Critical Issues (production risks)

| # | Issue | File/Location | Category | Impact |
|---|-------|--------------|----------|--------|
| 1 | **Zero authentication on entire HTTP API** | `server/server.go:196` | security | Anyone on the network can control agents, read secrets, run commands |
| 2 | **No request body size limits** | All POST/PUT handlers | security | DoS via memory exhaustion with large payloads |
| 3 | **Daemon endpoint allows arbitrary command execution** | `server/handlers/daemons.go:54` | security | POST /api/daemons with any cmd= runs it |
| 4 | **Secret API has no auth — plaintext metadata accessible** | `server/handlers/secrets.go` | security | Anyone can list/create/delete secrets |
| 5 | **RBAC permissions defined but never enforced at API layer** | `pkg/agent/agent.go:182` vs handlers | security | Permission system is dead code for HTTP API |
| 6 | **Unbounded event log read** | `pkg/events/store_sqlite.go:72` | performance | `Read()` returns ALL events, no LIMIT |
| 7 | **CORS Access-Control-Allow-Origin: * with no auth** | `server/handlers/helpers.go:48` | security | CSRF from any origin if port exposed |
| 8 | **`--dangerously-skip-permissions` in default provider config** | `config.toml:27` | security | Claude runs without permission checks by default |

## Major Issues (quality & scalability)

| # | Issue | File/Location | Category | Impact |
|---|-------|--------------|----------|--------|
| 9 | **No handler tests** (0 files in server/handlers/) | `server/handlers/` | testing | All API routes untested |
| 10 | **Multiple separate SQLite databases** with duplicated setup | `pkg/channel/sqlite.go:76`, `pkg/cost/cost.go:137` | data-layer | 5+ separate DB files, duplicated pragma config |
| 11 | **No pagination on list endpoints** | All list handlers | api | Will break with 1000+ agents/channels/records |
| 12 | **No rate limiting on any endpoint** | `server/server.go` | api | DoS vector |
| 13 | **Health check is shallow** — doesn't check dependencies | `server/server.go:82` | infra | Returns "ok" even if DB/Docker/tmux are down |
| 14 | **No request ID middleware** | `server/server.go` | observability | Can't correlate requests to logs |
| 15 | **No panic recovery middleware** | `server/server.go` | error-handling | Unhandled panic crashes entire daemon |
| 16 | **Agent `send` endpoint injects raw text into tmux** | `pkg/agent/agent.go` | security | Could inject shell commands via tmux send-keys |
| 17 | **ToggleReaction is not atomic** (check-then-act race) | `pkg/channel/sqlite.go:825` | data-layer | Concurrent toggles can double-add |
| 18 | **Channel history limit not capped** | `server/handlers/channels.go:124` | performance | `?limit=999999999` returns entire history |
| 19 | **Docker network=host by default** | `config.toml:113` | security | Containers share host network stack |

## Minor Issues & Improvements

| # | Issue | File/Location | Category | Impact |
|---|-------|--------------|----------|--------|
| 20 | `contains()` reimplements `strings.Contains` | `server/handlers/workspace.go:123` | dx | Unnecessary custom code |
| 21 | `trimPrefix()` reimplements `strings.TrimPrefix` | `server/handlers/events.go:75` | dx | Unnecessary custom code |
| 22 | No structured logging with request context | `pkg/log/log.go` | observability | Hard to trace requests |
| 23 | Cost `days` parameter not capped | `server/handlers/costs.go:88` | performance | `?days=99999` scans full table |
| 24 | Agent stats `limit` not capped | `server/handlers/agents.go:202` | performance | `?limit=999999` unbounded |
| 25 | Missing `Content-Type` validation on POST requests | All POST handlers | api | Accepts any content type |
| 26 | No response compression (gzip) | `server/server.go` | performance | Large JSON responses uncompressed |
| 27 | FTS trigger errors silently ignored | `pkg/channel/sqlite.go:203` | data-layer | FTS can silently fall out of sync |
| 28 | SSE hub drops events for slow clients silently | `server/ws/hub.go:131` | error-handling | No backpressure notification |
| 29 | Workspace status leaks `root_dir` absolute path | `server/handlers/workspace.go:49` | security | Information disclosure |
| 30 | `up` handler silently ignores body decode errors | `server/handlers/workspace.go:84` | error-handling | Malformed JSON accepted |

## What's Done Well

1. **Clean package architecture** — cmd imports pkg, never vice versa. Clear separation between CLI, daemon, and reusable packages.
2. **Comprehensive pkg-level test suite** — 90+ test files covering agent, channel, cost, cron, events, workspace, etc. Includes benchmarks.
3. **Proper SQLite configuration** — WAL mode, busy timeouts, single-writer connection pools, foreign keys enabled.
4. **Good error wrapping** — Consistent use of `fmt.Errorf("context: %w", err)` throughout.
5. **Graceful shutdown** — Signal handling with 10s timeout, deferred resource cleanup in bcd main.
6. **Secret encryption** — AES-256-GCM with PBKDF2-SHA256 (600k iterations per OWASP 2023), proper salt/nonce generation.
7. **Agent name validation** — `IsValidAgentName()` prevents path traversal via agent names.
8. **Idempotent schema creation** — All `CREATE TABLE IF NOT EXISTS` patterns.
9. **SSE implementation** — Clean hub pattern with subscriber management and buffer overflow handling.
10. **Transaction usage** — Channel deletion properly wraps multi-table cleanup in a transaction.

## Architecture Diagram

```mermaid
graph TB
    CLI[bc CLI] -->|HTTP| Daemon[bcd Daemon :9374]
    TUI[React/Ink TUI] -->|HTTP + SSE| Daemon
    WebUI[Web UI] -->|HTTP + SSE| Daemon

    Daemon --> AgentSvc[Agent Service]
    Daemon --> ChannelSvc[Channel Service]
    Daemon --> CostStore[Cost Store]
    Daemon --> SecretStore[Secret Store]
    Daemon --> CronStore[Cron Store]
    Daemon --> EventLog[Event Log]
    Daemon --> MCPSrv[MCP Server]

    AgentSvc --> AgentMgr[Agent Manager]
    AgentMgr --> TmuxRT[Tmux Runtime]
    AgentMgr --> DockerRT[Docker Runtime]

    TmuxRT --> Tmux[tmux sessions]
    DockerRT --> Docker[Docker containers]

    ChannelSvc --> SQLite1[(bc.db)]
    CostStore --> SQLite2[(costs.db)]
    SecretStore --> SQLite3[(secrets.db)]
    EventLog --> SQLite4[(state.db)]
    CronStore --> SQLite5[(cron.db)]

    Docker --> Claude[Claude Code]
    Docker --> Gemini[Gemini]
    Docker --> Cursor[Cursor]
    Tmux --> Claude2[Claude Code]
```

## Scalability Assessment

**Current bottleneck: SQLite single-writer model**

- **10x scale (50 agents):** Likely fine. SQLite WAL handles concurrent reads well. The 30s busy timeout prevents lock contention failures.
- **100x scale (500 agents):** Will break. Multiple SQLite databases mean multiple lock contention points. The `RefreshState()` call on every GET /api/agents lists all agents and reconciles with tmux/Docker — O(n) subprocess calls. No caching layer.
- **1000x scale:** Need PostgreSQL (already supported in code), proper connection pooling, caching layer, and async state reconciliation.

**Fix first:**
1. Consolidate SQLite databases into single bc.db (partially done for channels)
2. Add caching for agent state (currently polls tmux/Docker on every request)
3. Add pagination to all list endpoints
4. Cap query parameters (limit, days, lines)

## Action Plan

### Phase 1: Security & Data Integrity (immediate)
- Add API authentication (API key or localhost-only enforcement)
- Add request body size limits (http.MaxBytesReader)
- Enforce RBAC permissions at handler level
- Remove `--dangerously-skip-permissions` from default config
- Add rate limiting on state-changing endpoints

### Phase 2: Error Handling & Resilience (week 1)
- Add panic recovery middleware
- Add request ID middleware
- Make health check verify downstream dependencies
- Cap all query parameters (limit, days, lines)
- Fix ToggleReaction atomicity

### Phase 3: Performance & Query Optimization (week 2)
- Add pagination to all list endpoints
- Cap unbounded queries (events Read, history limit)
- Add response compression
- Consolidate SQLite databases
- Add agent state caching

### Phase 4: Testing & CI (week 3)
- Add handler integration tests (httptest)
- Add API contract tests
- Measure and improve coverage on server/ package

### Phase 5: Observability & Operational Readiness (week 4)
- Add structured logging with request context
- Add request duration metrics
- Add deep health check with dependency verification
- Add request/response logging middleware

### Phase 6: Documentation & API Standards (week 5)
- Add OpenAPI spec
- Standardize error envelope
- Add API versioning strategy
- Document deployment security requirements
