package channel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// channelNameRegex matches valid channel names: alphanumeric, hyphens, underscores,
// must start with an alphanumeric character.
var channelNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ErrChannelExists is returned when attempting to create a channel that already exists.
var ErrChannelExists = errors.New("channel already exists")

// ErrInvalidChannelName is returned when a channel name contains invalid characters.
var ErrInvalidChannelName = errors.New("invalid channel name: must start with alphanumeric and contain only alphanumeric, hyphens, or underscores")

// IsValidChannelName validates that a channel name contains only allowed characters.
func IsValidChannelName(name string) bool {
	return channelNameRegex.MatchString(name)
}

// ChannelDTO is the API representation of a channel.
type ChannelDTO struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Type         string    `json:"type"`
	Members      []string  `json:"members"`
	MemberCount  int       `json:"member_count"`
	MessageCount int       `json:"message_count"`
}

// MessageDTO is the API representation of a channel message.
type MessageDTO struct {
	Reactions map[string][]string `json:"reactions,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
	Channel   string              `json:"channel"`
	Sender    string              `json:"sender"`
	Content   string              `json:"content"`
	Type      string              `json:"type"`
	Metadata  string              `json:"metadata,omitempty"`
	ID        int64               `json:"id"`
}

// CreateChannelReq is the request to create a channel.
type CreateChannelReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateChannelReq is the request to update a channel.
type UpdateChannelReq struct {
	Description string `json:"description,omitempty"`
}

// HistoryOpts configures message history retrieval.
type HistoryOpts struct {
	Since  *time.Time `json:"since,omitempty"`
	Agent  string     `json:"agent,omitempty"`
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
}

// ChannelService encapsulates channel business logic, wrapping the Store.
// It provides the service layer between CLI/API handlers and storage,
// enforcing validation and returning DTOs.
type ChannelService struct {
	store     *Store
	OnMessage func(channel, sender, content string) // called after a message is stored
}

// NewChannelService creates a ChannelService backed by the given Store.
func NewChannelService(store *Store) *ChannelService {
	return &ChannelService{store: store}
}

// Store returns the underlying channel store.
func (s *ChannelService) Store() *Store {
	return s.store
}

// List returns all channels as DTOs.
func (s *ChannelService) List(_ context.Context) ([]ChannelDTO, error) {
	channels := s.store.List()
	dtos := make([]ChannelDTO, 0, len(channels))
	for _, ch := range channels {
		dto := channelToDTO(ch)
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

// Create creates a new channel and returns its DTO.
func (s *ChannelService) Create(_ context.Context, req CreateChannelReq) (*ChannelDTO, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("channel name is required")
	}

	if !IsValidChannelName(req.Name) {
		return nil, ErrInvalidChannelName
	}

	// Check for duplicate channel name
	if _, exists := s.store.Get(req.Name); exists {
		return nil, fmt.Errorf("%w: %q", ErrChannelExists, req.Name)
	}

	ch, err := s.store.Create(req.Name)
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	if req.Description != "" {
		if err := s.store.SetDescription(req.Name, req.Description); err != nil {
			return nil, fmt.Errorf("set description: %w", err)
		}
		ch.Description = req.Description
	}

	if err := s.store.Save(); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	dto := channelToDTO(ch)
	return &dto, nil
}

// Get returns a single channel detail with members.
func (s *ChannelService) Get(_ context.Context, name string) (*ChannelDTO, error) {
	ch, exists := s.store.Get(name)
	if !exists {
		return nil, fmt.Errorf("channel %q not found", name)
	}
	dto := channelToDTO(ch)
	return &dto, nil
}

// Update modifies a channel's settings (description).
func (s *ChannelService) Update(_ context.Context, name string, req UpdateChannelReq) (*ChannelDTO, error) {
	ch, exists := s.store.Get(name)
	if !exists {
		return nil, fmt.Errorf("channel %q not found", name)
	}

	if req.Description != "" {
		if err := s.store.SetDescription(name, req.Description); err != nil {
			return nil, fmt.Errorf("set description: %w", err)
		}
		ch.Description = req.Description
	}

	if err := s.store.Save(); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	dto := channelToDTO(ch)
	return &dto, nil
}

// Delete removes a channel.
func (s *ChannelService) Delete(_ context.Context, name string) error {
	if err := s.store.Delete(name); err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	return s.store.Save()
}

// AddMember adds an agent to a channel.
func (s *ChannelService) AddMember(_ context.Context, ch, agentID string) error {
	if err := s.store.AddMember(ch, agentID); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return s.store.Save()
}

// RemoveMember removes an agent from a channel.
func (s *ChannelService) RemoveMember(_ context.Context, ch, agentID string) error {
	if err := s.store.RemoveMember(ch, agentID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return s.store.Save()
}

// Send adds a message to a channel and returns the message DTO.
func (s *ChannelService) Send(_ context.Context, ch, sender, content string) (*MessageDTO, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("message content is required")
	}
	if strings.TrimSpace(sender) == "" {
		sender = "anonymous"
	}

	if err := s.store.AddHistory(ch, sender, content); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	if err := s.store.Save(); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	dto := &MessageDTO{
		Channel:   ch,
		Sender:    sender,
		Content:   content,
		Type:      string(TypeText),
		CreatedAt: time.Now(),
	}

	if s.OnMessage != nil {
		s.OnMessage(ch, sender, content)
	}

	return dto, nil
}

// History retrieves filtered message history for a channel.
func (s *ChannelService) History(_ context.Context, ch string, opts HistoryOpts) ([]MessageDTO, error) {
	history, err := s.store.GetHistory(ch)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	// Apply filters
	filtered := make([]HistoryEntry, 0, len(history))
	for _, entry := range history {
		if opts.Since != nil && entry.Time.Before(*opts.Since) {
			continue
		}
		if opts.Agent != "" && entry.Sender != opts.Agent {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Apply offset
	if opts.Offset > 0 {
		if opts.Offset >= len(filtered) {
			filtered = nil
		} else {
			filtered = filtered[opts.Offset:]
		}
	}

	// Apply limit (take last N messages)
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	dtos := make([]MessageDTO, 0, len(filtered))
	for _, entry := range filtered {
		dtos = append(dtos, MessageDTO{
			Channel:   ch,
			Sender:    entry.Sender,
			Content:   entry.Message,
			Type:      string(TypeText),
			CreatedAt: entry.Time,
			Reactions: entry.Reactions,
		})
	}
	return dtos, nil
}

// React toggles an emoji reaction on a message. Returns true if added.
func (s *ChannelService) React(_ context.Context, ch string, msgID int, emoji, user string) (bool, error) {
	added, err := s.store.ToggleReaction(ch, msgID, emoji, user)
	if err != nil {
		return false, fmt.Errorf("toggle reaction: %w", err)
	}

	if err := s.store.Save(); err != nil {
		return false, fmt.Errorf("save: %w", err)
	}

	return added, nil
}

// channelToDTO converts a Channel to a ChannelDTO.
func channelToDTO(ch *Channel) ChannelDTO {
	members := ch.Members
	if members == nil {
		members = []string{}
	}
	return ChannelDTO{
		Name:         ch.Name,
		Description:  ch.Description,
		Members:      members,
		MemberCount:  len(members),
		MessageCount: len(ch.History),
	}
}
