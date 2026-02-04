// Package tui provides a declarative TUI builder for creating terminal interfaces.
//
// The goal is to provide a simple, predictable API that AI agents can use to
// generate consistent terminal UIs without needing to understand the underlying
// Bubble Tea framework details.
//
// # Design Principles
//
//   - Declarative over imperative: Describe what you want, not how to build it
//   - Builder pattern: Chain methods for readable, AI-friendly code generation
//   - Sensible defaults: Minimal configuration for common cases
//   - Composable: Small components that combine into complex UIs
//   - Testable: Easy to verify generated UIs are correct
//
// # Basic Usage
//
//	// Create a simple table view
//	view := tui.NewTableView("agents").
//	    Title("Active Agents").
//	    Columns(
//	        tui.Col("NAME", 20),
//	        tui.Col("STATUS", 10),
//	    ).
//	    OnSelect(func(row tui.Row) tui.Cmd {
//	        // Handle selection
//	    }).
//	    Build()
//
//	// Run the app
//	app := tui.NewApp().
//	    AddView(view).
//	    Run()
//
// # Component Types
//
// The package provides these core components:
//
//   - TableView: Navigable data tables with vim-style keys
//   - DetailView: Key-value detail panels
//   - ListView: Simple scrollable lists
//   - FormView: Input forms with validation
//   - ModalView: Overlay dialogs (confirm, input, select)
//
// # Layouts
//
// Components can be arranged using layouts:
//
//   - Vertical: Stack components top to bottom
//   - Horizontal: Arrange components left to right
//   - Split: Two-pane layouts with adjustable divider
//
// # Styling
//
// Theming is handled via the style sub-package with predefined palettes.
package tui
