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
type Store struct {
	channels map[string]*Channel
	path     string
	mu       sync.RWMutex
}

// NewStore creates a new channel store for the given workspace.
func NewStore(workspacePath string) *Store {
	return &Store{
		path:     filepath.Join(workspacePath, ".bc", "channels.json"),
		channels: make(map[string]*Channel),
	}
}

// Load reads channels from disk.
func (s *Store) Load() error {
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

// Save writes channels to disk.
func (s *Store) Save() error {
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

// Create creates a new channel with the given name.
func (s *Store) Create(name string) (*Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.channels[name]; exists {
		return nil, fmt.Errorf("channel %q already exists", name)
	}

	ch := &Channel{
		Name:    name,
		Members: []string{},
	}
	s.channels[name] = ch

	return ch, nil
}

// Get returns a channel by name.
func (s *Store) Get(name string) (*Channel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[name]
	return ch, exists
}

// List returns all channels sorted by name for stable ordering.
func (s *Store) List() []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}

	// Sort by name for stable ordering in UI
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
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}

	// Check if already a member
	if slices.Contains(ch.Members, member) {
		return fmt.Errorf("%q is already a member of channel %q", member, channelName)
	}

	ch.Members = append(ch.Members, member)
	return nil
}

// RemoveMember removes a member from a channel.
func (s *Store) RemoveMember(channelName, member string) error {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	// Return a copy to prevent external modification
	members := make([]string, len(ch.Members))
	copy(members, ch.Members)
	return members, nil
}

// AddHistory adds a message to the channel's history.
func (s *Store) AddHistory(channelName, sender, message string) error {
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

	// Keep only the last 100 messages
	if len(ch.History) > 100 {
		ch.History = ch.History[len(ch.History)-100:]
	}

	return nil
}

// GetHistory returns the message history for a channel.
func (s *Store) GetHistory(channelName string) ([]HistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	// Return a copy to prevent external modification
	history := make([]HistoryEntry, len(ch.History))
	copy(history, ch.History)
	return history, nil
}

// SetDescription sets the description for a channel.
func (s *Store) SetDescription(channelName, description string) error {
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
