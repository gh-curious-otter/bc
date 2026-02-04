package tui

import tea "github.com/charmbracelet/bubbletea"

// isEnterKey returns true for Enter/Return in any form the terminal may send it.
// Some environments (e.g. Cursor) send KeyRunes with '\r' instead of KeyEnter.
func isEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter {
		return true
	}
	s := msg.String()
	if s == "enter" || s == "return" {
		return true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '\r' {
		return true
	}
	return false
}
