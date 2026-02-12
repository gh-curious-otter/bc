package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/config"
	itui "github.com/rpuneet/bc/internal/tui"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/workspace"
)

var homeCmd = &cobra.Command{
	Use:        "home",
	Short:      "Open the bc home screen TUI (deprecated)",
	Deprecated: "TUI is being rebuilt with Ink. Use 'bc agent list' and 'bc status' instead.",
	Long: `Open the interactive home screen showing all workspaces and agents.

DEPRECATED: This command is deprecated and will be removed in a future version.
The TUI is being rebuilt with Ink. Use 'bc agent list' and 'bc status' instead.

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
	// Load workspace registry (fast: single file read)
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

	// Start TUI immediately with loading state; load workspace list in background
	// so the UI appears without waiting for RefreshState() on every workspace.
	model := itui.NewHomeModel(nil, int(config.WorkspaceLegacy.MaxWorkers), true)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	go loadWorkspacesAndSend(reg, p)
	_, err = p.Run()
	return err
}

// loadWorkspacesAndSend builds workspace info (LoadState+RefreshState per workspace)
// and sends WorkspacesLoadedMsg so the TUI updates without blocking startup.
func loadWorkspacesAndSend(reg *workspace.Registry, p *tea.Program) {
	list := reg.List()
	workspaces := make([]itui.WorkspaceInfo, 0, len(list))
	for _, entry := range list {
		info := itui.WorkspaceInfo{
			Entry:      entry,
			MaxWorkers: int(config.WorkspaceLegacy.MaxWorkers),
		}
		mgr := agent.NewWorkspaceManager(
			entry.Path+"/.bc/agents",
			entry.Path,
		)
		_ = mgr.LoadState()
		_ = mgr.RefreshState()
		info.Total = mgr.AgentCount()
		info.Running = mgr.RunningCount()
		info.HasBeads = beads.HasBeads(entry.Path)
		if info.HasBeads {
			if issues, err := beads.ListIssues(entry.Path); err == nil {
				info.Issues = len(issues)
			}
		}
		workspaces = append(workspaces, info)
	}
	p.Send(itui.WorkspacesLoadedMsg{Workspaces: workspaces})
}
