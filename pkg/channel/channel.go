// Package channel provides a channels system for broadcasting messages to groups of agents.
//
// Channels are named groups of agent members. Messages sent to a channel are
// delivered to all member tmux sessions.
package channel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/rpuneet/bc/pkg/db"
	"github.com/rpuneet/bc/pkg/log"
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
// When backend is non-nil (SQLite or Postgres), all operations use it; otherwise JSON (.bc/channels.json).
type Store struct {
	channels map[string]*Channel
	backend  ChannelBackend // SQLiteStore or PostgresStore; nil = JSON fallback
	path     string
	mu       sync.RWMutex
}

// NewStore creates a new channel store for the given workspace (JSON backend only).
func NewStore(workspacePath string) *Store {
	return &Store{
		path:     filepath.Join(workspacePath, ".bc", "channels.json"),
		channels: make(map[string]*Channel),
	}
}

// OpenStore opens the channel store for the workspace.
// Priority: DATABASE_URL (Postgres) > .bc/channels.db (SQLite) > JSON fallback.
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
				return &Store{
					path:     filepath.Join(workspacePath, ".bc", "channels.json"),
					channels: make(map[string]*Channel),
					backend:  pg,
				}, nil
			}
		}
	}

	// Fall back to SQLite if .bc/channels.db exists
	dbPath := filepath.Join(workspacePath, ".bc", "bc.db")
	if _, err := os.Stat(dbPath); err == nil {
		s := NewSQLiteStore(workspacePath)
		if err := s.Open(); err != nil {
			return nil, fmt.Errorf("open channel store: %w", err)
		}
		return &Store{
			path:     filepath.Join(workspacePath, ".bc", "channels.json"),
			channels: make(map[string]*Channel),
			backend:  s,
		}, nil
	}
	return NewStore(workspacePath), nil
}

// Load reads channels from disk. When using SQLite backend, Load is a no-op (data read on demand).
func (s *Store) Load() error {
	if s.backend != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.channels = make(map[string]*Channel)
			return nil
		}
		return fmt.Errorf("failed to read channels file: %w", err)
	}

	var channels []*Channel
	if err := json.Unmarshal(data, &channels); err != nil {
		return fmt.Errorf("failed to parse channels file: %w", err)
	}

	s.channels = make(map[string]*Channel)
	for _, ch := range channels {
		s.channels[ch.Name] = ch
	}

	return nil
}

// Save writes channels to disk. When using SQLite backend, Save is a no-op (writes are immediate).
func (s *Store) Save() error {
	if s.backend != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice for JSON serialization
	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}

	// Sort by name for stable file output
	slices.SortFunc(channels, func(a, b *Channel) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	data, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channels: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return fmt.Errorf("failed to create channels directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write channels file: %w", err)
	}

	return nil
}

// Close closes the underlying database connection.
// Safe to call on JSON-backed stores (no-op).
func (s *Store) Close() error {
	if s.backend != nil {
		return s.backend.Close()
	}
	return nil
}

// Create creates a new channel with the given name.
func (s *Store) Create(name string) (*Channel, error) {
	if s.backend != nil {
		info, err := s.backend.CreateChannel(name, ChannelTypeGroup, "")
		if err != nil {
			return nil, err
		}
		return sqliteToChannel(info, nil, nil), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.channels[name]; exists {
		return nil, fmt.Errorf("channel %q already exists", name)
	}
	ch := &Channel{Name: name, Members: []string{}}
	s.channels[name] = ch
	return ch, nil
}

// Get returns a channel by name.
func (s *Store) Get(name string) (*Channel, bool) {
	if s.backend != nil {
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.channels[name]
	return ch, exists
}

// List returns all channels sorted by name for stable ordering.
func (s *Store) List() []*Channel {
	if s.backend != nil {
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}
	slices.SortFunc(channels, func(a, b *Channel) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return channels
}

// Delete removes a channel by name.
func (s *Store) Delete(name string) error {
	if s.backend != nil {
		return s.backend.DeleteChannel(name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.channels[name]; !exists {
		return fmt.Errorf("channel %q not found", name)
	}
	delete(s.channels, name)
	return nil
}

// AddMember adds a member to a channel.
func (s *Store) AddMember(channelName, member string) error {
	if s.backend != nil {
		return s.backend.AddMember(channelName, member)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}
	if slices.Contains(ch.Members, member) {
		return fmt.Errorf("%q is already a member of channel %q", member, channelName)
	}
	ch.Members = append(ch.Members, member)
	return nil
}

// RemoveMember removes a member from a channel.
func (s *Store) RemoveMember(channelName, member string) error {
	if s.backend != nil {
		return s.backend.RemoveMember(channelName, member)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}
	idx := slices.Index(ch.Members, member)
	if idx == -1 {
		return fmt.Errorf("%q is not a member of channel %q", member, channelName)
	}
	ch.Members = slices.Delete(ch.Members, idx, idx+1)
	return nil
}

// GetMembers returns the members of a channel.
func (s *Store) GetMembers(channelName string) ([]string, error) {
	if s.backend != nil {
		return s.backend.GetMembers(channelName)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}
	members := make([]string, len(ch.Members))
	copy(members, ch.Members)
	return members, nil
}

// AddHistory adds a message to the channel's history.
func (s *Store) AddHistory(channelName, sender, message string) error {
	if s.backend != nil {
		_, err := s.backend.AddMessage(channelName, sender, message, TypeText, "")
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}
	entry := HistoryEntry{
		Time:    time.Now(),
		Sender:  sender,
		Message: message,
	}
	ch.History = append(ch.History, entry)
	if len(ch.History) > 100 {
		ch.History = ch.History[len(ch.History)-100:]
	}
	return nil
}

// GetHistory returns the message history for a channel.
func (s *Store) GetHistory(channelName string) ([]HistoryEntry, error) {
	if s.backend != nil {
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}
	history := make([]HistoryEntry, len(ch.History))
	copy(history, ch.History)
	return history, nil
}

// SetDescription sets the description for a channel.
func (s *Store) SetDescription(channelName, description string) error {
	if s.backend != nil {
		return s.backend.SetChannelDescription(channelName, description)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}
	ch.Description = description
	return nil
}

// GetDescription returns the description for a channel.
func (s *Store) GetDescription(channelName string) (string, error) {
	if s.backend != nil {
		info, err := s.backend.GetChannel(channelName)
		if err != nil || info == nil {
			return "", fmt.Errorf("channel %q not found", channelName)
		}
		return info.Description, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return "", fmt.Errorf("channel %q not found", channelName)
	}
	return ch.Description, nil
}

// CommonReactions provides a set of commonly used emoji reactions.
var CommonReactions = []string{"👍", "👎", "❤️", "🎉", "👀", "🚀"}

// AddReaction adds an emoji reaction to a message.
// The messageIndex is the index into the channel's history.
func (s *Store) AddReaction(channelName string, messageIndex int, emoji, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}

	if messageIndex < 0 || messageIndex >= len(ch.History) {
		return fmt.Errorf("message index %d out of range", messageIndex)
	}

	entry := &ch.History[messageIndex]
	if entry.Reactions == nil {
		entry.Reactions = make(map[string][]string)
	}

	// Check if user already reacted with this emoji
	users := entry.Reactions[emoji]
	if slices.Contains(users, user) {
		return nil // Already reacted
	}

	entry.Reactions[emoji] = append(users, user)
	return nil
}

// RemoveReaction removes an emoji reaction from a message.
func (s *Store) RemoveReaction(channelName string, messageIndex int, emoji, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}

	if messageIndex < 0 || messageIndex >= len(ch.History) {
		return fmt.Errorf("message index %d out of range", messageIndex)
	}

	entry := &ch.History[messageIndex]
	if entry.Reactions == nil {
		return nil // No reactions
	}

	users := entry.Reactions[emoji]
	idx := slices.Index(users, user)
	if idx == -1 {
		return nil // User hasn't reacted
	}

	entry.Reactions[emoji] = slices.Delete(users, idx, idx+1)

	// Clean up empty reaction
	if len(entry.Reactions[emoji]) == 0 {
		delete(entry.Reactions, emoji)
	}

	return nil
}

// ToggleReaction toggles an emoji reaction on a message.
// Returns true if the reaction was added, false if removed.
func (s *Store) ToggleReaction(channelName string, messageIndex int, emoji, user string) (added bool, err error) {
	if s.backend != nil {
		msgs, err := s.backend.GetHistory(channelName, 100)
		if err != nil {
			return false, err
		}
		if messageIndex < 0 || messageIndex >= len(msgs) {
			return false, fmt.Errorf("message index %d out of range", messageIndex)
		}
		return s.backend.ToggleReaction(msgs[messageIndex].ID, emoji, user)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return false, fmt.Errorf("channel %q not found", channelName)
	}

	if messageIndex < 0 || messageIndex >= len(ch.History) {
		return false, fmt.Errorf("message index %d out of range", messageIndex)
	}

	entry := &ch.History[messageIndex]
	if entry.Reactions == nil {
		entry.Reactions = make(map[string][]string)
	}

	users := entry.Reactions[emoji]
	idx := slices.Index(users, user)

	if idx == -1 {
		// Add reaction
		entry.Reactions[emoji] = append(users, user)
		return true, nil
	}

	// Remove reaction
	entry.Reactions[emoji] = slices.Delete(users, idx, idx+1)
	if len(entry.Reactions[emoji]) == 0 {
		delete(entry.Reactions, emoji)
	}
	return false, nil
}

// GetReactions returns all reactions for a message.
func (s *Store) GetReactions(channelName string, messageIndex int) (map[string][]string, error) {
	if s.backend != nil {
		msgs, err := s.backend.GetHistory(channelName, 100)
		if err != nil {
			return nil, err
		}
		if messageIndex < 0 || messageIndex >= len(msgs) {
			return nil, fmt.Errorf("message index %d out of range", messageIndex)
		}
		return s.backend.GetReactions(msgs[messageIndex].ID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	if messageIndex < 0 || messageIndex >= len(ch.History) {
		return nil, fmt.Errorf("message index %d out of range", messageIndex)
	}

	entry := ch.History[messageIndex]
	if entry.Reactions == nil {
		return nil, nil
	}

	// Return a copy
	result := make(map[string][]string)
	for emoji, users := range entry.Reactions {
		usersCopy := make([]string, len(users))
		copy(usersCopy, users)
		result[emoji] = usersCopy
	}
	return result, nil
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
