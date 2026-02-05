package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsEnterKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{
			name: "KeyEnter type",
			msg:  tea.KeyMsg{Type: tea.KeyEnter},
			want: true,
		},
		{
			name: "runes with carriage return",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}},
			want: true,
		},
		{
			name: "regular rune j",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			want: false,
		},
		{
			name: "escape key",
			msg:  tea.KeyMsg{Type: tea.KeyEsc},
			want: false,
		},
		{
			name: "space key",
			msg:  tea.KeyMsg{Type: tea.KeySpace},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEnterKey(tt.msg)
			if got != tt.want {
				t.Errorf("isEnterKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
