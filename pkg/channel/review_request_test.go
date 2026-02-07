package channel

import (
	"testing"
)

func TestParseReviewRequest(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTarget string
		wantPR     int
		wantNil    bool
	}{
		{
			name:       "standard format",
			input:      "@tech-lead PR #123 ready for review",
			wantPR:     123,
			wantTarget: "tech-lead",
		},
		{
			name:       "with agent number",
			input:      "@tech-lead-01 please review PR #456",
			wantPR:     456,
			wantTarget: "tech-lead-01",
		},
		{
			name:       "mention at end",
			input:      "PR #789 ready for review @tech-lead",
			wantPR:     789,
			wantTarget: "tech-lead",
		},
		{
			name:       "lowercase",
			input:      "@tech-lead pr #100 ready for review",
			wantPR:     100,
			wantTarget: "tech-lead",
		},
		{
			name:       "no hash",
			input:      "@tech-lead PR 200 ready for review",
			wantPR:     200,
			wantTarget: "tech-lead",
		},
		{
			name:       "no mention",
			input:      "PR #300 is ready for review",
			wantPR:     300,
			wantTarget: "",
		},
		{
			name:    "not a review request",
			input:   "Hello everyone!",
			wantNil: true,
		},
		{
			name:    "has PR but no review",
			input:   "PR #123 is broken",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ParseReviewRequest(tt.input)
			if tt.wantNil {
				if req != nil {
					t.Errorf("expected nil, got %+v", req)
				}
				return
			}
			if req == nil {
				t.Fatal("expected non-nil result")
			}
			if req.PRNumber != tt.wantPR {
				t.Errorf("PRNumber = %d, want %d", req.PRNumber, tt.wantPR)
			}
			if req.Target != tt.wantTarget {
				t.Errorf("Target = %q, want %q", req.Target, tt.wantTarget)
			}
		})
	}
}

func TestFormatReviewRequest(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		want     string
		prNumber int
	}{
		{
			name:     "basic",
			prNumber: 123,
			target:   "tech-lead",
			want:     "@tech-lead PR #123 ready for review",
		},
		{
			name:     "with agent number",
			prNumber: 456,
			target:   "tech-lead-01",
			want:     "@tech-lead-01 PR #456 ready for review",
		},
		{
			name:     "empty target defaults",
			prNumber: 789,
			target:   "",
			want:     "@tech-lead PR #789 ready for review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatReviewRequest(tt.prNumber, tt.target)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatReviewRequestWithTitle(t *testing.T) {
	got := FormatReviewRequestWithTitle(123, "tech-lead", "Add new feature")
	want := "@tech-lead PR #123 ready for review: Add new feature"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Empty title should fall back to basic format
	got = FormatReviewRequestWithTitle(456, "tech-lead", "")
	want = "@tech-lead PR #456 ready for review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatReviewRequestWithURL(t *testing.T) {
	got := FormatReviewRequestWithURL(123, "tech-lead", "https://github.com/owner/repo/pull/123")
	want := "@tech-lead PR #123 ready for review: https://github.com/owner/repo/pull/123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewReviewRequestMessage(t *testing.T) {
	msg := NewReviewRequestMessage(123, "tech-lead-01", "engineer-01")

	if msg.Type != TypeReview {
		t.Errorf("Type = %v, want %v", msg.Type, TypeReview)
	}
	if msg.Sender != "engineer-01" {
		t.Errorf("Sender = %q, want %q", msg.Sender, "engineer-01")
	}
	if msg.Metadata["pr_number"] != "123" {
		t.Errorf("pr_number metadata = %q, want %q", msg.Metadata["pr_number"], "123")
	}
	if msg.Metadata["target"] != "tech-lead-01" {
		t.Errorf("target metadata = %q, want %q", msg.Metadata["target"], "tech-lead-01")
	}
}

func TestIsReviewRequest(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"@tech-lead PR #123 ready for review", true},
		{"please review PR #456", true},
		{"Hello world", false},
		{"PR #123 is broken", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsReviewRequest(tt.input); got != tt.want {
				t.Errorf("IsReviewRequest(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"PR #123", 123},
		{"pr 456", 456},
		{"#789", 789},
		{"no pr here", 0},
		{"multiple #1 and #2", 1}, // returns first match
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractPRNumber(tt.input); got != tt.want {
				t.Errorf("ExtractPRNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractMentions(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"@tech-lead review please", []string{"tech-lead"}},
		{"@alice and @bob please review", []string{"alice", "bob"}},
		{"no mentions here", []string{}},
		{"@user_name with underscore", []string{"user_name"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractMentions(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractMentions(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractMentions(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReviewRequestRoundTrip(t *testing.T) {
	// Format a request, then parse it back
	formatted := FormatReviewRequest(123, "tech-lead-01")
	parsed := ParseReviewRequest(formatted)

	if parsed == nil {
		t.Fatal("failed to parse formatted request")
	}
	if parsed.PRNumber != 123 {
		t.Errorf("PRNumber = %d, want 123", parsed.PRNumber)
	}
	if parsed.Target != "tech-lead-01" {
		t.Errorf("Target = %q, want %q", parsed.Target, "tech-lead-01")
	}
}
