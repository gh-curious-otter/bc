package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// Driver is the main runtime that communicates with an AI via stdin/stdout.
type Driver struct {
	// I/O
	input  io.Reader
	output io.Writer

	// State
	currentView  ViewType
	tableSpec    *TableSpec
	detailSpec   *DetailSpec
	modalSpec    *ModalSpec
	cursor       int
	width        int
	height       int

	// Components
	renderer *Renderer
	styles   style.Styles

	// App info
	title   string
	version string
}

// NewDriver creates a new runtime driver.
func NewDriver() *Driver {
	return &Driver{
		input:    os.Stdin,
		output:   os.Stdout,
		renderer: NewRenderer(),
		styles:   style.DefaultStyles(),
		title:    "bc",
		version:  "dev",
	}
}

// WithIO sets custom input/output streams (for testing).
func (d *Driver) WithIO(input io.Reader, output io.Writer) *Driver {
	d.input = input
	d.output = output
	return d
}

// WithTitle sets the app title.
func (d *Driver) WithTitle(title string) *Driver {
	d.title = title
	return d
}

// --- tea.Model Implementation ---

// Init implements tea.Model.
func (d *Driver) Init() tea.Cmd {
	return tea.Batch(
		d.readInput,
		d.sendReady,
	)
}

// Update implements tea.Model.
func (d *Driver) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.renderer.SetSize(msg.Width, msg.Height)
		return d, nil

	case tea.KeyMsg:
		return d.handleKey(msg)

	case incomingMessage:
		return d.handleIncoming(msg)

	case errMsg:
		// Log error but continue
		return d, d.readInput
	}

	return d, nil
}

func (d *Driver) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// Global quit
	if keyStr == "ctrl+c" || keyStr == "q" {
		return d, tea.Quit
	}

	// Navigation for table view
	if d.currentView == ViewTable && d.tableSpec != nil {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if d.cursor < len(d.tableSpec.Rows)-1 {
				d.cursor++
			}
			return d, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if d.cursor > 0 {
				d.cursor--
			}
			return d, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("g", "home"))):
			d.cursor = 0
			return d, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("G", "end"))):
			if len(d.tableSpec.Rows) > 0 {
				d.cursor = len(d.tableSpec.Rows) - 1
			}
			return d, nil
		}
	}

	// Send key event to AI
	return d, d.sendKeyEvent(keyStr)
}

func (d *Driver) handleIncoming(msg incomingMessage) (tea.Model, tea.Cmd) {
	msgType, err := ParseMessage(msg.data)
	if err != nil {
		return d, d.readInput
	}

	switch msgType {
	case MsgView:
		var vm ViewMessage
		if err := json.Unmarshal(msg.data, &vm); err == nil {
			d.handleViewMessage(&vm)
		}

	case MsgSet:
		var sm SetMessage
		if err := json.Unmarshal(msg.data, &sm); err == nil {
			d.handleSetMessage(&sm)
		}

	case MsgAppend:
		var am AppendMessage
		if err := json.Unmarshal(msg.data, &am); err == nil {
			d.handleAppendMessage(&am)
		}

	case MsgDone:
		// Mark loading as complete
		if d.tableSpec != nil {
			d.tableSpec.Loading = false
		}
		if d.detailSpec != nil {
			d.detailSpec.Loading = false
		}

	case MsgError:
		var em ErrorMessage
		if err := json.Unmarshal(msg.data, &em); err == nil {
			// Could show error in UI
			_ = em
		}
	}

	return d, d.readInput
}

func (d *Driver) handleViewMessage(msg *ViewMessage) {
	d.currentView = msg.View
	d.cursor = 0

	switch msg.View {
	case ViewTable:
		d.tableSpec = &TableSpec{
			ID:      msg.ID,
			Title:   msg.Title,
			Loading: msg.Loading,
		}
		d.detailSpec = nil
		d.modalSpec = nil

	case ViewDetail:
		d.detailSpec = &DetailSpec{
			ID:      msg.ID,
			Title:   msg.Title,
			Loading: msg.Loading,
		}
		d.tableSpec = nil
		d.modalSpec = nil

	case ViewModal:
		d.modalSpec = &ModalSpec{
			ID:    msg.ID,
			Title: msg.Title,
		}
	}
}

func (d *Driver) handleSetMessage(msg *SetMessage) {
	switch d.currentView {
	case ViewTable:
		if d.tableSpec != nil {
			d.setTablePath(msg.Path, msg.Value)
		}
	case ViewDetail:
		if d.detailSpec != nil {
			d.setDetailPath(msg.Path, msg.Value)
		}
	}
}

func (d *Driver) setTablePath(path string, value any) {
	switch path {
	case "title":
		if s, ok := value.(string); ok {
			d.tableSpec.Title = s
		}
	case "loading":
		if b, ok := value.(bool); ok {
			d.tableSpec.Loading = b
		}
	case "columns":
		if data, err := json.Marshal(value); err == nil {
			var cols []ColumnSpec
			if json.Unmarshal(data, &cols) == nil {
				d.tableSpec.Columns = cols
			}
		}
	case "rows":
		if data, err := json.Marshal(value); err == nil {
			var rows []RowSpec
			if json.Unmarshal(data, &rows) == nil {
				d.tableSpec.Rows = rows
			}
		}
	case "bindings":
		if data, err := json.Marshal(value); err == nil {
			var bindings []BindingSpec
			if json.Unmarshal(data, &bindings) == nil {
				d.tableSpec.Bindings = bindings
			}
		}
	}
}

func (d *Driver) setDetailPath(path string, value any) {
	switch path {
	case "title":
		if s, ok := value.(string); ok {
			d.detailSpec.Title = s
		}
	case "loading":
		if b, ok := value.(bool); ok {
			d.detailSpec.Loading = b
		}
	case "sections":
		if data, err := json.Marshal(value); err == nil {
			var sections []SectionSpec
			if json.Unmarshal(data, &sections) == nil {
				d.detailSpec.Sections = sections
			}
		}
	}
}

func (d *Driver) handleAppendMessage(msg *AppendMessage) {
	switch d.currentView {
	case ViewTable:
		if d.tableSpec != nil && msg.Path == "rows" {
			if data, err := json.Marshal(msg.Value); err == nil {
				var row RowSpec
				if json.Unmarshal(data, &row) == nil {
					d.tableSpec.Rows = append(d.tableSpec.Rows, row)
				}
			}
		}
	case ViewDetail:
		if d.detailSpec != nil {
			if strings.HasPrefix(msg.Path, "sections") {
				// Handle sections[0].fields append
				// Simplified: just append to first section's fields
				if len(d.detailSpec.Sections) == 0 {
					d.detailSpec.Sections = []SectionSpec{{}}
				}
				if data, err := json.Marshal(msg.Value); err == nil {
					var field FieldSpec
					if json.Unmarshal(data, &field) == nil {
						d.detailSpec.Sections[0].Fields = append(
							d.detailSpec.Sections[0].Fields, field)
					}
				}
			}
		}
	}
}

// View implements tea.Model.
func (d *Driver) View() string {
	var sections []string

	// Header
	viewID := ""
	switch d.currentView {
	case ViewTable:
		if d.tableSpec != nil {
			viewID = d.tableSpec.ID
		}
	case ViewDetail:
		if d.detailSpec != nil {
			viewID = d.detailSpec.ID
		}
	}
	sections = append(sections, d.renderer.RenderHeader(d.title, viewID))

	// Content
	var content string
	switch d.currentView {
	case ViewTable:
		if d.tableSpec != nil {
			content = d.renderer.RenderTable(d.tableSpec, d.cursor)
		} else {
			content = d.renderer.RenderLoading("")
		}
	case ViewDetail:
		if d.detailSpec != nil {
			content = d.renderer.RenderDetail(d.detailSpec)
		} else {
			content = d.renderer.RenderLoading("")
		}
	case ViewModal:
		if d.modalSpec != nil {
			content = d.renderer.RenderModal(d.modalSpec)
		}
	default:
		content = d.renderer.RenderLoading("Waiting for AI...")
	}
	sections = append(sections, content)

	// Status bar with bindings
	var bindings []BindingSpec
	switch d.currentView {
	case ViewTable:
		if d.tableSpec != nil {
			bindings = d.tableSpec.Bindings
		}
	case ViewDetail:
		if d.detailSpec != nil {
			bindings = d.detailSpec.Bindings
		}
	}
	sections = append(sections, d.renderer.RenderBindings(bindings))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- I/O Commands ---

type incomingMessage struct {
	data []byte
}

type errMsg struct {
	err error
}

func (d *Driver) readInput() tea.Msg {
	scanner := bufio.NewScanner(d.input)
	if scanner.Scan() {
		return incomingMessage{data: scanner.Bytes()}
	}
	if err := scanner.Err(); err != nil {
		return errMsg{err: err}
	}
	// EOF - AI closed connection
	return tea.Quit
}

func (d *Driver) sendReady() tea.Msg {
	event := ReadyEvent{
		Type:    MsgReady,
		Version: d.version,
		Width:   d.width,
		Height:  d.height,
	}
	d.sendEvent(event)
	return nil
}

func (d *Driver) sendKeyEvent(keyStr string) tea.Cmd {
	return func() tea.Msg {
		event := KeyEvent{
			Type: MsgKey,
			Key:  keyStr,
		}

		// Add context based on current view
		switch d.currentView {
		case ViewTable:
			if d.tableSpec != nil {
				event.View = d.tableSpec.ID
				if d.cursor < len(d.tableSpec.Rows) {
					row := d.tableSpec.Rows[d.cursor]
					event.Selected = &RowRef{
						ID:     row.ID,
						Index:  d.cursor,
						Values: row.Values,
					}
				}
			}
		case ViewDetail:
			if d.detailSpec != nil {
				event.View = d.detailSpec.ID
			}
		}

		d.sendEvent(event)
		return nil
	}
}

func (d *Driver) sendEvent(event any) {
	data, err := Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprint(d.output, string(data))
}

// Run starts the runtime driver.
func (d *Driver) Run() error {
	p := tea.NewProgram(d, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
