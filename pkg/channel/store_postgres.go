package channel

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PostgresStore provides Postgres-backed channel storage.
// It implements ChannelBackend with the same API as SQLiteStore.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a PostgresStore from an existing *sql.DB connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// InitSchema creates the channel tables in Postgres if they don't exist.
func (s *PostgresStore) InitSchema() error {
	ctx := context.Background()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS channels (
			id          BIGSERIAL PRIMARY KEY,
			name        TEXT NOT NULL UNIQUE,
			type        TEXT NOT NULL DEFAULT 'group' CHECK (type IN ('group', 'direct')),
			description TEXT,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)`,

		`CREATE TABLE IF NOT EXISTS channel_members (
			id               BIGSERIAL PRIMARY KEY,
			channel_id       BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			agent_id         TEXT NOT NULL,
			joined_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_read_msg_id BIGINT DEFAULT 0,
			UNIQUE(channel_id, agent_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_members_agent   ON channel_members(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_members_channel ON channel_members(channel_id)`,

		`CREATE TABLE IF NOT EXISTS messages (
			id         BIGSERIAL PRIMARY KEY,
			channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			sender     TEXT NOT NULL,
			content    TEXT NOT NULL,
			type       TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text','task','review','approval','merge','status')),
			metadata   TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_channel_time ON messages(channel_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender       ON messages(sender)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_type         ON messages(type)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_channel_id   ON messages(channel_id, id)`,

		`CREATE TABLE IF NOT EXISTS mentions (
			id           BIGSERIAL PRIMARY KEY,
			message_id   BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			agent_id     TEXT NOT NULL,
			acknowledged BOOLEAN NOT NULL DEFAULT FALSE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mentions_agent   ON mentions(agent_id, acknowledged)`,
		`CREATE INDEX IF NOT EXISTS idx_mentions_message ON mentions(message_id)`,

		`CREATE TABLE IF NOT EXISTS reactions (
			id         BIGSERIAL PRIMARY KEY,
			message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			emoji      TEXT NOT NULL,
			user_id    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(message_id, emoji, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reactions_message ON reactions(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reactions_user    ON reactions(user_id)`,

		// Seed default channels
		`INSERT INTO channels (name, type, description, created_at, updated_at) VALUES
			('general',     'group', 'General discussion for all agents',  NOW(), NOW()),
			('engineering', 'group', 'Engineering team coordination',      NOW(), NOW()),
			('all',         'group', 'Broadcast channel for announcements', NOW(), NOW())
		ON CONFLICT (name) DO NOTHING`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("postgres channel schema: %w\nSQL: %s", err, stmt[:min(60, len(stmt))])
		}
	}
	return nil
}

// Close is a no-op — the shared DB is owned by the caller.
func (s *PostgresStore) Close() error {
	return nil
}

// --- Channel operations ---

func (s *PostgresStore) CreateChannel(name string, channelType ChannelType, description string) (*ChannelInfo, error) {
	ctx := context.Background()
	var info ChannelInfo
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO channels (name, type, description)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (name) DO UPDATE SET updated_at = NOW()
		 RETURNING id, name, type, COALESCE(description,''), created_at, updated_at`,
		name, string(channelType), description,
	).Scan(&info.ID, &info.Name, &info.Type, &info.Description, &info.CreatedAt, &info.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create channel %q: %w", name, err)
	}
	return &info, nil
}

func (s *PostgresStore) GetChannel(name string) (*ChannelInfo, error) {
	ctx := context.Background()
	var info ChannelInfo
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, type, COALESCE(description,''), created_at, updated_at
		 FROM channels WHERE name = $1`, name,
	).Scan(&info.ID, &info.Name, &info.Type, &info.Description, &info.CreatedAt, &info.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("channel %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	return &info, nil
}

func (s *PostgresStore) ListChannels() ([]*ChannelInfo, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, COALESCE(description,''), created_at, updated_at
		 FROM channels ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var channels []*ChannelInfo
	for rows.Next() {
		var info ChannelInfo
		if err := rows.Scan(&info.ID, &info.Name, &info.Type, &info.Description, &info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, &info)
	}
	return channels, rows.Err()
}

func (s *PostgresStore) DeleteChannel(name string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE name = $1`, name)
	return err
}

func (s *PostgresStore) SetChannelDescription(channelName, description string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`UPDATE channels SET description = $1, updated_at = NOW() WHERE name = $2`,
		description, channelName)
	return err
}

// --- Member operations ---

func (s *PostgresStore) AddMember(channelName, agentID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO channel_members (channel_id, agent_id)
		 SELECT id, $1 FROM channels WHERE name = $2
		 ON CONFLICT (channel_id, agent_id) DO NOTHING`,
		agentID, channelName)
	return err
}

func (s *PostgresStore) RemoveMember(channelName, agentID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM channel_members
		 WHERE channel_id = (SELECT id FROM channels WHERE name = $1)
		   AND agent_id = $2`,
		channelName, agentID)
	return err
}

func (s *PostgresStore) GetMembers(channelName string) ([]string, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT cm.agent_id
		 FROM channel_members cm
		 JOIN channels c ON c.id = cm.channel_id
		 WHERE c.name = $1
		 ORDER BY cm.joined_at`, channelName)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []string
	for rows.Next() {
		var agent string
		if err := rows.Scan(&agent); err != nil {
			return nil, err
		}
		members = append(members, agent)
	}
	return members, rows.Err()
}

// --- Message operations ---

func (s *PostgresStore) AddMessage(channelName, sender, content string, msgType MessageType, metadata string) (*Message, error) {
	ctx := context.Background()
	var msg Message
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO messages (channel_id, sender, content, type, metadata)
		 SELECT id, $1, $2, $3, NULLIF($4,'')
		 FROM channels WHERE name = $5
		 RETURNING id, channel_id, sender, content, type, COALESCE(metadata,''), created_at`,
		sender, content, string(msgType), metadata, channelName,
	).Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &msg.Metadata, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("add message to %q: %w", channelName, err)
	}
	msg.CreatedAt = createdAt
	return &msg, nil
}

func (s *PostgresStore) GetHistory(channelName string, limit int) ([]*Message, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.channel_id, m.sender, m.content, m.type, COALESCE(m.metadata,''), m.created_at
		 FROM messages m
		 JOIN channels c ON c.id = m.channel_id
		 WHERE c.name = $1
		 ORDER BY m.created_at ASC
		 LIMIT $2`,
		channelName, limit)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgs []*Message
	for rows.Next() {
		var msg Message
		var createdAt time.Time
		if err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &msg.Metadata, &createdAt); err != nil {
			return nil, err
		}
		msg.CreatedAt = createdAt
		msgs = append(msgs, &msg)
	}
	return msgs, rows.Err()
}

// --- Reaction operations ---

func (s *PostgresStore) AddReaction(messageID int64, emoji, userID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reactions (message_id, emoji, user_id) VALUES ($1, $2, $3)
		 ON CONFLICT (message_id, emoji, user_id) DO NOTHING`,
		messageID, emoji, userID)
	if err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}
	return nil
}

func (s *PostgresStore) RemoveReaction(messageID int64, emoji, userID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM reactions WHERE message_id = $1 AND emoji = $2 AND user_id = $3`,
		messageID, emoji, userID)
	if err != nil {
		return fmt.Errorf("remove reaction: %w", err)
	}
	return nil
}

func (s *PostgresStore) ToggleReaction(messageID int64, emoji, userID string) (bool, error) {
	ctx := context.Background()

	// Try insert first
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO reactions (message_id, emoji, user_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (message_id, emoji, user_id) DO NOTHING`,
		messageID, emoji, userID)
	if err != nil {
		return false, fmt.Errorf("toggle reaction: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		return true, nil // added
	}

	// Already existed — remove it
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM reactions WHERE message_id = $1 AND emoji = $2 AND user_id = $3`,
		messageID, emoji, userID)
	return false, err
}

func (s *PostgresStore) GetReactions(messageID int64) (map[string][]string, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT emoji, user_id FROM reactions WHERE message_id = $1 ORDER BY created_at`,
		messageID)
	if err != nil {
		return nil, fmt.Errorf("get reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string][]string)
	for rows.Next() {
		var emoji, user string
		if err := rows.Scan(&emoji, &user); err != nil {
			return nil, err
		}
		result[emoji] = append(result[emoji], user)
	}
	return result, rows.Err()
}

// min returns the smaller of two ints. Used for error message truncation.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
