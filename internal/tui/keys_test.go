package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsEnterKey_KeyEnter(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	if !isEnterKey(msg) {
		t.Error("KeyEnter should return true")
	}
}

func TestIsEnterKey_CarriageReturn(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}}
	if !isEnterKey(msg) {
		t.Error("rune '\\r' should return true")
	}
}

func TestIsEnterKey_RegularKey(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	if isEnterKey(msg) {
		t.Error("regular key 'a' should return false")
	}
}

func TestIsEnterKey_EscKey(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	if isEnterKey(msg) {
		t.Error("escape should return false")
	}
}

func TestIsEnterKey_MultipleRunes(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}}
	if isEnterKey(msg) {
		t.Error("multiple runes should return false")
	}
}
