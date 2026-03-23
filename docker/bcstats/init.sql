-- bcstats schema — TimescaleDB hypertables for bc workspace metrics
-- Column names must match pkg/stats/store.go ensureSchema().

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- System Metrics
CREATE TABLE IF NOT EXISTS system_metrics (
    time        TIMESTAMPTZ NOT NULL,
    cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_bytes   BIGINT NOT NULL DEFAULT 0,
    mem_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    disk_bytes  BIGINT NOT NULL DEFAULT 0,
    goroutines  INT NOT NULL DEFAULT 0,
    hostname    TEXT NOT NULL DEFAULT ''
);
SELECT create_hypertable('system_metrics', 'time', if_not_exists => TRUE);

-- Agent Metrics
CREATE TABLE IF NOT EXISTS agent_metrics (
    time       TIMESTAMPTZ NOT NULL,
    agent_name TEXT NOT NULL,
    agent_id   TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT '',
    state      TEXT NOT NULL DEFAULT '',
    cpu_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_bytes  BIGINT NOT NULL DEFAULT 0,
    uptime_sec BIGINT NOT NULL DEFAULT 0
);
SELECT create_hypertable('agent_metrics', 'time', if_not_exists => TRUE);

-- Token Metrics
CREATE TABLE IF NOT EXISTS token_metrics (
    time          TIMESTAMPTZ NOT NULL,
    agent_id      TEXT NOT NULL DEFAULT '',
    agent_name    TEXT NOT NULL DEFAULT '',
    provider      TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    input_tokens  BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cost_usd      DOUBLE PRECISION NOT NULL DEFAULT 0
);
SELECT create_hypertable('token_metrics', 'time', if_not_exists => TRUE);

-- Channel Metrics
CREATE TABLE IF NOT EXISTS channel_metrics (
    time          TIMESTAMPTZ NOT NULL,
    channel_name  TEXT NOT NULL,
    messages_sent BIGINT NOT NULL DEFAULT 0,
    messages_read BIGINT NOT NULL DEFAULT 0,
    participants  INT NOT NULL DEFAULT 0
);
SELECT create_hypertable('channel_metrics', 'time', if_not_exists => TRUE);

-- Daemon Metrics
CREATE TABLE IF NOT EXISTS daemon_metrics (
    time        TIMESTAMPTZ NOT NULL,
    daemon_name TEXT NOT NULL,
    state       TEXT NOT NULL DEFAULT '',
    pid         INT NOT NULL DEFAULT 0,
    cpu_pct     DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_bytes   BIGINT NOT NULL DEFAULT 0,
    restarts    INT NOT NULL DEFAULT 0
);
SELECT create_hypertable('daemon_metrics', 'time', if_not_exists => TRUE);

-- Retention policies
SELECT add_retention_policy('system_metrics',  INTERVAL '7 days',  if_not_exists => TRUE);
SELECT add_retention_policy('agent_metrics',   INTERVAL '7 days',  if_not_exists => TRUE);
SELECT add_retention_policy('token_metrics',   INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('channel_metrics', INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('daemon_metrics',  INTERVAL '7 days',  if_not_exists => TRUE);
