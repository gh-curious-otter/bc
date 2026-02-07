-- Channel SQLite Schema for bc v2
-- Part of Epic #26 (Channels Infrastructure)
-- Replaces JSON-based storage with SQLite for better querying and scalability

-- Schema version for migrations
PRAGMA user_version = 1;

-- Enable foreign keys
PRAGMA foreign_keys = ON;

--------------------------------------------------------------------------------
-- CHANNELS TABLE
--------------------------------------------------------------------------------
-- Stores channel metadata. Each channel is a named group for message routing.
-- Types: 'group' (multi-member), 'direct' (per-agent DM channel)
CREATE TABLE IF NOT EXISTS channels (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'group' CHECK (type IN ('group', 'direct')),
    description TEXT,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Index for channel lookup by name (most common query)
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

-- Index for filtering by type
CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);

--------------------------------------------------------------------------------
-- CHANNEL MEMBERS TABLE
--------------------------------------------------------------------------------
-- Many-to-many relationship between channels and agents.
-- Tracks when each member joined and their read position.
CREATE TABLE IF NOT EXISTS channel_members (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id      INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    agent_id        TEXT NOT NULL,
    joined_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_read_msg_id INTEGER DEFAULT 0,  -- For unread message tracking
    UNIQUE(channel_id, agent_id)
);

-- Index for finding all channels an agent belongs to
CREATE INDEX IF NOT EXISTS idx_channel_members_agent ON channel_members(agent_id);

-- Index for finding all members of a channel
CREATE INDEX IF NOT EXISTS idx_channel_members_channel ON channel_members(channel_id);

--------------------------------------------------------------------------------
-- MESSAGES TABLE
--------------------------------------------------------------------------------
-- Stores all channel messages with type classification for work coordination.
-- Message types enable filtering for task assignments, reviews, approvals, etc.
CREATE TABLE IF NOT EXISTS messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender      TEXT NOT NULL,              -- Agent ID or 'system'
    content     TEXT NOT NULL,
    type        TEXT NOT NULL DEFAULT 'text' CHECK (type IN (
        'text',      -- Normal conversation message
        'task',      -- Work assignment (@mention with task)
        'review',    -- PR review request
        'approval',  -- Tech lead approval notification
        'merge',     -- Merge request/notification
        'status'     -- Agent status update
    )),
    metadata    TEXT,                       -- JSON blob for type-specific data
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Primary query pattern: messages in a channel, ordered by time
CREATE INDEX IF NOT EXISTS idx_messages_channel_time ON messages(channel_id, created_at DESC);

-- Filter messages by sender (e.g., "show me all messages from engineer-01")
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender);

-- Filter messages by type (e.g., "show me all task assignments")
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages(type);

-- Composite index for unread queries: messages in channel after a certain ID
CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id, id);

--------------------------------------------------------------------------------
-- MENTIONS TABLE
--------------------------------------------------------------------------------
-- Tracks @mentions for notification and filtering.
-- Extracted from message content for fast lookup.
CREATE TABLE IF NOT EXISTS mentions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id  INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    agent_id    TEXT NOT NULL,              -- The mentioned agent
    acknowledged INTEGER NOT NULL DEFAULT 0  -- 0 = unread, 1 = read
);

-- Find all mentions for an agent (e.g., "messages mentioning me")
CREATE INDEX IF NOT EXISTS idx_mentions_agent ON mentions(agent_id, acknowledged);

-- Find all mentions in a message
CREATE INDEX IF NOT EXISTS idx_mentions_message ON mentions(message_id);

--------------------------------------------------------------------------------
-- FULL-TEXT SEARCH (FTS5)
--------------------------------------------------------------------------------
-- Virtual table for fast full-text search across message content.
-- Enables queries like: bc channel search "authentication bug"
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    sender,
    content='messages',
    content_rowid='id'
);

-- Triggers to keep FTS index in sync with messages table
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content, sender)
    VALUES (new.id, new.content, new.sender);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, sender)
    VALUES ('delete', old.id, old.content, old.sender);
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, sender)
    VALUES ('delete', old.id, old.content, old.sender);
    INSERT INTO messages_fts(rowid, content, sender)
    VALUES (new.id, new.content, new.sender);
END;

--------------------------------------------------------------------------------
-- DEFAULT CHANNELS
--------------------------------------------------------------------------------
-- Seed default channels that every workspace should have
INSERT OR IGNORE INTO channels (name, type, description) VALUES
    ('general', 'group', 'General discussion for all agents'),
    ('engineering', 'group', 'Engineering team coordination'),
    ('all', 'group', 'Broadcast channel for announcements');
