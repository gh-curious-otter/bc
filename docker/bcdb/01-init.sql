-- bc database initialisation
-- Runs once on first Postgres startup.
-- Tables are also created by bcd on connect (migrations), so this
-- file just ensures the database and search_path are correct.

\connect bc

-- Channels
CREATE TABLE IF NOT EXISTS channels (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    topic      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS channel_members (
    channel_name TEXT NOT NULL,
    agent        TEXT NOT NULL,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel_name, agent)
);

CREATE TABLE IF NOT EXISTS channel_messages (
    id           SERIAL PRIMARY KEY,
    channel_name TEXT        NOT NULL,
    sender       TEXT        NOT NULL,
    body         TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channel_messages_channel ON channel_messages (channel_name, created_at DESC);
