package channel

import (
	"regexp"
	"slices"
	"sort"
)

// Highlight represents a highlighted segment in a message.
type Highlight struct {
	// Text is the matched text.
	Text string
	// StartIndex is the position of the highlight start in the original text.
	StartIndex int
	// EndIndex is the position after the highlight in the original text.
	EndIndex int
	// Type identifies the highlight type.
	Type HighlightType
}

// HighlightType identifies the type of highlight.
type HighlightType int

const (
	HighlightMention HighlightType = iota
	HighlightChannel
	HighlightGitHubLink
)

// channelPattern matches #channel-name patterns in message text.
// Supports alphanumeric names with hyphens and underscores.
var channelPattern = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)

// githubLinkPattern matches GitHub issue/PR references.
// Matches: #123, PR #456, issue #789, github.com URLs
var githubLinkPattern = regexp.MustCompile(`(?i)(?:(?:PR|issue)\s*#(\d+))|(?:https?://github\.com/[^\s]+(?:/(?:issues|pull)/\d+)?)|(?:(?:^|[^\w])#(\d+)(?:[^\w]|$))`)

// ParseChannelRefs extracts all #channel references from a message.
func ParseChannelRefs(message string) []Highlight {
	matches := channelPattern.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		return nil
	}

	highlights := make([]Highlight, 0, len(matches))
	for _, match := range matches {
		// Skip if this looks like a GitHub issue number (all digits after #)
		channelName := message[match[2]:match[3]]
		if isAllDigits(channelName) {
			continue
		}

		highlight := Highlight{
			StartIndex: match[0],
			EndIndex:   match[1],
			Type:       HighlightChannel,
			Text:       message[match[0]:match[1]],
		}
		highlights = append(highlights, highlight)
	}

	return highlights
}

// isAllDigits checks if a string contains only digits.
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// ParseGitHubLinks extracts GitHub issue/PR references from a message.
func ParseGitHubLinks(message string) []Highlight {
	matches := githubLinkPattern.FindAllStringIndex(message, -1)
	if len(matches) == 0 {
		return nil
	}

	highlights := make([]Highlight, 0, len(matches))
	for _, match := range matches {
		// Trim leading/trailing non-link characters that may be captured
		start, end := match[0], match[1]
		for start < end && !isLinkChar(rune(message[start])) {
			start++
		}
		for end > start && !isLinkChar(rune(message[end-1])) {
			end--
		}
		if start >= end {
			continue
		}

		highlight := Highlight{
			Text:       message[start:end],
			StartIndex: start,
			EndIndex:   end,
			Type:       HighlightGitHubLink,
		}
		highlights = append(highlights, highlight)
	}

	return highlights
}

// isLinkChar checks if a character is part of a link or reference.
func isLinkChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '#' || c == '/' || c == ':' ||
		c == '.' || c == '-' || c == '_' || c == '?' || c == '=' || c == '&'
}

// ParseAllHighlights extracts all highlight types from a message.
// Returns highlights sorted by start position.
func ParseAllHighlights(message string) []Highlight {
	var all []Highlight

	// Parse mentions using existing function
	mentions := ParseMentions(message)
	for _, m := range mentions {
		all = append(all, Highlight{
			StartIndex: m.StartIndex,
			EndIndex:   m.EndIndex,
			Type:       HighlightMention,
			Text:       "@" + m.Name,
		})
	}

	// Parse GitHub links first (before channel refs to avoid conflicts)
	githubLinks := ParseGitHubLinks(message)
	all = append(all, githubLinks...)

	// Parse channel refs (excluding GitHub issue numbers)
	channelRefs := ParseChannelRefs(message)
	// Filter out channel refs that overlap with GitHub links
	for _, ch := range channelRefs {
		overlaps := false
		for _, gh := range githubLinks {
			if ch.StartIndex < gh.EndIndex && ch.EndIndex > gh.StartIndex {
				overlaps = true
				break
			}
		}
		if !overlaps {
			all = append(all, ch)
		}
	}

	// Sort by start position
	sort.Slice(all, func(i, j int) bool {
		return all[i].StartIndex < all[j].StartIndex
	})

	return all
}

// FormatFunc is a function that formats a highlight for display.
type FormatFunc func(text string, highlightType HighlightType) string

// ApplyHighlights applies formatting to all highlights in a message.
// The format function receives the matched text and highlight type.
func ApplyHighlights(message string, format FormatFunc) string {
	highlights := ParseAllHighlights(message)
	if len(highlights) == 0 {
		return message
	}

	// Process in reverse order to preserve indices
	result := message
	slices.Reverse(highlights)
	for _, h := range highlights {
		formatted := format(h.Text, h.Type)
		result = result[:h.StartIndex] + formatted + result[h.EndIndex:]
	}

	return result
}
