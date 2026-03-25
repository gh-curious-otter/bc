package gateway

import "testing"

func TestSanitizeChannelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Marketing", "marketing"},
		{"All BC Infra", "all-bc-infra"},
		{"dev-chat", "dev-chat"},
		{"hello_world", "hello_world"},
		{"café ☕", "caf-"},
		{"UPPER CASE", "upper-case"},
		{"a/b\\c", "abc"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeChannelName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeChannelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		want  string
		n     int
	}{
		{"hello", "hello", 10},
		{"hello world", "hello...", 5},
		{"", "", 5},
		{"abc", "abc", 3},
		{"abcd", "abc...", 3},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Truncate(tt.input, tt.n)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.want)
			}
		})
	}
}

func TestManagerIsGatewayChannel(t *testing.T) {
	m := NewManager()
	if m.IsGatewayChannel("telegram:marketing") {
		t.Error("expected false for unknown channel")
	}

	m.channelMap["telegram:marketing"] = channelRoute{Platform: "telegram", ChannelID: "123"}
	if !m.IsGatewayChannel("telegram:marketing") {
		t.Error("expected true for known channel")
	}
}

func TestManagerExternalChannels(t *testing.T) {
	m := NewManager()
	if len(m.ExternalChannels()) != 0 {
		t.Error("expected empty list")
	}

	m.channelMap["telegram:marketing"] = channelRoute{Platform: "telegram"}
	m.channelMap["slack:general"] = channelRoute{Platform: "slack"}

	channels := m.ExternalChannels()
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}
