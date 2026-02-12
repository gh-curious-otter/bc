package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/team"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams",
	Long: `Manage organizational teams.

Teams group agents together for collaboration and organization.

Examples:
  bc team create engineering
  bc team list
  bc team show engineering
  bc team add engineering eng-01
  bc team remove engineering eng-01
  bc team rename engineering backend
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
	Long: `Add an agent to a team.

Examples:
  bc team add engineering eng-01
  bc team add frontend designer-01`,
	Args: cobra.ExactArgs(2),
	RunE: runTeamAdd,
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <team> <agent>",
	Short: "Remove an agent from a team",
	Long: `Remove an agent from a team.

Examples:
  bc team remove engineering eng-01
  bc team remove frontend designer-01`,
	Args: cobra.ExactArgs(2),
	RunE: runTeamRemove,
}

var teamRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a team",
	Long: `Rename a team while preserving all members and settings.

Examples:
  bc team rename frontend web-team
  bc team rename backend api-team`,
	Args: cobra.ExactArgs(2),
	RunE: runTeamRename,
}

func init() {
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamShowCmd)
	teamCmd.AddCommand(teamDeleteCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	teamCmd.AddCommand(teamRenameCmd)
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

	// Check for JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Wrap in object for TUI compatibility
		if teams == nil {
			teams = []*team.Team{}
		}
		response := struct {
			Teams []*team.Team `json:"teams"`
		}{Teams: teams}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
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

func runTeamRename(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	oldName := args[0]
	newName := args[1]
	store := team.NewStore(ws.RootDir)

	// Get old team
	oldTeam, err := store.Get(oldName)
	if err != nil {
		return err
	}
	if oldTeam == nil {
		return fmt.Errorf("team %q not found", oldName)
	}

	// Check new name doesn't exist
	if store.Exists(newName) {
		return fmt.Errorf("team %q already exists", newName)
	}

	// Create new team
	newTeam, err := store.Create(newName)
	if err != nil {
		return fmt.Errorf("failed to create new team: %w", err)
	}

	// Copy properties from old team
	if updateErr := store.Update(newName, func(t *team.Team) {
		t.Description = oldTeam.Description
		t.Members = oldTeam.Members
		t.Lead = oldTeam.Lead
		t.CreatedAt = oldTeam.CreatedAt // Preserve original creation time
	}); updateErr != nil {
		// Cleanup: delete the new team we just created
		_ = store.Delete(newName)
		return fmt.Errorf("failed to update new team: %w", updateErr)
	}

	// Delete old team
	if deleteErr := store.Delete(oldName); deleteErr != nil {
		cmd.PrintErrf("Warning: renamed to %q but failed to delete old team %q: %v\n", newName, oldName, deleteErr)
	}

	cmd.Printf("✓ Renamed team %q to %q\n", oldName, newName)
	if len(newTeam.Members) > 0 || oldTeam.Lead != "" {
		cmd.Printf("  Members: %d\n", len(oldTeam.Members))
		if oldTeam.Lead != "" {
			cmd.Printf("  Lead: %s\n", oldTeam.Lead)
		}
	}

	return nil
}
