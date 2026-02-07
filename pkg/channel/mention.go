// Package channel provides a channels system for broadcasting messages to groups of agents.
package channel

import (
	"regexp"
	"slices"
	"strings"
)

// MentionAll is the special mention that targets all channel members.
const MentionAll = "all"

// mentionPattern matches @agent-name patterns in message text.
// Supports alphanumeric names with hyphens and underscores.
var mentionPattern = regexp.MustCompile(`@([a-zA-Z][a-zA-Z0-9_-]*)`)

// Mention represents a parsed mention from a message.
type Mention struct {
	// Name is the agent name (without @prefix).
	Name string
	// StartIndex is the position of @ in the original text.
	StartIndex int
	// EndIndex is the position after the mention in the original text.
	EndIndex int
	// IsAll is true if this is an @all mention.
	IsAll bool
}

// ParseMentions extracts all @mentions from a message.
// Returns a slice of Mention structs with their positions.
func ParseMentions(message string) []Mention {
	matches := mentionPattern.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		return nil
	}

	mentions := make([]Mention, 0, len(matches))
	for _, match := range matches {
		// match[0:1] is full match, match[2:3] is capture group
		name := message[match[2]:match[3]]
		mention := Mention{
			Name:       name,
			StartIndex: match[0],
			EndIndex:   match[1],
			IsAll:      strings.EqualFold(name, MentionAll),
		}
		mentions = append(mentions, mention)
	}

	return mentions
}

// ExtractMentionedAgents returns unique agent names mentioned in a message.
// If @all is mentioned, returns nil (caller should expand to all members).
func ExtractMentionedAgents(message string) (agents []string, hasAll bool) {
	mentions := ParseMentions(message)
	if len(mentions) == 0 {
		return nil, false
	}

	seen := make(map[string]bool)
	for _, m := range mentions {
		if m.IsAll {
			hasAll = true
			continue
		}
		lower := strings.ToLower(m.Name)
		if !seen[lower] {
			seen[lower] = true
			agents = append(agents, m.Name)
		}
	}

	return agents, hasAll
}

// ResolveMentions expands mentions to actual agent names.
// If @all is present, returns all channel members.
// Otherwise returns the unique mentioned agents that are channel members.
func ResolveMentions(message string, channelMembers []string) []string {
	agents, hasAll := ExtractMentionedAgents(message)
	if hasAll {
		// Return all channel members for @all
		result := make([]string, len(channelMembers))
		copy(result, channelMembers)
		return result
	}

	if len(agents) == 0 {
		return nil
	}

	// Filter to only include valid channel members
	memberSet := make(map[string]bool)
	for _, m := range channelMembers {
		memberSet[strings.ToLower(m)] = true
	}

	result := make([]string, 0, len(agents))
	for _, agent := range agents {
		if memberSet[strings.ToLower(agent)] {
			result = append(result, agent)
		}
	}

	return result
}

// ContainsMention checks if a message mentions a specific agent.
// Also returns true if @all is mentioned.
func ContainsMention(message, agentName string) bool {
	agents, hasAll := ExtractMentionedAgents(message)
	if hasAll {
		return true
	}

	agentLower := strings.ToLower(agentName)
	for _, a := range agents {
		if strings.ToLower(a) == agentLower {
			return true
		}
	}
	return false
}

// StripMentions removes all @mentions from a message, leaving just the text.
func StripMentions(message string) string {
	return mentionPattern.ReplaceAllString(message, "")
}

// HighlightMentions wraps mentions in a message with markers for display.
// The format function receives the agent name and returns the highlighted form.
func HighlightMentions(message string, format func(name string) string) string {
	mentions := ParseMentions(message)
	if len(mentions) == 0 {
		return message
	}

	// Process in reverse order to preserve indices
	result := message
	slices.Reverse(mentions)
	for _, m := range mentions {
		highlighted := format(m.Name)
		result = result[:m.StartIndex] + highlighted + result[m.EndIndex:]
	}

	return result
}
