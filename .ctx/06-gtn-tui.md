# gtn - Gas Town Navigator TUI

> **Note**: This is reference documentation from the Gas Town project. It documents the gtn TUI patterns and Bubble Tea architecture that can inform bc's TUI development.

A k9s-style terminal user interface for Gas Town multi-agent orchestration.

## Overview

gtn (Gas Town Navigator) is a terminal UI built with Bubble Tea and lipgloss that provides hierarchical navigation, dense status displays, and manual intervention workflows for managing Gas Town agents and work items.

### Key Features
- Real-time agent status monitoring
- Tmux session attachment for direct interaction
- Peek (view output) and nudge (send messages) capabilities
- Convoy and merge queue tracking
- Cost monitoring per session

### Technology Stack
- **UI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Elm-inspired terminal UI framework
- **Styling**: [lipgloss](https://github.com/charmbracelet/lipgloss) - CSS-like terminal styling
- **CLI**: [Cobra](https://github.com/spf13/cobra) - Command-line interface
- **Backend**: Shells out to `gt` (Gas Town) and `bd` (beads) CLI commands

---

## Architecture

### Package Structure

```
gtn/
├── cmd/gtn/           # Main entry point
│   └── main.go        # Cobra CLI setup, launches Bubble Tea program
├── app/               # Main application logic
│   ├── app.go         # Root Model, Update, View - orchestrates all views
│   ├── keys.go        # Global KeyMap bindings
│   └── navigation.go  # ViewType enum definitions
├── views/             # View implementations (one per feature area)
│   ├── status.go      # Home view - agent overview with rig filtering
│   ├── sessions.go    # Tmux session listing
│   ├── convoys.go     # Batch work tracking
│   ├── mq.go          # Merge queue per rig
│   ├── costs.go       # Session cost tracking
│   ├── peek.go        # Output viewing with viewport scrolling
│   ├── nudge.go       # Message sending
│   ├── help.go        # Keyboard shortcut reference
│   ├── rigs.go        # Rig listing
│   ├── polecats.go    # Ephemeral AI worker listing
│   ├── crew.go        # Persistent human worker listing
│   ├── beads.go       # Issue/work item listing
│   ├── bead_detail.go # Single bead detail view
│   ├── ready.go       # Unblocked work ready for assignment
│   ├── blocked.go     # Blocked work items
│   ├── escalations.go # Items needing manual intervention
│   ├── mayor.go       # Mayor agent status
│   ├── witness.go     # Witness agent status
│   └── refinery.go    # Refinery agent status
├── components/        # Reusable UI widgets
│   ├── table.go       # Navigable table with vim keys
│   ├── header.go      # Top bar component
│   ├── statusbar.go   # Bottom bar with hints
│   ├── detail.go      # Rich detail panel with sections
│   ├── modal.go       # Dialog boxes (confirm, input, select, form)
│   ├── tree.go        # Hierarchical tree display
│   └── breadcrumb.go  # Navigation path display
├── gt/                # Gas Town client wrapper
│   ├── client.go      # Methods wrapping gt/bd CLI commands
│   └── types.go       # Data structures matching JSON output
├── tmux/              # Tmux session management
│   └── session.go     # Session attach, capture, list operations
├── style/             # Theme and styling
│   └── theme.go       # Ayu-inspired color palette and styles
├── bin/               # Build output (gitignored)
└── Makefile           # Build automation
```

### Message Flow in Bubble Tea

Bubble Tea follows the Elm architecture (Model-Update-View):

```
┌─────────────────────────────────────────────────────────────────┐
│                        Bubble Tea Runtime                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   1. Init() ──────> Returns initial Cmd (async data fetch)       │
│                                                                  │
│   2. Update(msg) ─> Handles messages, returns (Model, Cmd)       │
│      │                                                           │
│      ├── tea.KeyMsg ──────> Keyboard input                       │
│      ├── tea.WindowSizeMsg ──> Terminal resize                   │
│      ├── *DataMsg ────────> Async data loaded                    │
│      └── sessionExitMsg ──> Returned from tmux attach            │
│                                                                  │
│   3. View() ─────> Renders current state to string               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Message Types:**
```go
// Custom data messages for async loading
type statusDataMsg struct {
    status *gt.TownStatus
    err    error
}

type sessionsDataMsg struct {
    sessions []gt.Session
    err      error
}

// External process completion
type sessionExitMsg struct {
    err error
}
```

### Async Data Pattern

Each view follows a consistent pattern for loading data asynchronously:

```go
// 1. Init() triggers async data fetch
func (v *StatusView) Init() tea.Cmd {
    return func() tea.Msg {
        status, err := v.client.GetStatus()
        return statusDataMsg{status: status, err: err}
    }
}

// 2. HandleMsg() processes the async result
func (v *StatusView) HandleMsg(msg tea.Msg) tea.Cmd {
    if m, ok := msg.(statusDataMsg); ok {
        v.status = m.status
        v.err = m.err
        v.updateTable()
    }
    return nil
}

// 3. Update() handles keyboard input
func (v *StatusView) Update(msg tea.KeyMsg) tea.Cmd {
    v.table.Update(msg)
    return nil
}

// 4. View() renders the current state
func (v *StatusView) View() string {
    if v.err != nil {
        return style.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
    }
    if v.status == nil {
        return "Loading..."
    }
    return v.table.View()
}
```

---

## Views

### StatusView (Home) - `0` key

The default view showing all agents across the town with rig filtering.

**Features:**
- Lists global agents (mayor, deacon) and per-rig agents
- Tab navigation to filter by specific rig
- Quick stats showing running/total agents
- Actions: attach, peek, nudge

**Data Source:** `gt status --json`

**Columns:** RIG | AGENT | ROLE | STATUS | MAIL | SESSION

### SessionsView - `s` key

Lists all active tmux sessions managed by Gas Town.

**Features:**
- Shows rig, polecat name, session ID, running status
- Direct attach to tmux session with Enter

**Data Source:** `gt session list --json`

**Columns:** RIG | NAME | SESSION | STATUS

### ConvoysView - `c` key

Displays batch work tracking across rigs.

**Features:**
- Shows convoy ID, title, and status
- Status indicators for open/closed

**Data Source:** `gt convoy list --json`

**Columns:** ID | TITLE | STATUS

### MQView (Merge Queue) - `m` key

Shows merge requests per rig with tab navigation between rigs.

**Features:**
- Tab/Shift+Tab to switch between rigs
- Displays MR ID, title, status, creator

**Data Source:** `gt mq list <rig> --json`

**Columns:** ID | TITLE | STATUS | CREATED BY

### CostsView - `$` key

Tracks session costs with running totals.

**Features:**
- Per-session cost breakdown
- Total cost calculation at top
- Running/stopped status indicators

**Data Source:** `gt costs --json`

**Columns:** SESSION | RIG | ROLE | WORKER | COST | STATUS

### PeekView - `p` key on selected row

Displays captured output from a session using a scrollable viewport.

**Features:**
- Viewport with j/k scrolling
- Refresh with `r` key
- Returns to previous view with Esc

**Data Source:** `gt peek <address> -n <lines>`

### NudgeView - `n` key on selected row

Send a message to an agent's session.

**Features:**
- Text input with placeholder
- Enter to send, Esc to cancel
- Success/error feedback

**Data Source:** `gt nudge <address> <message>`

### HelpView - `?` key

Displays keyboard shortcuts organized by section:
- Agents View (home)
- Switch Views
- General
- Commands
- In Session (tmux)

---

## Components

### Table (`components/table.go`)

A navigable table component with vim-style keyboard shortcuts.

**Features:**
- Column definitions with width
- Row data with optional ID and status for styling
- Cursor tracking with viewport scrolling
- Status-based row coloring

**Navigation:**
| Key | Action |
|-----|--------|
| `j` / `down` | Move down |
| `k` / `up` | Move up |
| `g` | Go to top |
| `G` | Go to bottom |

### Header (`components/header.go`)

Top bar showing application title, current view, and connection status.

**Layout:** `gtn <View> [status]`

### StatusBar (`components/statusbar.go`)

Bottom bar with contextual keyboard hints or command input.

**Modes:**
- Normal: Shows key hints
- Command: Shows `:` prefix with typed command
- Message: Shows status message

### Detail (`components/detail.go`)

Rich detail panel for showing entity information.

**Features:**
- Title with icon
- Multiple sections with fields
- Field styles: normal, status (colored), link, code, list
- Action hints at bottom

**Helper Functions:**
- `BeadDetail()` - Pre-built detail view for beads
- `AgentDetail()` - Pre-built detail view for agents
- `QuickDetail()` - Create from map of values

### Modal (`components/modal.go`)

Dialog boxes for user interaction.

**Types:**
- `ModalConfirm` - Yes/No confirmation
- `ModalInput` - Single text input
- `ModalSelect` - Selection list
- `ModalForm` - Multiple input fields

**Navigation:**
| Key | Action |
|-----|--------|
| `Tab` / `down` | Next field |
| `Shift+Tab` / `up` | Previous field |
| `Enter` | Confirm / Next |
| `Esc` | Cancel |
| `y/Y` | Confirm (confirm modal) |
| `n/N` | Cancel (confirm modal) |

### Tree (`components/tree.go`)

Hierarchical tree display for nested data like beads.

**Features:**
- Expand/collapse with Enter
- Type icons (epic, task, bug, etc.)
- Status indicators
- Depth-based indentation

**Navigation:**
| Key | Action |
|-----|--------|
| `j/k` | Up/down |
| `Enter` | Toggle expand/collapse |

### Breadcrumb (`components/breadcrumb.go`)

Navigation path display showing current location.

**Example:** `Town > gastown > Polecats > Toast`

**Methods:**
- `Push(element)` - Add to path
- `Pop()` - Remove last element
- `Clear()` - Reset to root

---

## Keyboard Shortcuts

### Global Navigation

| Key | View |
|-----|------|
| `0` | Status/Agents (home) |
| `s` | Sessions |
| `c` | Convoys |
| `m` | Merge Queue |
| `$` | Costs |
| `f` | Feed (launches gt feed TUI) |
| `?` | Toggle Help |
| `:` | Command mode |
| `q` / `Ctrl+c` | Quit |
| `r` | Refresh current view |

### Vim-Style List Navigation

| Key | Action |
|-----|--------|
| `j` / `down` | Move down |
| `k` / `up` | Move up |
| `g` | Go to top |
| `G` | Go to bottom |
| `Tab` / `l` | Next tab/rig filter |
| `Shift+Tab` / `h` | Previous tab/rig filter |

### Context Actions

| Key | Action |
|-----|--------|
| `Enter` | Select / Attach to session |
| `p` | Peek (view output) |
| `n` | Nudge (send message) |
| `Esc` | Back / Cancel |

### Command Mode (`:`)

| Command | Action |
|---------|--------|
| `:quit` / `:q` | Exit application |
| `:status` / `:0` | Go to status view |
| `:sessions` / `:s` | Go to sessions view |
| `:convoys` / `:c` | Go to convoys view |
| `:mq` / `:m` | Go to merge queue |
| `:costs` / `:$` | Go to costs view |
| `:feed` / `:f` | Launch gt feed |
| `:up` | Start all services |
| `:down` | Stop all services |
| `:help` / `:?` | Show help |

### In tmux Session

| Key | Action |
|-----|--------|
| `Ctrl+b d` | Detach and return to gtn |

---

## gt Client Wrapper

The `gt` package provides a Go wrapper around the `gt` and `bd` CLI commands.

### Client Structure

```go
type Client struct {
    workspace string  // Gas Town workspace path (e.g., ~/Projects/.gt)
}
```

### Workspace Discovery

The client searches for Gas Town workspace in these locations:
1. `~/Projects/.gt`
2. `~/.gt`
3. `~/gt`

Validation: Checks for `.beads` directory as workspace marker.

### Core Methods

```go
// Status and overview
func (c *Client) GetStatus() (*TownStatus, error)
func (c *Client) GetSessions() ([]Session, error)
func (c *Client) GetConvoys() ([]Convoy, error)
func (c *Client) GetCosts() (*CostsResponse, error)
func (c *Client) GetMergeQueue(rig string) ([]MergeRequest, error)

// Agent interaction
func (c *Client) Peek(address string, lines int) (string, error)
func (c *Client) Nudge(address, message string) error
func (c *Client) Broadcast(message string) error

// Agent-specific status
func (c *Client) GetMayorStatus() (*MayorStatus, error)
func (c *Client) GetWitnessStatus(rig string) (*WitnessStatus, error)
func (c *Client) GetRefineryStatus(rig string) (*RefineryStatus, error)

// Polecat management
func (c *Client) ListPolecats(rig string) ([]Polecat, error)
func (c *Client) NukePolecat(address string) error
func (c *Client) SpawnPolecat(rig, name string) error

// Crew management
func (c *Client) ListCrew(rig string) ([]Crew, error)
func (c *Client) AddCrew(rig, name string) error
func (c *Client) RemoveCrew(address string) error

// Bead operations (via bd command)
func (c *Client) ListBeads(opts BeadOpts) ([]Bead, error)
func (c *Client) GetBead(id string) (*Bead, error)
func (c *Client) CreateBead(opts CreateBeadOpts) (*Bead, error)
func (c *Client) CloseBead(id string) error
func (c *Client) ReadyBeads() ([]Bead, error)
func (c *Client) BlockedBeads() ([]Bead, error)

// Work assignment
func (c *Client) Sling(beadID, agent string) error
func (c *Client) GetEscalations() ([]Escalation, error)

// Service control
func (c *Client) Up() error
func (c *Client) Down() error
func (c *Client) Doctor() (string, error)

// Commands that return exec.Cmd for external process execution
func (c *Client) FeedCmd() *exec.Cmd
func (c *Client) AttachSession(sessionID string) *exec.Cmd
func (c *Client) MayorAttachCmd() *exec.Cmd
func (c *Client) MayorStartCmd() *exec.Cmd
```

### JSON Parsing

All data methods:
1. Execute the CLI command with `--json` flag
2. Parse JSON output into typed Go structs
3. Return structured data or error

```go
func (c *Client) run(args ...string) ([]byte, error) {
    cmd := exec.Command("gt", args...)
    cmd.Dir = c.workspace
    return cmd.Output()
}

func (c *Client) runBd(args ...string) ([]byte, error) {
    cmd := exec.Command("bd", args...)
    cmd.Dir = c.workspace
    return cmd.Output()
}
```

### Key Data Types

```go
type TownStatus struct {
    Name     string   `json:"name"`
    Location string   `json:"location"`
    Overseer Overseer `json:"overseer"`
    Agents   []Agent  `json:"agents"`
    Rigs     []Rig    `json:"rigs"`
}

type Agent struct {
    Name       string `json:"name"`
    Address    string `json:"address"`
    Session    string `json:"session"`
    Role       string `json:"role"`
    Running    bool   `json:"running"`
    State      string `json:"state"`
    UnreadMail int    `json:"unread_mail"`
}

type Polecat struct {
    Name       string `json:"name"`
    Address    string `json:"address"`
    Session    string `json:"session"`
    Rig        string `json:"rig"`
    State      string `json:"state"`  // working, done, stuck, spawning, awaiting
    Running    bool   `json:"running"`
    HookBead   string `json:"hook_bead,omitempty"`
}

type Bead struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      string   `json:"status"`
    Type        string   `json:"type"`  // task, epic, bug, story
    Priority    int      `json:"priority"`
    Assignee    string   `json:"assignee,omitempty"`
    DependsOn   []string `json:"depends_on,omitempty"`
    AgentState  string   `json:"agent_state,omitempty"`
}
```

### Tmux Integration

The `tmux` package provides session management:

```go
type SessionManager struct{}

func (m *SessionManager) Attach(name string) *exec.Cmd
func (m *SessionManager) Detach(name string) error
func (m *SessionManager) List() ([]SessionInfo, error)
func (m *SessionManager) Kill(name string) error
func (m *SessionManager) Capture(name string, lines int) (string, error)
func (m *SessionManager) HasSession(name string) bool
func (m *SessionManager) SendKeys(name string, keys string) error
func (m *SessionManager) CreateSession(name string, cmd string) error
```

---

## Styling

### Color Palette (Ayu-inspired)

```go
// Base colors
Background = "#0B0E14"
Foreground = "#BFBDB6"
Selection  = "#409FFF"

// Accent colors
Accent       = "#E6B450"  // Yellow/gold (primary accent)
AccentBlue   = "#59C2FF"
AccentGreen  = "#AAD94C"
AccentRed    = "#F07178"
AccentOrange = "#FF8F40"

// UI colors
Border    = "#565B66"
Muted     = "#565B66"
BarBg     = "#1C2028"

// Status colors
StatusOK      = AccentGreen
StatusError   = AccentRed
StatusWarning = AccentOrange
StatusInfo    = AccentBlue
```

### Common Styles

- `HeaderStyle` - Bold accent on bar background
- `SelectedStyle` - Bold with inverted selection colors
- `NormalStyle` - Standard foreground
- `MutedStyle` - Dimmed text for secondary info
- `ErrorStyle` - Red text for errors
- `SuccessStyle` - Green text for success
- `BorderStyle` - Rounded border with border color

---

## Build Commands

Always use Makefile commands (never raw `go build`):

```bash
# Build (outputs to bin/gtn)
make build

# Build and run
make run

# Run tests
make test

# Lint code
make lint

# Clean build artifacts
make clean

# Tidy dependencies
make deps

# Install to GOPATH
make install
```

The Makefile handles version injection via ldflags:
```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
```
