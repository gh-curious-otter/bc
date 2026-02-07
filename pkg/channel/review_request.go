// Package channel - review_request.go provides PR review request parsing and formatting.
package channel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ReviewRequest represents a parsed PR review request message.
type ReviewRequest struct {
	// Target is the mentioned agent or role (e.g., "tech-lead-01")
	Target string `json:"target,omitempty"`

	// Title is the optional PR title
	Title string `json:"title,omitempty"`

	// URL is the GitHub PR URL (if determinable)
	URL string `json:"url,omitempty"`

	// Branch is the source branch name
	Branch string `json:"branch,omitempty"`

	// Author is who requested the review
	Author string `json:"author,omitempty"`

	// Raw is the original message content
	Raw string `json:"raw"`

	// PRNumber is the pull request number (e.g., 123)
	PRNumber int `json:"pr_number"`
}

// prNumberRegex matches PR references like "PR #123", "PR 123", "#123"
// Must have "PR" prefix or "#" prefix to avoid matching agent numbers
var prNumberRegex = regexp.MustCompile(`(?i)(?:pr\s*#?|#)(\d+)`)

// mentionRegex matches @mentions like "@tech-lead-01"
var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)

// ParseReviewRequest parses a review request message.
// Expected formats:
//   - "@tech-lead PR #123 ready for review"
//   - "@tech-lead-01 please review PR #456"
//   - "PR #789 ready for review @tech-lead"
//
// Returns nil if the message doesn't appear to be a review request.
func ParseReviewRequest(content string) *ReviewRequest {
	lower := strings.ToLower(content)

	// Must contain review-related keywords
	if !strings.Contains(lower, "review") {
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

	req := &ReviewRequest{
		PRNumber: prNumber,
		Raw:      content,
	}

	// Extract @mentions
	mentions := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(mentions) > 0 {
		req.Target = mentions[0][1]
	}

	return req
}

// FormatReviewRequest creates a standardized review request message.
// Format: "@<target> PR #<number> ready for review"
func FormatReviewRequest(prNumber int, target string) string {
	if target == "" {
		target = "tech-lead"
	}
	return fmt.Sprintf("@%s PR #%d ready for review", target, prNumber)
}

// FormatReviewRequestWithTitle includes the PR title.
// Format: "@<target> PR #<number> ready for review: <title>"
func FormatReviewRequestWithTitle(prNumber int, target, title string) string {
	if target == "" {
		target = "tech-lead"
	}
	if title == "" {
		return FormatReviewRequest(prNumber, target)
	}
	return fmt.Sprintf("@%s PR #%d ready for review: %s", target, prNumber, title)
}

// FormatReviewRequestWithURL includes the full GitHub URL.
// Format: "@<target> PR #<number> ready for review: <url>"
func FormatReviewRequestWithURL(prNumber int, target, url string) string {
	if target == "" {
		target = "tech-lead"
	}
	return fmt.Sprintf("@%s PR #%d ready for review: %s", target, prNumber, url)
}

// NewReviewRequestMessage creates a TypedMessage for a review request.
func NewReviewRequestMessage(prNumber int, target, sender string) *TypedMessage {
	content := FormatReviewRequest(prNumber, target)
	msg := NewTypedMessage(content, TypeReview, sender)
	msg.WithMetadata("pr_number", strconv.Itoa(prNumber))
	if target != "" {
		msg.WithMetadata("target", target)
	}
	return msg
}

// IsReviewRequest checks if a message content looks like a review request.
func IsReviewRequest(content string) bool {
	return ParseReviewRequest(content) != nil
}

// ExtractPRNumber extracts just the PR number from a message.
// Returns 0 if no PR number found.
func ExtractPRNumber(content string) int {
	match := prNumberRegex.FindStringSubmatch(content)
	if match == nil {
		return 0
	}
	num, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return num
}

// ExtractMentions returns all @mentions from a message.
func ExtractMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	mentions := make([]string, 0, len(matches))
	for _, m := range matches {
		mentions = append(mentions, m[1])
	}
	return mentions
}
