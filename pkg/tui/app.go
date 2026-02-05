package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rpuneet/bc/pkg/tui/style"
)

// App is the main application container that manages views and navigation.
type App struct {
	views      map[string]Model
	viewOrder  []string // For navigation order
	activeView string

	// Layout
	showHeader    bool
	showStatusBar bool
	title         string

	// State
	width  int
	height int

	// Callbacks
	globalBindings []KeyBinding
	onQuit         func() Cmd

	// Styling
	styles style.Styles
}

// AppBuilder provides a fluent API for constructing App.
type AppBuilder struct {
	app *App
}

// NewApp creates a new AppBuilder.
func NewApp() *AppBuilder {
	return &AppBuilder{
		app: &App{
			views:         make(map[string]Model),
			showHeader:    true,
			showStatusBar: true,
			styles:        style.DefaultStyles(),
		},
	}
}

// Title sets the application title shown in the header.
func (b *AppBuilder) Title(title string) *AppBuilder {
	b.app.title = title
	return b
}

// AddView registers a view with an ID.
func (b *AppBuilder) AddView(id string, view Model) *AppBuilder {
	b.app.views[id] = view
	b.app.viewOrder = append(b.app.viewOrder, id)
	if b.app.activeView == "" {
		b.app.activeView = id
	}
	return b
}

// DefaultView sets which view to show initially.
func (b *AppBuilder) DefaultView(id string) *AppBuilder {
	b.app.activeView = id
	return b
}

// Bind adds a global key binding (works in all views).
func (b *AppBuilder) Bind(key, label string, handler func() Cmd) *AppBuilder {
	b.app.globalBindings = append(b.app.globalBindings, KeyBinding{
		Key:     key,
		Label:   label,
		Handler: handler,
	})
	return b
}

// ShowHeader enables/disables the header bar.
func (b *AppBuilder) ShowHeader(show bool) *AppBuilder {
	b.app.showHeader = show
	return b
}

// ShowStatusBar enables/disables the status bar.
func (b *AppBuilder) ShowStatusBar(show bool) *AppBuilder {
	b.app.showStatusBar = show
	return b
}

// Styles sets custom styles.
func (b *AppBuilder) Styles(s style.Styles) *AppBuilder {
	b.app.styles = s
	return b
}

// Build returns the constructed App.
func (b *AppBuilder) Build() *App {
	return b.app
}

// --- tea.Model Implementation ---

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, view := range a.views {
		if cmd := view.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate to active view with adjusted height
		return a.updateActiveView(msg)

	case tea.KeyMsg:
		// Global quit
		switch msg.String() {
		case "ctrl+c", "q":
			if a.onQuit != nil {
				return a, a.onQuit()
			}
			return a, tea.Quit
		}

		// Check global bindings
		for _, binding := range a.globalBindings {
			if msg.String() == binding.Key {
				if binding.Handler != nil {
					return a, binding.Handler()
				}
			}
		}

		// Pass to active view
		return a.updateActiveView(msg)
	}

	return a.updateActiveView(msg)
}

func (a *App) updateActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if view, ok := a.views[a.activeView]; ok {
		// Adjust window size for header/status bar
		if wsm, ok := msg.(tea.WindowSizeMsg); ok {
			contentHeight := wsm.Height
			if a.showHeader {
				contentHeight--
			}
			if a.showStatusBar {
				contentHeight--
			}
			msg = tea.WindowSizeMsg{Width: wsm.Width, Height: contentHeight}
		}

		newView, cmd := view.Update(msg)
		a.views[a.activeView] = newView
		return a, cmd
	}
	return a, nil
}

// View implements tea.Model.
func (a *App) View() string {
	var sections []string

	// Header
	if a.showHeader {
		sections = append(sections, a.renderHeader())
	}

	// Active view content
	if view, ok := a.views[a.activeView]; ok {
		sections = append(sections, view.View())
	} else {
		sections = append(sections, a.styles.Error.Render("No view loaded"))
	}

	// Status bar
	if a.showStatusBar {
		sections = append(sections, a.renderStatusBar())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (a *App) renderHeader() string {
	title := a.title
	if title == "" {
		title = "bc"
	}

	left := a.styles.Header.Render(title)

	viewIndicator := ""
	if a.activeView != "" {
		viewIndicator = a.styles.Muted.Render(fmt.Sprintf(" [%s]", a.activeView))
	}

	return left + viewIndicator
}

func (a *App) renderStatusBar() string {
	// Build hints from global bindings
	var hints []string
	for _, b := range a.globalBindings {
		if !b.Hidden {
			hints = append(hints, fmt.Sprintf("%s:%s", b.Key, b.Label))
		}
	}
	hints = append(hints, "q:quit")

	hintStr := strings.Join(hints, " | ")
	return a.styles.StatusBar.Width(a.width).Render(hintStr)
}

// --- Navigation Methods ---

// SetView switches to the specified view.
func (a *App) SetView(id string) tea.Cmd {
	if _, ok := a.views[id]; ok {
		a.activeView = id
	}
	return nil
}

// ActiveView returns the currently active view ID.
func (a *App) ActiveView() string {
	return a.activeView
}

// NextView cycles to the next view.
func (a *App) NextView() {
	for i, id := range a.viewOrder {
		if id == a.activeView {
			next := (i + 1) % len(a.viewOrder)
			a.activeView = a.viewOrder[next]
			return
		}
	}
}

// PrevView cycles to the previous view.
func (a *App) PrevView() {
	for i, id := range a.viewOrder {
		if id == a.activeView {
			prev := (i - 1 + len(a.viewOrder)) % len(a.viewOrder)
			a.activeView = a.viewOrder[prev]
			return
		}
	}
}

// Run starts the Bubble Tea program.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunSimple starts without alternate screen (for debugging).
func (a *App) RunSimple() error {
	p := tea.NewProgram(a)
	_, err := p.Run()
	return err
}
