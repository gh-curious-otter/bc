-- bcstats schema — TimescaleDB hypertables for bc workspace metrics
-- Column names must match pkg/stats/store.go ensureSchema().

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- System Metrics — bc-daemon, bc-sql, bc-stats containers
CREATE TABLE IF NOT EXISTS system_metrics (
    time             TIMESTAMPTZ NOT NULL,
    system_name      TEXT NOT NULL,
    cpu_percent      DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_used_bytes   BIGINT NOT NULL DEFAULT 0,
    mem_limit_bytes  BIGINT NOT NULL DEFAULT 0,
    mem_percent      DOUBLE PRECISION NOT NULL DEFAULT 0,
    net_rx_bytes     BIGINT NOT NULL DEFAULT 0,
    net_tx_bytes     BIGINT NOT NULL DEFAULT 0,
    disk_read_bytes  BIGINT NOT NULL DEFAULT 0,
    disk_write_bytes BIGINT NOT NULL DEFAULT 0
);
SELECT create_hypertable('system_metrics', 'time', if_not_exists => TRUE);

-- Agent Metrics — per-agent container stats
CREATE TABLE IF NOT EXISTS agent_metrics (
    time             TIMESTAMPTZ NOT NULL,
    agent_name       TEXT NOT NULL,
    role             TEXT NOT NULL DEFAULT '',
    tool             TEXT NOT NULL DEFAULT '',
    runtime          TEXT NOT NULL DEFAULT 'docker',
    state            TEXT NOT NULL DEFAULT '',
    cpu_percent      DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem_used_bytes   BIGINT NOT NULL DEFAULT 0,
    mem_limit_bytes  BIGINT NOT NULL DEFAULT 0,
    mem_percent      DOUBLE PRECISION NOT NULL DEFAULT 0,
    net_rx_bytes     BIGINT NOT NULL DEFAULT 0,
    net_tx_bytes     BIGINT NOT NULL DEFAULT 0,
    disk_read_bytes  BIGINT NOT NULL DEFAULT 0,
    disk_write_bytes BIGINT NOT NULL DEFAULT 0
);
SELECT create_hypertable('agent_metrics', 'time', if_not_exists => TRUE);

-- Token Metrics — per-agent token consumption from JSONL
CREATE TABLE IF NOT EXISTS token_metrics (
    time          TIMESTAMPTZ NOT NULL,
    agent_name    TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    input_tokens  BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cache_read    BIGINT NOT NULL DEFAULT 0,
    cache_create  BIGINT NOT NULL DEFAULT 0,
    cost_usd      DOUBLE PRECISION NOT NULL DEFAULT 0
);
SELECT create_hypertable('token_metrics', 'time', if_not_exists => TRUE);

-- Channel Metrics — message/member/reaction counts
CREATE TABLE IF NOT EXISTS channel_metrics (
    time           TIMESTAMPTZ NOT NULL,
    channel_name   TEXT NOT NULL,
    message_count  BIGINT NOT NULL DEFAULT 0,
    member_count   INT NOT NULL DEFAULT 0,
    reaction_count BIGINT NOT NULL DEFAULT 0
);
SELECT create_hypertable('channel_metrics', 'time', if_not_exists => TRUE);

-- Retention policies
SELECT add_retention_policy('system_metrics',  INTERVAL '7 days',  if_not_exists => TRUE);
SELECT add_retention_policy('agent_metrics',   INTERVAL '7 days',  if_not_exists => TRUE);
SELECT add_retention_policy('token_metrics',   INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('channel_metrics', INTERVAL '30 days', if_not_exists => TRUE);
