// Package channel provides SQLite-backed channel storage for bc v2.
package channel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rpuneet/bc/pkg/log"
)

// ChannelType represents the type of a channel.
type ChannelType string

const (
	ChannelTypeGroup  ChannelType = "group"
	ChannelTypeDirect ChannelType = "direct"
)

// Message represents a channel message.
type Message struct {
	CreatedAt time.Time   `json:"created_at"`
	Sender    string      `json:"sender"`
	Content   string      `json:"content"`
	Metadata  string      `json:"metadata,omitempty"`
	Type      MessageType `json:"type"`
	ID        int64       `json:"id"`
	ChannelID int64       `json:"channel_id"`
}

// ChannelInfo represents channel metadata.
type ChannelInfo struct {
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        ChannelType `json:"type"`
	ID          int64       `json:"id"`
}

// MentionRecord represents a tracked @mention in the database.
type MentionRecord struct {
	AgentID      string `json:"agent_id"`
	ID           int64  `json:"id"`
	MessageID    int64  `json:"message_id"`
	Acknowledged bool   `json:"acknowledged"`
}

// SQLiteStore provides SQLite-backed channel storage.
type SQLiteStore struct {
	db           *sql.DB
	path         string
	ftsAvailable bool
}

// NewSQLiteStore creates a new SQLite store for the given workspace.
func NewSQLiteStore(workspacePath string) *SQLiteStore {
	return &SQLiteStore{
		path: filepath.Join(workspacePath, ".bc", "bc.db"),
	}
}

// Open initializes the SQLite database.
func (s *SQLiteStore) Open() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// #1011: Add WAL mode and busy timeout for better concurrency
	db, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// #1011: Configure connection pool for SQLite's single-writer model
	// SQLite only allows one writer at a time, so limit connections
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err := s.initSchema(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// #1011: Set optimal SQLite pragmas for performance
	ctx := context.Background()
	pragmas := `
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -2000;
		PRAGMA temp_store = MEMORY;
	`
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		log.Warn("failed to set SQLite pragmas", "error", err)
	}

	s.db = db
	return nil
}

// initSchema executes the schema SQL.
func (s *SQLiteStore) initSchema(db *sql.DB) error {
	ctx := context.Background()

	coreSchema := `
		PRAGMA foreign_keys = ON;

		CREATE TABLE IF NOT EXISTS channels (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL UNIQUE,
			type        TEXT NOT NULL DEFAULT 'group' CHECK (type IN ('group', 'direct')),
			description TEXT,
			created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
		CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);

		CREATE TABLE IF NOT EXISTS channel_members (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id      INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			agent_id        TEXT NOT NULL,
			joined_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			last_read_msg_id INTEGER DEFAULT 0,
			UNIQUE(channel_id, agent_id)
		);
		CREATE INDEX IF NOT EXISTS idx_channel_members_agent ON channel_members(agent_id);
		CREATE INDEX IF NOT EXISTS idx_channel_members_channel ON channel_members(channel_id);

		CREATE TABLE IF NOT EXISTS messages (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id  INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			sender      TEXT NOT NULL,
			content     TEXT NOT NULL,
			type        TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text', 'task', 'review', 'approval', 'merge', 'status')),
			metadata    TEXT,
			created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		CREATE INDEX IF NOT EXISTS idx_messages_channel_time ON messages(channel_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender);
		CREATE INDEX IF NOT EXISTS idx_messages_type ON messages(type);
		CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id, id);

		CREATE TABLE IF NOT EXISTS mentions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id  INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			agent_id    TEXT NOT NULL,
			acknowledged INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_mentions_agent ON mentions(agent_id, acknowledged);
		CREATE INDEX IF NOT EXISTS idx_mentions_message ON mentions(message_id);

		CREATE TABLE IF NOT EXISTS reactions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id  INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			emoji       TEXT NOT NULL,
			user_id     TEXT NOT NULL,
			created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			UNIQUE(message_id, emoji, user_id)
		);
		CREATE INDEX IF NOT EXISTS idx_reactions_message ON reactions(message_id);
		CREATE INDEX IF NOT EXISTS idx_reactions_user ON reactions(user_id);

		INSERT OR IGNORE INTO channels (name, type, description) VALUES
			('general', 'group', 'General discussion for all agents'),
			('engineering', 'group', 'Engineering team coordination'),
			('all', 'group', 'Broadcast channel for announcements');
	`

	if _, err := db.ExecContext(ctx, coreSchema); err != nil {
		return err
	}

	// Try FTS5, then FTS4, then skip
	ftsSchema := `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(content, sender, content='messages', content_rowid='id');`
	if _, err := db.ExecContext(ctx, ftsSchema); err != nil {
		fts4Schema := `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts4(content, sender, content='messages');`
		if _, err := db.ExecContext(ctx, fts4Schema); err != nil {
			s.ftsAvailable = false
			return nil
		}
	}

	ftsTriggers := `
		CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, content, sender) VALUES (new.id, new.content, new.sender);
		END;
		CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, content, sender) VALUES ('delete', old.id, old.content, old.sender);
		END;
		CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, content, sender) VALUES ('delete', old.id, old.content, old.sender);
			INSERT INTO messages_fts(rowid, content, sender) VALUES (new.id, new.content, new.sender);
		END;
	`
	_, _ = db.ExecContext(ctx, ftsTriggers) // Ignore errors for triggers
	s.ftsAvailable = true
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB returns the underlying database connection.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// CreateChannel creates a new channel.
func (s *SQLiteStore) CreateChannel(name string, channelType ChannelType, description string) (*ChannelInfo, error) {
	ctx := context.Background()
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO channels (name, type, description) VALUES (?, ?, ?)",
		name, channelType, description,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel insert ID: %w", err)
	}
	return s.GetChannelByID(id)
}

// GetChannel returns a channel by name.
func (s *SQLiteStore) GetChannel(name string) (*ChannelInfo, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		"SELECT id, name, type, description, created_at, updated_at FROM channels WHERE name = ?",
		name,
	)
	return s.scanChannel(row)
}

// GetChannelByID returns a channel by ID.
func (s *SQLiteStore) GetChannelByID(id int64) (*ChannelInfo, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		"SELECT id, name, type, description, created_at, updated_at FROM channels WHERE id = ?",
		id,
	)
	return s.scanChannel(row)
}

func (s *SQLiteStore) scanChannel(row *sql.Row) (*ChannelInfo, error) {
	var ch ChannelInfo
	var createdAt, updatedAt string
	var desc sql.NullString

	err := row.Scan(&ch.ID, &ch.Name, &ch.Type, &desc, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan channel: %w", err)
	}

	ch.Description = desc.String
	if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
		log.Warn("failed to parse channel created_at", "value", createdAt, "error", err)
	} else {
		ch.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		log.Warn("failed to parse channel updated_at", "value", updatedAt, "error", err)
	} else {
		ch.UpdatedAt = t
	}
	return &ch, nil
}

// ListChannels returns all channels.
func (s *SQLiteStore) ListChannels() ([]*ChannelInfo, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, type, description, created_at, updated_at FROM channels ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var channels []*ChannelInfo
	for rows.Next() {
		var ch ChannelInfo
		var createdAt, updatedAt string
		var desc sql.NullString

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &desc, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		ch.Description = desc.String
		if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
			log.Warn("failed to parse channel created_at", "value", createdAt, "error", err)
		} else {
			ch.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err != nil {
			log.Warn("failed to parse channel updated_at", "value", updatedAt, "error", err)
		} else {
			ch.UpdatedAt = t
		}
		channels = append(channels, &ch)
	}
	return channels, rows.Err()
}

// DeleteChannel removes a channel by name.
// This explicitly deletes related records and handles FTS cleanup
// to avoid trigger issues with external content FTS tables (issue #738).
func (s *SQLiteStore) DeleteChannel(name string) error {
	ctx := context.Background()

	// Get the channel ID first
	ch, err := s.GetChannel(name)
	if err != nil {
		return err
	}
	if ch == nil {
		return fmt.Errorf("channel %q not found", name)
	}

	// Use a transaction for atomic deletion
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Temporarily drop the FTS delete trigger to avoid issues with
	// external content FTS5 tables (they don't support the delete command)
	_, _ = tx.ExecContext(ctx, "DROP TRIGGER IF EXISTS messages_ad")

	// Delete in correct order to satisfy foreign key constraints:
	// 1. Delete reactions (references messages)
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM reactions WHERE message_id IN (SELECT id FROM messages WHERE channel_id = ?)",
		ch.ID); err != nil {
		return fmt.Errorf("failed to delete reactions: %w", err)
	}

	// 2. Delete mentions (references messages)
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM mentions WHERE message_id IN (SELECT id FROM messages WHERE channel_id = ?)",
		ch.ID); err != nil {
		return fmt.Errorf("failed to delete mentions: %w", err)
	}

	// 3. Delete messages
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM messages WHERE channel_id = ?",
		ch.ID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// 4. Delete channel members
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM channel_members WHERE channel_id = ?",
		ch.ID); err != nil {
		return fmt.Errorf("failed to delete members: %w", err)
	}

	// 5. Delete the channel
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM channels WHERE id = ?",
		ch.ID); err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	// Recreate the FTS delete trigger
	_, _ = tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, content, sender) VALUES ('delete', old.id, old.content, old.sender);
		END
	`)

	// Rebuild FTS index to stay consistent after deletions
	if s.ftsAvailable {
		_, _ = tx.ExecContext(ctx, "INSERT INTO messages_fts(messages_fts) VALUES('rebuild')")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}
	return nil
}

// SetChannelDescription updates the description for a channel.
func (s *SQLiteStore) SetChannelDescription(channelName, description string) error {
	ctx := context.Background()
	result, err := s.db.ExecContext(ctx,
		"UPDATE channels SET description = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE name = ?",
		description, channelName,
	)
	if err != nil {
		return fmt.Errorf("failed to set description: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("channel %q not found", channelName)
	}
	return nil
}

// AddMember adds a member to a channel.
func (s *SQLiteStore) AddMember(channelName, agentID string) error {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return err
	}
	if ch == nil {
		return fmt.Errorf("channel %q not found", channelName)
	}

	ctx := context.Background()
	_, err = s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO channel_members (channel_id, agent_id) VALUES (?, ?)",
		ch.ID, agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}
	return nil
}

// RemoveMember removes a member from a channel.
func (s *SQLiteStore) RemoveMember(channelName, agentID string) error {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return err
	}
	if ch == nil {
		return fmt.Errorf("channel %q not found", channelName)
	}

	ctx := context.Background()
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM channel_members WHERE channel_id = ? AND agent_id = ?",
		ch.ID, agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("%q is not a member of channel %q", agentID, channelName)
	}
	return nil
}

// GetMembers returns all members of a channel.
func (s *SQLiteStore) GetMembers(channelName string) ([]string, error) {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		"SELECT agent_id FROM channel_members WHERE channel_id = ? ORDER BY agent_id",
		ch.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []string
	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		members = append(members, agentID)
	}
	return members, rows.Err()
}

// GetChannelsForAgent returns all channels an agent is a member of.
func (s *SQLiteStore) GetChannelsForAgent(agentID string) ([]*ChannelInfo, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.type, c.description, c.created_at, c.updated_at
		FROM channels c
		JOIN channel_members m ON c.id = m.channel_id
		WHERE m.agent_id = ?
		ORDER BY c.name
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channels for agent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var channels []*ChannelInfo
	for rows.Next() {
		var ch ChannelInfo
		var createdAt, updatedAt string
		var desc sql.NullString

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &desc, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		ch.Description = desc.String
		if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
			log.Warn("failed to parse channel created_at", "value", createdAt, "error", err)
		} else {
			ch.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err != nil {
			log.Warn("failed to parse channel updated_at", "value", updatedAt, "error", err)
		} else {
			ch.UpdatedAt = t
		}
		channels = append(channels, &ch)
	}
	return channels, rows.Err()
}

// AddMessage adds a message to a channel.
func (s *SQLiteStore) AddMessage(channelName, sender, content string, msgType MessageType, metadata string) (*Message, error) {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	if msgType == "" {
		msgType = TypeText
	}

	var metadataPtr *string
	if metadata != "" {
		metadataPtr = &metadata
	}

	ctx := context.Background()
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (channel_id, sender, content, type, metadata) VALUES (?, ?, ?, ?, ?)",
		ch.ID, sender, content, msgType, metadataPtr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get message insert ID: %w", err)
	}
	return s.GetMessage(id)
}

// GetMessage returns a message by ID.
func (s *SQLiteStore) GetMessage(id int64) (*Message, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		"SELECT id, channel_id, sender, content, type, metadata, created_at FROM messages WHERE id = ?",
		id,
	)

	var msg Message
	var createdAt string
	var metadata sql.NullString

	err := row.Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &metadata, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	msg.Metadata = metadata.String
	if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
		log.Warn("failed to parse message created_at", "value", createdAt, "error", err)
	} else {
		msg.CreatedAt = t
	}
	return &msg, nil
}

// GetHistory returns messages for a channel.
func (s *SQLiteStore) GetHistory(channelName string, limit int) ([]*Message, error) {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	if limit <= 0 {
		limit = 100
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, sender, content, type, metadata, created_at
		FROM messages WHERE channel_id = ? ORDER BY created_at DESC LIMIT ?
	`, ch.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var createdAt string
		var metadata sql.NullString

		if err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &metadata, &createdAt); err != nil {
			return nil, err
		}
		msg.Metadata = metadata.String
		if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
			log.Warn("failed to parse message created_at", "value", createdAt, "error", err)
		} else {
			msg.CreatedAt = t
		}
		messages = append(messages, &msg)
	}

	// Reverse for chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, rows.Err()
}

// GetMessagesByType returns messages of a specific type.
func (s *SQLiteStore) GetMessagesByType(channelName string, msgType MessageType, limit int) ([]*Message, error) {
	ch, err := s.GetChannel(channelName)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	if limit <= 0 {
		limit = 100
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, sender, content, type, metadata, created_at
		FROM messages WHERE channel_id = ? AND type = ? ORDER BY created_at DESC LIMIT ?
	`, ch.ID, msgType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var createdAt string
		var metadata sql.NullString

		if err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &metadata, &createdAt); err != nil {
			return nil, err
		}
		msg.Metadata = metadata.String
		if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
			log.Warn("failed to parse message created_at", "value", createdAt, "error", err)
		} else {
			msg.CreatedAt = t
		}
		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}

// SearchMessages performs full-text search on messages.
func (s *SQLiteStore) SearchMessages(query string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 50
	}

	ctx := context.Background()
	var rows *sql.Rows
	var err error

	if s.ftsAvailable {
		rows, err = s.db.QueryContext(ctx, `
			SELECT m.id, m.channel_id, m.sender, m.content, m.type, m.metadata, m.created_at
			FROM messages m JOIN messages_fts f ON m.id = f.rowid
			WHERE messages_fts MATCH ? ORDER BY m.created_at DESC LIMIT ?
		`, query, limit)
	} else {
		likePattern := "%" + query + "%"
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, channel_id, sender, content, type, metadata, created_at
			FROM messages WHERE content LIKE ? OR sender LIKE ?
			ORDER BY created_at DESC LIMIT ?
		`, likePattern, likePattern, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var createdAt string
		var metadata sql.NullString

		if err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.Sender, &msg.Content, &msg.Type, &metadata, &createdAt); err != nil {
			return nil, err
		}
		msg.Metadata = metadata.String
		if t, err := time.Parse(time.RFC3339, createdAt); err != nil {
			log.Warn("failed to parse message created_at", "value", createdAt, "error", err)
		} else {
			msg.CreatedAt = t
		}
		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}

// AddMention records a mention in a message.
func (s *SQLiteStore) AddMention(messageID int64, agentID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO mentions (message_id, agent_id) VALUES (?, ?)",
		messageID, agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to add mention: %w", err)
	}
	return nil
}

// GetUnreadMentions returns unacknowledged mentions for an agent.
func (s *SQLiteStore) GetUnreadMentions(agentID string) ([]*MentionRecord, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, message_id, agent_id, acknowledged FROM mentions WHERE agent_id = ? AND acknowledged = 0",
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread mentions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var mentions []*MentionRecord
	for rows.Next() {
		var m MentionRecord
		var ack int
		if err := rows.Scan(&m.ID, &m.MessageID, &m.AgentID, &ack); err != nil {
			return nil, err
		}
		m.Acknowledged = ack == 1
		mentions = append(mentions, &m)
	}
	return mentions, rows.Err()
}

// AcknowledgeMentions marks all mentions for an agent as read.
func (s *SQLiteStore) AcknowledgeMentions(agentID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		"UPDATE mentions SET acknowledged = 1 WHERE agent_id = ? AND acknowledged = 0",
		agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to acknowledge mentions: %w", err)
	}
	return nil
}

// AddReaction adds a reaction to a message.
func (s *SQLiteStore) AddReaction(messageID int64, emoji, userID string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO reactions (message_id, emoji, user_id) VALUES (?, ?, ?)",
		messageID, emoji, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	return nil
}

// RemoveReaction removes a reaction from a message.
func (s *SQLiteStore) RemoveReaction(messageID int64, emoji, userID string) error {
	ctx := context.Background()
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM reactions WHERE message_id = ? AND emoji = ? AND user_id = ?",
		messageID, emoji, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil // Reaction didn't exist, not an error
	}
	return nil
}

// ToggleReaction toggles a reaction on a message. Returns true if added, false if removed.
func (s *SQLiteStore) ToggleReaction(messageID int64, emoji, userID string) (added bool, err error) {
	ctx := context.Background()
	// Check if reaction exists
	var count int
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reactions WHERE message_id = ? AND emoji = ? AND user_id = ?",
		messageID, emoji, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check reaction: %w", err)
	}

	if count > 0 {
		// Remove existing reaction
		if err := s.RemoveReaction(messageID, emoji, userID); err != nil {
			return false, err
		}
		return false, nil
	}

	// Add new reaction
	if err := s.AddReaction(messageID, emoji, userID); err != nil {
		return false, err
	}
	return true, nil
}

// GetReactions returns all reactions for a message as emoji -> []userID.
func (s *SQLiteStore) GetReactions(messageID int64) (map[string][]string, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		"SELECT emoji, user_id FROM reactions WHERE message_id = ? ORDER BY created_at",
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string][]string)
	for rows.Next() {
		var emoji, userID string
		if err := rows.Scan(&emoji, &userID); err != nil {
			return nil, err
		}
		result[emoji] = append(result[emoji], userID)
	}
	return result, rows.Err()
}

// MigrateFromJSON migrates data from the legacy JSON store.
func (s *SQLiteStore) MigrateFromJSON(jsonPath string) error {
	data, err := os.ReadFile(jsonPath) //nolint:gosec // path provided by caller
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var channels []*Channel
	if unmarshalErr := json.Unmarshal(data, &channels); unmarshalErr != nil {
		return fmt.Errorf("failed to parse JSON: %w", unmarshalErr)
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, ch := range channels {
		result, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO channels (name, type) VALUES (?, ?)",
			ch.Name, ChannelTypeGroup,
		)
		if err != nil {
			return fmt.Errorf("failed to insert channel %s: %w", ch.Name, err)
		}

		var channelID int64
		id, idErr := result.LastInsertId()
		if idErr != nil {
			return fmt.Errorf("failed to get channel insert ID during migration: %w", idErr)
		}
		if id > 0 {
			channelID = id
		} else {
			row := tx.QueryRowContext(ctx, "SELECT id FROM channels WHERE name = ?", ch.Name)
			if err := row.Scan(&channelID); err != nil {
				return fmt.Errorf("failed to get channel ID: %w", err)
			}
		}

		for _, member := range ch.Members {
			if _, err := tx.ExecContext(ctx,
				"INSERT OR IGNORE INTO channel_members (channel_id, agent_id) VALUES (?, ?)",
				channelID, member,
			); err != nil {
				return fmt.Errorf("failed to insert member %s: %w", member, err)
			}
		}

		for _, entry := range ch.History {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO messages (channel_id, sender, content, type, created_at) VALUES (?, ?, ?, ?, ?)",
				channelID, entry.Sender, entry.Message, TypeText, entry.Time.Format(time.RFC3339),
			); err != nil {
				return fmt.Errorf("failed to insert message: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	if err := os.Rename(jsonPath, jsonPath+".migrated"); err != nil {
		fmt.Printf("Warning: could not rename %s: %v\n", jsonPath, err)
	}
	return nil
}
