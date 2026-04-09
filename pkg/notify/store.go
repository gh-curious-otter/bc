package notify

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gh-curious-otter/bc/pkg/db"
)

// Store is the SQLite/Postgres-backed persistence layer for subscriptions
// and the delivery log. Uses the shared workspace database.
type Store struct {
	db *db.DB
}

// OpenStore opens the notify store using the shared workspace database.
func OpenStore(workspacePath string) (*Store, error) {
	shared := db.SharedWrapped()
	if shared == nil {
		return nil, fmt.Errorf("notify store requires shared database (none available for workspace %s)", workspacePath)
	}
	s := &Store{db: shared}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("init notify schema: %w", err)
	}
	return s, nil
}

// Close is a no-op — the shared DB is owned by the caller.
func (s *Store) Close() error { return nil }

func (s *Store) initSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS notify_subscriptions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    channel      TEXT NOT NULL,
    agent        TEXT NOT NULL,
    mention_only INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(channel, agent)
);

CREATE TABLE IF NOT EXISTS notify_delivery_log (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    logged_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    channel   TEXT NOT NULL,
    agent     TEXT NOT NULL,
    status    TEXT NOT NULL CHECK(status IN ('delivered', 'failed', 'pending')),
    error     TEXT,
    preview   TEXT
);

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
`
	_, err := s.db.ExecContext(context.TODO(), schema)
	return err
}

// Subscribe adds an agent to a channel. If already subscribed, this is a no-op.
func (s *Store) Subscribe(ctx context.Context, channel, agent string, mentionOnly bool) error {
	mentionInt := 0
	if mentionOnly {
		mentionInt = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notify_subscriptions (channel, agent, mention_only)
		 VALUES (?, ?, ?)
		 ON CONFLICT(channel, agent) DO UPDATE SET mention_only = excluded.mention_only`,
		channel, agent, mentionInt)
	return err
}

// Unsubscribe removes an agent from a channel.
func (s *Store) Unsubscribe(ctx context.Context, channel, agent string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM notify_subscriptions WHERE channel = ? AND agent = ?`,
		channel, agent)
	return err
}

// SetMentionOnly updates the mention_only flag for a subscription.
func (s *Store) SetMentionOnly(ctx context.Context, channel, agent string, mentionOnly bool) error {
	mentionInt := 0
	if mentionOnly {
		mentionInt = 1
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE notify_subscriptions SET mention_only = ? WHERE channel = ? AND agent = ?`,
		mentionInt, channel, agent)
	return err
}

// Subscribers returns all subscriptions for a channel.
func (s *Store) Subscribers(ctx context.Context, channel string) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, channel, agent, mention_only, created_at FROM notify_subscriptions WHERE channel = ?`,
		channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var mentionInt int
		var createdStr string
		if err := rows.Scan(&sub.ID, &sub.Channel, &sub.Agent, &mentionInt, &createdStr); err != nil {
			return nil, err
		}
		sub.MentionOnly = mentionInt != 0
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// AllSubscriptions returns all subscriptions across all channels.
func (s *Store) AllSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, channel, agent, mention_only, created_at FROM notify_subscriptions ORDER BY channel, agent`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var mentionInt int
		var createdStr string
		if err := rows.Scan(&sub.ID, &sub.Channel, &sub.Agent, &mentionInt, &createdStr); err != nil {
			return nil, err
		}
		sub.MentionOnly = mentionInt != 0
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// LogDelivery records a delivery attempt.
func (s *Store) LogDelivery(ctx context.Context, e DeliveryEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notify_delivery_log (channel, agent, status, error, preview)
		 VALUES (?, ?, ?, ?, ?)`,
		e.Channel, e.Agent, string(e.Status), e.Error, e.Preview)
	return err
}

// RecentActivity returns the most recent delivery log entries for a channel.
func (s *Store) RecentActivity(ctx context.Context, channel string, limit int) ([]DeliveryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, logged_at, channel, agent, status, COALESCE(error, ''), COALESCE(preview, '')
		 FROM notify_delivery_log
		 WHERE channel = ?
		 ORDER BY id DESC
		 LIMIT ?`,
		channel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []DeliveryEntry
	for rows.Next() {
		var e DeliveryEntry
		var loggedStr string
		if err := rows.Scan(&e.ID, &loggedStr, &e.Channel, &e.Agent, &e.Status, &e.Error, &e.Preview); err != nil {
			return nil, err
		}
		e.LoggedAt, _ = time.Parse(time.RFC3339, loggedStr)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PruneActivity deletes old delivery log entries, keeping the most recent keepLast per channel.
func (s *Store) PruneActivity(ctx context.Context, channel string, keepLast int) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM notify_delivery_log
		 WHERE channel = ? AND id NOT IN (
		     SELECT id FROM notify_delivery_log WHERE channel = ? ORDER BY id DESC LIMIT ?
		 )`,
		channel, channel, keepLast)
	return err
}

// UpsertGateway inserts or updates a gateway record.
func (s *Store) UpsertGateway(ctx context.Context, name string, enabled, connected bool) error {
	enabledInt, connectedInt := 0, 0
	if enabled {
		enabledInt = 1
	}
	if connected {
		connectedInt = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notify_gateways (name, enabled, connected)
		 VALUES (?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		     enabled = excluded.enabled,
		     connected = excluded.connected,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		name, enabledInt, connectedInt)
	return err
}

// SetGatewayConnected updates the connected status and last_seen_at.
func (s *Store) SetGatewayConnected(ctx context.Context, name string, connected bool) error {
	connectedInt := 0
	if connected {
		connectedInt = 1
	}
	var lastSeen sql.NullString
	if connected {
		lastSeen = sql.NullString{String: time.Now().UTC().Format(time.RFC3339), Valid: true}
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE notify_gateways SET connected = ?, last_seen_at = COALESCE(?, last_seen_at),
		 updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE name = ?`,
		connectedInt, lastSeen, name)
	return err
}

// ListGateways returns all registered gateways.
func (s *Store) ListGateways(ctx context.Context) ([]GatewayInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, enabled, connected, last_seen_at, updated_at FROM notify_gateways ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []GatewayInfo
	for rows.Next() {
		var g GatewayInfo
		var enabledInt, connectedInt int
		var lastSeenStr, updatedStr sql.NullString
		if err := rows.Scan(&g.Name, &enabledInt, &connectedInt, &lastSeenStr, &updatedStr); err != nil {
			return nil, err
		}
		g.Enabled = enabledInt != 0
		g.Connected = connectedInt != 0
		if lastSeenStr.Valid {
			t, _ := time.Parse(time.RFC3339, lastSeenStr.String)
			g.LastSeenAt = &t
		}
		if updatedStr.Valid {
			g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr.String)
		}
		gateways = append(gateways, g)
	}
	return gateways, rows.Err()
}
