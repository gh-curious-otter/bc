# Proposal: Channels Revamp — Notification Gateway

**Status:** Proposal  
**Author:** zen-zebra (root agent)  
**Date:** 2026-04-09

## TL;DR

Replace the over-engineered internal messaging/chat system with a **simple notification gateway**. Channels become a one-way pipe: external app events → subscribed agents via tmux send-keys. Agents respond using the app's own MCP tools. Delete ~3,500 lines of chat infrastructure (reactions, FTS, approvals, mentions, message types). Add ~800 lines of clean gateway + notification dispatch code.

## Problem

The current channel system was built as a full messaging platform with:
- SQLite/Postgres message store with FTS5 full-text search
- Reactions system (add/remove/toggle with emoji)
- Approval automation (detect "LGTM", auto-create merge requests)
- Review request parsing
- Mention system (@mentions, @all, resolution, highlighting)
- 6 message types with content-based inference
- 14 CLI commands
- Chat UI with message composer, member panel, chat rooms
- Dual backend (SQLite + Postgres)

**None of this is needed.** Channels should just deliver notifications from external apps to agents.

### Issues Found (from 42+ conversations, 4 agent histories)

**Architecture:**
- Fundamental mismatch: built as messaging system, used as notification pipe
- Two parallel communication systems (bc send vs bc channel send) with no guidance
- MCP SSE delivery attempted and abandoned — caused agent hangs
- OnMessage hook too limited (no type/metadata access, can't reject messages)

**Bugs:**
- Missing `Close()` on Store — resource leaks
- Silent error swallowing in 7+ `LastInsertId`/`RowsAffected`/`time.Parse` calls
- Reaction methods incompatible with SQLite backend
- Duplicate Slack events from Socket Mode redelivery
- Agent identity loss (`?agent=` query param dropped from MCP URL)
- MCP route conflict (`/mcp` prefix intercepted browser requests)
- `database is closed` SQLite errors under load
- FTS availability flag set inconsistently

**Dead Code:**
- Message type system (223 lines) — inferred but never displayed
- Approval automation (220 lines) — rarely used
- Query system (199 lines) — advanced search never exposed in CLI/TUI
- `Store.Load()`/`Store.Save()` — no-ops for SQLite
- `MigrateFromJSON()` — completed migration, dead code
- `mentions` table — fully implemented, never called from service layer
- `last_read_msg_id` — in schema, never written

**UX:**
- `bc channel send` produces 12+ lines of output for a 10-member channel
- Default channels in help text don't match what's actually created
- Agents have zero awareness of channels (no prompt references)

## Design

### Core Principle

```
External App (Slack/Telegram/Discord/GitHub)
    ↓ adapter receives event
    ↓ filter by agent subscriptions  
    ↓ tmux send-keys (JSON) to subscribed agents
Done.
```

No reactions. No file sharing. No message store. No FTS. No approval automation. Agent receives notification, decides what to do, uses the app's MCP tools to respond.

### Package Structure

```
DELETE:  pkg/channel/           (all 20 files — chat store, reactions, FTS, mentions)
KEEP:   pkg/gateway/            (adapter interface refined, adapters kept)
NEW:    pkg/notify/             (subscriptions, delivery dispatch, activity log)
```

### New Interfaces

#### pkg/gateway/gateway.go (simplified)

```go
// Adapter connects to an external platform and routes inbound events.
// Send/file capabilities are NOT in this interface — agents use the
// platform's own MCP tools to respond.
type Adapter interface {
    Name() string
    Start(ctx context.Context, onMessage func(InboundMessage)) error
    Stop(ctx context.Context) error
    Channels(ctx context.Context) ([]ExternalChannel, error)
    Health(ctx context.Context) error
}

type InboundMessage struct {
    Timestamp   time.Time
    ChannelID   string   // platform-native ID ("C0123")
    ChannelName string   // human-readable ("engineering")
    Platform    string   // "slack" | "telegram" | "discord" | "github" | "gmail"
    Sender      string
    SenderID    string
    Content     string
    MessageID   string
}
```

Key change: `Send()` removed from Adapter interface. Agents use MCP.

#### pkg/notify/ (new package)

```go
// Notification is the JSON payload sent to agents via tmux send-keys.
type Notification struct {
    Timestamp string `json:"timestamp"`
    Channel   string `json:"channel"`    // "slack:engineering"
    Platform  string `json:"platform"`
    Sender    string `json:"sender"`
    Content   string `json:"content"`
    MessageID string `json:"message_id,omitempty"`
}

// DeliveryEntry records one delivery attempt.
type DeliveryEntry struct {
    Timestamp time.Time      `json:"timestamp"`
    Channel   string         `json:"channel"`
    Agent     string         `json:"agent"`
    Status    DeliveryStatus `json:"status"`   // delivered | failed | pending
    Error     string         `json:"error,omitempty"`
    Preview   string         `json:"preview"`  // first 120 chars
}

// Service dispatches notifications to subscribed agents.
type Service struct {
    store  *Store
    agents AgentSender  // interface: SendToAgent(ctx, name, msg) error
}
```

### SQLite Schema (minimal — 3 tables)

```sql
CREATE TABLE IF NOT EXISTS subscriptions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    channel    TEXT NOT NULL,       -- "slack:engineering"
    agent      TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(channel, agent)
);

CREATE TABLE IF NOT EXISTS delivery_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    logged_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    channel   TEXT NOT NULL,
    agent     TEXT NOT NULL,
    status    TEXT NOT NULL CHECK(status IN ('delivered', 'failed', 'pending')),
    error     TEXT,
    preview   TEXT
);

CREATE TABLE IF NOT EXISTS gateways (
    name         TEXT PRIMARY KEY,
    enabled      INTEGER NOT NULL DEFAULT 0,
    connected    INTEGER NOT NULL DEFAULT 0,
    last_seen_at TEXT,
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
```

No message content tables. No reactions. No FTS. No mentions. Delivery log is pruned to last 1000 entries per channel.

### Data Flow

```
Slack/Telegram/Discord
    ↓ WebSocket/polling
pkg/gateway/{platform}/adapter
    ↓ onMessage(InboundMessage)
pkg/gateway/manager
    ↓ dispatch(InboundMessage)
pkg/notify/service.Dispatch()      [goroutine — non-blocking]
    ├─ store.Subscribers(channel)  [SQLite read]
    ├─ agents.SendToAgent()        [tmux send-keys per subscriber]
    ├─ store.LogDelivery()         [SQLite write]
    └─ hub.Publish("gateway.notification")  [SSE to web UI]
```

### REST API

```
GET    /api/gateways                                    — list gateways + status
POST   /api/gateways                                    — enable gateway
DELETE /api/gateways/{name}                              — disable gateway
GET    /api/gateways/{name}/channels                     — list channels (auto-discovered)
GET    /api/gateways/{name}/channels/{channel}/activity  — delivery log
POST   /api/gateways/{name}/channels/{channel}/subscribe — subscribe agent
DELETE /api/gateways/{name}/channels/{channel}/subscribe/{agent} — unsubscribe
GET    /api/gateways/{name}/health                       — live health check
```

Removed: all `/api/channels/*` endpoints (15 endpoints → 8 new ones).

### CLI Commands (14 → 4)

```
bc channel list         — all channels across gateways
bc channel subscribe    — subscribe agent to channel
bc channel unsubscribe  — unsubscribe agent
bc channel status       — gateway connection status
```

Removed: `create`, `delete`, `send`, `add`, `remove`, `join`, `leave`, `history`, `react`, `show`, `desc`, `edit`.

### Web UI

```
/channels                     → GatewayList (cards for each supported gateway)
/channels/:gateway            → ChannelList for that gateway
/channels/:gateway/:channel   → ActivityFeed + SubscriptionPanel
```

**GatewayList:** Cards for Slack, Telegram, Discord, GitHub, Gmail. Each shows connection status (green/red dot), channel count, "Setup" button.

**ChannelList:** Sidebar of discovered channels for one gateway.

**ActivityFeed:** Delivery log entries (timestamp, agent, status badge, preview). NOT a chat — shows delivery confirmations. Polls every 5s + live WebSocket updates.

**SubscriptionPanel:** Agent list with online dots, subscribe/unsubscribe buttons.

**SetupWizard:** Modal for connecting a new gateway. Shows docs link, token input fields, stores tokens via `/api/secrets`.

### Token Management

Tokens stored in `pkg/secret` (AES-256-GCM encrypted), NOT in settings.json.

Convention: `GATEWAY_SLACK_BOT_TOKEN`, `GATEWAY_TELEGRAM_BOT_TOKEN`, etc.

Setup flow: SetupWizard → store token via secrets API → enable gateway via gateways API.

### Supported Gateways

| Gateway | Status | Adapter Exists | Inbound Events |
|---------|--------|---------------|----------------|
| **Slack** | Ready | Yes | Messages, mentions, thread replies |
| **Telegram** | Ready | Yes | Messages to bot |
| **Discord** | Ready | Yes | Messages, mentions |
| **GitHub** | Planned | No | PR comments, review requests, issue assignments |
| **Gmail** | Planned | No | Incoming emails |

GitHub and Gmail show in UI as "not configured" with setup docs link.

## Build Sequence

| Phase | Scope | Depends On |
|-------|-------|-----------|
| 1 | `pkg/notify/` — types, store, service, tests | Nothing |
| 2 | `pkg/gateway/` — remove Send from interface, update manager | Phase 1 |
| 3 | Server handlers — rewrite gateways.go, remove channel handlers | Phase 1+2 |
| 4 | Workspace config — remove gateway config structs | Phase 3 |
| 5 | Delete `pkg/channel/` — remove all 20 files | Phase 3+4 |
| 6 | CLI — rewrite channel.go (14→4 commands), new client | Phase 3 |
| 7 | Frontend — new components, delete old ones | Phase 3 |
| 8 | Integration testing — smoke test with Slack adapter | Phase 1-7 |

## What Gets Deleted

| Component | Files | Lines (approx) |
|-----------|-------|----------------|
| pkg/channel/ core | 9 source files | ~2,500 |
| pkg/channel/ tests | 11 test files | ~2,500 |
| server/handlers/channels.go | 1 file | ~230 |
| server/handlers/channel_stats.go | 1 file | ~50 |
| server/handlers/stats_channels.go | 1 file | ~80 |
| MCP channel tools | 4 tool functions | ~200 |
| CLI channel commands | 10 commands removed | ~750 |
| pkg/client/channels.go | 1 file | ~100 |
| Frontend channel components | 8 files | ~1,200 |
| **Total deleted** | **~45 files** | **~7,600 lines** |

## What Gets Created

| Component | Files | Lines (approx) |
|-----------|-------|----------------|
| pkg/notify/ | 5 files (types, store, service, schema, tests) | ~500 |
| server/handlers/gateways.go | 1 file (rewrite) | ~200 |
| pkg/client/gateways.go | 1 file | ~80 |
| Frontend components | 6 files | ~600 |
| **Total created** | **~13 files** | **~1,380 lines** |

**Net reduction: ~6,200 lines of code.**

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Breaking existing agent workflows that use `send_message` MCP tool | Agents already use Slack/Telegram MCP for responses. The bc `send_message` was rarely used by agents for external comms. |
| Loss of message history | Messages are not bc's responsibility. Each platform keeps its own history. Agents can use MCP `read_channel` on the platform directly. |
| Gateway adapter reconnection | Each adapter owns its own reconnection with exponential backoff (already implemented in Slack adapter). |
| Delivery failures not visible | DeliveryLog + ActivityFeed make failures visible in real-time. |
| Token security | Already solved — `pkg/secret` with AES-256-GCM encryption. |

## Open Questions

1. **Should we keep `bc send <agent> <message>` for direct agent-to-agent messaging?** This is separate from channels but currently shares some infrastructure.
2. **GitHub adapter priority** — should we implement the GitHub gateway adapter in this phase or defer?
3. **TUI (Ink) channel view** — the TUI has a channels tab. Should we update it in this phase or deprecate it in favor of the web UI?
4. **Migration path** — existing workspaces have channel data in SQLite. Should we provide a migration script or just clean-slate?
