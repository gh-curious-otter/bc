package channel

import (
	"testing"
)

func TestParseApprovalMessage(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus ApprovalStatus
		wantPR     int
		wantNil    bool
	}{
		{
			name:       "approved with checkmark",
			input:      "PR #123 approved ✓",
			wantPR:     123,
			wantStatus: StatusApproved,
		},
		{
			name:       "lgtm",
			input:      "LGTM PR #456",
			wantPR:     456,
			wantStatus: StatusApproved,
		},
		{
			name:       "looks good",
			input:      "PR #789 looks good to me",
			wantPR:     789,
			wantStatus: StatusApproved,
		},
		{
			name:       "needs changes",
			input:      "PR #100 needs changes: fix the tests",
			wantPR:     100,
			wantStatus: StatusChangesRequested,
		},
		{
			name:       "please fix",
			input:      "PR #200 please fix the formatting",
			wantPR:     200,
			wantStatus: StatusChangesRequested,
		},
		{
			name:       "emoji approval",
			input:      "PR #300 ✅",
			wantPR:     300,
			wantStatus: StatusApproved,
		},
		{
			name:    "not an approval",
			input:   "Hello world",
			wantNil: true,
		},
		{
			name:    "no PR number",
			input:   "LGTM on this change",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseApprovalMessage(tt.input)
			if tt.wantNil {
				if msg != nil {
					t.Errorf("expected nil, got %+v", msg)
				}
				return
			}
			if msg == nil {
				t.Fatal("expected non-nil result")
			}
			if msg.PRNumber != tt.wantPR {
				t.Errorf("PRNumber = %d, want %d", msg.PRNumber, tt.wantPR)
			}
			if msg.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", msg.Status, tt.wantStatus)
			}
		})
	}
}

func TestFormatApprovalMessage(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		status   ApprovalStatus
		prNumber int
	}{
		{
			name:     "approved",
			prNumber: 123,
			status:   StatusApproved,
			want:     "PR #123 approved ✓",
		},
		{
			name:     "changes requested",
			prNumber: 456,
			status:   StatusChangesRequested,
			want:     "PR #456 needs changes",
		},
		{
			name:     "commented",
			prNumber: 789,
			status:   StatusCommented,
			want:     "PR #789 reviewed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatApprovalMessage(tt.prNumber, tt.status)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatApprovalWithComment(t *testing.T) {
	got := FormatApprovalWithComment(123, StatusApproved, "Great work!")
	want := "PR #123 approved ✓: Great work!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Empty comment
	got = FormatApprovalWithComment(456, StatusChangesRequested, "")
	want = "PR #456 needs changes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewApprovalMessage(t *testing.T) {
	msg := NewApprovalMessage(123, StatusApproved, "tech-lead-01")

	if msg.Type != TypeApproval {
		t.Errorf("Type = %v, want %v", msg.Type, TypeApproval)
	}
	if msg.Sender != "tech-lead-01" {
		t.Errorf("Sender = %q, want %q", msg.Sender, "tech-lead-01")
	}
	if msg.Metadata["pr_number"] != "123" {
		t.Errorf("pr_number metadata = %q, want %q", msg.Metadata["pr_number"], "123")
	}
	if msg.Metadata["status"] != string(StatusApproved) {
		t.Errorf("status metadata = %q, want %q", msg.Metadata["status"], string(StatusApproved))
	}
}

func TestIsApproved(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"PR #123 approved ✓", true},
		{"LGTM PR #456", true},
		{"PR #789 needs changes", false},
		{"Hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsApproved(tt.input); got != tt.want {
				t.Errorf("IsApproved(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsChangesRequested(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"PR #123 needs changes", true},
		{"PR #456 please fix the tests", true},
		{"PR #789 approved ✓", false},
		{"Hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsChangesRequested(tt.input); got != tt.want {
				t.Errorf("IsChangesRequested(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMergeNotification(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantBranch string
		wantPR     int
		wantNil    bool
	}{
		{
			name:       "merged to main",
			input:      "PR #123 merged to main",
			wantPR:     123,
			wantBranch: "main",
		},
		{
			name:       "merged PR",
			input:      "Merged PR #456",
			wantPR:     456,
			wantBranch: "",
		},
		{
			name:       "pushed to develop",
			input:      "PR #789 pushed to develop",
			wantPR:     789,
			wantBranch: "develop",
		},
		{
			name:    "not a merge",
			input:   "PR #100 approved",
			wantNil: true,
		},
		{
			name:    "no PR number",
			input:   "Merged to main",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseMergeNotification(tt.input)
			if tt.wantNil {
				if msg != nil {
					t.Errorf("expected nil, got %+v", msg)
				}
				return
			}
			if msg == nil {
				t.Fatal("expected non-nil result")
			}
			if msg.PRNumber != tt.wantPR {
				t.Errorf("PRNumber = %d, want %d", msg.PRNumber, tt.wantPR)
			}
			if msg.Branch != tt.wantBranch {
				t.Errorf("Branch = %q, want %q", msg.Branch, tt.wantBranch)
			}
		})
	}
}

func TestFormatMergeNotification(t *testing.T) {
	tests := []struct {
		branch   string
		want     string
		prNumber int
	}{
		{"main", "PR #123 merged to main", 123},
		{"develop", "PR #456 merged to develop", 456},
		{"", "PR #789 merged to main", 789}, // default to main
	}

	for _, tt := range tests {
		got := FormatMergeNotification(tt.prNumber, tt.branch)
		if got != tt.want {
			t.Errorf("FormatMergeNotification(%d, %q) = %q, want %q",
				tt.prNumber, tt.branch, got, tt.want)
		}
	}
}

func TestNewMergeNotificationMessage(t *testing.T) {
	msg := NewMergeNotificationMessage(123, "main", "manager")

	if msg.Type != TypeMerge {
		t.Errorf("Type = %v, want %v", msg.Type, TypeMerge)
	}
	if msg.Sender != "manager" {
		t.Errorf("Sender = %q, want %q", msg.Sender, "manager")
	}
	if msg.Metadata["pr_number"] != "123" {
		t.Errorf("pr_number metadata = %q, want %q", msg.Metadata["pr_number"], "123")
	}
	if msg.Metadata["branch"] != "main" {
		t.Errorf("branch metadata = %q, want %q", msg.Metadata["branch"], "main")
	}
}

func TestIsMergeNotification(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"PR #123 merged to main", true},
		{"Merged PR #456", true},
		{"PR #789 approved ✓", false},
		{"Hello world", false},
	}

	for _, tt := range tests {
		if got := IsMergeNotification(tt.input); got != tt.want {
			t.Errorf("IsMergeNotification(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestApprovalRoundTrip(t *testing.T) {
	// Format a message, then parse it back
	formatted := FormatApprovalMessage(123, StatusApproved)
	parsed := ParseApprovalMessage(formatted)

	if parsed == nil {
		t.Fatal("failed to parse formatted approval")
	}
	if parsed.PRNumber != 123 {
		t.Errorf("PRNumber = %d, want 123", parsed.PRNumber)
	}
	if parsed.Status != StatusApproved {
		t.Errorf("Status = %v, want %v", parsed.Status, StatusApproved)
	}
}

func TestMergeRoundTrip(t *testing.T) {
	// Format a message, then parse it back
	formatted := FormatMergeNotification(456, "main")
	parsed := ParseMergeNotification(formatted)

	if parsed == nil {
		t.Fatal("failed to parse formatted merge notification")
	}
	if parsed.PRNumber != 456 {
		t.Errorf("PRNumber = %d, want 456", parsed.PRNumber)
	}
	if parsed.Branch != "main" {
		t.Errorf("Branch = %q, want %q", parsed.Branch, "main")
	}
}

func TestIsApprovalMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"approved message", "PR #123 approved ✓", true},
		{"lgtm message", "LGTM PR #456", true},
		{"needs changes", "PR #100 needs changes", true},
		{"regular message", "Hello everyone", false},
		{"empty message", "", false},
		{"pr mention context", "I submitted PR #999", true}, // Contains "pr" keyword
		{"emoji approval", "PR #300 ✅", true},
		{"no pr number", "This looks good", false},
		{"unrelated text", "The weather is nice", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsApprovalMessage(tc.content)
			if got != tc.want {
				t.Errorf("IsApprovalMessage(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}
