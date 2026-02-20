package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
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

var teamCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Find and remove orphaned team members",
	Long: `Find and remove team members that reference non-existent agents.

By default, performs a dry-run showing what would be removed.
Use --fix to actually remove the orphaned members.

Examples:
  bc team cleanup           # Dry-run: show orphaned members
  bc team cleanup --fix     # Remove orphaned members`,
	RunE: runTeamCleanup,
}

var teamExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export team configuration",
	Long: `Export team configuration to JSON for sharing or backup.

The output includes all teams, their members, leads, and descriptions.
Pipe to a file to save: bc team export > teams.json

Examples:
  bc team export                    # Export all teams to stdout
  bc team export > teams.json       # Save to file
  bc team export --teams            # Export teams only (no roles/channels)`,
	RunE: runTeamExport,
}

var teamImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import team configuration",
	Long: `Import team configuration from a JSON file.

By default, performs a preview showing what will be imported.
Use --apply to actually import the configuration.

Examples:
  bc team import teams.json           # Preview what will be imported
  bc team import teams.json --apply   # Actually import
  bc team import teams.json --merge   # Merge with existing (don't overwrite)`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamImport,
}

func init() {
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamShowCmd)
	teamCmd.AddCommand(teamDeleteCmd)
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	teamCmd.AddCommand(teamRenameCmd)
	teamCmd.AddCommand(teamCleanupCmd)
	teamCmd.AddCommand(teamExportCmd)
	teamCmd.AddCommand(teamImportCmd)
	teamCleanupCmd.Flags().Bool("fix", false, "Actually remove orphaned members (default: dry-run)")
	teamImportCmd.Flags().Bool("apply", false, "Actually import the configuration (default: preview)")
	teamImportCmd.Flags().Bool("merge", false, "Merge with existing teams (don't overwrite)")
	rootCmd.AddCommand(teamCmd)
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
	}

	name := args[0]
	store := team.NewStore(ws.RootDir)

	t, err := store.Get(name)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("team %q not found (use 'bc team list' to see available teams)", name)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
	}

	teamName := args[0]
	agentName := args[1]
	store := team.NewStore(ws.RootDir)

	// Validate team exists
	if !store.Exists(teamName) {
		return fmt.Errorf("team %q not found. Create it first with: bc team create %s", teamName, teamName)
	}

	// Validate agent exists
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	_ = mgr.LoadState() // nolint:errcheck - continue even if state doesn't load
	if mgr.GetAgent(agentName) == nil {
		return fmt.Errorf("agent %q does not exist. Create it first with: bc agent create %s", agentName, agentName)
	}

	if err := store.AddMember(teamName, agentName); err != nil {
		return err
	}

	cmd.Printf("Added %q to team %q\n", agentName, teamName)
	return nil
}

func runTeamRemove(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	teamName := args[0]
	agentName := args[1]
	store := team.NewStore(ws.RootDir)

	// Validate team exists
	if !store.Exists(teamName) {
		return fmt.Errorf("team %q not found (use 'bc team list' to see available teams)", teamName)
	}

	if err := store.RemoveMember(teamName, agentName); err != nil {
		return err
	}

	cmd.Printf("Removed %q from team %q\n", agentName, teamName)
	return nil
}

func runTeamRename(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
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
		return fmt.Errorf("team %q not found (use 'bc team list' to see available teams)", oldName)
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

func runTeamCleanup(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	fix, _ := cmd.Flags().GetBool("fix")

	// Set up agent existence check
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	_ = mgr.LoadState() //nolint:errcheck // continue even if state doesn't load

	agentExists := func(name string) bool {
		return mgr.GetAgent(name) != nil
	}

	store := team.NewStore(ws.RootDir)

	if fix {
		// Actually remove orphaned members
		removed, cleanupErr := store.CleanupOrphanedMembers(agentExists)
		if cleanupErr != nil {
			return cleanupErr
		}
		if removed == 0 {
			cmd.Println("No orphaned members found")
		} else {
			cmd.Printf("Removed %d orphaned member(s)\n", removed)
		}
	} else {
		// Dry-run: just show orphaned members
		orphans, findErr := store.FindOrphanedMembers(agentExists)
		if findErr != nil {
			return findErr
		}
		if len(orphans) == 0 {
			cmd.Println("No orphaned members found")
		} else {
			cmd.Printf("Found %d orphaned member(s):\n", len(orphans))
			cmd.Println()
			for _, o := range orphans {
				role := "member"
				if o.IsLead {
					role = "lead"
				}
				cmd.Printf("  %-20s %s (%s)\n", o.TeamName, o.MemberName, role)
			}
			cmd.Println()
			cmd.Println("Run with --fix to remove these orphaned members")
		}
	}

	return nil
}

// TeamExport is the export format for team configurations.
type TeamExport struct {
	Version string       `json:"version"`
	Teams   []*team.Team `json:"teams"`
}

func runTeamExport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store := team.NewStore(ws.RootDir)
	teams, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	if teams == nil {
		teams = []*team.Team{}
	}

	export := TeamExport{
		Version: "1",
		Teams:   teams,
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(export)
}

func runTeamImport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	apply, _ := cmd.Flags().GetBool("apply")
	merge, _ := cmd.Flags().GetBool("merge")

	// Read import file
	filename := args[0]
	data, err := os.ReadFile(filename) //nolint:gosec // user-provided path
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var importData TeamExport
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	if importData.Version == "" {
		return fmt.Errorf("invalid export format: missing version field")
	}

	store := team.NewStore(ws.RootDir)

	// Preview mode
	if !apply {
		cmd.Printf("Preview import from %s:\n", filename)
		cmd.Printf("  Version: %s\n", importData.Version)
		cmd.Printf("  Teams:   %d\n", len(importData.Teams))
		cmd.Println()

		for _, t := range importData.Teams {
			existing, _ := store.Get(t.Name)
			status := "NEW"
			if existing != nil {
				if merge {
					status = "MERGE"
				} else {
					status = "OVERWRITE"
				}
			}
			cmd.Printf("  [%s] %s (%d members)\n", status, t.Name, len(t.Members))
		}
		cmd.Println()
		cmd.Println("Run with --apply to import these teams")
		return nil
	}

	// Apply import
	var created, updated int
	for _, t := range importData.Teams {
		existing, _ := store.Get(t.Name)

		if existing == nil {
			// Create new team
			if _, createErr := store.Create(t.Name); createErr != nil {
				cmd.PrintErrf("Warning: failed to create team %q: %v\n", t.Name, createErr)
				continue
			}
			created++
		} else if merge {
			// Skip existing in merge mode
			continue
		} else {
			updated++
		}

		// Update team properties
		if updateErr := store.Update(t.Name, func(existing *team.Team) {
			existing.Description = t.Description
			existing.Lead = t.Lead
			existing.Members = t.Members
		}); updateErr != nil {
			cmd.PrintErrf("Warning: failed to update team %q: %v\n", t.Name, updateErr)
		}
	}

	cmd.Printf("Import complete: %d created, %d updated\n", created, updated)
	return nil
}
