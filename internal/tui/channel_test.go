package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
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

// newTestChannelModelWithStore returns a ChannelModel with store and manager set
// so that sendMessage can run without panicking (e.g. for testing send key behavior).
func newTestChannelModelWithStore(t *testing.T) *ChannelModel {
	t.Helper()
	dir := t.TempDir()
	store := channel.NewStore(dir)
	if _, err := store.Create("standup"); err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"coordinator", "eng-01", "eng-02"} {
		if err := store.AddMember("standup", m); err != nil {
			t.Fatal(err)
		}
	}
	mgr := agent.NewManager(dir)
	ch, _ := store.Get("standup")
	ch.History = []channel.HistoryEntry{
		{Sender: "eng-01", Message: "started work", Time: time.Now().Add(-5 * time.Minute)},
		{Sender: "eng-02", Message: "fixing tests", Time: time.Now().Add(-3 * time.Minute)},
		{Sender: "coordinator", Message: "good progress", Time: time.Now().Add(-1 * time.Minute)},
	}
	return &ChannelModel{
		channel: ch,
		store:   store,
		manager: mgr,
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

// TestClampScroll_KeepsScrollInBounds ensures scroll is clamped after history shrinks or view changes.
func TestClampScroll_KeepsScrollInBounds(t *testing.T) {
	m := newTestChannelModel()
	for i := 0; i < 50; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{
			Sender: "u", Message: "m", Time: time.Now(),
		})
	}
	m.scroll = 1000 // out of bounds
	m.clampScroll()
	maxScroll := len(m.channel.History) - m.visibleMsgCount()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll != maxScroll {
		t.Errorf("clampScroll: scroll should be %d, got %d", maxScroll, m.scroll)
	}
}

// TestClampScroll_ZeroStaysZero ensures scroll 0 is not changed when valid.
func TestClampScroll_ZeroStaysZero(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "a", Message: "hi", Time: time.Now()},
	}
	m.scroll = 0
	m.clampScroll()
	if m.scroll != 0 {
		t.Errorf("scroll should stay 0, got %d", m.scroll)
	}
}

func TestChannelHandleMouse_WheelDown(t *testing.T) {
	m := newTestChannelModel()
	for i := 0; i < 25; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{Sender: "u", Message: "m", Time: time.Now()})
	}
	m.scroll = 10
	m.HandleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if m.scroll != 9 {
		t.Errorf("wheel down should decrease scroll to 9, got %d", m.scroll)
	}
	m.scroll = 0
	m.HandleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if m.scroll != 0 {
		t.Errorf("wheel down at 0 should stay 0, got %d", m.scroll)
	}
}

func TestChannelHandleMouse_WheelUp(t *testing.T) {
	m := newTestChannelModel()
	for i := 0; i < 25; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{Sender: "u", Message: "m", Time: time.Now()})
	}
	m.scroll = 5
	m.HandleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	if m.scroll != 6 {
		t.Errorf("wheel up should increase scroll to 6, got %d", m.scroll)
	}
	maxScroll := len(m.channel.History) - m.visibleMsgCount()
	if maxScroll < 0 {
		maxScroll = 0
	}
	for m.scroll < maxScroll {
		m.HandleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	}
	m.HandleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	if m.scroll != maxScroll {
		t.Errorf("wheel up at max should stay %d, got %d", maxScroll, m.scroll)
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
	// visibleCount is the window size (virtualized), not a fixed 20 cap
	got := m.visibleCount()
	want := m.visibleMsgCount()
	if got != want {
		t.Errorf("visibleCount = %d, want window size visibleMsgCount() = %d", got, want)
	}
	if got > 50 || got <= 0 {
		t.Errorf("visibleCount = %d, want in (0, 50] for 50 messages", got)
	}
}

// --- visibleWindow / virtualization tests (#326) ---

func TestVisibleWindow_LongHistory(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = nil
	for i := 0; i < 100; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{
			Sender:  "u",
			Message: "msg",
			Time:    time.Now(),
		})
	}
	start, end := m.visibleWindow()
	windowSize := end - start
	if windowSize != m.visibleMsgCount() {
		t.Errorf("at scroll=0 window size = %d, want visibleMsgCount() = %d", windowSize, m.visibleMsgCount())
	}
	if end != 100 {
		t.Errorf("end = %d, want 100", end)
	}
	// Scroll up (older messages)
	m.scroll = 50
	start, end = m.visibleWindow()
	if start < 0 || end > 100 || end <= start {
		t.Errorf("visibleWindow [%d,%d) invalid for 100 messages", start, end)
	}
	if end-start != windowSize && end-start != 100-start {
		t.Errorf("window size changed unexpectedly to %d", end-start)
	}
}

func TestSelectedMessageIndex_LongHistory(t *testing.T) {
	m := newTestChannelModel()
	for i := 0; i < 50; i++ {
		m.channel.History = append(m.channel.History, channel.HistoryEntry{
			Sender:  "u",
			Message: "msg",
			Time:    time.Now(),
		})
	}
	m.cursor = 2
	idx := m.selectedMessageIndex()
	start, _ := m.visibleWindow()
	if idx != start+2 {
		t.Errorf("selectedMessageIndex = %d, want start+cursor = %d", idx, start+2)
	}
	_, ok := m.selectedMessage()
	if !ok {
		t.Fatal("selectedMessage should be ok")
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

func TestHandleSendKey_EnterSendsSingleLine(t *testing.T) {
	m := newTestChannelModelWithStore(t)
	m.sendMode = true
	m.input = "hello"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.sendMode {
		t.Error("enter on single line should send and exit sendMode")
	}
	if !strings.Contains(m.sendMsg, "Sent to") && !strings.Contains(m.sendMsg, "members") {
		t.Errorf("expected send status, got %q", m.sendMsg)
	}
}

func TestHandleSendKey_EnterAddsNewlineWhenMultiline(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "line1\n"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.sendMode {
		t.Error("enter in multi-line should stay in sendMode and add newline")
	}
	if m.input != "line1\n\n" {
		t.Errorf("expected 'line1\\n\\n', got %q", m.input)
	}
}

func TestHandleSendKey_AltEnterSends(t *testing.T) {
	m := newTestChannelModelWithStore(t)
	m.sendMode = true
	m.input = "test message"

	// Alt+Enter is the reliable send shortcut (Ctrl+Enter equals Enter in most terminals)
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if m.sendMode {
		t.Error("alt+enter should send and exit sendMode")
	}
	if !strings.Contains(m.sendMsg, "Sent to") && !strings.Contains(m.sendMsg, "members") {
		t.Errorf("expected send status, got %q", m.sendMsg)
	}
}

func TestHandleSendKey_CtrlEnterSends(t *testing.T) {
	m := newTestChannelModelWithStore(t)
	m.sendMode = true
	m.input = "test message"
	// Ctrl+J is a reliable send shortcut (Ctrl+Enter often sends as plain Enter in terminals)
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if m.sendMode {
		t.Error("sendMode should be false after send key")
	}
	if m.sendMsg == "" {
		t.Error("sendMsg should be set after send (e.g. 'No members in channel' or 'Sent to...')")
	}
}

func TestHandleSendKey_MultilineInput(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	// Start with one line and newline so Enter adds newline (single-line Enter would send)
	m.input = "Hi\n"

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEnter})
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T', 'e', 's', 't'}})

	if m.input != "Hi\n\nTest" {
		t.Errorf("expected 'Hi\\n\\nTest', got %q", m.input)
	}
}

// --- Autocomplete tests ---

func TestAutocomplete_TriggerMention(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	// Type @
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})

	if m.autocompleteType != AutocompleteMention {
		t.Errorf("expected AutocompleteMention, got %v", m.autocompleteType)
	}
	if len(m.autocompleteSuggestions) == 0 {
		t.Error("expected suggestions after @")
	}
}

func TestAutocomplete_TriggerChannel(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	// Type #
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})

	if m.autocompleteType != AutocompleteChannel {
		t.Errorf("expected AutocompleteChannel, got %v", m.autocompleteType)
	}
}

func TestAutocomplete_DismissOnSpace(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = ""

	// Trigger autocomplete
	m.handleSendKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	if m.autocompleteType != AutocompleteMention {
		t.Fatal("autocomplete should be active")
	}

	// Space dismisses it
	m.handleSendKey(tea.KeyMsg{Type: tea.KeySpace})
	if m.autocompleteType != AutocompleteNone {
		t.Error("space should dismiss autocomplete")
	}
}

func TestAutocomplete_DismissOnEsc(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "@"
	m.autocompleteType = AutocompleteMention
	m.autocompleteSuggestions = []string{"all", "eng-01"}

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.autocompleteType != AutocompleteNone {
		t.Error("esc should dismiss autocomplete")
	}
	// Esc only dismisses autocomplete, keeps send mode active
	if !m.sendMode {
		t.Error("esc should keep send mode active when autocomplete was shown")
	}
}

func TestAutocomplete_NavigateDown(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.autocompleteType = AutocompleteMention
	m.autocompleteSuggestions = []string{"all", "eng-01", "eng-02"}
	m.autocompleteSelected = 0

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.autocompleteSelected != 1 {
		t.Errorf("expected selection 1, got %d", m.autocompleteSelected)
	}
}

func TestAutocomplete_NavigateUp(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.autocompleteType = AutocompleteMention
	m.autocompleteSuggestions = []string{"all", "eng-01", "eng-02"}
	m.autocompleteSelected = 2

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyUp})
	if m.autocompleteSelected != 1 {
		t.Errorf("expected selection 1, got %d", m.autocompleteSelected)
	}
}

func TestAutocomplete_SelectWithTab(t *testing.T) {
	m := newTestChannelModel()
	m.sendMode = true
	m.input = "@e"
	m.autocompleteType = AutocompleteMention
	m.autocompleteSuggestions = []string{"eng-01", "eng-02"}
	m.autocompleteSelected = 0

	m.handleSendKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.autocompleteType != AutocompleteNone {
		t.Error("tab should dismiss autocomplete after selection")
	}
	if !strings.Contains(m.input, "@eng-01") {
		t.Errorf("expected @eng-01 in input, got %q", m.input)
	}
}

func TestGetMentionSuggestions(t *testing.T) {
	m := newTestChannelModel()

	suggestions := m.getMentionSuggestions("e")
	// Should include channel members starting with 'e'
	found := false
	for _, s := range suggestions {
		if strings.HasPrefix(strings.ToLower(s), "e") || s == "all" {
			found = true
			break
		}
	}
	if !found && len(suggestions) > 0 {
		t.Error("suggestions should include items starting with 'e' or be empty")
	}
}

func TestGetMentionSuggestions_All(t *testing.T) {
	m := newTestChannelModel()

	suggestions := m.getMentionSuggestions("a")
	// Should include @all
	found := false
	for _, s := range suggestions {
		if s == "all" {
			found = true
			break
		}
	}
	if !found {
		t.Error("suggestions should include 'all'")
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

func TestChannelView_SummaryVisible(t *testing.T) {
	m := newTestChannelModel()
	m.channel.Description = "Daily sync and status"

	output := m.View()
	if !strings.Contains(output, "Summary:") {
		t.Error("expected 'Summary:' label in header")
	}
	if !strings.Contains(output, "Daily sync and status") {
		t.Error("expected summary text visible and readable")
	}
}

func TestChannelView_MemberListVisible(t *testing.T) {
	m := newTestChannelModel()
	// newTestChannelModel has Members: coordinator, eng-01, eng-02
	output := m.View()
	if !strings.Contains(output, "Members:") {
		t.Error("expected 'Members:' label in channel view")
	}
	for _, name := range m.channel.Members {
		if !strings.Contains(output, name) {
			t.Errorf("expected member %q visible in member list", name)
		}
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

func TestChannelView_MemberListInSummary(t *testing.T) {
	m := newTestChannelModel()

	output := m.View()
	if !strings.Contains(output, "Members:") {
		t.Errorf("expected 'Members:' label in channel summary, got: %s", output)
	}
	// Member names from newTestChannelModel should appear
	for _, name := range []string{"coordinator", "eng-01", "eng-02"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected member %q in view", name)
		}
	}
}

func TestChannelView_MemberListEmpty(t *testing.T) {
	m := newTestChannelModel()
	m.channel.Members = nil

	output := m.View()
	if !strings.Contains(output, "Members:") {
		t.Errorf("expected 'Members:' label even with no members, got: %s", output)
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

// --- formatRelativeTime tests ---

func TestFormatRelativeTime_Now(t *testing.T) {
	now := time.Now()
	result := formatRelativeTimeFrom(now, now)
	if result != "now" {
		t.Errorf("expected 'now', got %q", result)
	}
}

func TestFormatRelativeTime_Minutes(t *testing.T) {
	now := time.Now()
	msgTime := now.Add(-5 * time.Minute)
	result := formatRelativeTimeFrom(msgTime, now)
	if result != "5m ago" {
		t.Errorf("expected '5m ago', got %q", result)
	}
}

func TestFormatRelativeTime_Hours(t *testing.T) {
	now := time.Now()
	msgTime := now.Add(-3 * time.Hour)
	result := formatRelativeTimeFrom(msgTime, now)
	if result != "3h ago" {
		t.Errorf("expected '3h ago', got %q", result)
	}
}

func TestFormatRelativeTime_Yesterday(t *testing.T) {
	now := time.Now()
	msgTime := now.Add(-30 * time.Hour)
	result := formatRelativeTimeFrom(msgTime, now)
	if !strings.HasPrefix(result, "yesterday") {
		t.Errorf("expected result to start with 'yesterday', got %q", result)
	}
}

func TestFormatRelativeTime_OlderDate(t *testing.T) {
	now := time.Now()
	msgTime := now.Add(-72 * time.Hour) // 3 days ago
	result := formatRelativeTimeFrom(msgTime, now)
	// Should contain the month abbreviation
	if !strings.Contains(result, msgTime.Format("Jan")) {
		t.Errorf("expected date format with month, got %q", result)
	}
}

// --- isSameDay tests ---

func TestIsSameDay_Same(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 23, 59, 0, 0, time.UTC)
	if !isSameDay(t1, t2) {
		t.Error("same day should return true")
	}
}

func TestIsSameDay_Different(t *testing.T) {
	t1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC)
	if isSameDay(t1, t2) {
		t.Error("different days should return false")
	}
}

// --- formatDateSeparator tests ---

func TestFormatDateSeparator_Today(t *testing.T) {
	now := time.Now()
	result := formatDateSeparatorFrom(now, now)
	if result != "Today" {
		t.Errorf("expected 'Today', got %q", result)
	}
}

func TestFormatDateSeparator_Yesterday(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	result := formatDateSeparatorFrom(yesterday, now)
	if result != "Yesterday" {
		t.Errorf("expected 'Yesterday', got %q", result)
	}
}

func TestFormatDateSeparator_OlderDate(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	oldDate := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	result := formatDateSeparatorFrom(oldDate, now)
	// Should include day name and date
	if !strings.Contains(result, "Wednesday") || !strings.Contains(result, "Jan 10") {
		t.Errorf("expected 'Wednesday, Jan 10, 2024', got %q", result)
	}
}

// --- View tests with date separators ---

func TestChannelView_DateSeparators(t *testing.T) {
	m := newTestChannelModel()
	now := time.Now()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "eng-01", Message: "old message", Time: now.AddDate(0, 0, -2)},
		{Sender: "eng-02", Message: "yesterday message", Time: now.AddDate(0, 0, -1)},
		{Sender: "eng-03", Message: "today message", Time: now},
	}

	output := m.View()
	if !strings.Contains(output, "Today") {
		t.Errorf("expected 'Today' separator in output")
	}
	if !strings.Contains(output, "Yesterday") {
		t.Errorf("expected 'Yesterday' separator in output")
	}
}

func TestChannelView_RelativeTimestamps(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "eng-01", Message: "recent message", Time: time.Now().Add(-5 * time.Minute)},
	}

	output := m.View()
	if !strings.Contains(output, "5m ago") {
		t.Errorf("expected '5m ago' in output, got: %s", output)
	}
}

// --- Message grouping tests ---

func TestChannelView_MessageGrouping(t *testing.T) {
	m := newTestChannelModel()
	now := time.Now()

	// Add consecutive messages from the same sender
	m.channel.History = []channel.HistoryEntry{
		{Sender: "engineer-01", Message: "first message", Time: now.Add(-2 * time.Minute)},
		{Sender: "engineer-01", Message: "second message", Time: now.Add(-1 * time.Minute)},
		{Sender: "engineer-02", Message: "different sender", Time: now},
	}

	output := m.View()

	// All messages should be in output
	if !strings.Contains(output, "first message") {
		t.Error("expected 'first message' in output")
	}
	if !strings.Contains(output, "second message") {
		t.Error("expected 'second message' in output")
	}
	if !strings.Contains(output, "different sender") {
		t.Error("expected 'different sender' in output")
	}

	// Both senders should be present (grouping reduces headers, not removes them)
	if !strings.Contains(output, "engineer-01") {
		t.Error("expected engineer-01 in output")
	}
	if !strings.Contains(output, "engineer-02") {
		t.Error("expected engineer-02 in output")
	}
}

func TestChannelView_MessageBubbleRendering(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "engineer-01", Message: "test bubble content", Time: time.Now()},
	}

	output := m.View()

	// Verify the message content is rendered
	if !strings.Contains(output, "test bubble content") {
		t.Error("expected message content in output")
	}

	// Verify sender is rendered
	if !strings.Contains(output, "engineer-01") {
		t.Error("expected sender name in output")
	}
}

func TestChannelView_EmptyMessageShowsPlaceholder(t *testing.T) {
	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "engineer-01", Message: "", Time: time.Now()},
	}

	output := m.View()

	if !strings.Contains(output, "(empty)") {
		t.Error("expected (empty) placeholder for empty message, got output without it")
	}
	if !strings.Contains(output, "engineer-01") {
		t.Error("expected sender name in output")
	}
}

func TestChannelView_OwnMessageUsesDistinctBubble(t *testing.T) {
	prev := os.Getenv("BC_AGENT_ID")
	defer func() { _ = os.Setenv("BC_AGENT_ID", prev) }()
	_ = os.Setenv("BC_AGENT_ID", "engineer-02")

	m := newTestChannelModel()
	m.channel.History = []channel.HistoryEntry{
		{Sender: "engineer-02", Message: "my own message", Time: time.Now()},
		{Sender: "engineer-01", Message: "other message", Time: time.Now()},
	}

	output := m.View()

	// Both messages must appear; own message uses MessageBubbleOwn (we can't assert ANSI, just content)
	if !strings.Contains(output, "my own message") {
		t.Error("expected own message in output")
	}
	if !strings.Contains(output, "other message") {
		t.Error("expected other message in output")
	}
	if !strings.Contains(output, "engineer-02") || !strings.Contains(output, "engineer-01") {
		t.Error("expected both senders in output")
	}
}
