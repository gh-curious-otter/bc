package channel

import (
	"strings"
	"testing"
)

func TestParseMentions_Single(t *testing.T) {
	mentions := ParseMentions("Hello @engineer-01, please review this")

	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}

	m := mentions[0]
	if m.Name != "engineer-01" {
		t.Errorf("Name = %q, want engineer-01", m.Name)
	}
	if m.IsAll {
		t.Error("IsAll should be false")
	}
	if m.StartIndex != 6 {
		t.Errorf("StartIndex = %d, want 6", m.StartIndex)
	}
}

func TestParseMentions_Multiple(t *testing.T) {
	mentions := ParseMentions("@manager please assign @engineer-01 and @qa-02")

	if len(mentions) != 3 {
		t.Fatalf("expected 3 mentions, got %d", len(mentions))
	}

	names := []string{mentions[0].Name, mentions[1].Name, mentions[2].Name}
	expected := []string{"manager", "engineer-01", "qa-02"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("mention[%d].Name = %q, want %q", i, name, expected[i])
		}
	}
}

func TestParseMentions_All(t *testing.T) {
	mentions := ParseMentions("@all please check in for standup")

	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}

	if mentions[0].Name != "all" {
		t.Errorf("Name = %q, want all", mentions[0].Name)
	}
	if !mentions[0].IsAll {
		t.Error("IsAll should be true")
	}
}

func TestParseMentions_AllCaseInsensitive(t *testing.T) {
	tests := []string{"@ALL", "@All", "@aLl"}
	for _, msg := range tests {
		mentions := ParseMentions(msg + " check in")
		if len(mentions) != 1 {
			t.Errorf("expected 1 mention for %q", msg)
			continue
		}
		if !mentions[0].IsAll {
			t.Errorf("IsAll should be true for %q", msg)
		}
	}
}

func TestParseMentions_None(t *testing.T) {
	mentions := ParseMentions("No mentions here, just email@example.com")

	// email@example.com should match "example" after @
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention (example from email), got %d", len(mentions))
	}
	if mentions[0].Name != "example" {
		t.Errorf("Name = %q, want example", mentions[0].Name)
	}
}

func TestParseMentions_NoMentions(t *testing.T) {
	mentions := ParseMentions("Just a regular message")

	if mentions != nil {
		t.Errorf("expected nil, got %v", mentions)
	}
}

func TestParseMentions_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{"start of message", "@engineer-01 do this", []string{"engineer-01"}},
		{"end of message", "assign to @engineer-01", []string{"engineer-01"}},
		{"with punctuation", "@engineer-01: please help", []string{"engineer-01"}},
		{"with underscore", "@tech_lead_01 review", []string{"tech_lead_01"}},
		{"consecutive", "@a@b@c mentions", []string{"a", "b", "c"}},
		{"just @", "@ alone is not a mention", nil},
		{"@123", "@123 numbers only not valid", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mentions := ParseMentions(tt.message)
			if tt.expected == nil {
				if mentions != nil {
					t.Errorf("expected nil, got %v", mentions)
				}
				return
			}
			if len(mentions) != len(tt.expected) {
				t.Fatalf("expected %d mentions, got %d", len(tt.expected), len(mentions))
			}
			for i, m := range mentions {
				if m.Name != tt.expected[i] {
					t.Errorf("mention[%d].Name = %q, want %q", i, m.Name, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractMentionedAgents_Unique(t *testing.T) {
	agents, hasAll := ExtractMentionedAgents("@engineer-01 and @ENGINEER-01 again")

	if hasAll {
		t.Error("hasAll should be false")
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 unique agent, got %d", len(agents))
	}
}

func TestExtractMentionedAgents_WithAll(t *testing.T) {
	agents, hasAll := ExtractMentionedAgents("@engineer-01 and @all need to see this")

	if !hasAll {
		t.Error("hasAll should be true")
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent (besides @all), got %d", len(agents))
	}
}

func TestResolveMentions_ValidMembers(t *testing.T) {
	members := []string{"engineer-01", "engineer-02", "manager"}
	result := ResolveMentions("@engineer-01 please help @manager", members)

	if len(result) != 2 {
		t.Fatalf("expected 2 resolved mentions, got %d", len(result))
	}
}

func TestResolveMentions_InvalidMember(t *testing.T) {
	members := []string{"engineer-01", "engineer-02"}
	result := ResolveMentions("@engineer-01 and @nonexistent", members)

	if len(result) != 1 {
		t.Fatalf("expected 1 resolved mention, got %d", len(result))
	}
	if result[0] != "engineer-01" {
		t.Errorf("expected engineer-01, got %s", result[0])
	}
}

func TestResolveMentions_All(t *testing.T) {
	members := []string{"engineer-01", "engineer-02", "manager"}
	result := ResolveMentions("@all standup time", members)

	if len(result) != 3 {
		t.Fatalf("expected all 3 members, got %d", len(result))
	}
}

func TestResolveMentions_NoMentions(t *testing.T) {
	members := []string{"engineer-01", "engineer-02"}
	result := ResolveMentions("no mentions here", members)

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestContainsMention_Direct(t *testing.T) {
	if !ContainsMention("@engineer-01 do this", "engineer-01") {
		t.Error("should contain mention for engineer-01")
	}
	if ContainsMention("@engineer-01 do this", "engineer-02") {
		t.Error("should not contain mention for engineer-02")
	}
}

func TestContainsMention_All(t *testing.T) {
	if !ContainsMention("@all standup", "engineer-01") {
		t.Error("@all should match any agent")
	}
	if !ContainsMention("@all standup", "manager") {
		t.Error("@all should match any agent")
	}
}

func TestContainsMention_CaseInsensitive(t *testing.T) {
	if !ContainsMention("@ENGINEER-01 do this", "engineer-01") {
		t.Error("mention matching should be case insensitive")
	}
	if !ContainsMention("@engineer-01 do this", "ENGINEER-01") {
		t.Error("agent name matching should be case insensitive")
	}
}

func TestStripMentions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@engineer-01 do this", " do this"},
		{"Hello @all", "Hello "},
		{"@a @b @c test", "   test"},
		{"no mentions", "no mentions"},
	}

	for _, tt := range tests {
		result := StripMentions(tt.input)
		if result != tt.expected {
			t.Errorf("StripMentions(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestHighlightMentions(t *testing.T) {
	format := func(name string) string {
		return "[" + strings.ToUpper(name) + "]"
	}

	result := HighlightMentions("Hello @engineer-01, please help @manager", format)
	expected := "Hello [ENGINEER-01], please help [MANAGER]"

	if result != expected {
		t.Errorf("HighlightMentions = %q, want %q", result, expected)
	}
}

func TestHighlightMentions_NoMentions(t *testing.T) {
	format := func(name string) string {
		return "[" + name + "]"
	}

	input := "no mentions here"
	result := HighlightMentions(input, format)

	if result != input {
		t.Errorf("HighlightMentions = %q, want %q", result, input)
	}
}
