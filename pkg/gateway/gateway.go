// Package gateway provides external messaging platform integrations.
// It bridges bc channels to platforms like Telegram, Discord, and Slack.
package gateway

import (
	"context"
	"time"
)

// Adapter handles the platform connection lifecycle and message routing.
type Adapter interface {
	// Name returns the platform identifier ("telegram", "discord", "slack").
	Name() string

	// Start connects to the platform and begins receiving messages.
	// Calls onMessage for each inbound message. Blocks until ctx is canceled.
	Start(ctx context.Context, onMessage func(InboundMessage)) error

	// Stop gracefully disconnects from the platform.
	Stop(ctx context.Context) error

	// Send delivers a message to a platform channel.
	Send(ctx context.Context, channelID, sender, content string) error

	// Channels returns all channels/groups the bot is a member of.
	Channels(ctx context.Context) ([]ExternalChannel, error)

	// Health returns nil if the adapter is connected and operational.
	Health(ctx context.Context) error
}

// FileSender is optionally implemented by adapters that support file uploads.
type FileSender interface {
	// SendFile uploads a file to a platform channel.
	SendFile(ctx context.Context, channelID, sender, filename string, data []byte, mimeType string) error
}

// StatusReporter is optionally implemented by adapters that report connection state.
type StatusReporter interface {
	// Status returns the current connection state for UI display.
	Status() AdapterStatus
}

// AdapterStatus reports connection state for the web UI.
type AdapterStatus struct {
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
	Error         string    `json:"error,omitempty"`
	Connected     bool      `json:"connected"`
}

// Attachment represents a file attached to a message.
type Attachment struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Source   string `json:"source"`
	FileID   string `json:"file_id,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// InboundMessage is a normalized message from an external platform.
type InboundMessage struct {
	Timestamp   time.Time
	ChannelID   string
	ChannelName string
	Sender      string
	SenderID    string
	Content     string
	MessageID   string
	Attachments []Attachment
}

// ExternalChannel represents a channel/group on an external platform.
type ExternalChannel struct {
	ID   string
	Name string
	Type string // "group", "channel", "dm"
}

// Truncate shortens a string to n characters, appending "..." if truncated.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
