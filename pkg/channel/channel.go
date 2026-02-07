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

// MessageType defines the type of message for routing and filtering.
type MessageType string

const (
	// TypeMessage is a regular chat message (default).
	TypeMessage MessageType = "message"
	// TypeTask is a work assignment.
	TypeTask MessageType = "task"
	// TypeReview is a PR review request.
	TypeReview MessageType = "review"
	// TypeApproval is a PR approval notification.
	TypeApproval MessageType = "approval"
	// TypeMerge is a merge request.
	TypeMerge MessageType = "merge"
)

// ValidMessageTypes returns all valid message types.
func ValidMessageTypes() []MessageType {
	return []MessageType{TypeMessage, TypeTask, TypeReview, TypeApproval, TypeMerge}
}

// IsValidMessageType checks if a string is a valid message type.
func IsValidMessageType(t string) bool {
	for _, valid := range ValidMessageTypes() {
		if string(valid) == t {
			return true
		}
	}
	return false
}

// HistoryEntry represents a message in channel history.
type HistoryEntry struct {
	Time    time.Time   `json:"time"`
	Sender  string      `json:"sender,omitempty"`
	Message string      `json:"message"`
	Type    MessageType `json:"type,omitempty"`
}

// Channel represents a named communication channel with a list of members.
type Channel struct {
	Name    string         `json:"name"`
	Members []string       `json:"members"`
	History []HistoryEntry `json:"history,omitempty"`
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

// List returns all channels.
func (s *Store) List() []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}
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
// Deprecated: Use AddHistoryWithType instead.
func (s *Store) AddHistory(channelName, sender, message string) error {
	return s.AddHistoryWithType(channelName, sender, message, TypeMessage)
}

// AddHistoryWithType adds a typed message to the channel's history.
func (s *Store) AddHistoryWithType(channelName, sender, message string, msgType MessageType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}

	// Default to TypeMessage if empty
	if msgType == "" {
		msgType = TypeMessage
	}

	entry := HistoryEntry{
		Time:    time.Now(),
		Sender:  sender,
		Message: message,
		Type:    msgType,
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

// GetHistoryByType returns messages filtered by type.
func (s *Store) GetHistoryByType(channelName string, msgType MessageType) ([]HistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	// Filter by type
	filtered := make([]HistoryEntry, 0)
	for _, entry := range ch.History {
		// Match type (empty type matches TypeMessage for backward compatibility)
		entryType := entry.Type
		if entryType == "" {
			entryType = TypeMessage
		}
		if entryType == msgType {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}
