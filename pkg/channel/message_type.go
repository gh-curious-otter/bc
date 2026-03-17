// Package channel provides a channels system for broadcasting messages to groups of agents.
package channel

import (
	"fmt"
	"strings"
)

// MessageType represents the type of a channel message.
// Different types enable filtering and routing for work coordination.
type MessageType string

const (
	// TypeText is a regular conversation message (default).
	TypeText MessageType = "text"

	// TypeTask is a work assignment, typically with @mention.
	TypeTask MessageType = "task"

	// TypeReview is a PR review request.
	TypeReview MessageType = "review"

	// TypeApproval is a tech lead approval notification.
	TypeApproval MessageType = "approval"

	// TypeMerge is a merge request or notification.
	TypeMerge MessageType = "merge"

	// TypeStatus is an agent status update.
	TypeStatus MessageType = "status"
)

// AllMessageTypes returns all valid message types.
func AllMessageTypes() []MessageType {
	return []MessageType{
		TypeText,
		TypeTask,
		TypeReview,
		TypeApproval,
		TypeMerge,
		TypeStatus,
	}
}

// ValidMessageTypes returns a comma-separated list of valid types for help text.
func ValidMessageTypes() string {
	types := AllMessageTypes()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}

// IsValidMessageType checks if a type string is a valid MessageType.
func IsValidMessageType(t string) bool {
	switch MessageType(strings.ToLower(t)) {
	case TypeText, TypeTask, TypeReview, TypeApproval, TypeMerge, TypeStatus:
		return true
	default:
		return false
	}
}

// ParseMessageType converts a string to a MessageType.
// Returns TypeText for empty string, error for invalid types.
func ParseMessageType(s string) (MessageType, error) {
	if s == "" {
		return TypeText, nil
	}

	t := MessageType(strings.ToLower(s))
	if !IsValidMessageType(string(t)) {
		return "", fmt.Errorf("invalid message type %q, valid types: %s", s, ValidMessageTypes())
	}
	return t, nil
}

// String returns the string representation of the message type.
func (t MessageType) String() string {
	return string(t)
}

// Emoji returns an emoji representation for display.
func (t MessageType) Emoji() string {
	switch t {
	case TypeTask:
		return "📋"
	case TypeReview:
		return "👀"
	case TypeApproval:
		return "✅"
	case TypeMerge:
		return "🔀"
	case TypeStatus:
		return "📊"
	default:
		return "💬"
	}
}

// Description returns a human-readable description of the type.
func (t MessageType) Description() string {
	switch t {
	case TypeText:
		return "Regular message"
	case TypeTask:
		return "Work assignment"
	case TypeReview:
		return "PR review request"
	case TypeApproval:
		return "Tech lead approval"
	case TypeMerge:
		return "Merge request/notification"
	case TypeStatus:
		return "Agent status update"
	default:
		return "Unknown type"
	}
}

// IsWorkItem returns true if the message type represents actionable work.
func (t MessageType) IsWorkItem() bool {
	switch t {
	case TypeTask, TypeReview, TypeMerge:
		return true
	default:
		return false
	}
}

// TargetRole returns the role that typically handles this message type.
// Returns empty string if no specific role is targeted.
func (t MessageType) TargetRole() string {
	switch t {
	case TypeTask:
		return "engineer"
	case TypeReview:
		return "tech-lead"
	case TypeApproval:
		return "manager"
	case TypeMerge:
		return "manager"
	default:
		return ""
	}
}

// TypedMessage represents a message with its type and metadata.
type TypedMessage struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Content  string            `json:"content"`
	Sender   string            `json:"sender"`
	Type     MessageType       `json:"type"`
}

// NewTypedMessage creates a new typed message.
func NewTypedMessage(content string, msgType MessageType, sender string) *TypedMessage {
	return &TypedMessage{
		Content: content,
		Type:    msgType,
		Sender:  sender,
	}
}

// WithMetadata adds metadata to the message.
func (m *TypedMessage) WithMetadata(key, value string) *TypedMessage {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	return m
}

// FormatForDisplay returns a formatted string for CLI display.
func (m *TypedMessage) FormatForDisplay() string {
	return fmt.Sprintf("%s [%s] %s", m.Type.Emoji(), m.Type, m.Content)
}

// InferMessageType attempts to infer the message type from content.
// Returns TypeText if no specific type can be inferred.
func InferMessageType(content string) MessageType {
	lower := strings.ToLower(content)

	// Check for PR review patterns
	if strings.Contains(lower, "please review") ||
		strings.Contains(lower, "ready for review") ||
		strings.Contains(lower, "pr #") && strings.Contains(lower, "review") {
		return TypeReview
	}

	// Check for approval patterns
	if strings.Contains(lower, "approved") ||
		strings.Contains(lower, "lgtm") ||
		strings.Contains(lower, "looks good") {
		return TypeApproval
	}

	// Check for merge patterns
	if strings.Contains(lower, "merged") ||
		strings.Contains(lower, "merge to main") ||
		strings.Contains(lower, "ready to merge") {
		return TypeMerge
	}

	// Check for task patterns (typically has @mention with action verb)
	if strings.Contains(content, "@") &&
		(strings.Contains(lower, "please") ||
			strings.Contains(lower, "implement") ||
			strings.Contains(lower, "fix") ||
			strings.Contains(lower, "add") ||
			strings.Contains(lower, "create")) {
		return TypeTask
	}

	// Check for status patterns
	if strings.HasPrefix(lower, "status:") ||
		strings.Contains(lower, "bc report") ||
		strings.Contains(lower, "agent report") {
		return TypeStatus
	}

	return TypeText
}
