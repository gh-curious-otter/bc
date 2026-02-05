package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/workspace"
)

func newTestHomeModel() *HomeModel {
	return &HomeModel{
		screen: ScreenHome,
		styles: newTestModel().styles,
		width:  120,
		height: 40,
		workspaces: []WorkspaceInfo{
			{
				Entry:      workspace.RegistryEntry{Name: "project-a", Path: "/home/user/project-a"},
				Running:    3,
				Total:      5,
				MaxWorkers: 10,
				Issues:     12,
				HasBeads:   true,
			},
			{
				Entry:      workspace.RegistryEntry{Name: "project-b", Path: "/home/user/project-b"},
				Running:    0,
				Total:      0,
				MaxWorkers: 10,
				Issues:     0,
				HasBeads:   false,
			},
		},
		maxWorkers: 10,
	}
}

// --- NewHomeModel tests ---

func TestNewHomeModel(t *testing.T) {
	ws := []WorkspaceInfo{
		{Entry: workspace.RegistryEntry{Name: "test"}},
	}
	m := NewHomeModel(ws, 5)

	if m.screen != ScreenHome {
		t.Errorf("screen = %d, want ScreenHome", m.screen)
	}
	if len(m.workspaces) != 1 {
		t.Errorf("workspaces count = %d, want 1", len(m.workspaces))
	}
	if m.maxWorkers != 5 {
		t.Errorf("maxWorkers = %d, want 5", m.maxWorkers)
	}
}

// --- renderHeader tests ---

func TestRenderHeader_HomeScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenHome

	output := m.renderHeader()
	if !strings.Contains(output, "bc") {
		t.Errorf("expected 'bc' in header, got: %s", output)
	}
	if !strings.Contains(output, "home") {
		t.Errorf("expected 'home' label, got: %s", output)
	}
}

func TestRenderHeader_WorkspaceScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace
	m.wsModel = newTestModel()

	output := m.renderHeader()
	if !strings.Contains(output, "test-project") {
		t.Errorf("expected workspace name in header, got: %s", output)
	}
}

func TestRenderHeader_ChannelScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenChannel
	m.channelModel = &ChannelModel{
		channel: &channel.Channel{Name: "standup"},
		styles:  m.styles,
	}

	output := m.renderHeader()
	if !strings.Contains(output, "#standup") {
		t.Errorf("expected '#standup' in header, got: %s", output)
	}
}

// --- renderHomeScreen tests ---

func TestRenderHomeScreen_WithWorkspaces(t *testing.T) {
	m := newTestHomeModel()

	output := m.renderHomeScreen()
	if !strings.Contains(output, "Workspaces") {
		t.Errorf("expected 'Workspaces' title, got: %s", output)
	}
	if !strings.Contains(output, "project-a") {
		t.Errorf("expected project-a in listing, got: %s", output)
	}
	if !strings.Contains(output, "project-b") {
		t.Errorf("expected project-b in listing, got: %s", output)
	}
	if !strings.Contains(output, "3 running") {
		t.Errorf("expected '3 running' for project-a, got: %s", output)
	}
	if !strings.Contains(output, "stopped") {
		t.Errorf("expected 'stopped' for project-b, got: %s", output)
	}
}

func TestRenderHomeScreen_NoWorkspaces(t *testing.T) {
	m := newTestHomeModel()
	m.workspaces = nil

	output := m.renderHomeScreen()
	if !strings.Contains(output, "No workspaces registered") {
		t.Errorf("expected 'No workspaces registered', got: %s", output)
	}
}

func TestRenderHomeScreen_OverLimit(t *testing.T) {
	m := newTestHomeModel()
	m.maxWorkers = 2 // project-a has 3 running, which exceeds 2

	output := m.renderHomeScreen()
	// Should still render without error
	if !strings.Contains(output, "project-a") {
		t.Errorf("expected project-a even when over limit")
	}
}

// --- renderHelp tests ---

func TestRenderHelp_HomeScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenHome

	output := m.renderHelp()
	if !strings.Contains(output, "Keyboard Shortcuts") {
		t.Errorf("expected 'Keyboard Shortcuts' title, got: %s", output)
	}
	if !strings.Contains(output, "Global") {
		t.Errorf("expected 'Global' section, got: %s", output)
	}
	if !strings.Contains(output, "Home") {
		t.Errorf("expected 'Home' section, got: %s", output)
	}
	if !strings.Contains(output, "Enter") {
		t.Errorf("expected Enter shortcut, got: %s", output)
	}
}

func TestRenderHelp_WorkspaceScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace

	output := m.renderHelp()
	if !strings.Contains(output, "Workspace") {
		t.Errorf("expected 'Workspace' section, got: %s", output)
	}
	if !strings.Contains(output, "Tab") {
		t.Errorf("expected Tab shortcut, got: %s", output)
	}
}

func TestRenderHelp_AgentScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenAgent

	output := m.renderHelp()
	if !strings.Contains(output, "Agent") {
		t.Errorf("expected 'Agent' section, got: %s", output)
	}
	if !strings.Contains(output, "Peek") {
		t.Errorf("expected Peek shortcut, got: %s", output)
	}
}

func TestRenderHelp_ChannelScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenChannel

	output := m.renderHelp()
	if !strings.Contains(output, "Channel") {
		t.Errorf("expected 'Channel' section, got: %s", output)
	}
}

func TestRenderHelp_IssueScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenIssue

	output := m.renderHelp()
	if !strings.Contains(output, "Detail View") {
		t.Errorf("expected 'Detail View' section, got: %s", output)
	}
}

// --- renderStatusBar tests ---

func TestRenderStatusBar_HomeScreen(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenHome

	output := m.renderStatusBar()
	if !strings.Contains(output, "navigate") {
		t.Errorf("expected navigation hints, got: %s", output)
	}
	if !strings.Contains(output, "help") {
		t.Errorf("expected help hint, got: %s", output)
	}
}

func TestRenderStatusBar_WithStatusMsg(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenHome
	m.statusMsg = "Refreshed"

	output := m.renderStatusBar()
	if !strings.Contains(output, "Refreshed") {
		t.Errorf("expected status message, got: %s", output)
	}
}

func TestRenderStatusBar_HelpActive(t *testing.T) {
	m := newTestHomeModel()
	m.helpActive = true

	output := m.renderStatusBar()
	if !strings.Contains(output, "close help") {
		t.Errorf("expected 'close help' when help is active, got: %s", output)
	}
}

func TestRenderStatusBar_AllScreens(t *testing.T) {
	screens := []Screen{ScreenHome, ScreenWorkspace, ScreenAgent, ScreenChannel, ScreenIssue, ScreenQueueItem}
	for _, screen := range screens {
		m := newTestHomeModel()
		m.screen = screen
		output := m.renderStatusBar()
		if output == "" {
			t.Errorf("status bar for screen %d is empty", screen)
		}
	}
}

// --- handleKey tests ---

func TestHandleKey_QuitKeys(t *testing.T) {
	keys := []string{"ctrl+c", "q"}
	for _, key := range keys {
		m := newTestHomeModel()
		var msg tea.KeyMsg
		if key == "ctrl+c" {
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		}
		_, cmd := m.handleKey(msg)
		if cmd == nil {
			t.Errorf("expected quit command for key %q", key)
		}
	}
}

func TestHandleKey_HelpToggle(t *testing.T) {
	m := newTestHomeModel()

	// Toggle help on
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.helpActive {
		t.Error("expected helpActive after ?")
	}

	// Toggle help off
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.helpActive {
		t.Error("expected helpActive=false after second ?")
	}
}

func TestHandleKey_EscClosesHelp(t *testing.T) {
	m := newTestHomeModel()
	m.helpActive = true

	m.handleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.helpActive {
		t.Error("esc should close help")
	}
}

func TestHandleKey_QClosesHelp(t *testing.T) {
	m := newTestHomeModel()
	m.helpActive = true

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.helpActive {
		t.Error("q should close help when active")
	}
	if cmd != nil {
		t.Error("q on help overlay should not quit")
	}
}

func TestHandleKey_AnyKeyClosesHelp(t *testing.T) {
	m := newTestHomeModel()
	m.helpActive = true

	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.helpActive {
		t.Error("any key should close help")
	}
}

// --- handleHomeKey tests ---

func TestHandleHomeKey_CursorDown(t *testing.T) {
	m := newTestHomeModel()
	m.homeCursor = 0

	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.homeCursor != 1 {
		t.Errorf("cursor = %d, want 1", m.homeCursor)
	}

	// Should not go past last workspace
	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.homeCursor != 1 {
		t.Errorf("cursor should not exceed max, got %d", m.homeCursor)
	}
}

func TestHandleHomeKey_CursorUp(t *testing.T) {
	m := newTestHomeModel()
	m.homeCursor = 1

	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.homeCursor != 0 {
		t.Errorf("cursor = %d, want 0", m.homeCursor)
	}

	// Should not go below 0
	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.homeCursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", m.homeCursor)
	}
}

func TestHandleHomeKey_HomeEnd(t *testing.T) {
	m := newTestHomeModel()
	m.homeCursor = 0

	// End
	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.homeCursor != 1 {
		t.Errorf("G should go to end, got %d", m.homeCursor)
	}

	// Home
	m.handleHomeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.homeCursor != 0 {
		t.Errorf("g should go to home, got %d", m.homeCursor)
	}
}

// --- View tests ---

func TestView_HomeScreen(t *testing.T) {
	m := newTestHomeModel()
	output := m.View()

	if !strings.Contains(output, "bc") {
		t.Errorf("expected 'bc' in view")
	}
	if !strings.Contains(output, "project-a") {
		t.Errorf("expected workspace name in view")
	}
}

func TestView_HelpOverlay(t *testing.T) {
	m := newTestHomeModel()
	m.helpActive = true

	output := m.View()
	if !strings.Contains(output, "Keyboard Shortcuts") {
		t.Errorf("expected help content when helpActive, got: %s", output)
	}
}

// --- handleAgentKey tests ---

func TestHandleAgentKey_EscGoesBack(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenAgent
	m.agentModel = &AgentModel{
		agent:  &agent.Agent{Name: "eng-01"},
		styles: m.styles,
	}

	m.handleAgentKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenWorkspace {
		t.Errorf("esc should go back to workspace, got screen %d", m.screen)
	}
	if m.agentModel != nil {
		t.Error("agentModel should be nil after esc")
	}
}

// --- handleIssueKey tests ---

func TestHandleIssueKey_EscGoesBack(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenIssue
	m.issueModel = &IssueModel{
		styles: m.styles,
	}

	m.handleIssueKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenWorkspace {
		t.Errorf("esc should go back to workspace, got screen %d", m.screen)
	}
	if m.issueModel != nil {
		t.Error("issueModel should be nil after esc")
	}
}

// --- handleWorkspaceKey tests ---

func TestHandleWorkspaceKey_EscGoesHome(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace
	m.wsModel = newTestModel()
	m.statusMsg = "some status"

	m.handleWorkspaceKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenHome {
		t.Errorf("esc should go to home, got screen %d", m.screen)
	}
	if m.wsModel != nil {
		t.Error("wsModel should be nil after esc")
	}
	if m.statusMsg != "" {
		t.Error("statusMsg should be cleared after esc")
	}
}

// --- Update tests ---

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestHomeModel()
	m.wsModel = newTestModel()

	msg := tea.WindowSizeMsg{Width: 200, Height: 50}
	m.Update(msg)

	if m.width != 200 || m.height != 50 {
		t.Errorf("width/height not updated: %d x %d", m.width, m.height)
	}
	if m.wsModel.width != 200 || m.wsModel.height != 50 {
		t.Errorf("wsModel width/height not updated: %d x %d", m.wsModel.width, m.wsModel.height)
	}
}

func TestUpdate_WindowSizeMsg_AllModels(t *testing.T) {
	m := newTestHomeModel()
	m.agentModel = &AgentModel{agent: &agent.Agent{Name: "x"}, styles: m.styles}
	m.channelModel = &ChannelModel{channel: &channel.Channel{Name: "y"}, styles: m.styles}
	m.issueModel = &IssueModel{styles: m.styles}

	msg := tea.WindowSizeMsg{Width: 150, Height: 45}
	m.Update(msg)

	if m.agentModel.width != 150 {
		t.Errorf("agentModel width = %d, want 150", m.agentModel.width)
	}
	if m.channelModel.width != 150 {
		t.Errorf("channelModel width = %d, want 150", m.channelModel.width)
	}
	if m.issueModel.width != 150 {
		t.Errorf("issueModel width = %d, want 150", m.issueModel.width)
	}
}
