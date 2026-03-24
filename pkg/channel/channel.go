// Package channel provides a channels system for broadcasting messages to groups of agents.
//
// Channels are named groups of agent members. Messages sent to a channel are
// delivered to all member tmux sessions.
package channel

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gh-curious-otter/bc/pkg/db"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// HistoryEntry represents a message in channel history.
type HistoryEntry struct {
	Reactions map[string][]string `json:"reactions,omitempty"` // emoji -> list of users
	Time      time.Time           `json:"time"`
	Sender    string              `json:"sender,omitempty"`
	Message   string              `json:"message"`
}

// Channel represents a named communication channel with a list of members.
type Channel struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Members     []string       `json:"members"`
	History     []HistoryEntry `json:"history,omitempty"`
}

// Store manages channel persistence and operations.
// All operations are delegated to the backend (SQLite or Postgres).
type Store struct {
	backend ChannelBackend // SQLiteStore or PostgresStore
}

// NewStore creates a new channel store backed by SQLite for the given workspace.
// It creates the .bc directory and bc.db file if they do not exist.
func NewStore(workspacePath string) *Store {
	bcDir := filepath.Join(workspacePath, ".bc")
	_ = os.MkdirAll(bcDir, 0750) //nolint:errcheck // best-effort dir creation

	sqlite := NewSQLiteStore(workspacePath)
	if err := sqlite.Open(); err != nil {
		log.Warn("failed to open SQLite channel store in NewStore", "error", err)
		// Return a store with backend set; operations will fail with clear errors.
		return &Store{backend: sqlite}
	}
	return &Store{backend: sqlite}
}

// OpenStore opens the channel store for the workspace.
// Priority: DATABASE_URL (Postgres) > SQLite (.bc/bc.db).
func OpenStore(workspacePath string) (*Store, error) {
	// Try Postgres first when DATABASE_URL is set
	if db.IsPostgresEnabled() {
		pgDB, err := db.TryOpenPostgres()
		if err != nil {
			log.Warn("failed to connect to Postgres, falling back to SQLite", "error", err)
		} else if pgDB != nil {
			pg := NewPostgresStore(pgDB)
			if schemaErr := pg.InitSchema(); schemaErr != nil {
				_ = pg.Close()
				log.Warn("failed to init Postgres channel schema, falling back to SQLite", "error", schemaErr)
			} else {
				log.Debug("channel store: using Postgres backend")
				return &Store{backend: pg}, nil
			}
		}
	}

	// SQLite backend
	s := NewSQLiteStore(workspacePath)
	if err := s.Open(); err != nil {
		return nil, fmt.Errorf("open channel store: %w", err)
	}
	return &Store{backend: s}, nil
}

// Load is a no-op retained for backward compatibility. Data is read on demand from the backend.
func (s *Store) Load() error {
	return nil
}

// Save is a no-op retained for backward compatibility. Writes are immediate in the backend.
func (s *Store) Save() error {
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}

// Create creates a new channel with the given name.
func (s *Store) Create(name string) (*Channel, error) {
	info, err := s.backend.CreateChannel(name, ChannelTypeGroup, "")
	if err != nil {
		return nil, err
	}
	return sqliteToChannel(info, nil, nil), nil
}

// Get returns a channel by name.
func (s *Store) Get(name string) (*Channel, bool) {
	info, err := s.backend.GetChannel(name)
	if err != nil || info == nil {
		return nil, false
	}
	members, err := s.backend.GetMembers(name)
	if err != nil {
		log.Warn("failed to get channel members", "channel", name, "error", err)
	}
	msgs, err := s.backend.GetHistory(name, 100)
	if err != nil {
		log.Warn("failed to get channel history", "channel", name, "error", err)
	}
	ch := sqliteToChannel(info, members, msgs)
	return ch, true
}

// List returns all channels sorted by name for stable ordering.
func (s *Store) List() []*Channel {
	infos, err := s.backend.ListChannels()
	if err != nil {
		return nil
	}
	out := make([]*Channel, 0, len(infos))
	for _, info := range infos {
		members, err := s.backend.GetMembers(info.Name)
		if err != nil {
			log.Warn("failed to get channel members", "channel", info.Name, "error", err)
		}
		msgs, err := s.backend.GetHistory(info.Name, 100)
		if err != nil {
			log.Warn("failed to get channel history", "channel", info.Name, "error", err)
		}
		out = append(out, sqliteToChannel(info, members, msgs))
	}
	return out
}

// Delete removes a channel by name.
func (s *Store) Delete(name string) error {
	return s.backend.DeleteChannel(name)
}

// AddMember adds a member to a channel.
func (s *Store) AddMember(channelName, member string) error {
	return s.backend.AddMember(channelName, member)
}

// RemoveMember removes a member from a channel.
func (s *Store) RemoveMember(channelName, member string) error {
	return s.backend.RemoveMember(channelName, member)
}

// GetMembers returns the members of a channel.
func (s *Store) GetMembers(channelName string) ([]string, error) {
	return s.backend.GetMembers(channelName)
}

// AddHistory adds a message to the channel's history.
func (s *Store) AddHistory(channelName, sender, message string) error {
	_, err := s.backend.AddMessage(channelName, sender, message, TypeText, "")
	return err
}

// GetHistory returns the message history for a channel.
func (s *Store) GetHistory(channelName string) ([]HistoryEntry, error) {
	msgs, err := s.backend.GetHistory(channelName, 100)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, 0, len(msgs))
	for _, m := range msgs {
		entry := HistoryEntry{Time: m.CreatedAt, Sender: m.Sender, Message: m.Content}
		// Fetch reactions for this message
		if reactions, reactErr := s.backend.GetReactions(m.ID); reactErr == nil && len(reactions) > 0 {
			entry.Reactions = reactions
		}
		out = append(out, entry)
	}
	return out, nil
}

// SetDescription sets the description for a channel.
func (s *Store) SetDescription(channelName, description string) error {
	return s.backend.SetChannelDescription(channelName, description)
}

// GetDescription returns the description for a channel.
func (s *Store) GetDescription(channelName string) (string, error) {
	info, err := s.backend.GetChannel(channelName)
	if err != nil || info == nil {
		return "", fmt.Errorf("channel %q not found", channelName)
	}
	return info.Description, nil
}

// CommonReactions provides a set of commonly used emoji reactions.
var CommonReactions = []string{"👍", "👎", "❤️", "🎉", "👀", "🚀"}

// AddReaction adds an emoji reaction to a message.
// The messageIndex is the index into the channel's history.
func (s *Store) AddReaction(channelName string, messageIndex int, emoji, user string) error {
	msgs, err := s.backend.GetHistory(channelName, 100)
	if err != nil {
		return err
	}
	if messageIndex < 0 || messageIndex >= len(msgs) {
		return fmt.Errorf("message index %d out of range", messageIndex)
	}
	return s.backend.AddReaction(msgs[messageIndex].ID, emoji, user)
}

// RemoveReaction removes an emoji reaction from a message.
func (s *Store) RemoveReaction(channelName string, messageIndex int, emoji, user string) error {
	msgs, err := s.backend.GetHistory(channelName, 100)
	if err != nil {
		return err
	}
	if messageIndex < 0 || messageIndex >= len(msgs) {
		return fmt.Errorf("message index %d out of range", messageIndex)
	}
	return s.backend.RemoveReaction(msgs[messageIndex].ID, emoji, user)
}

// ToggleReaction toggles an emoji reaction on a message.
// Returns true if the reaction was added, false if removed.
func (s *Store) ToggleReaction(channelName string, messageIndex int, emoji, user string) (added bool, err error) {
	msgs, err := s.backend.GetHistory(channelName, 100)
	if err != nil {
		return false, err
	}
	if messageIndex < 0 || messageIndex >= len(msgs) {
		return false, fmt.Errorf("message index %d out of range", messageIndex)
	}
	return s.backend.ToggleReaction(msgs[messageIndex].ID, emoji, user)
}

// GetReactions returns all reactions for a message.
func (s *Store) GetReactions(channelName string, messageIndex int) (map[string][]string, error) {
	msgs, err := s.backend.GetHistory(channelName, 100)
	if err != nil {
		return nil, err
	}
	if messageIndex < 0 || messageIndex >= len(msgs) {
		return nil, fmt.Errorf("message index %d out of range", messageIndex)
	}
	return s.backend.GetReactions(msgs[messageIndex].ID)
}

// sqliteToChannel builds a *Channel from SQLite data for use by Store callers.
func sqliteToChannel(info *ChannelInfo, members []string, msgs []*Message) *Channel {
	if members == nil {
		members = []string{}
	}
	ch := &Channel{
		Name:        info.Name,
		Description: info.Description,
		Members:     members,
		History:     make([]HistoryEntry, 0, len(msgs)),
	}
	for _, m := range msgs {
		ch.History = append(ch.History, HistoryEntry{
			Time:    m.CreatedAt,
			Sender:  m.Sender,
			Message: m.Content,
		})
	}
	return ch
}
