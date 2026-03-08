package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
)

// CompleteAgentNames returns a completion function for agent names
func CompleteAgentNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	mgr := newAgentManager(ws)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Debug("completion: failed to load agent state", "error", loadErr)
	}

	agents := mgr.ListAgents()
	names := make([]string, 0, len(agents))
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// CompleteChannelNames returns a completion function for channel names
func CompleteChannelNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	store := channel.NewStore(filepath.Join(ws.StateDir(), "channels"))
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			log.Debug("completion: failed to close channel store", "error", closeErr)
		}
	}()

	channels := store.List()
	names := make([]string, 0, len(channels))
	for _, ch := range channels {
		names = append(names, ch.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// CompleteRoleNames returns a completion function for role names
func CompleteRoleNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ws, err := getWorkspace()
	if err != nil {
		// Return built-in roles as fallback
		return []string{"root", "manager", "engineer", "tech-lead", "product-manager"}, cobra.ShellCompDirectiveNoFileComp
	}

	rm := workspace.NewRoleManager(ws.StateDir())
	roles, rolesErr := rm.LoadAllRoles()
	if rolesErr != nil {
		log.Debug("completion: failed to list roles", "error", rolesErr)
		return []string{"root", "manager", "engineer", "tech-lead", "product-manager"}, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Metadata.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for bc.

To load completions:

Bash:
  $ source <(bc completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ bc completion bash > /etc/bash_completion.d/bc
  # macOS:
  $ bc completion bash > $(brew --prefix)/etc/bash_completion.d/bc

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ bc completion zsh > "${fpath[1]}/_bc"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ bc completion fish | source

  # To load completions for each session, execute once:
  $ bc completion fish > ~/.config/fish/completions/bc.fish

PowerShell:
  PS> bc completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> bc completion powershell > bc.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
