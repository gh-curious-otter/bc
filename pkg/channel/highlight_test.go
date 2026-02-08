package channel

import (
	"strings"
	"testing"
)

func TestParseChannelRefs(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "single channel ref",
			message:  "Check out #engineering for updates",
			expected: []string{"#engineering"},
		},
		{
			name:     "multiple channel refs",
			message:  "See #general and #standup for details",
			expected: []string{"#general", "#standup"},
		},
		{
			name:     "no channel refs",
			message:  "Just a regular message",
			expected: nil,
		},
		{
			name:     "channel with hyphens",
			message:  "Join #ui-design for the meeting",
			expected: []string{"#ui-design"},
		},
		{
			name:     "channel with underscores",
			message:  "Check #dev_tools channel",
			expected: []string{"#dev_tools"},
		},
		{
			name:     "github issue number should not match",
			message:  "Fixed in #123",
			expected: nil,
		},
		{
			name:     "mixed channels and issue numbers",
			message:  "See #engineering about issue #456",
			expected: []string{"#engineering"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highlights := ParseChannelRefs(tt.message)

			if len(highlights) != len(tt.expected) {
				t.Errorf("got %d highlights, want %d", len(highlights), len(tt.expected))
				return
			}

			for i, h := range highlights {
				if h.Text != tt.expected[i] {
					t.Errorf("highlight %d: got %q, want %q", i, h.Text, tt.expected[i])
				}
				if h.Type != HighlightChannel {
					t.Errorf("highlight %d: got type %d, want %d", i, h.Type, HighlightChannel)
				}
			}
		})
	}
}

func TestParseGitHubLinks(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "issue number",
			message:  "Fixed in #123",
			expected: []string{"#123"},
		},
		{
			name:     "PR reference",
			message:  "See PR #456 for the fix",
			expected: []string{"PR #456"},
		},
		{
			name:     "issue reference",
			message:  "Related to issue #789",
			expected: []string{"issue #789"},
		},
		{
			name:     "github URL",
			message:  "Check https://github.com/rpuneet/bc/issues/123",
			expected: []string{"https://github.com/rpuneet/bc/issues/123"},
		},
		{
			name:     "github PR URL",
			message:  "See https://github.com/rpuneet/bc/pull/456",
			expected: []string{"https://github.com/rpuneet/bc/pull/456"},
		},
		{
			name:     "no github links",
			message:  "Just a regular message",
			expected: nil,
		},
		{
			name:     "multiple issue numbers",
			message:  "Fixed #123 and #456",
			expected: []string{"#123", "#456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highlights := ParseGitHubLinks(tt.message)

			if len(highlights) != len(tt.expected) {
				t.Errorf("got %d highlights, want %d", len(highlights), len(tt.expected))
				for i, h := range highlights {
					t.Logf("  highlight %d: %q at [%d:%d]", i, h.Text, h.StartIndex, h.EndIndex)
				}
				return
			}

			for i, h := range highlights {
				if h.Text != tt.expected[i] {
					t.Errorf("highlight %d: got %q, want %q", i, h.Text, tt.expected[i])
				}
				if h.Type != HighlightGitHubLink {
					t.Errorf("highlight %d: got type %d, want %d", i, h.Type, HighlightGitHubLink)
				}
			}
		})
	}
}

func TestParseAllHighlights(t *testing.T) {
	tests := []struct {
		name    string
		message string
		types   []HighlightType
		texts   []string
	}{
		{
			name:    "all types",
			message: "Hey @engineer-01, check #general about issue #123",
			types:   []HighlightType{HighlightMention, HighlightChannel, HighlightGitHubLink},
			texts:   []string{"@engineer-01", "#general", "issue #123"},
		},
		{
			name:    "mentions only",
			message: "@tech-lead-01 and @manager please review",
			types:   []HighlightType{HighlightMention, HighlightMention},
			texts:   []string{"@tech-lead-01", "@manager"},
		},
		{
			name:    "empty message",
			message: "",
			types:   nil,
			texts:   nil,
		},
		{
			name:    "sorted by position",
			message: "#channel first, then @user, finally #456",
			types:   []HighlightType{HighlightChannel, HighlightMention, HighlightGitHubLink},
			texts:   []string{"#channel", "@user", "#456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highlights := ParseAllHighlights(tt.message)

			if len(highlights) != len(tt.types) {
				t.Errorf("got %d highlights, want %d", len(highlights), len(tt.types))
				for i, h := range highlights {
					t.Logf("  highlight %d: type=%d text=%q", i, h.Type, h.Text)
				}
				return
			}

			for i, h := range highlights {
				if h.Type != tt.types[i] {
					t.Errorf("highlight %d: got type %d, want %d", i, h.Type, tt.types[i])
				}
				if h.Text != tt.texts[i] {
					t.Errorf("highlight %d: got text %q, want %q", i, h.Text, tt.texts[i])
				}
			}

			// Verify sorted by position
			for i := 1; i < len(highlights); i++ {
				if highlights[i].StartIndex < highlights[i-1].StartIndex {
					t.Errorf("highlights not sorted: [%d].StartIndex=%d < [%d].StartIndex=%d",
						i, highlights[i].StartIndex, i-1, highlights[i-1].StartIndex)
				}
			}
		})
	}
}

func TestApplyHighlights(t *testing.T) {
	format := func(text string, highlightType HighlightType) string {
		switch highlightType {
		case HighlightMention:
			return "[M:" + text + "]"
		case HighlightChannel:
			return "[C:" + text + "]"
		case HighlightGitHubLink:
			return "[L:" + text + "]"
		default:
			return text
		}
	}

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "mention",
			message:  "Hello @user",
			expected: "Hello [M:@user]",
		},
		{
			name:     "channel",
			message:  "See #general",
			expected: "See [C:#general]",
		},
		{
			name:     "github link",
			message:  "Fixed in #123",
			expected: "Fixed in [L:#123]",
		},
		{
			name:     "all types",
			message:  "@user check #engineering for #456",
			expected: "[M:@user] check [C:#engineering] for [L:#456]",
		},
		{
			name:     "no highlights",
			message:  "Just a plain message",
			expected: "Just a plain message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyHighlights(tt.message, format)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"123abc", false},
		{"abc", false},
		{"", false},
		{"12.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isAllDigits(tt.input); got != tt.expected {
				t.Errorf("isAllDigits(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHighlightIndicesPreserved(t *testing.T) {
	// Test that highlights preserve correct indices in original text
	message := "Hey @alice, see #dev about #123"

	highlights := ParseAllHighlights(message)
	if len(highlights) != 3 {
		t.Fatalf("expected 3 highlights, got %d", len(highlights))
	}

	// Verify each highlight extracts the correct substring
	for _, h := range highlights {
		extracted := message[h.StartIndex:h.EndIndex]
		if extracted != h.Text {
			t.Errorf("indices mismatch: message[%d:%d]=%q, h.Text=%q",
				h.StartIndex, h.EndIndex, extracted, h.Text)
		}
	}
}

func TestNoOverlappingHighlights(t *testing.T) {
	// Test that channel refs and github links don't overlap
	message := "Issue #123 in #engineering channel"

	highlights := ParseAllHighlights(message)

	// Check for overlaps
	for i := 0; i < len(highlights); i++ {
		for j := i + 1; j < len(highlights); j++ {
			if highlights[i].EndIndex > highlights[j].StartIndex &&
				highlights[i].StartIndex < highlights[j].EndIndex {
				t.Errorf("overlapping highlights: %q [%d:%d] and %q [%d:%d]",
					highlights[i].Text, highlights[i].StartIndex, highlights[i].EndIndex,
					highlights[j].Text, highlights[j].StartIndex, highlights[j].EndIndex)
			}
		}
	}
}

func TestApplyHighlightsWithANSI(t *testing.T) {
	// Test that ANSI codes work correctly in formatted output
	format := func(text string, _ HighlightType) string {
		return "\x1b[34m" + text + "\x1b[0m" // Blue text
	}

	message := "Hello @user"
	result := ApplyHighlights(message, format)

	if !strings.Contains(result, "\x1b[34m@user\x1b[0m") {
		t.Errorf("ANSI formatting not applied correctly: %q", result)
	}
}
