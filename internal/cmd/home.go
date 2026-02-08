package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/config"
	itui "github.com/rpuneet/bc/internal/tui"
	"github.com/rpuneet/bc/pkg/workspace"
)

var homeCmd = &cobra.Command{
	Use:   "home",
	Short: "Open the bc home screen TUI",
	Long: `Open the interactive home screen showing all workspaces and agents.

The TUI updates in real-time as agents start, stop, and report progress.
You can drill into workspaces, view agents/issues/PRs, and peek at output.

Navigation:
  j/k      Move up/down
  Enter    Select / drill down
  Tab      Switch tabs (in workspace view)
  p        Peek at agent output
  a        Attach to agent (in agent view)
  Esc      Go back
  r        Refresh
  q        Quit`,
	RunE: runHome,
}

func init() {
	rootCmd.AddCommand(homeCmd)
}

func runHome(cmd *cobra.Command, args []string) error {
	// Load workspace registry
	reg, err := workspace.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load workspace registry: %w", err)
	}
	reg.Prune()

	// If no workspaces registered, try to register the current one
	if len(reg.Workspaces) == 0 {
		ws, wsErr := getWorkspace()
		if wsErr == nil {
			reg.Register(ws.RootDir, ws.Config.Name)
			_ = reg.Save()
		}
	}

	// Build workspace list with zero counts; TUI will fill counts on first tick (#310).
	// This avoids N Manager loads and RefreshState() at CLI startup, reducing memory and improving TUI appearance time.
	list := reg.List()
	workspaces := make([]itui.WorkspaceInfo, 0, len(list))
	for _, entry := range list {
		workspaces = append(workspaces, itui.WorkspaceInfo{
			Entry:      entry,
			MaxWorkers: int(config.Workspace.MaxWorkers),
		})
	}

	// Run the Bubble Tea TUI
	model := itui.NewHomeModel(workspaces, int(config.Workspace.MaxWorkers))
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
