package channel

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// channelNameRegex matches valid channel names: alphanumeric, hyphens, underscores, colons (for gateway channels like telegram:marketing),
// must start with an alphanumeric character.
var channelNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_:-]*$`)

// ErrChannelExists is returned when attempting to create a channel that already exists.
var ErrChannelExists = errors.New("channel already exists")

// MaxChannelNameLength is the maximum allowed length for a channel name.
const MaxChannelNameLength = 64

// ErrInvalidChannelName is returned when a channel name is invalid.
var ErrInvalidChannelName = errors.New("invalid channel name: must be 1-64 chars, start with alphanumeric, contain only alphanumeric, hyphens, or underscores")

// IsValidChannelName validates channel name format and length.
func IsValidChannelName(name string) bool {
	return len(name) > 0 && len(name) <= MaxChannelNameLength && channelNameRegex.MatchString(name)
}

// ChannelDTO is the API representation of a channel.
type ChannelDTO struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Type         string    `json:"type"`
	Platform     string    `json:"platform"` // "bc", "slack", "telegram", "discord"
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
	Order  string     `json:"order,omitempty"` // "asc" (default) or "desc"
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
	Before int        `json:"before,omitempty"` // cursor: messages before this ID
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

	// Cursor-based pagination: filter entries before the given ID
	if opts.Before > 0 && opts.Before <= len(filtered) {
		filtered = filtered[:opts.Before-1]
	}

	// Apply limit (take last N messages for asc, first N for desc)
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(filtered) > limit {
		if opts.Order == "desc" {
			// desc: take newest N (end of slice), then reverse
			filtered = filtered[len(filtered)-limit:]
		} else {
			// asc: take last N
			filtered = filtered[len(filtered)-limit:]
		}
	}

	// Reverse for descending order
	if opts.Order == "desc" {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	dtos := make([]MessageDTO, 0, len(filtered))
	for i, entry := range filtered {
		dtos = append(dtos, MessageDTO{
			ID:        int64(i + 1),
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

// SenderCount represents a sender and their message count.
type SenderCount struct {
	Sender string `json:"sender"`
	Count  int    `json:"count"`
}

// ChannelStatsDTO is the API representation of per-channel activity statistics.
type ChannelStatsDTO struct {
	LastActivity *time.Time    `json:"last_activity"`
	Name         string        `json:"name"`
	TopSenders   []SenderCount `json:"top_senders"`
	MessageCount int           `json:"message_count"`
	MemberCount  int           `json:"member_count"`
}

// Stats returns per-channel activity statistics for all channels.
func (s *ChannelService) Stats(_ context.Context) ([]ChannelStatsDTO, error) {
	channels := s.store.List()
	stats := make([]ChannelStatsDTO, 0, len(channels))
	for _, ch := range channels {
		dto := ChannelStatsDTO{
			Name:         ch.Name,
			MessageCount: len(ch.History),
			MemberCount:  len(ch.Members),
			TopSenders:   computeTopSenders(ch.History, 5),
		}
		if len(ch.History) > 0 {
			last := ch.History[len(ch.History)-1].Time
			dto.LastActivity = &last
		}
		stats = append(stats, dto)
	}
	return stats, nil
}

// computeTopSenders returns the top N senders by message count from history.
func computeTopSenders(history []HistoryEntry, n int) []SenderCount {
	counts := make(map[string]int)
	for _, entry := range history {
		counts[entry.Sender]++
	}
	senders := make([]SenderCount, 0, len(counts))
	for sender, count := range counts {
		senders = append(senders, SenderCount{Sender: sender, Count: count})
	}
	sort.Slice(senders, func(i, j int) bool {
		if senders[i].Count != senders[j].Count {
			return senders[i].Count > senders[j].Count
		}
		return senders[i].Sender < senders[j].Sender
	})
	if len(senders) > n {
		senders = senders[:n]
	}
	return senders
}

// channelToDTO converts a Channel to a ChannelDTO.
func channelToDTO(ch *Channel) ChannelDTO {
	members := ch.Members
	if members == nil {
		members = []string{}
	}
	platform := "bc"
	for _, prefix := range []string{"slack:", "telegram:", "discord:"} {
		if strings.HasPrefix(ch.Name, prefix) {
			platform = strings.TrimSuffix(prefix, ":")
			break
		}
	}
	return ChannelDTO{
		Name:         ch.Name,
		Description:  ch.Description,
		Platform:     platform,
		Members:      members,
		MemberCount:  len(members),
		MessageCount: len(ch.History),
	}
}
