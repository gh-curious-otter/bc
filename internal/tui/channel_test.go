package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/tui/style"
)

func newTestChannelModel() *ChannelModel {
	ch := &channel.Channel{
		Name:    "standup",
		Members: []string{"coordinator", "eng-01", "eng-02"},
		History: []channel.HistoryEntry{
			{Sender: "eng-01", Message: "started work", Time: time.Now().Add(-5 * time.Minute)},
			{Sender: "eng-02", Message: "fixing tests", Time: time.Now().Add(-3 * time.Minute)},
			{Sender: "coordinator", Message: "good progress", Time: time.Now().Add(-1 * time.Minute)},
		},
	}
	return &ChannelModel{
		channel: ch,
		styles:  style.DefaultStyles(),
		width:   120,
		height:  40,
	}
}

// --- HandleKey tests ---

func TestChannelHandleKey_Esc(t *testing.T) {
	m := newTestChannelModel()
	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if action.Type != ActionBack {
		t.Errorf("esc should return ActionBack, got %d", action.Type)
	}
}

func TestChannelHandleKey_ScrollDown(t *testing.T) {
	m := newTestChannelModel()
	// Initially scroll is 0 (at bottom)
	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action.Type != ActionNone {
		t.Errorf("j should return NoAction, got %d", action.Type)
	}
}

func TestChannelHandleKey_ScrollUp(t *testing.T) {
	m := newTestChannelModel()
	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if action.Type != ActionNone {
		t.Errorf("k should return NoAction, got %d", action.Type)
	}
}

func TestChannelHandleKey_SendMode(t *testing.T) {
	m := newTestChannelModel()
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.sendMode {
		t.Error("s should enable sendMode")
	}
}

func TestChannelHandleKey_HomeEnd(t *testing.T) {
	m := newTestChannelModel()
	// Add enough history to scroll
	for i := 0; i < 30; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{
			Sender:  "bot",
			Message: "msg",
			Time:    time.Now(),
		})
	}

	// Home - scroll to oldest
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.scroll == 0 {
		t.Error("g should scroll to oldest messages")
	}

	// End - scroll to newest
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.scroll != 0 {
		t.Errorf("G should scroll to newest (scroll=0), got %d", m.scroll)
	}
}

func TestChannelHandleKey_CreateIssue(t *testing.T) {
	m := newTestChannelModel()
	m.cursor = 0

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if action.Type != ActionCreateIssue {
		t.Errorf("i should return ActionCreateIssue, got %d", action.Type)
	}
}

func TestChannelHandleKey_CreateIssue_EmptyHistory(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = nil

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if action.Type != ActionNone {
		t.Errorf("i on empty channel should return NoAction, got %d", action.Type)
	}
}

// --- visibleCount tests ---

func TestVisibleCount_Small(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "a", Message: "hi"},
	}

	if m.visibleCount() != 1 {
		t.Errorf("visibleCount = %d, want 1", m.visibleCount())
	}
}

func TestVisibleCount_Large(t *testing.T) {
	m := newTestChannelModel()
	for i := 0; i < 50; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{Sender: "a", Message: "msg"})
	}

	if m.visibleCount() != 20 {
		t.Errorf("visibleCount = %d, want 20 (max)", m.visibleCount())
	}
}

// --- selectedMessage tests ---

func TestSelectedMessage_Valid(t *testing.T) {
	m := newTestChannelModel()
	m.cursor = 1

	entry, ok := m.selectedMessage()
	if !ok {
		t.Fatal("expected selectedMessage to return true")
	}
	if entry.Sender != "eng-02" {
		t.Errorf("expected eng-02, got %s", entry.Sender)
	}
}

func TestSelectedMessage_EmptyHistory(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = nil
	m.cursor = 0

	_, ok := m.selectedMessage()
	if ok {
		t.Error("expected false for empty history")
	}
}

func TestSelectedMessage_OutOfBounds(t *testing.T) {
	m := newTestChannelModel()
	m.cursor = 100

	_, ok := m.selectedMessage()
	if ok {
		t.Error("expected false for out of bounds cursor")
	}
}

// --- handleSendKey tests ---

func TestHandleSendKey_Esc(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "partial message"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.sendMode {
		t.Error("esc should exit sendMode")
	}
	if m.input != "" {
		t.Error("esc should clear input")
	}
}

func TestHandleSendKey_Backspace(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "hello"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.input != "hell" {
		t.Errorf("backspace should remove last char, got %q", m.input)
	}
}

func TestHandleSendKey_BackspaceEmpty(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.input != "" {
		t.Errorf("backspace on empty should stay empty, got %q", m.input)
	}
}

func TestHandleSendKey_TypeRunes(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.input != "hi" {
		t.Errorf("expected 'hi', got %q", m.input)
	}
}

func TestHandleSendKey_Space(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "hello"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeySpace})
	if m.input != "hello " {
		t.Errorf("expected 'hello ', got %q", m.input)
	}
}

func TestHandleSendKey_EnterEmptyDoesNotSend(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.sendMode {
		t.Error("enter should exit sendMode even with empty input")
	}
}

// --- wrapText tests ---

func TestWrapText_ShortText(t *testing.T) {
	lines := wrapText("hello", 80)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected single line 'hello', got %v", lines)
	}
}

func TestWrapText_ExactWidth(t *testing.T) {
	text := "12345"
	lines := wrapText(text, 5)
	if len(lines) != 1 {
		t.Errorf("text at exact width should be 1 line, got %d", len(lines))
	}
}

func TestWrapText_WrapsAtSpace(t *testing.T) {
	text := "hello world foo"
	lines := wrapText(text, 12)
	if len(lines) < 2 {
		t.Errorf("expected wrapping, got %d lines: %v", len(lines), lines)
	}
	if lines[0] != "hello world" {
		t.Errorf("first line = %q, expected 'hello world'", lines[0])
	}
}

func TestWrapText_NoSpaces(t *testing.T) {
	text := "abcdefghijklmnop"
	lines := wrapText(text, 5)
	if len(lines) < 2 {
		t.Errorf("expected wrapping even without spaces, got %d lines", len(lines))
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	lines := wrapText("hello", 0)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("zero width should return original, got %v", lines)
	}
}

// --- View tests ---

func TestChannelView_NoMessages(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = nil

	output := m.View()
	if !strings.Contains(output, "# standup") {
		t.Errorf("expected channel name, got: %s", output)
	}
	if !strings.Contains(output, "No messages") {
		t.Errorf("expected 'No messages', got: %s", output)
	}
}

func TestChannelView_WithMessages(t *testing.T) {
	m := newTestChannelModel()

	output := m.View()
	if !strings.Contains(output, "# standup") {
		t.Errorf("expected channel name, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected sender in output, got: %s", output)
	}
	if !strings.Contains(output, "started work") {
		t.Errorf("expected message content, got: %s", output)
	}
	if !strings.Contains(output, "members") {
		t.Errorf("expected member count, got: %s", output)
	}
}

func TestChannelView_SendMode(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "test message"

	output := m.View()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected input in send mode, got: %s", output)
	}
}

func TestChannelView_SendMsg(t *testing.T) {
	m := newTestChannelModel()
	m.sendMsg = "Sent to 3/3 members"

	output := m.View()
	if !strings.Contains(output, "Sent to 3/3 members") {
		t.Errorf("expected send status message, got: %s", output)
	}
}

func TestChannelView_NoSender(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Message: "system message", Time: time.Now()},
	}

	output := m.View()
	if !strings.Contains(output, "system") {
		t.Errorf("expected 'system' for empty sender, got: %s", output)
	}
}

func TestChannelView_WithDescription(t *testing.T) {
	m := newTestChannelModel()
	m.channel.Description = "Team standup updates"

	output := m.View()
	if !strings.Contains(output, "Team standup updates") {
		t.Errorf("expected description in header, got: %s", output)
	}
}

func TestChannelView_OnlineIndicator(t *testing.T) {
	m := newTestChannelModel()

	output := m.View()
	// Should show member count with online indicator
	if !strings.Contains(output, "members") {
		t.Errorf("expected member count in header, got: %s", output)
	}
}

func TestChannelView_QuickActions(t *testing.T) {
	m := newTestChannelModel()

	output := m.View()
	// Should show quick action hints in header
	if !strings.Contains(output, "[s]") {
		t.Errorf("expected send action hint, got: %s", output)
	}
	if !strings.Contains(output, "[r]") {
		t.Errorf("expected refresh action hint, got: %s", output)
	}
	if !strings.Contains(output, "[esc]") {
		t.Errorf("expected back action hint, got: %s", output)
	}
}

// --- visibleMsgCount tests ---

func TestVisibleMsgCount(t *testing.T) {
	m := newTestChannelModel()
	m.height = 40

	count := m.visibleMsgCount()
	if count < 1 {
		t.Errorf("visibleMsgCount should be positive, got %d", count)
	}
}
