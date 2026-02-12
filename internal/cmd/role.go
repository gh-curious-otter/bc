package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	if len(roles) == 0 {
		fmt.Println("No roles defined (besides root)")
		return nil
	}

	fmt.Println("Workspace Roles")
	fmt.Println(strings.Repeat("=", 60))
	for name, role := range roles {
		fmt.Printf("\n%s\n", name)
		if role.Metadata.IsSingleton {
			fmt.Println("  [singleton]")
		}
		if len(role.Metadata.Capabilities) > 0 {
			fmt.Printf("  capabilities: %s\n", strings.Join(role.Metadata.Capabilities, ", "))
		}
		if len(role.Metadata.ParentRoles) > 0 {
			fmt.Printf("  parent roles: %s\n", strings.Join(role.Metadata.ParentRoles, ", "))
		}
	}

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
