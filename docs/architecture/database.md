# Database Architecture

## Overview

All data stored in `.bc/bc.db` (SQLite, WAL mode). PostgreSQL supported via `[database]` config but not yet connected end-to-end.

## Connection Management

Centralized in `pkg/db/db.go`:
- WAL journal mode
- Foreign keys enabled
- 30s busy timeout (handles concurrent agent access)
- MaxOpenConns=1, MaxIdleConns=1 (SQLite single-writer)
- `PRAGMA synchronous = NORMAL`, `cache_size = -2000`, `temp_store = MEMORY`, `mmap_size = 268435456`

**Known issue:** 5 stores still bypass `pkg/db.Open()` and use raw `sql.Open()` with inconsistent pragmas (5s busy timeout vs 30s). See #2026.

## Schema

### agents
```sql
CREATE TABLE agents (
    name            TEXT PRIMARY KEY,
    role            TEXT NOT NULL,
    state           TEXT NOT NULL DEFAULT 'idle',
    task            TEXT,
    tool            TEXT,
    session         TEXT,
    session_id      TEXT,
    parent_id       TEXT,
    workspace       TEXT,
    worktree_dir    TEXT,
    log_file        TEXT,
    team            TEXT,
    env_file        TEXT,
    runtime_backend TEXT DEFAULT 'tmux',
    hooked_work     TEXT,
    children        TEXT, -- JSON array
    is_root         INTEGER DEFAULT 0,
    created_at      TEXT NOT NULL,
    started_at      TEXT,
    updated_at      TEXT NOT NULL,
    stopped_at      TEXT
);
```

### channels, channel_members, messages, mentions, reactions
```sql
CREATE TABLE channels (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'group' CHECK (type IN ('group', 'direct')),
    description TEXT,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
-- Indexes: name, type

CREATE TABLE messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender      TEXT NOT NULL,
    content     TEXT NOT NULL,
    type        TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text','task','review','approval','merge','status')),
    metadata    TEXT,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
-- Indexes: (channel_id, created_at DESC), sender, type, (channel_id, id)
-- FTS5: messages_fts on (content, sender)
```

### cost_records, cost_budgets, cost_imports
```sql
CREATE TABLE cost_records (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id             TEXT,
    team_id              TEXT,
    model                TEXT,
    session_id           TEXT,
    input_tokens         INTEGER DEFAULT 0,
    output_tokens        INTEGER DEFAULT 0,
    total_tokens         INTEGER DEFAULT 0,
    cache_creation_tokens INTEGER DEFAULT 0,
    cache_read_tokens    INTEGER DEFAULT 0,
    cost_usd             REAL DEFAULT 0,
    timestamp            TEXT NOT NULL
);
-- Indexes: agent_id, model, timestamp DESC, team_id
-- Missing: composite (agent_id, timestamp) for budget queries — see #2110
```

### events
```sql
CREATE TABLE events (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    type      TEXT NOT NULL,
    agent     TEXT,
    message   TEXT,
    data      TEXT, -- JSON
    timestamp TEXT NOT NULL
);
-- Indexes: agent, timestamp DESC
```

### secrets, secret_meta
```sql
CREATE TABLE secrets (
    name        TEXT PRIMARY KEY,
    value       TEXT NOT NULL, -- AES-256-GCM encrypted, base64
    description TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
```

### cron_jobs, cron_logs
### mcp_servers
### tools
### daemons

## Database Evolution

```
Feb 7-10:  agents.json, channels.json (flat files)
Mar 6:     state.db for agents + events (#1934)
Mar 18:    bc.db consolidation — all 9 stores into one (#2017)
Mar 19:    Agent store fixed to use bc.db (#2039)
Current:   bc.db is primary, but 5 stores still open own connections
```

## Postgres Support

Config: `[database] driver = "postgres"` with connection URL. Channel store has a full Postgres backend (`pkg/channel/store_postgres.go`). Other stores have not been ported. The `bcdb` Docker image (postgres:17) exists but nothing connects to it in production.
