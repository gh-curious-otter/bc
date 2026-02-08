package channel

import "testing"

func TestFormatAgentComment(t *testing.T) {
	tests := []struct {
		agentID string
		body    string
		want    string
	}{
		{"engineer-01", "Done with login API", "**[engineer-01]** Done with login API"},
		{"manager", "Please review PR #42", "**[manager]** Please review PR #42"},
		{"cli", "", "**[cli]**"},
		{"", "No agent", "No agent"},
		{"bot", "single", "**[bot]** single"},
	}
	for _, tt := range tests {
		got := FormatAgentComment(tt.agentID, tt.body)
		if got != tt.want {
			t.Errorf("FormatAgentComment(%q, %q) = %q, want %q", tt.agentID, tt.body, got, tt.want)
		}
	}
}
