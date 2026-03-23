-- bcsql schema — Postgres replacement for all SQLite stores
-- All bc workspace data consolidated into one Postgres database.

-- =============================================================================
-- Agents
-- =============================================================================

CREATE TABLE IF NOT EXISTS agents (
    name            TEXT PRIMARY KEY,
    role            TEXT NOT NULL,
    state           TEXT NOT NULL DEFAULT 'idle',
    tool            TEXT,
    parent_id       TEXT,
    team            TEXT,
    task            TEXT,
    session         TEXT,
    workspace       TEXT NOT NULL,
    worktree_dir    TEXT,
    log_file        TEXT,
    hooked_work     TEXT,
    children        TEXT,
    is_root         BOOLEAN NOT NULL DEFAULT FALSE,
    crash_count     INTEGER NOT NULL DEFAULT 0,
    last_crash_time TIMESTAMPTZ,
    recovered_from  TEXT,
    runtime_backend TEXT,
    session_id      TEXT,
    ttl             INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agents_state  ON agents(state);
CREATE INDEX IF NOT EXISTS idx_agents_role   ON agents(role);
CREATE INDEX IF NOT EXISTS idx_agents_parent ON agents(parent_id);

-- =============================================================================
-- Agent Stats (non-timeseries, kept here for agent CRUD; timeseries in bcstats)
-- =============================================================================

CREATE TABLE IF NOT EXISTS agent_stats (
    id             BIGSERIAL PRIMARY KEY,
    agent_name     TEXT NOT NULL,
    collected_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_pct        REAL NOT NULL DEFAULT 0,
    mem_used_mb    REAL NOT NULL DEFAULT 0,
    mem_limit_mb   REAL NOT NULL DEFAULT 0,
    net_rx_mb      REAL NOT NULL DEFAULT 0,
    net_tx_mb      REAL NOT NULL DEFAULT 0,
    block_read_mb  REAL NOT NULL DEFAULT 0,
    block_write_mb REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_agent_stats_agent ON agent_stats(agent_name);
CREATE INDEX IF NOT EXISTS idx_agent_stats_time  ON agent_stats(collected_at);

-- =============================================================================
-- Channels
-- =============================================================================

CREATE TABLE IF NOT EXISTS channels (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'group' CHECK (type IN ('group', 'direct')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);

CREATE TABLE IF NOT EXISTS channel_members (
    id               BIGSERIAL PRIMARY KEY,
    channel_id       BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    agent_id         TEXT NOT NULL,
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_msg_id BIGINT DEFAULT 0,
    UNIQUE(channel_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_members_agent   ON channel_members(agent_id);
CREATE INDEX IF NOT EXISTS idx_channel_members_channel ON channel_members(channel_id);

CREATE TABLE IF NOT EXISTS messages (
    id         BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender     TEXT NOT NULL,
    content    TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text','task','review','approval','merge','status')),
    metadata   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_time ON messages(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sender       ON messages(sender);
CREATE INDEX IF NOT EXISTS idx_messages_type         ON messages(type);
CREATE INDEX IF NOT EXISTS idx_messages_channel_id   ON messages(channel_id, id);

CREATE TABLE IF NOT EXISTS mentions (
    id           BIGSERIAL PRIMARY KEY,
    message_id   BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    agent_id     TEXT NOT NULL,
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_mentions_agent   ON mentions(agent_id, acknowledged);
CREATE INDEX IF NOT EXISTS idx_mentions_message ON mentions(message_id);

CREATE TABLE IF NOT EXISTS reactions (
    id         BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    emoji      TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(message_id, emoji, user_id)
);

CREATE INDEX IF NOT EXISTS idx_reactions_message ON reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_reactions_user    ON reactions(user_id);

-- Seed default channels
INSERT INTO channels (name, type, description) VALUES
    ('general',     'group', 'General discussion for all agents'),
    ('engineering', 'group', 'Engineering team coordination'),
    ('all',         'group', 'Broadcast channel for announcements')
ON CONFLICT (name) DO NOTHING;

-- =============================================================================
-- Cost Records
-- =============================================================================

CREATE TABLE IF NOT EXISTS cost_records (
    id                    BIGSERIAL PRIMARY KEY,
    agent_id              TEXT NOT NULL,
    team_id               TEXT,
    model                 TEXT NOT NULL,
    input_tokens          BIGINT NOT NULL DEFAULT 0,
    output_tokens         BIGINT NOT NULL DEFAULT 0,
    total_tokens          BIGINT NOT NULL DEFAULT 0,
    cost_usd              DOUBLE PRECISION NOT NULL DEFAULT 0,
    session_id            TEXT,
    cache_creation_tokens BIGINT DEFAULT 0,
    cache_read_tokens     BIGINT DEFAULT 0,
    timestamp             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_records_agent     ON cost_records(agent_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_team      ON cost_records(team_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_model     ON cost_records(model);
CREATE INDEX IF NOT EXISTS idx_cost_records_timestamp ON cost_records(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cost_records_agent_time ON cost_records(agent_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_cost_records_team_time  ON cost_records(team_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS cost_budgets (
    id         BIGSERIAL PRIMARY KEY,
    scope      TEXT NOT NULL UNIQUE,
    period     TEXT NOT NULL DEFAULT 'monthly' CHECK (period IN ('daily', 'weekly', 'monthly')),
    limit_usd  DOUBLE PRECISION NOT NULL DEFAULT 0,
    alert_at   DOUBLE PRECISION NOT NULL DEFAULT 0.8,
    hard_stop  BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_budgets_scope ON cost_budgets(scope);

CREATE TABLE IF NOT EXISTS cost_imports (
    source_path  TEXT PRIMARY KEY,
    watermark    TEXT NOT NULL DEFAULT '',
    record_count BIGINT NOT NULL DEFAULT 0,
    imported_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Events
-- =============================================================================

CREATE TABLE IF NOT EXISTS events (
    id        BIGSERIAL PRIMARY KEY,
    type      TEXT NOT NULL,
    agent     TEXT,
    message   TEXT,
    data      TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_agent     ON events(agent);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);

-- =============================================================================
-- Cron Jobs
-- =============================================================================

CREATE TABLE IF NOT EXISTS cron_jobs (
    name       TEXT PRIMARY KEY,
    schedule   TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    prompt     TEXT,
    command    TEXT,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    last_run   TIMESTAMPTZ,
    next_run   TIMESTAMPTZ,
    run_count  INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cron_logs (
    id          BIGSERIAL PRIMARY KEY,
    job_name    TEXT NOT NULL REFERENCES cron_jobs(name) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    duration_ms BIGINT DEFAULT 0,
    cost_usd    DOUBLE PRECISION DEFAULT 0,
    output      TEXT,
    run_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cron_logs_job ON cron_logs(job_name);

-- =============================================================================
-- MCP Servers
-- =============================================================================

CREATE TABLE IF NOT EXISTS mcp_servers (
    name       TEXT PRIMARY KEY,
    transport  TEXT NOT NULL DEFAULT 'stdio',
    command    TEXT,
    args       TEXT,
    url        TEXT,
    env        TEXT,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Secrets
-- =============================================================================

CREATE TABLE IF NOT EXISTS secrets (
    name        TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS secret_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- =============================================================================
-- Tools
-- =============================================================================

CREATE TABLE IF NOT EXISTS tools (
    name        TEXT PRIMARY KEY,
    command     TEXT NOT NULL,
    install_cmd TEXT,
    upgrade_cmd TEXT,
    slash_cmds  TEXT,
    mcp_servers TEXT,
    config      TEXT,
    builtin     BOOLEAN NOT NULL DEFAULT FALSE,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Roles
-- =============================================================================

CREATE TABLE IF NOT EXISTS roles (
    name           TEXT PRIMARY KEY,
    description    TEXT,
    prompt         TEXT,
    mcp_servers    TEXT,
    parent_roles   TEXT,
    secrets        TEXT,
    plugins        TEXT,
    settings       TEXT,
    rules          TEXT,
    agents         TEXT,
    skills         TEXT,
    commands       TEXT,
    prompt_create  TEXT,
    prompt_start   TEXT,
    prompt_stop    TEXT,
    prompt_delete  TEXT,
    review         TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Daemons
-- =============================================================================

CREATE TABLE IF NOT EXISTS daemons (
    name         TEXT PRIMARY KEY,
    runtime      TEXT NOT NULL,
    cmd          TEXT,
    image        TEXT,
    status       TEXT NOT NULL DEFAULT 'stopped',
    pid          INTEGER,
    container_id TEXT,
    ports        TEXT,
    volumes      TEXT,
    env          TEXT,
    restart      TEXT DEFAULT 'no',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    stopped_at   TIMESTAMPTZ
);
