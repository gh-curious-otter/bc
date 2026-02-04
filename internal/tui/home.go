// Package tui implements the hardcoded TUI screens for bc home.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/tui/style"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Screen identifies which screen is currently active.
type Screen int

const (
	ScreenHome Screen = iota
	ScreenWorkspace
	ScreenAgent
)

// TickMsg triggers a periodic refresh.
type TickMsg struct{}

// WorkspaceInfo holds summary data for a workspace in the home view.
type WorkspaceInfo struct {
	Entry    workspace.RegistryEntry
	Running  int
	Total    int
	Issues   int
	PRs      int
	HasBeads bool
}

// HomeModel is the root TUI model for bc home.
type HomeModel struct {
	screen Screen
	styles style.Styles
	width  int
	height int

	// Home screen state
	workspaces []WorkspaceInfo
	homeCursor int

	// Workspace detail state
	wsModel *WorkspaceModel

	// Agent detail state
	agentModel *AgentModel

	// Status message
	statusMsg string
}

// NewHomeModel creates the root TUI model.
func NewHomeModel(workspaces []WorkspaceInfo) *HomeModel {
	return &HomeModel{
		screen:     ScreenHome,
		styles:     style.DefaultStyles(),
		workspaces: workspaces,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(config.Tui.RefreshInterval, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

// Init implements tea.Model.
func (m *HomeModel) Init() tea.Cmd {
	return tickCmd()
}

// Update implements tea.Model.
func (m *HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.wsModel != nil {
			m.wsModel.width = msg.Width
			m.wsModel.height = msg.Height
		}
		if m.agentModel != nil {
			m.agentModel.width = msg.Width
			m.agentModel.height = msg.Height
		}
		return m, nil

	case TickMsg:
		if m.wsModel != nil {
			m.wsModel.refresh()
		}
		return m, tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *HomeModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.statusMsg = "j/k:navigate  enter:select  esc:back  tab:switch  q:quit"
		return m, nil
	}

	// Dispatch to active screen
	switch m.screen {
	case ScreenHome:
		return m.handleHomeKey(msg)
	case ScreenWorkspace:
		return m.handleWorkspaceKey(msg)
	case ScreenAgent:
		return m.handleAgentKey(msg)
	}

	return m, nil
}

func (m *HomeModel) handleHomeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "j", "down":
		if m.homeCursor < len(m.workspaces)-1 {
			m.homeCursor++
		}
	case "k", "up":
		if m.homeCursor > 0 {
			m.homeCursor--
		}
	case "g", "home":
		m.homeCursor = 0
	case "G", "end":
		if len(m.workspaces) > 0 {
			m.homeCursor = len(m.workspaces) - 1
		}
	case "r":
		m.statusMsg = "Refreshed"
	}
	if isEnterKey(msg) {
		if m.homeCursor < len(m.workspaces) {
			ws := m.workspaces[m.homeCursor]
			m.wsModel = NewWorkspaceModel(ws, m.styles)
			m.wsModel.width = m.width
			m.wsModel.height = m.height
			m.screen = ScreenWorkspace
			m.statusMsg = ""
		}
	}
	return m, nil
}

func (m *HomeModel) handleWorkspaceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.screen = ScreenHome
		m.wsModel = nil
		m.statusMsg = ""
		return m, nil
	}

	if m.wsModel != nil {
		action := m.wsModel.HandleKey(msg)
		switch action.Type {
		case ActionDrillAgent:
			if a, ok := action.Data.(*agent.Agent); ok {
				m.agentModel = NewAgentModel(a, m.wsModel.manager, m.styles)
				m.agentModel.width = m.width
				m.agentModel.height = m.height
				m.screen = ScreenAgent
			}
		}
	}

	return m, nil
}

func (m *HomeModel) handleAgentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.screen = ScreenWorkspace
		m.agentModel = nil
		return m, nil
	}

	if m.agentModel != nil {
		action := m.agentModel.HandleKey(msg)
		switch action.Type {
		case ActionBack:
			m.screen = ScreenWorkspace
			m.agentModel = nil
		case ActionAttach:
			// Exit TUI to attach to tmux
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m *HomeModel) View() string {
	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Content
	switch m.screen {
	case ScreenHome:
		sections = append(sections, m.renderHomeScreen())
	case ScreenWorkspace:
		if m.wsModel != nil {
			sections = append(sections, m.wsModel.View())
		}
	case ScreenAgent:
		if m.agentModel != nil {
			sections = append(sections, m.agentModel.View())
		}
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *HomeModel) renderHeader() string {
	title := m.styles.Header.Render("bc")
	var screenLabel string
	switch m.screen {
	case ScreenHome:
		screenLabel = "home"
	case ScreenWorkspace:
		if m.wsModel != nil {
			screenLabel = m.wsModel.info.Entry.Name
		}
	case ScreenAgent:
		if m.agentModel != nil {
			screenLabel = m.agentModel.agent.Name
		}
	}
	if screenLabel != "" {
		title += m.styles.Muted.Render(" [" + screenLabel + "]")
	}
	return title
}

func (m *HomeModel) renderHomeScreen() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("Workspaces"))
	b.WriteString("\n\n")

	if len(m.workspaces) == 0 {
		b.WriteString(m.styles.Muted.Render("  No workspaces registered. Run 'bc init' in a project directory."))
		b.WriteString("\n")
		return b.String()
	}

	// Header row
	header := fmt.Sprintf("  %-25s %-18s %-12s %-8s", "NAME", "PATH", "AGENTS", "ISSUES")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for i, ws := range m.workspaces {
		selected := i == m.homeCursor

		agentStr := fmt.Sprintf("%d running", ws.Running)
		if ws.Running == 0 {
			agentStr = "stopped"
		}

		issueStr := "-"
		if ws.HasBeads {
			issueStr = fmt.Sprintf("%d", ws.Issues)
		}

		// Truncate path for display
		path := ws.Entry.Path
		if len(path) > 16 {
			path = "..." + path[len(path)-13:]
		}

		line := fmt.Sprintf("  %-25s %-18s %-12s %-8s",
			ws.Entry.Name,
			path,
			agentStr,
			issueStr,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else if ws.Running > 0 {
			b.WriteString(m.styles.Success.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *HomeModel) renderStatusBar() string {
	var hints string
	switch m.screen {
	case ScreenHome:
		hints = "j/k:navigate | enter:open | r:refresh | ?:help | q:quit"
	case ScreenWorkspace:
		hints = "j/k:navigate | tab:switch tab | enter:details | esc:back | q:quit"
	case ScreenAgent:
		hints = "p:peek | a:attach | esc:back | q:quit"
	}

	if m.statusMsg != "" {
		hints = m.statusMsg + "  |  " + hints
	}

	return m.styles.StatusBar.Width(m.width).Render(hints)
}
