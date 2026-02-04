package cmd

import (
	"github.com/prai-git/bc/pkg/tui"
	"github.com/spf13/cobra"
)

// exampleCmd demonstrates the TUI builder.
var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Run an example TUI to demonstrate the builder",
	Long: `Launches an example TUI showing the declarative builder pattern.

This demonstrates how AI agents can generate predictable TUI code:

  table := tui.NewTableView("agents").
      Title("Active Agents").
      Columns(
          tui.Col("NAME", 15),
          tui.Col("STATUS", 10),
          tui.Col("TASK", 30),
      ).
      Rows(...).
      Build()

Navigation: j/k to move, q to quit.`,
	RunE: runExample,
}

func init() {
	rootCmd.AddCommand(exampleCmd)
}

func runExample(cmd *cobra.Command, args []string) error {
	// Build a sample table view - this is the pattern AI agents would generate
	agentTable := tui.NewTableView("agents").
		Title("Active Agents").
		Columns(
			tui.Col("NAME", 15),
			tui.Col("STATUS", 10),
			tui.Col("RIG", 12),
			tui.Col("TASK", 35),
		).
		Rows(
			tui.Row{ID: "1", Values: []string{"coordinator", "running", "bc", "Managing work queue"}, Status: "ok"},
			tui.Row{ID: "2", Values: []string{"worker-01", "working", "bc", "Implementing TUI builder"}, Status: "ok"},
			tui.Row{ID: "3", Values: []string{"worker-02", "idle", "bc", "-"}, Status: "info"},
			tui.Row{ID: "4", Values: []string{"worker-03", "stuck", "api", "Waiting for review"}, Status: "warning"},
			tui.Row{ID: "5", Values: []string{"worker-04", "error", "api", "Build failed"}, Status: "error"},
		).
		OnSelect(func(row tui.Row) tui.Cmd {
			// In a real app, this would open details or attach to session
			return nil
		}).
		Build()

	// Build the app
	app := tui.NewApp().
		Title("bc - Example TUI").
		AddView("agents", agentTable).
		Bind("?", "Help", func() tui.Cmd {
			// Would show help view
			return nil
		}).
		Bind("r", "Refresh", func() tui.Cmd {
			// Would refresh data
			return nil
		}).
		Build()

	return app.Run()
}
