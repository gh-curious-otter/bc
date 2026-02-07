// Package channel - approval_message.go provides PR approval message parsing and formatting.
package channel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ApprovalStatus represents the status of a PR review.
type ApprovalStatus string

const (
	// StatusApproved indicates the PR was approved.
	StatusApproved ApprovalStatus = "approved"

	// StatusChangesRequested indicates changes are needed.
	StatusChangesRequested ApprovalStatus = "changes_requested"

	// StatusCommented indicates a comment without approval/rejection.
	StatusCommented ApprovalStatus = "commented"
)

// ApprovalMessage represents a parsed PR approval/review message.
type ApprovalMessage struct {
	// Reviewer is the name of the reviewer
	Reviewer string `json:"reviewer,omitempty"`

	// Comment is any additional comment
	Comment string `json:"comment,omitempty"`

	// Raw is the original message content
	Raw string `json:"raw"`

	// Status is the approval status
	Status ApprovalStatus `json:"status"`

	// PRNumber is the pull request number
	PRNumber int `json:"pr_number"`
}

// approvalPatterns matches approval keywords
var approvalPatterns = []string{
	"approved",
	"lgtm",
	"looks good",
	"ship it",
	"✅",
	"👍",
}

// changesPatterns matches change request keywords
var changesPatterns = []string{
	"needs changes",
	"changes requested",
	"please fix",
	"needs work",
	"❌",
	"👎",
}

// ParseApprovalMessage parses an approval/review message.
// Expected formats:
//   - "PR #123 approved ✓"
//   - "LGTM PR #456"
//   - "PR #789 needs changes: please fix X"
//
// Returns nil if the message doesn't appear to be an approval message.
func ParseApprovalMessage(content string) *ApprovalMessage {
	lower := strings.ToLower(content)

	// Try to extract PR number first
	prMatch := prNumberRegex.FindStringSubmatch(content)
	if prMatch == nil {
		return nil
	}

	prNumber, err := strconv.Atoi(prMatch[1])
	if err != nil {
		return nil
	}

	msg := &ApprovalMessage{
		PRNumber: prNumber,
		Raw:      content,
		Status:   StatusCommented, // Default
	}

	// Check for approval patterns
	for _, pattern := range approvalPatterns {
		if strings.Contains(lower, pattern) {
			msg.Status = StatusApproved
			break
		}
	}

	// Check for changes requested patterns (takes precedence)
	for _, pattern := range changesPatterns {
		if strings.Contains(lower, pattern) {
			msg.Status = StatusChangesRequested
			break
		}
	}

	// Only return if it's actually an approval-related message
	if msg.Status == StatusCommented {
		// Check if it has any review-related context
		if !strings.Contains(lower, "review") &&
			!strings.Contains(lower, "pr") &&
			!strings.Contains(lower, "comment") {
			return nil
		}
	}

	// Extract reviewer from @mention
	mentions := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(mentions) > 0 {
		msg.Reviewer = mentions[0][1]
	}

	return msg
}

// FormatApprovalMessage creates a standardized approval message.
// Format: "PR #<number> approved ✓"
func FormatApprovalMessage(prNumber int, status ApprovalStatus) string {
	switch status {
	case StatusApproved:
		return fmt.Sprintf("PR #%d approved ✓", prNumber)
	case StatusChangesRequested:
		return fmt.Sprintf("PR #%d needs changes", prNumber)
	default:
		return fmt.Sprintf("PR #%d reviewed", prNumber)
	}
}

// FormatApprovalWithComment includes a comment with the approval.
func FormatApprovalWithComment(prNumber int, status ApprovalStatus, comment string) string {
	base := FormatApprovalMessage(prNumber, status)
	if comment == "" {
		return base
	}
	return fmt.Sprintf("%s: %s", base, comment)
}

// NewApprovalMessage creates a TypedMessage for an approval.
func NewApprovalMessage(prNumber int, status ApprovalStatus, sender string) *TypedMessage {
	content := FormatApprovalMessage(prNumber, status)
	msg := NewTypedMessage(content, TypeApproval, sender)
	msg.WithMetadata("pr_number", strconv.Itoa(prNumber))
	msg.WithMetadata("status", string(status))
	return msg
}

// IsApprovalMessage checks if a message content looks like an approval.
func IsApprovalMessage(content string) bool {
	return ParseApprovalMessage(content) != nil
}

// IsApproved checks if a message indicates PR approval.
func IsApproved(content string) bool {
	msg := ParseApprovalMessage(content)
	return msg != nil && msg.Status == StatusApproved
}

// IsChangesRequested checks if a message indicates changes are needed.
func IsChangesRequested(content string) bool {
	msg := ParseApprovalMessage(content)
	return msg != nil && msg.Status == StatusChangesRequested
}

// MergeNotification represents a parsed merge notification message.
type MergeNotification struct {
	// Branch is the target branch (e.g., "main")
	Branch string `json:"branch,omitempty"`

	// MergedBy is who performed the merge
	MergedBy string `json:"merged_by,omitempty"`

	// Raw is the original message content
	Raw string `json:"raw"`

	// PRNumber is the pull request number
	PRNumber int `json:"pr_number"`
}

// mergePatterns matches merge-related keywords
var mergePatterns = regexp.MustCompile(`(?i)(?:merged|merge(?:d)?(?:\s+to)?|pushed to)`)

// ParseMergeNotification parses a merge notification message.
// Expected formats:
//   - "PR #123 merged to main"
//   - "Merged PR #456"
//   - "#789 pushed to main"
//
// Returns nil if the message doesn't appear to be a merge notification.
func ParseMergeNotification(content string) *MergeNotification {
	// Must contain merge-related keywords
	if !mergePatterns.MatchString(content) {
		return nil
	}

	// Try to extract PR number
	prMatch := prNumberRegex.FindStringSubmatch(content)
	if prMatch == nil {
		return nil
	}

	prNumber, err := strconv.Atoi(prMatch[1])
	if err != nil {
		return nil
	}

	msg := &MergeNotification{
		PRNumber: prNumber,
		Raw:      content,
	}

	// Extract target branch
	branchMatch := regexp.MustCompile(`(?i)(?:to|into)\s+(\w+)`).FindStringSubmatch(content)
	if branchMatch != nil {
		msg.Branch = branchMatch[1]
	}

	// Extract who merged from @mention
	mentions := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(mentions) > 0 {
		msg.MergedBy = mentions[0][1]
	}

	return msg
}

// FormatMergeNotification creates a standardized merge notification.
func FormatMergeNotification(prNumber int, branch string) string {
	if branch == "" {
		branch = "main"
	}
	return fmt.Sprintf("PR #%d merged to %s", prNumber, branch)
}

// NewMergeNotificationMessage creates a TypedMessage for a merge.
func NewMergeNotificationMessage(prNumber int, branch, sender string) *TypedMessage {
	content := FormatMergeNotification(prNumber, branch)
	msg := NewTypedMessage(content, TypeMerge, sender)
	msg.WithMetadata("pr_number", strconv.Itoa(prNumber))
	if branch != "" {
		msg.WithMetadata("branch", branch)
	}
	return msg
}

// IsMergeNotification checks if a message looks like a merge notification.
func IsMergeNotification(content string) bool {
	return ParseMergeNotification(content) != nil
}
