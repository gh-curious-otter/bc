# Proposal: Channels Revamp — Notification Gateway

**Status:** Proposal (v2)
**Author:** zen-zebra (root agent)
**Date:** 2026-04-09
**Issue:** #2947

---

## What Are Channels?

Channels are how agents stay connected to the outside world. When someone messages you on Slack, comments on a GitHub PR, or sends a Telegram message to your bot — that event flows through a channel and lands in the right agent's terminal as a structured notification.

**Channels are a notification gateway — not a chat system.**

An agent receives a notification and decides what to do. If it wants to reply, post a file, or react — it uses that app's own MCP tools directly. bc doesn't reinvent Slack or Telegram; it bridges them to agents.

### What You See

**Channels page** — a chatroom-style view showing live activity from connected apps:

```
┌─────────────────┬──────────────────────────────────────────────┐
│ GATEWAYS        │  #engineering (Slack)                         │
│                 │                                              │
│ ▼ Slack     (3) │  [10:32] @alice: Can someone review PR #428? │
│   #engineering  │  [10:33] @bob: On it, looking now            │
│   #all-bc       │  → delivered to eng-01, eng-02               │
│   #infra        │  [10:35] @alice: [shared screenshot.png]     │
│                 │  → delivered to eng-01, eng-02               │
│ ▼ Telegram  (1) │                                              │
│   bc-dev        │  ┌─ Subscribed Agents ──────────────────┐    │
│                 │  │ ● eng-01 (engineer)    [Unsubscribe] │    │
│ ▼ Discord   (1) │  │ ● eng-02 (engineer)    [Unsubscribe] │    │
│   #general      │  │ ○ lead-01 (tech-lead)  [Subscribe]   │    │
│                 │  │ ○ root (manager)        [Subscribe]   │    │
│ + Connect app   │  └──────────────────────────────────────┘    │
└─────────────────┴──────────────────────────────────────────────┘
```

- **Left sidebar**: Gateway dropdowns (Slack, Telegram, Discord) each listing their channels
- **Main area**: Chatroom-style activity feed with rich metadata — sender, timestamp, content, delivery status
- **Right panel**: Subscribed agents with online indicators (green dot = running)
- **No channels?**: "Connect app" button with platform-specific setup instructions

### What Agents Experience

An agent subscribed to `slack:engineering` receives this in its tmux session:

```json
{
  "timestamp": "2026-04-09T10:32:15Z",
  "channel": "slack:engineering",
  "platform": "slack",
  "sender": "alice",
  "content": "Can someone review PR #428?",
  "message_id": "1712657535.000200",
  "attachments": []
}
```

The agent reads this, decides to act, and uses `mcp__slack__post_message` to respond. bc never touches the response.

### File & Image Handling

When someone shares a file on Slack/Telegram/Discord:
- **Inbound**: The adapter receives the event. For files under 10MB, the adapter downloads and stores it in `.bc/attachments/<hash>`. The notification includes an `attachments` array with `filename`, `mime_type`, `size`, and `local_path`. The agent can read the file from that path.
- **Outbound**: Agents use the platform's MCP tools to send files (e.g., `mcp__slack__files_upload`). bc is not involved.

```json
{
  "timestamp": "...",
  "channel": "slack:engineering",
  "sender": "alice",
  "content": "[shared a file]",
  "attachments": [
    {
      "filename": "screenshot.png",
      "mime_type": "image/png",
      "size": 245760,
      "url": "https://files.slack.com/...",
      "local_path": ".bc/attachments/a1b2c3d4.png"
    }
  ]
}
```

For Docker agents: `.bc/attachments/` is mounted as a shared volume, so files are accessible across containers.

---

## Problem With Current System

The current channel system (~7,600 lines) was built as a full messaging platform:
- SQLite/Postgres message store with FTS5 search, reactions, approval automation, mention parsing, 6 message types, 14 CLI commands, chat UI with composer
- 26+ bugs documented: resource leaks, silent errors, SQLite "database is closed", broken reactions, agent identity loss
- ~3,500 lines of dead code (message types never displayed, automation rarely used, query system never exposed)
- Tokens stored in plaintext in settings.json causing 10 documented security/config problems

See [#2947](https://github.com/gh-curious-otter/bc/issues/2947) for the full issue list.

---

## Gateway Adapter Design

### Current State (from code exploration)

Three adapters exist today with different connection models:

| | Slack | Telegram | Discord |
|---|---|---|---|
| Transport | WebSocket (Socket Mode) | HTTP long-polling | WebSocket (Gateway) |
| Tokens needed | 2 (bot + app) | 1 (bot) | 1 (bot) |
| Channel discovery | API call at startup | Lazy (first message) | Event-driven (READY) |
| Event model | Pull from channel | Pull from channel | Push via callbacks |
| File support | Yes (FileSender) | No | No |
| Reconnection | Library-managed | Library-managed | Library-managed |
| Rate limiting | None | None | None |

**Common patterns** across all 3:
- `Start()` blocks until context cancellation (manager runs each in a goroutine)
- Bot self-message filtering
- `Health()` is a nil-check (not a live probe)
- `Channels()` reads from in-memory map
- `onMessage` callback pattern
- Sender formatted into message body for outbound (platform can't post-as-user)

**What's missing**:
- No reconnection signaling (manager can't tell if adapter lost connection)
- No live health checks (just nil-check on client struct)
- `InboundMessage.Attachments` field exists but is never populated
- No rate limiting (Slack allows 1 msg/sec/channel, Discord 5/5sec, Telegram 30/sec global)
- `Timestamp` not set by Slack adapter (zero value)

### New Adapter Interface

The interface stays close to what works today but adds what's missing:

```go
// Adapter connects to an external platform and routes inbound events to agents.
// Outbound messaging is NOT part of this interface — agents use the platform's
// own MCP tools to respond.
type Adapter interface {
    // Identity
    Name() string                    // "slack", "telegram", "discord"

    // Lifecycle — Start blocks until ctx is cancelled
    Start(ctx context.Context, handler EventHandler) error
    Stop(ctx context.Context) error

    // Discovery — returns channels the bot can see
    Channels(ctx context.Context) ([]ExternalChannel, error)

    // Health — MUST be a live probe (API call), not a nil-check
    Health(ctx context.Context) error

    // Status — connection state for UI display
    Status() AdapterStatus
}

// EventHandler is called by the adapter when an event arrives.
// Implementations must be safe for concurrent calls.
type EventHandler interface {
    OnMessage(msg InboundMessage)
    OnFile(msg InboundMessage, att Attachment)  // called for file/image shares
}

// AdapterStatus reports connection state for the web UI.
type AdapterStatus struct {
    Connected     bool
    LastMessageAt time.Time
    Error         string       // last error, if disconnected
}

// InboundMessage is the normalized event envelope.
type InboundMessage struct {
    Timestamp   time.Time
    Platform    string        // "slack" | "telegram" | "discord" | "github"
    ChannelID   string        // platform-native ID
    ChannelName string        // human-readable name
    Sender      string        // resolved display name
    SenderID    string        // platform-native user ID
    Content     string        // text content (or "[shared a file]")
    MessageID   string        // platform-native message ID
}

// Attachment describes a file shared on a channel.
type Attachment struct {
    Filename string
    MimeType string
    Size     int64
    URL      string  // platform download URL
}

// ExternalChannel describes a discoverable channel on a platform.
type ExternalChannel struct {
    ID   string
    Name string
    Type string  // "channel", "group", "dm"
}
```

**Key changes from current interface:**
- `Send()` removed — agents use MCP
- `FileSender` removed — agents use MCP
- `EventHandler` interface replaces raw callback (cleaner, supports file events)
- `Status()` added for connection state reporting
- `Health()` contract: must make a live API call
- `Attachment` type for inbound file handling

### Adding New Gateways

To add a new gateway (e.g., GitHub), implement the `Adapter` interface:

```go
type GitHubAdapter struct { ... }
func (a *GitHubAdapter) Name() string { return "github" }
func (a *GitHubAdapter) Start(ctx context.Context, h gateway.EventHandler) error {
    // Listen for webhooks or poll GitHub API
    // Call h.OnMessage() for PR comments, review requests, etc.
}
// ... implement remaining methods
```

The adapter handles its own:
- Connection mode (polling, WebSocket, webhook)
- Reconnection and backoff
- Rate limiting per platform rules
- User/channel name resolution
- Bot self-filtering

---

## Storage Design

### Use the Shared DB (pkg/db)

The codebase uses a shared singleton pattern for database access. All packages call `db.SharedWrapped()` or `db.Shared()` — they do NOT open their own connections.

Pattern (from `pkg/events`, `pkg/cost`, `pkg/cron`):

```go
func OpenStore(workspacePath string) (*Store, error) {
    driver := db.SharedDriver()  // "sqlite" or "timescale"
    if driver == "timescale" {
        pg := NewPostgresStore(db.Shared())
        _ = pg.InitSchema()
        return &Store{pg: pg}, nil
    }
    d := db.SharedWrapped()
    // CREATE TABLE IF NOT EXISTS ...
    return &Store{db: d}, nil
}

// Close is a no-op — the shared DB is owned by the caller (bcd main.go)
func (s *Store) Close() error { return nil }
```

`pkg/notify` will follow this exact pattern. No separate SQLite file.

### Schema

Tables are added to the shared `bc.db` (SQLite) or the shared TimescaleDB:

```sql
-- Agent subscriptions to gateway channels
CREATE TABLE IF NOT EXISTS notify_subscriptions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    channel    TEXT NOT NULL,       -- "slack:engineering"
    agent      TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(channel, agent)
);

-- Rolling delivery log (pruned to last 1000 per channel)
CREATE TABLE IF NOT EXISTS notify_delivery_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    logged_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    channel   TEXT NOT NULL,
    agent     TEXT NOT NULL,
    status    TEXT NOT NULL CHECK(status IN ('delivered', 'failed', 'pending')),
    error     TEXT,
    preview   TEXT
);

-- Gateway registry (connection state, enabled/disabled)
CREATE TABLE IF NOT EXISTS notify_gateways (
    name         TEXT PRIMARY KEY,
    enabled      INTEGER NOT NULL DEFAULT 0,
    connected    INTEGER NOT NULL DEFAULT 0,
    last_seen_at TEXT,
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_notify_subs_channel ON notify_subscriptions(channel);
CREATE INDEX IF NOT EXISTS idx_notify_subs_agent ON notify_subscriptions(agent);
CREATE INDEX IF NOT EXISTS idx_notify_delivery_channel ON notify_delivery_log(channel, id DESC);
```

Postgres variants use `BIGSERIAL`, `TIMESTAMPTZ`, `$1` placeholders.

Tables are prefixed with `notify_` to avoid collision with existing `channels` tables during migration.

### What's NOT Stored

- Message content (platforms keep their own history)
- Reactions, mentions, approvals, message types
- FTS indexes
- File content (stored in `.bc/attachments/`, not DB)

---

## Token & Secret Management

Tokens are stored in `pkg/secret` (AES-256-GCM encrypted), **never in settings.json**.

| Gateway | Secrets Required |
|---------|-----------------|
| Slack | `GATEWAY_SLACK_BOT_TOKEN`, `GATEWAY_SLACK_APP_TOKEN` |
| Telegram | `GATEWAY_TELEGRAM_BOT_TOKEN` |
| Discord | `GATEWAY_DISCORD_BOT_TOKEN` |
| GitHub | `GATEWAY_GITHUB_TOKEN` |

**Setup flow:**
1. User clicks "Connect Slack" in web UI
2. SetupWizard shows platform-specific instructions (create bot, get tokens, invite to channels)
3. User enters tokens → stored via `POST /api/secrets`
4. Gateway enabled via `POST /api/gateways`
5. Adapter starts, discovers channels, shows in sidebar

No manual file editing. No settings.json. No plaintext tokens.

---

## REST API

Consistent naming: `/api/gateways/{gateway}/channels/{channel}/...`

```
GET    /api/gateways                                           — list all gateways + status
POST   /api/gateways                                           — enable gateway
DELETE /api/gateways/{gateway}                                  — disable gateway
GET    /api/gateways/{gateway}/health                           — live health check
GET    /api/gateways/{gateway}/channels                         — list discovered channels
GET    /api/gateways/{gateway}/channels/{channel}               — channel detail + subscriptions
GET    /api/gateways/{gateway}/channels/{channel}/activity      — delivery log (paginated)
POST   /api/gateways/{gateway}/channels/{channel}/subscribe     — subscribe agent
DELETE /api/gateways/{gateway}/channels/{channel}/subscribe/{agent} — unsubscribe
```

### DTOs

```go
type GatewayDTO struct {
    Name         string         `json:"name"`
    Label        string         `json:"label"`          // "Slack"
    Connected    bool           `json:"connected"`
    Enabled      bool           `json:"enabled"`
    ChannelCount int            `json:"channel_count"`
    SetupDocsURL string         `json:"setup_docs_url"`
    LastSeenAt   *time.Time     `json:"last_seen_at,omitempty"`
    Error        string         `json:"error,omitempty"`
}

type ChannelDTO struct {
    ID            string         `json:"id"`
    Name          string         `json:"name"`
    ChannelKey    string         `json:"channel_key"`    // "slack:engineering"
    Type          string         `json:"type"`
    Subscribers   []SubscriberDTO `json:"subscribers"`
}

type SubscriberDTO struct {
    Agent   string `json:"agent"`
    Online  bool   `json:"online"`
    Role    string `json:"role"`
}

type DeliveryEntryDTO struct {
    Timestamp time.Time `json:"timestamp"`
    Channel   string    `json:"channel"`
    Agent     string    `json:"agent"`
    Status    string    `json:"status"`
    Error     string    `json:"error,omitempty"`
    Preview   string    `json:"preview"`
}
```

---

## Web UI

### Route Structure

```
/channels                                → GatewayList (when no gateways connected)
/channels                                → Sidebar + first channel (when gateways exist)
/channels/:gateway/:channel              → Activity feed for that channel
```

Frontend routes mirror API: `/channels/slack/engineering` maps to `/api/gateways/slack/channels/engineering`.

### Component Hierarchy

```
Channels.tsx (top-level view)
├── GatewaySidebar.tsx
│   ├── GatewayDropdown.tsx (one per connected gateway)
│   │   └── ChannelItem.tsx (click to select)
│   └── ConnectButton.tsx ("+ Connect app")
├── ChannelView.tsx (main area)
│   ├── ChannelHeader.tsx (channel name, gateway icon, connection status)
│   ├── ActivityFeed.tsx (chatroom-style message list with rich metadata)
│   │   └── ActivityEntry.tsx (sender, content, delivery badges, file previews)
│   └── (no input box — agents respond via MCP, not bc)
├── SubscriptionPanel.tsx (right panel)
│   └── AgentRow.tsx (name, role badge, online dot, subscribe/unsubscribe)
└── SetupWizard.tsx (modal for connecting new gateways)
    └── Platform-specific token fields and setup instructions
```

### Empty State

When no gateways are connected, the channels page shows:

```
┌──────────────────────────────────────────┐
│                                          │
│   Connect your first app                 │
│                                          │
│   ┌──────┐  ┌──────┐  ┌──────┐         │
│   │Slack │  │Telegram│ │Discord│         │
│   └──────┘  └──────┘  └──────┘         │
│   ┌──────┐  ┌──────┐                    │
│   │GitHub│  │ Gmail │                    │
│   └──────┘  └──────┘                    │
│                                          │
│   Click to connect and start receiving   │
│   notifications in your agents.          │
└──────────────────────────────────────────┘
```

### Live Updates

WebSocket events (via existing SSE hub):
- `gateway.message` — new message in a channel (appended to activity feed)
- `gateway.connected` / `gateway.disconnected` — gateway status change
- `gateway.delivery` — delivery status update (agent received/failed)

---

## CLI Commands (14 → 4)

```
bc channel list         — all channels across gateways with subscriber counts
bc channel subscribe    — subscribe agent to channel
bc channel unsubscribe  — unsubscribe agent
bc channel status       — gateway connection status + health
```

---

## Dispatch Flow

```
External Platform (Slack/Telegram/Discord)
    │ WebSocket / polling / webhook
    ▼
pkg/gateway/{platform}/adapter.Start()
    │ calls handler.OnMessage(InboundMessage) or handler.OnFile(msg, att)
    ▼
pkg/gateway/manager.go — fan-out to dispatch
    │ non-blocking goroutine
    ▼
pkg/notify/service.Dispatch()
    ├── store.Subscribers("slack:engineering")     [db read]
    ├── for each subscriber:
    │   ├── agent.SendToAgent(ctx, name, jsonPayload)  [tmux send-keys]
    │   └── store.LogDelivery(entry)                    [db write]
    └── hub.Publish("gateway.message", payload)         [SSE to web UI]
```

**File handling flow:**
```
Slack file_share event
    ▼
adapter calls handler.OnFile(msg, attachment)
    ▼
manager downloads file to .bc/attachments/<hash>.<ext>
    ▼
notification includes attachments[] with local_path
    ▼
agent reads file from local_path, decides what to do
```

---

## Build Sequence

| Phase | Scope | Size | Issue |
|-------|-------|------|-------|
| 1 | `pkg/notify/` — types, store (shared db pattern), service, tests | Small | |
| 2 | `pkg/gateway/` — new EventHandler interface, Status(), live Health(), attachment support | Small | |
| 3 | Server handlers — rewrite gateways.go, wire notify service | Medium | |
| 4 | Delete `pkg/channel/` + workspace config cleanup | Medium | |
| 5 | CLI — rewrite channel.go (14→4 commands) | Small | |
| 6 | Frontend — gateway sidebar, activity feed, subscription panel, setup wizard | Large | |
| 7 | File handling — adapter download, .bc/attachments/, shared volume for Docker | Medium | |
| 8 | Integration testing — smoke test with Slack adapter | Small | |

---

## What Gets Deleted vs Created

| Deleted | ~45 files | ~7,600 lines |
|---------|-----------|-------------|
| Created | ~15 files | ~1,800 lines |
| **Net reduction** | **~30 files** | **~5,800 lines** |

---

## Open Questions

1. **Keep `bc send <agent>`?** — Direct agent-to-agent messaging is separate from channels but shares infrastructure. Keep or remove?
2. **GitHub adapter** — implement in this phase or defer?
3. **TUI channels tab** — update or deprecate in favor of web UI?
4. **Migration** — clean-slate or provide migration script for existing channel data?
