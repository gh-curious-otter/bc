# TUI Builder Design

A declarative TUI builder for predictable AI-generated terminal interfaces.

---

## Problem

Building Bubble Tea UIs requires understanding:
- Elm architecture (Model-Update-View)
- Message types and command handling
- Lipgloss styling
- Component state management

This complexity makes it hard for AI agents to reliably generate TUI code.

---

## Solution

A builder pattern that hides complexity behind a simple, declarative API:

```go
// AI can reliably generate this pattern
table := tui.NewTableView("agents").
    Title("Active Agents").
    Columns(
        tui.Col("NAME", 15),
        tui.Col("STATUS", 10),
    ).
    Rows(
        tui.Row{ID: "1", Values: []string{"worker-01", "running"}},
        tui.Row{ID: "2", Values: []string{"worker-02", "idle"}},
    ).
    OnSelect(handleSelect).
    Build()

app := tui.NewApp().
    Title("bc").
    AddView("agents", table).
    Build()

app.Run()
```

---

## Design Principles

### 1. Builder Pattern Everywhere

Every component uses the same pattern:
```go
NewXxxView(id string) → *XxxBuilder
    .Property(value)
    .Property(value)
    .Build() → *XxxView
```

This is:
- Predictable: AI always knows the shape
- Composable: Chain any methods
- Self-documenting: IDE autocomplete works

### 2. Sensible Defaults

Components work with minimal configuration:
```go
// Minimal table - still functional
tui.NewTableView("data").
    Columns(tui.Col("Name", 20)).
    Build()
```

### 3. Status-Based Styling

Pass status strings, get automatic coloring:
```go
tui.Row{Values: [...], Status: "ok"}      // Green
tui.Row{Values: [...], Status: "error"}   // Red
tui.Row{Values: [...], Status: "warning"} // Orange
```

### 4. Built-in Navigation

All views include vim-style keys:
- `j/k` or arrows: Move up/down
- `g/G`: Top/bottom
- `Enter`: Select
- `q`: Quit

---

## Component Types

| Component | Use Case | Builder |
|-----------|----------|---------|
| `TableView` | Data grids, lists | `NewTableView(id)` |
| `DetailView` | Key-value info | `NewDetailView(id)` |
| `FormView` | Input forms | `NewFormView(id)` |
| `ModalView` | Dialogs | `NewModal(type)` |
| `App` | Container | `NewApp()` |

---

## Data Types

### Row
```go
type Row struct {
    ID     string   // Unique identifier
    Values []string // Cell values
    Data   any      // Attached data (for callbacks)
    Status string   // For styling: ok, error, warning, info
}
```

### Column
```go
type Column struct {
    Name      string
    Width     int       // 0 = auto
    Alignment Alignment // Left, Center, Right
}

// Convenience constructors
tui.Col("Name", 20)       // Left-aligned
tui.ColRight("Cost", 10)  // Right-aligned
tui.ColCenter("Status", 8) // Centered
```

### KeyBinding
```go
type KeyBinding struct {
    Key     string     // "enter", "p", "ctrl+c"
    Label   string     // Shown in status bar
    Handler func() Cmd // Action
}

// Convenience constructor
tui.Bind("p", "Peek", peekHandler)
```

---

## Theming

Styles are centralized in `pkg/tui/style`:

```go
// Use default theme
styles := style.DefaultStyles()

// Access specific styles
styles.Header.Render("Title")
styles.Error.Render("Failed")
styles.StatusStyle("ok").Render("Running") // Auto status color
```

### Color Palette (Ayu Dark)

| Role | Color | Hex |
|------|-------|-----|
| Background | Dark blue-black | #0B0E14 |
| Foreground | Light gray | #BFBDB6 |
| Primary | Gold | #E6B450 |
| Success | Green | #AAD94C |
| Warning | Orange | #FF8F40 |
| Error | Red | #F07178 |
| Info | Blue | #59C2FF |

---

## AI Generation Guidelines

When generating TUI code, AI should:

1. **Always use the builder pattern**
   ```go
   // Good
   NewTableView("id").Columns(...).Build()
   
   // Avoid - direct struct creation
   &TableView{columns: ...}
   ```

2. **Use meaningful IDs**
   ```go
   NewTableView("agents")      // Good
   NewTableView("table1")      // Avoid
   ```

3. **Set status for colored rows**
   ```go
   Row{Values: [...], Status: "error"}  // Will be red
   ```

4. **Add key bindings for actions**
   ```go
   .Bind("p", "Peek", peekHandler)
   .Bind("n", "Nudge", nudgeHandler)
   ```

5. **Handle empty states**
   - Tables automatically show "No data"
   - No special handling needed

---

## Future Extensions

### Planned Components
- `ListView` - Simple scrollable list
- `TreeView` - Hierarchical data
- `SplitView` - Two-pane layouts
- `TabView` - Tabbed navigation

### Planned Features
- YAML/JSON spec parsing
- Component composition
- Async data loading helpers
- Search/filter built-in

---

## Example: Complete Agent Status View

```go
func buildAgentView(client *gt.Client) *tui.App {
    // Fetch data
    agents, _ := client.ListAgents()
    
    // Build rows from data
    var rows []tui.Row
    for _, a := range agents {
        rows = append(rows, tui.Row{
            ID:     a.ID,
            Values: []string{a.Name, a.Status, a.Rig, a.Task},
            Status: a.Status, // "running", "idle", "error"
            Data:   a,        // Attach for callbacks
        })
    }
    
    // Build table
    table := tui.NewTableView("agents").
        Title("Agents").
        Columns(
            tui.Col("NAME", 15),
            tui.Col("STATUS", 10),
            tui.Col("RIG", 12),
            tui.Col("TASK", 30),
        ).
        Rows(rows...).
        OnSelect(func(row tui.Row) tui.Cmd {
            agent := row.Data.(*Agent)
            return attachToAgent(agent)
        }).
        Bind("p", "Peek", func() tui.Cmd {
            return peekAgent(table.SelectedRow())
        }).
        Bind("n", "Nudge", func() tui.Cmd {
            return nudgeAgent(table.SelectedRow())
        }).
        Build()
    
    // Build app
    return tui.NewApp().
        Title("bc - Agent Status").
        AddView("agents", table).
        Bind("r", "Refresh", refreshData).
        Bind("?", "Help", showHelp).
        Build()
}
```

This pattern is predictable, testable, and easy for AI to generate correctly.
