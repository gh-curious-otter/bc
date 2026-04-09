// Package notify implements the notification gateway for delivering
// external platform events (Slack, Telegram, Discord, etc.) to subscribed
// bc agents via tmux send-keys.
package notify

import "time"

// ChannelKey is the canonical identifier for an external channel.
// Format: "<platform>:<channel_name>", e.g., "slack:engineering".
type ChannelKey = string

// DeliveryStatus is the outcome of a tmux send-keys attempt.
type DeliveryStatus string

const (
	StatusDelivered DeliveryStatus = "delivered"
	StatusFailed    DeliveryStatus = "failed"
	StatusPending   DeliveryStatus = "pending"
)

// Notification is the JSON payload sent to subscribed agents via tmux send-keys.
type Notification struct {
	Timestamp   string       `json:"timestamp"`
	Channel     string       `json:"channel"`               // "slack:engineering"
	Platform    string       `json:"platform"`               // "slack"
	Sender      string       `json:"sender"`                 // resolved display name
	Content     string       `json:"content"`                // text content
	MessageID   string       `json:"message_id,omitempty"`   // platform-native message ID
	Mentions    []string     `json:"mentions,omitempty"`     // extracted @mentions
	Attachments []Attachment `json:"attachments,omitempty"`  // files shared
}

// Attachment describes a file shared on a channel.
type Attachment struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	URL       string `json:"url,omitempty"`        // platform download URL
	LocalPath string `json:"local_path,omitempty"` // path in .bc/attachments/
}

// Subscription ties an agent to a channel with delivery settings.
type Subscription struct {
	ID          int64     `json:"id"`
	Channel     string    `json:"channel"`      // "slack:engineering"
	Agent       string    `json:"agent"`
	MentionOnly bool      `json:"mention_only"` // only deliver when @agent in content
	CreatedAt   time.Time `json:"created_at"`
}

// DeliveryEntry records one delivery attempt in the activity log.
type DeliveryEntry struct {
	ID        int64          `json:"id"`
	LoggedAt  time.Time      `json:"logged_at"`
	Channel   string         `json:"channel"`
	Agent     string         `json:"agent"`
	Status    DeliveryStatus `json:"status"`
	Error     string         `json:"error,omitempty"`
	Preview   string         `json:"preview"` // first 120 chars of content
}

// GatewayInfo holds gateway state from the database.
type GatewayInfo struct {
	Name       string     `json:"name"`
	Enabled    bool       `json:"enabled"`
	Connected  bool       `json:"connected"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// truncate returns the first n characters of s.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
