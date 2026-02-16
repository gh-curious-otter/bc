package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/workspace"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage agent roles in the workspace",
	Long: `Manage custom agent roles stored in .bc/roles/*.md files.

Each role file contains YAML frontmatter with metadata and a Markdown prompt.

Examples:
  bc role list                                      # List all roles
  bc role show engineer                             # Show engineer role details
  bc role create --name my-role --prompt "..."      # Create role with inline prompt
  bc role create --name my-role --prompt-file x.md  # Create role from file
  bc role edit engineer                             # Edit engineer role in $EDITOR
  bc role delete custom                             # Delete a role
  bc role validate                                  # Validate all role files`,
	RunE: runRoleList,
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all roles in the workspace",
	RunE:  runRoleList,
}

var roleShowCmd = &cobra.Command{
	Use:   "show <role>",
	Short: "Show role definition and metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleShow,
}

var roleCreateCmd = &cobra.Command{
	Use:   "create --name <role>",
	Short: "Create a new role",
	Long: `Create a new role with a custom prompt.

Examples:
  bc role create --name my-role --prompt "You are a specialized agent..."
  bc role create --name my-role --prompt-file ./prompts/custom.md
  bc role create --name my-role --description "Code reviewer" --prompt "Review code..."
  bc role create --name my-role  # Creates blank role for editing`,
	RunE: runRoleCreate,
}

var roleEditCmd = &cobra.Command{
	Use:   "edit <role>",
	Short: "Edit a role in your $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleEdit,
}

var roleDeleteCmd = &cobra.Command{
	Use:   "delete <role>",
	Short: "Delete a role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleDelete,
}

var roleValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all role files",
	RunE:  runRoleValidate,
}

var roleRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a role",
	Long: `Rename an existing role, updating both the file name and metadata.

Example:
  bc role rename engineer senior-engineer`,
	Args: cobra.ExactArgs(2),
	RunE: runRoleRename,
}

var roleCloneCmd = &cobra.Command{
	Use:   "clone <source> <destination>",
	Short: "Clone an existing role",
	Long: `Create a copy of an existing role with a new name.
This is useful when creating a new role based on an existing one.

Example:
  bc role clone engineer frontend-engineer`,
	Args: cobra.ExactArgs(2),
	RunE: runRoleClone,
}

var roleDiffCmd = &cobra.Command{
	Use:   "diff <role-a> <role-b>",
	Short: "Compare two roles",
	Long: `Show differences between two roles.
Compares metadata (capabilities, parent roles) and prompts.

Example:
  bc role diff engineer manager`,
	Args: cobra.ExactArgs(2),
	RunE: runRoleDiff,
}

// Flags
var (
	roleName        string
	roleTemplate    string
	rolePrompt      string
	rolePromptFile  string
	roleDescription string
	roleForce       bool
)

func init() {
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleShowCmd)
	roleCmd.AddCommand(roleCreateCmd)
	roleCmd.AddCommand(roleEditCmd)
	roleCmd.AddCommand(roleDeleteCmd)
	roleCmd.AddCommand(roleValidateCmd)
	roleCmd.AddCommand(roleRenameCmd)
	roleCmd.AddCommand(roleCloneCmd)
	roleCmd.AddCommand(roleDiffCmd)

	roleCreateCmd.Flags().StringVar(&roleName, "name", "", "Name for the new role (required)")
	roleCreateCmd.Flags().StringVar(&rolePrompt, "prompt", "", "Inline prompt text for the role")
	roleCreateCmd.Flags().StringVar(&rolePromptFile, "prompt-file", "", "Path to file containing prompt text")
	roleCreateCmd.Flags().StringVar(&roleDescription, "description", "", "Brief description of the role")
	roleCreateCmd.Flags().StringVar(&roleTemplate, "template", "", "Template to use (engineer, manager, qa, blank) [deprecated]")
	_ = roleCreateCmd.MarkFlagRequired("name")

	roleDeleteCmd.Flags().BoolVar(&roleForce, "force", false, "Skip confirmation")

	rootCmd.AddCommand(roleCmd)
}

func getWorkspaceRoleManager() (*workspace.Workspace, *workspace.RoleManager, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, nil, fmt.Errorf("not in a bc workspace: %w", err)
	}

	rm := workspace.NewRoleManager(ws.StateDir())
	return ws, rm, nil
}

func runRoleList(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	roles, err := rm.LoadAllRoles()
	if err != nil {
		return fmt.Errorf("failed to load roles: %w", err)
	}

	// Check for JSON output flag
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Build JSON response matching TUI RolesResponse type
		type jsonRole struct {
			Name         string   `json:"name"`
			Description  string   `json:"description,omitempty"`
			Parent       string   `json:"parent,omitempty"`
			Capabilities []string `json:"capabilities"`
			AgentCount   int      `json:"agent_count"`
		}
		type jsonResponse struct {
			Roles []jsonRole `json:"roles"`
		}

		resp := jsonResponse{Roles: make([]jsonRole, 0, len(roles))}
		for name, role := range roles {
			caps := role.Metadata.Capabilities
			if caps == nil {
				caps = []string{}
			}
			parent := ""
			if len(role.Metadata.ParentRoles) > 0 {
				parent = role.Metadata.ParentRoles[0]
			}
			resp.Roles = append(resp.Roles, jsonRole{
				Name:         name,
				Description:  role.Description(),
				Capabilities: caps,
				Parent:       parent,
				AgentCount:   0, // TODO: Count agents with this role
			})
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	if len(roles) == 0 {
		fmt.Println("No roles defined (besides root)")
		return nil
	}

	// Collect role data and calculate column widths
	type roleRow struct {
		name         string
		description  string
		flags        string
		capabilities int
	}
	rows := make([]roleRow, 0, len(roles))
	maxNameLen := 4  // "ROLE"
	maxDescLen := 11 // "DESCRIPTION"

	for name, role := range roles {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}

		desc := role.Description()
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		if len(desc) > maxDescLen {
			maxDescLen = len(desc)
		}

		flags := ""
		if role.Metadata.IsSingleton {
			flags = "[singleton]"
		}

		rows = append(rows, roleRow{
			name:         name,
			capabilities: len(role.Metadata.Capabilities),
			description:  desc,
			flags:        flags,
		})
	}

	// Sort roles alphabetically
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].name < rows[j].name
	})

	// Check if any roles have capabilities defined
	hasCapabilities := false
	for _, r := range rows {
		if r.capabilities > 0 {
			hasCapabilities = true
			break
		}
	}

	// Print table header (hide CAPS column if all roles have 0 capabilities)
	if hasCapabilities {
		fmt.Printf("%-*s  %-4s  %-*s  %s\n", maxNameLen, "ROLE", "CAPS", maxDescLen, "DESCRIPTION", "FLAGS")
		fmt.Println(strings.Repeat("-", maxNameLen+maxDescLen+20))
	} else {
		fmt.Printf("%-*s  %-*s  %s\n", maxNameLen, "ROLE", maxDescLen, "DESCRIPTION", "FLAGS")
		fmt.Println(strings.Repeat("-", maxNameLen+maxDescLen+14))
	}

	// Print rows
	for _, r := range rows {
		if hasCapabilities {
			fmt.Printf("%-*s  %-4d  %-*s  %s\n", maxNameLen, r.name, r.capabilities, maxDescLen, r.description, r.flags)
		} else {
			fmt.Printf("%-*s  %-*s  %s\n", maxNameLen, r.name, maxDescLen, r.description, r.flags)
		}
	}

	fmt.Printf("\n%d role(s) defined\n", len(rows))

	return nil
}

func runRoleShow(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	roleArg := args[0]
	role, err := rm.LoadRole(roleArg)
	if err != nil {
		return fmt.Errorf("failed to load role %q: %w", roleArg, err)
	}

	fmt.Printf("Role: %s\n", role.Metadata.Name)
	if role.Metadata.IsSingleton {
		fmt.Println("Singleton: true")
	}
	if len(role.Metadata.Capabilities) > 0 {
		fmt.Printf("Capabilities: %s\n", strings.Join(role.Metadata.Capabilities, ", "))
	}
	if len(role.Metadata.ParentRoles) > 0 {
		fmt.Printf("Parent Roles: %s\n", strings.Join(role.Metadata.ParentRoles, ", "))
	}
	fmt.Printf("File: %s\n\n", role.FilePath)
	fmt.Println("Prompt:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(role.Prompt)

	return nil
}

func runRoleCreate(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	if !isValidRoleName(roleName) {
		return fmt.Errorf("invalid role name %q (must be alphanumeric with hyphens, max 50 chars)", roleName)
	}

	// Check if role already exists
	if rm.HasRole(roleName) {
		return fmt.Errorf("role %q already exists", roleName)
	}

	// Warn about deprecated --template flag
	if roleTemplate != "" {
		fmt.Fprintln(os.Stderr, "Warning: --template is deprecated. Use --prompt or --prompt-file instead.")
	}

	// Determine prompt content
	var promptContent string
	switch {
	case rolePromptFile != "":
		// Read prompt from file
		content, readErr := os.ReadFile(rolePromptFile) //nolint:gosec // G304: File path is user-provided via CLI flag
		if readErr != nil {
			return fmt.Errorf("failed to read prompt file %q: %w", rolePromptFile, readErr)
		}
		promptContent = string(content)
	case rolePrompt != "":
		// Use inline prompt
		promptContent = rolePrompt
	case roleTemplate != "":
		// Backwards compatibility: use template
		templateContent := getTemplate(roleTemplate)
		if templateContent == "" {
			return fmt.Errorf("unknown template %q", roleTemplate)
		}
		role, parseErr := workspace.ParseRoleFile([]byte(templateContent))
		if parseErr != nil {
			return fmt.Errorf("failed to parse template: %w", parseErr)
		}
		role.Metadata.Name = roleName
		if err := rm.WriteRole(role); err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
		fmt.Printf("✓ Created role %q\n", roleName)
		fmt.Printf("  File: .bc/roles/%s.md\n\n", roleName)
		fmt.Println("To edit the role:")
		fmt.Printf("  bc role edit %s\n", roleName)
		return nil
	default:
		// Create blank role
		promptContent = fmt.Sprintf("# %s\n\nDefine the purpose and responsibilities of this role.", roleName)
	}

	// Build role with custom prompt
	role := &workspace.Role{
		Metadata: workspace.RoleMetadata{
			Name:         roleName,
			Capabilities: []string{},
			ParentRoles:  []string{},
			IsSingleton:  false,
		},
		Prompt: promptContent,
	}

	// Write role file
	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	fmt.Printf("✓ Created role %q\n", roleName)
	fmt.Printf("  File: .bc/roles/%s.md\n\n", roleName)
	fmt.Println("To edit the role:")
	fmt.Printf("  bc role edit %s\n", roleName)

	return nil
}

func runRoleEdit(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	roleArg := args[0]
	role, err := rm.LoadRole(roleArg)
	if err != nil {
		return fmt.Errorf("failed to load role %q: %w", roleArg, err)
	}

	// Protect root role from editing
	if roleArg == "root" {
		return fmt.Errorf("cannot edit root role (it is hardcoded in the system)")
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	roleFile := role.FilePath
	if roleFile == "" {
		return fmt.Errorf("role file path not set")
	}

	// #nosec G204 - editor command is from user's EDITOR env var
	editorCmd := exec.CommandContext(context.Background(), editor, roleFile)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	runErr := editorCmd.Run()
	if runErr != nil {
		return fmt.Errorf("failed to open editor: %w", runErr)
	}

	// Reload and validate
	updatedRole, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to reload role after edit: %w", err)
	}

	fmt.Printf("✓ Updated role %q\n", roleName)
	fmt.Printf("  Name: %s\n", updatedRole.Metadata.Name)
	if len(updatedRole.Metadata.Capabilities) > 0 {
		fmt.Printf("  Capabilities: %s\n", strings.Join(updatedRole.Metadata.Capabilities, ", "))
	}

	return nil
}

func runRoleDelete(cmd *cobra.Command, args []string) error {
	ws, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	roleName := args[0]

	// Protect root role
	if roleName == "root" {
		return fmt.Errorf("cannot delete root role (it is hardcoded in the system)")
	}

	// Check if role exists
	if !rm.HasRole(roleName) {
		return fmt.Errorf("role %q not found", roleName)
	}

	if !roleForce {
		fmt.Printf("Delete role %q? [y/N]: ", roleName)
		var response string
		fmt.Scanln(&response) //nolint:errcheck
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
	}

	// Delete role file
	roleFile := fmt.Sprintf("%s/roles/%s.md", ws.StateDir(), roleName)
	if err := os.Remove(roleFile); err != nil {
		return fmt.Errorf("failed to delete role file: %w", err)
	}

	fmt.Printf("✓ Deleted role %q\n", roleName)
	return nil
}

func runRoleValidate(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	roles, err := rm.LoadAllRoles()
	if err != nil {
		return fmt.Errorf("failed to load roles: %w", err)
	}

	if len(roles) == 0 {
		fmt.Println("✓ No custom roles to validate (root only)")
		return nil
	}

	fmt.Println("Validating roles...")
	for name, role := range roles {
		if role.Metadata.Name != name {
			fmt.Printf("⚠ Role %q: name mismatch (metadata says %q)\n", name, role.Metadata.Name)
		} else {
			fmt.Printf("✓ %s\n", name)
		}
	}

	fmt.Println("\nAll roles validated")
	return nil
}

func runRoleRename(cmd *cobra.Command, args []string) error {
	ws, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	oldName := args[0]
	newName := args[1]

	// Protect root role
	if oldName == "root" {
		return fmt.Errorf("cannot rename root role (it is hardcoded in the system)")
	}

	// Validate new name
	if !isValidRoleName(newName) {
		return fmt.Errorf("invalid role name %q (must be alphanumeric with hyphens, max 50 chars)", newName)
	}

	// Check source exists
	if !rm.HasRole(oldName) {
		return fmt.Errorf("role %q not found", oldName)
	}

	// Check destination doesn't exist
	if rm.HasRole(newName) {
		return fmt.Errorf("role %q already exists", newName)
	}

	// Load the source role
	role, err := rm.LoadRole(oldName)
	if err != nil {
		return fmt.Errorf("failed to load role %q: %w", oldName, err)
	}

	// Update metadata with new name
	role.Metadata.Name = newName

	// Write the new role file
	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to write renamed role: %w", err)
	}

	// Delete the old role file
	oldFile := fmt.Sprintf("%s/roles/%s.md", ws.StateDir(), oldName)
	if err := os.Remove(oldFile); err != nil {
		return fmt.Errorf("failed to delete old role file: %w", err)
	}

	fmt.Printf("✓ Renamed role %q to %q\n", oldName, newName)
	return nil
}

func runRoleClone(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	srcName := args[0]
	dstName := args[1]

	// Validate destination name
	if !isValidRoleName(dstName) {
		return fmt.Errorf("invalid role name %q (must be alphanumeric with hyphens, max 50 chars)", dstName)
	}

	// Check source exists
	if !rm.HasRole(srcName) {
		return fmt.Errorf("source role %q not found", srcName)
	}

	// Check destination doesn't exist
	if rm.HasRole(dstName) {
		return fmt.Errorf("destination role %q already exists", dstName)
	}

	// Load the source role
	srcRole, err := rm.LoadRole(srcName)
	if err != nil {
		return fmt.Errorf("failed to load source role %q: %w", srcName, err)
	}

	// Create a copy with new name
	dstRole := &workspace.Role{
		Metadata: workspace.RoleMetadata{
			Name:         dstName,
			Description:  srcRole.Metadata.Description,
			Capabilities: append([]string{}, srcRole.Metadata.Capabilities...),
			ParentRoles:  append([]string{}, srcRole.Metadata.ParentRoles...),
			IsSingleton:  false, // Clones should not be singletons by default
		},
		Prompt: srcRole.Prompt,
	}

	// Write the new role
	if err := rm.WriteRole(dstRole); err != nil {
		return fmt.Errorf("failed to write cloned role: %w", err)
	}

	fmt.Printf("✓ Cloned role %q to %q\n", srcName, dstName)
	fmt.Printf("  File: .bc/roles/%s.md\n\n", dstName)
	fmt.Println("To edit the cloned role:")
	fmt.Printf("  bc role edit %s\n", dstName)
	return nil
}

func runRoleDiff(cmd *cobra.Command, args []string) error {
	_, rm, err := getWorkspaceRoleManager()
	if err != nil {
		return err
	}

	nameA := args[0]
	nameB := args[1]

	// Load both roles
	roleA, err := rm.LoadRole(nameA)
	if err != nil {
		return fmt.Errorf("failed to load role %q: %w", nameA, err)
	}

	roleB, err := rm.LoadRole(nameB)
	if err != nil {
		return fmt.Errorf("failed to load role %q: %w", nameB, err)
	}

	fmt.Printf("Comparing roles: %s vs %s\n", nameA, nameB)
	fmt.Println(strings.Repeat("=", 60))

	hasDiffs := false

	// Compare descriptions
	if roleA.Metadata.Description != roleB.Metadata.Description {
		hasDiffs = true
		fmt.Println("\nDescription:")
		fmt.Printf("  %s: %s\n", nameA, roleA.Metadata.Description)
		fmt.Printf("  %s: %s\n", nameB, roleB.Metadata.Description)
	}

	// Compare capabilities
	capsA := make(map[string]bool)
	capsB := make(map[string]bool)
	for _, c := range roleA.Metadata.Capabilities {
		capsA[c] = true
	}
	for _, c := range roleB.Metadata.Capabilities {
		capsB[c] = true
	}

	var onlyInA, onlyInB []string
	for c := range capsA {
		if !capsB[c] {
			onlyInA = append(onlyInA, c)
		}
	}
	for c := range capsB {
		if !capsA[c] {
			onlyInB = append(onlyInB, c)
		}
	}

	if len(onlyInA) > 0 || len(onlyInB) > 0 {
		hasDiffs = true
		fmt.Println("\nCapabilities:")
		sort.Strings(onlyInA)
		sort.Strings(onlyInB)
		for _, c := range onlyInA {
			fmt.Printf("  - %s (only in %s)\n", c, nameA)
		}
		for _, c := range onlyInB {
			fmt.Printf("  + %s (only in %s)\n", c, nameB)
		}
	}

	// Compare parent roles
	parentsA := make(map[string]bool)
	parentsB := make(map[string]bool)
	for _, p := range roleA.Metadata.ParentRoles {
		parentsA[p] = true
	}
	for _, p := range roleB.Metadata.ParentRoles {
		parentsB[p] = true
	}

	var parentsOnlyA, parentsOnlyB []string
	for p := range parentsA {
		if !parentsB[p] {
			parentsOnlyA = append(parentsOnlyA, p)
		}
	}
	for p := range parentsB {
		if !parentsA[p] {
			parentsOnlyB = append(parentsOnlyB, p)
		}
	}

	if len(parentsOnlyA) > 0 || len(parentsOnlyB) > 0 {
		hasDiffs = true
		fmt.Println("\nParent Roles:")
		sort.Strings(parentsOnlyA)
		sort.Strings(parentsOnlyB)
		for _, p := range parentsOnlyA {
			fmt.Printf("  - %s (only in %s)\n", p, nameA)
		}
		for _, p := range parentsOnlyB {
			fmt.Printf("  + %s (only in %s)\n", p, nameB)
		}
	}

	// Compare singleton status
	if roleA.Metadata.IsSingleton != roleB.Metadata.IsSingleton {
		hasDiffs = true
		fmt.Println("\nSingleton:")
		fmt.Printf("  %s: %v\n", nameA, roleA.Metadata.IsSingleton)
		fmt.Printf("  %s: %v\n", nameB, roleB.Metadata.IsSingleton)
	}

	// Compare prompts (line count and first difference)
	linesA := strings.Split(roleA.Prompt, "\n")
	linesB := strings.Split(roleB.Prompt, "\n")
	if roleA.Prompt != roleB.Prompt {
		hasDiffs = true
		fmt.Println("\nPrompt:")
		fmt.Printf("  %s: %d lines\n", nameA, len(linesA))
		fmt.Printf("  %s: %d lines\n", nameB, len(linesB))

		// Find first differing line
		maxLines := len(linesA)
		if len(linesB) < maxLines {
			maxLines = len(linesB)
		}
		for i := 0; i < maxLines; i++ {
			if linesA[i] != linesB[i] {
				fmt.Printf("  First difference at line %d:\n", i+1)
				if len(linesA[i]) > 60 {
					fmt.Printf("    %s: %s...\n", nameA, linesA[i][:60])
				} else {
					fmt.Printf("    %s: %s\n", nameA, linesA[i])
				}
				if len(linesB[i]) > 60 {
					fmt.Printf("    %s: %s...\n", nameB, linesB[i][:60])
				} else {
					fmt.Printf("    %s: %s\n", nameB, linesB[i])
				}
				break
			}
		}
	}

	if !hasDiffs {
		fmt.Println("\nNo differences found (roles are identical)")
	}

	return nil
}

// getTemplate returns the template content for the given template name.
func getTemplate(template string) string {
	switch strings.ToLower(template) {
	case "engineer":
		return engineerTemplate
	case "manager":
		return managerTemplate
	case "qa":
		return qaTemplate
	case "blank", "custom", "":
		return blankTemplate
	default:
		return ""
	}
}

// isValidRoleName checks if a role name is valid.
func isValidRoleName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}
	for _, ch := range name {
		isLower := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		isValid := isLower || isDigit || ch == '-' || ch == '_'
		if !isValid {
			return false
		}
	}
	return true
}

// Role templates

const blankTemplate = `---
name: custom
capabilities: []
parent_roles: []
is_singleton: false
---

# Custom Role

Define the purpose and responsibilities of this role.
`

const engineerTemplate = `---
name: engineer
capabilities:
  - implement_tasks
  - write_code
parent_roles: []
is_singleton: false
---

# Engineer

You are an engineer agent in the bc workspace.

## Responsibilities
- Implement assigned tasks
- Write code and tests
- Execute work items from the queue
- Report progress via bc report

## Guidelines
1. Focus on code quality and testing
2. Write clear commit messages
3. Communicate blockers immediately
4. Update task status regularly
`

const managerTemplate = `---
name: manager
capabilities:
  - create_agents
  - assign_work
  - review_work
parent_roles: []
is_singleton: false
---

# Manager

You are a manager agent in the bc workspace.

## Responsibilities
- Break down epics into tasks
- Assign work to engineers
- Review code and pull requests
- Create child agents (engineers, QA)
- Report team progress

## Guidelines
1. Ensure clear task descriptions
2. Review code before merging
3. Monitor team health and morale
4. Escalate blockers to leadership
5. Maintain project momentum
`

const qaTemplate = `---
name: qa
capabilities:
  - test_work
  - review_work
parent_roles: []
is_singleton: false
---

# QA Agent

You are a QA/testing agent in the bc workspace.

## Responsibilities
- Test implemented features
- Verify quality standards
- Review test coverage
- Report bugs and issues
- Validate completed work

## Guidelines
1. Create comprehensive test cases
2. Test edge cases and error paths
3. Verify requirements are met
4. Document test results
5. Suggest improvements
`
