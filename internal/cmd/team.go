package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/team"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams",
	Long: `Manage organizational teams.

Teams group agents together for collaboration and organization.

Example:
  bc team create engineering
  bc team list
  bc team show engineering
  bc team delete engineering`,
}

var teamCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamCreate,
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all teams",
	RunE:  runTeamList,
}

var teamShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show team details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamShow,
}

var teamDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamDelete,
}

var teamAddCmd = &cobra.Command{
	Use:   "add <team> <agent>",
	Short: "Add an agent to a team",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamAdd,
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <team> <agent>",
	Short: "Remove an agent from a team",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamRemove,
}

func init() {
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamShowCmd)
	teamCmd.AddCommand(teamDeleteCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	rootCmd.AddCommand(teamCmd)
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := team.NewStore(ws.RootDir)

	t, err := store.Create(name)
	if err != nil {
		return err
	}

	cmd.Printf("Created team %q\n", t.Name)
	return nil
}

func runTeamList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store := team.NewStore(ws.RootDir)
	teams, err := store.List()
	if err != nil {
		return err
	}

	if len(teams) == 0 {
		cmd.Println("No teams configured")
		cmd.Println()
		cmd.Println("Create one with: bc team create <name>")
		return nil
	}

	cmd.Printf("%-20s %-10s %-20s %s\n", "NAME", "MEMBERS", "LEAD", "DESCRIPTION")
	cmd.Println("--------------------------------------------------------------------")
	for _, t := range teams {
		lead := t.Lead
		if lead == "" {
			lead = "-"
		}
		desc := t.Description
		if len(desc) > 25 {
			desc = desc[:22] + "..."
		}
		if desc == "" {
			desc = "-"
		}
		cmd.Printf("%-20s %-10d %-20s %s\n", t.Name, len(t.Members), lead, desc)
	}

	return nil
}

func runTeamShow(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := team.NewStore(ws.RootDir)

	t, err := store.Get(name)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("team %q not found", name)
	}

	cmd.Printf("Name:        %s\n", t.Name)
	if t.Description != "" {
		cmd.Printf("Description: %s\n", t.Description)
	}
	if t.Lead != "" {
		cmd.Printf("Lead:        %s\n", t.Lead)
	}
	cmd.Printf("Members:     %d\n", len(t.Members))
	if len(t.Members) > 0 {
		for _, m := range t.Members {
			cmd.Printf("  - %s\n", m)
		}
	}
	cmd.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runTeamDelete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := team.NewStore(ws.RootDir)

	if err := store.Delete(name); err != nil {
		return err
	}

	cmd.Printf("Deleted team %q\n", name)
	return nil
}

func runTeamAdd(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	teamName := args[0]
	agentName := args[1]
	store := team.NewStore(ws.RootDir)

	if err := store.AddMember(teamName, agentName); err != nil {
		return err
	}

	cmd.Printf("Added %q to team %q\n", agentName, teamName)
	return nil
}

func runTeamRemove(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	teamName := args[0]
	agentName := args[1]
	store := team.NewStore(ws.RootDir)

	if err := store.RemoveMember(teamName, agentName); err != nil {
		return err
	}

	cmd.Printf("Removed %q from team %q\n", agentName, teamName)
	return nil
}
