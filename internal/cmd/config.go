package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/workspace"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage workspace configuration",
	Long: `Commands for managing workspace configuration (.bc/config.toml).

Configuration uses a hierarchical key structure with dot notation:
  workspace.name
  tools.claude.command
  roster.engineers

Examples:
  bc config show                        # Show all config
  bc config get tools.default           # Get a specific value
  bc config set roster.engineers 6      # Set a value
  bc config list                        # List all config keys
  bc config edit                        # Open config in editor
  bc config validate                    # Validate config file
  bc config reset                       # Reset to defaults`,
	RunE: runConfigShow,
}

var configShowCmd = &cobra.Command{
	Use:   "show [key]",
	Short: "Show configuration",
	Long: `Display the current workspace configuration.

If a key is specified, shows only that section. Otherwise shows entire config.

Examples:
  bc config show                  # Show all config
  bc config show tools            # Show tools section
  bc config show tools.claude     # Show specific tool config
  bc config show --json           # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigShow,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a specific configuration value using dot notation.

Examples:
  bc config get workspace.name
  bc config get tools.default
  bc config get roster.engineers
  bc config get tools.claude.command`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a specific configuration value using dot notation.

The value type is automatically inferred (string, number, boolean).

Examples:
  bc config set roster.engineers 6
  bc config set tools.default claude
  bc config set worktrees.auto_cleanup true
  bc config set tools.claude.command "claude --force"`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration keys",
	Long: `List all available configuration keys in the workspace config.

Examples:
  bc config list
  bc config list --json           # Output as JSON array`,
	RunE: runConfigList,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration file",
	Long: `Open the workspace configuration file in your default editor.

Uses $EDITOR environment variable, falls back to 'nano' if not set.

Examples:
  bc config edit`,
	RunE: runConfigEdit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate the workspace configuration file for errors.

Checks for:
  - Valid TOML syntax
  - Required fields present
  - Valid values and types
  - Tool references exist

Examples:
  bc config validate`,
	RunE: runConfigValidate,
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long: `Reset the workspace configuration to default values.

WARNING: This will overwrite your current config. Back up first if needed.

Examples:
  bc config reset
  bc config reset --force         # Skip confirmation`,
	RunE: runConfigReset,
}

// #1197: Config export/import commands

var configExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export workspace configuration",
	Long: `Export workspace configuration to a file for sharing with team members.

The export includes:
- Workspace settings (name, version)
- Tool configurations
- Channel definitions
- Roster settings
- Performance tuning

The export excludes:
- User-specific settings (nickname)
- Local paths (worktrees.path, memory.path)
- Secrets and API keys

Examples:
  bc config export                        # Export to stdout
  bc config export team-config.toml       # Export to file
  bc config export --include-roles        # Include role definitions
  bc config export --format json          # Export as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigExport,
}

var configImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import workspace configuration",
	Long: `Import workspace configuration from a file.

Imports team configuration settings while preserving local paths and user settings.

Examples:
  bc config import team-config.toml       # Import from file
  bc config import --merge config.toml    # Merge with existing config
  bc config import --force config.toml    # Overwrite without confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigImport,
}

// #1160: User-level config (.bcrc) commands

var configUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage user-level configuration (~/.bcrc)",
	Long: `Manage user-level configuration stored in ~/.bcrc.

User configuration provides defaults that apply across all bc workspaces:
  - Your nickname for channel messages
  - Default role for new agents
  - Preferred AI tools

Workspace config (.bc/config.toml) takes precedence over user config.

Examples:
  bc config user init   # Create ~/.bcrc with guided prompts
  bc config user show   # Show user config
  bc config user path   # Show user config path`,
	RunE: runConfigUserShow,
}

var configUserInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create user configuration file (~/.bcrc)",
	Long: `Create a new ~/.bcrc file with guided prompts.

Examples:
  bc config user init          # Interactive setup
  bc config user init --quick  # Use defaults without prompts`,
	RunE: runConfigUserInit,
}

var configUserShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show user configuration",
	Long: `Display the current user configuration from ~/.bcrc.

Examples:
  bc config user show`,
	RunE: runConfigUserShow,
}

var configUserPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show user configuration file path",
	Long: `Display the path to the user configuration file.

Examples:
  bc config user path`,
	RunE: runConfigUserPath,
}

// Flags
var (
	configForce        bool
	configExportFormat string
	configIncludeRoles bool
	configImportMerge  bool
	configImportForce  bool
)

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configResetCmd)

	// #1160: User config subcommands
	configCmd.AddCommand(configUserCmd)
	configUserCmd.AddCommand(configUserInitCmd)
	configUserCmd.AddCommand(configUserShowCmd)
	configUserCmd.AddCommand(configUserPathCmd)

	// #1197: Export/import subcommands
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)

	configResetCmd.Flags().BoolVar(&configForce, "force", false, "Skip confirmation prompt")
	configUserInitCmd.Flags().Bool("quick", false, "Use defaults without prompts")

	// Export flags
	configExportCmd.Flags().StringVar(&configExportFormat, "format", "toml", "Output format: toml, json")
	configExportCmd.Flags().BoolVar(&configIncludeRoles, "include-roles", false, "Include role definitions")

	// Import flags
	configImportCmd.Flags().BoolVar(&configImportMerge, "merge", false, "Merge with existing config instead of replacing")
	configImportCmd.Flags().BoolVar(&configImportForce, "force", false, "Skip confirmation prompt")

	rootCmd.AddCommand(configCmd)
}

func loadWorkspaceConfig() (*workspace.V2Config, string, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, "", fmt.Errorf("not in a bc workspace: %w", err)
	}

	if ws.V2Config == nil {
		return nil, "", fmt.Errorf("workspace is using v1 config format. Run 'bc init' to upgrade to v2")
	}

	configPath := workspace.ConfigPath(ws.RootDir)
	return ws.V2Config, configPath, nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	// If a key is specified, show only that section
	if len(args) > 0 {
		key := args[0]
		value, err := getConfigValue(cfg, key)
		if err != nil {
			return err
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(value)
		}

		return printValue(key, value, 0)
	}

	// Show entire config
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	}

	printConfig(cfg)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	key := args[0]
	value, err := getConfigValue(cfg, key)
	if err != nil {
		return err
	}

	// Print just the value
	fmt.Println(formatValue(value))
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	key := args[0]
	valueStr := args[1]

	if err := setConfigValue(cfg, key, valueStr); err != nil {
		return err
	}

	// Save the config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, valueStr)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	keys := listConfigKeys(cfg, "")

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(keys)
	}

	for _, key := range keys {
		fmt.Println(key)
	}
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	_, configPath, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	// #nosec G204 - editor command is from user's EDITOR env var
	editorCmd := exec.CommandContext(context.Background(), editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Config validation failed: %v\n", err)
		fmt.Printf("   File: %s\n", configPath)
		return err
	}

	fmt.Printf("✓ Config is valid\n")
	fmt.Printf("  File: %s\n", configPath)
	return nil
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	configPath := workspace.ConfigPath(ws.RootDir)

	if !configForce {
		fmt.Printf("⚠️  This will overwrite your config at: %s\n", configPath)
		fmt.Print("Continue? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Handle scan error or empty input
			fmt.Println("Canceled")
			return nil
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
	}

	// Create default config
	defaultCfg := workspace.DefaultV2Config(ws.Config.Name)

	// Save it
	if err := defaultCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Config reset to defaults\n")
	fmt.Printf("  File: %s\n", configPath)
	return nil
}

// #1197: Export config implementation

// ExportableConfig is a subset of V2Config safe for sharing
// Excludes user-specific settings and local paths
type ExportableConfig struct {
	Tools       workspace.ToolsConfig       `toml:"tools" json:"tools"`
	TUI         workspace.TUIConfig         `toml:"tui" json:"tui"`
	Workspace   ExportableWorkspace         `toml:"workspace" json:"workspace"`
	Channels    workspace.ChannelsConfig    `toml:"channels" json:"channels"`
	Performance workspace.PerformanceConfig `toml:"performance" json:"performance"`
	Roster      workspace.RosterConfig      `toml:"roster" json:"roster"`
}

// ExportableWorkspace excludes version (schema-specific)
type ExportableWorkspace struct {
	Name string `toml:"name" json:"name"`
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Create exportable config (excludes user settings and local paths)
	exportCfg := ExportableConfig{
		Tools: cfg.Tools,
		TUI:   cfg.TUI,
		Workspace: ExportableWorkspace{
			Name: cfg.Workspace.Name,
		},
		Channels:    cfg.Channels,
		Performance: cfg.Performance,
		Roster:      cfg.Roster,
	}

	// Determine output destination
	var output io.Writer = os.Stdout
	if len(args) > 0 {
		outputPath := filepath.Clean(args[0])
		f, err := os.Create(outputPath) //nolint:gosec // user-provided output path is intentional
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = f.Close() }()
		output = f
	}

	// Export in requested format
	switch configExportFormat {
	case "toml":
		enc := toml.NewEncoder(output)
		if err := enc.Encode(exportCfg); err != nil {
			return fmt.Errorf("failed to encode TOML: %w", err)
		}
	case "json":
		enc := json.NewEncoder(output)
		enc.SetIndent("", "  ")
		if err := enc.Encode(exportCfg); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use toml or json)", configExportFormat)
	}

	// Include roles if requested
	if configIncludeRoles && len(args) > 0 {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		rolesDir := filepath.Join(ws.RootDir, ".bc", "roles")
		outputDir := filepath.Dir(args[0])
		rolesOutputDir := filepath.Join(outputDir, "roles")

		if err := copyRoles(rolesDir, rolesOutputDir); err != nil {
			fmt.Printf("Warning: could not export roles: %v\n", err)
		} else {
			fmt.Printf("✓ Exported roles to %s\n", rolesOutputDir)
		}
	}

	if len(args) > 0 {
		fmt.Printf("✓ Config exported to %s\n", args[0])
	}

	return nil
}

func copyRoles(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No roles to copy
		}
		return err
	}

	if err := os.MkdirAll(dstDir, 0750); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		data, err := os.ReadFile(srcPath) //nolint:gosec // copying from known roles directory
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, data, 0600); err != nil {
			return err
		}
	}

	return nil
}

func runConfigImport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	importPath := filepath.Clean(args[0])

	// Read import file
	data, err := os.ReadFile(importPath) //nolint:gosec // user-provided import path is intentional
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Determine format from extension
	var importCfg ExportableConfig
	ext := strings.ToLower(filepath.Ext(importPath))

	switch ext {
	case ".toml":
		if _, err := toml.Decode(string(data), &importCfg); err != nil {
			return fmt.Errorf("failed to parse TOML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &importCfg); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		// Try TOML first, then JSON
		if _, err := toml.Decode(string(data), &importCfg); err != nil {
			if jsonErr := json.Unmarshal(data, &importCfg); jsonErr != nil {
				return fmt.Errorf("failed to parse config (tried TOML and JSON)")
			}
		}
	}

	configPath := workspace.ConfigPath(ws.RootDir)

	// Confirm unless force flag
	if !configImportForce {
		fmt.Printf("⚠️  This will modify your config at: %s\n", configPath)
		if configImportMerge {
			fmt.Println("   Mode: merge (preserves existing values not in import)")
		} else {
			fmt.Println("   Mode: replace (overwrites with imported values)")
		}
		fmt.Print("Continue? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Println("Canceled")
			return nil
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
	}

	// Load existing config
	existingCfg := ws.V2Config
	if existingCfg == nil {
		return fmt.Errorf("workspace is using v1 config format. Run 'bc init' to upgrade to v2")
	}

	// Apply imported values
	if configImportMerge {
		// Merge: only overwrite non-zero values
		mergeConfig(existingCfg, &importCfg)
	} else {
		// Replace: overwrite all importable fields
		existingCfg.Tools = importCfg.Tools
		existingCfg.TUI = importCfg.TUI
		existingCfg.Channels = importCfg.Channels
		existingCfg.Performance = importCfg.Performance
		existingCfg.Roster = importCfg.Roster
		if importCfg.Workspace.Name != "" {
			existingCfg.Workspace.Name = importCfg.Workspace.Name
		}
	}

	// Save the config
	if err := existingCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Config imported from %s\n", importPath)
	fmt.Printf("  File: %s\n", configPath)

	// Check for roles directory alongside import file
	rolesDir := filepath.Join(filepath.Dir(importPath), "roles")
	if info, err := os.Stat(rolesDir); err == nil && info.IsDir() {
		destRolesDir := filepath.Join(ws.RootDir, ".bc", "roles")
		if err := copyRoles(rolesDir, destRolesDir); err != nil {
			fmt.Printf("Warning: could not import roles: %v\n", err)
		} else {
			fmt.Printf("✓ Imported roles to %s\n", destRolesDir)
		}
	}

	return nil
}

// mergeConfig merges importCfg into existingCfg, only setting non-zero values
func mergeConfig(existing *workspace.V2Config, imported *ExportableConfig) {
	// Tools
	if imported.Tools.Default != "" {
		existing.Tools.Default = imported.Tools.Default
	}
	if imported.Tools.Claude != nil {
		if existing.Tools.Claude == nil {
			existing.Tools.Claude = imported.Tools.Claude
		} else {
			if imported.Tools.Claude.Command != "" {
				existing.Tools.Claude.Command = imported.Tools.Claude.Command
			}
			if imported.Tools.Claude.Enabled {
				existing.Tools.Claude.Enabled = true
			}
		}
	}
	if imported.Tools.Cursor != nil {
		if existing.Tools.Cursor == nil {
			existing.Tools.Cursor = imported.Tools.Cursor
		} else {
			if imported.Tools.Cursor.Command != "" {
				existing.Tools.Cursor.Command = imported.Tools.Cursor.Command
			}
			if imported.Tools.Cursor.Enabled {
				existing.Tools.Cursor.Enabled = true
			}
		}
	}

	// Roster (only non-zero values)
	if imported.Roster.Engineers > 0 {
		existing.Roster.Engineers = imported.Roster.Engineers
	}
	if imported.Roster.TechLeads > 0 {
		existing.Roster.TechLeads = imported.Roster.TechLeads
	}
	if imported.Roster.QA > 0 {
		existing.Roster.QA = imported.Roster.QA
	}

	// Channels
	if len(imported.Channels.Default) > 0 {
		existing.Channels.Default = imported.Channels.Default
	}

	// Performance (only non-zero values)
	if imported.Performance.PollIntervalAgents > 0 {
		existing.Performance.PollIntervalAgents = imported.Performance.PollIntervalAgents
	}
	if imported.Performance.PollIntervalChannels > 0 {
		existing.Performance.PollIntervalChannels = imported.Performance.PollIntervalChannels
	}
	if imported.Performance.AdaptiveFastInterval > 0 {
		existing.Performance.AdaptiveFastInterval = imported.Performance.AdaptiveFastInterval
	}
	if imported.Performance.AdaptiveNormalInterval > 0 {
		existing.Performance.AdaptiveNormalInterval = imported.Performance.AdaptiveNormalInterval
	}
	if imported.Performance.AdaptiveSlowInterval > 0 {
		existing.Performance.AdaptiveSlowInterval = imported.Performance.AdaptiveSlowInterval
	}

	// Workspace name
	if imported.Workspace.Name != "" {
		existing.Workspace.Name = imported.Workspace.Name
	}
}

// Helper functions

func getConfigValue(cfg *workspace.V2Config, key string) (any, error) {
	parts := strings.Split(key, ".")
	v := reflect.ValueOf(*cfg)

	for i, part := range parts {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("invalid key path at '%s'", strings.Join(parts[:i+1], "."))
		}

		// Find field (case-insensitive)
		field := findField(v, part)
		if !field.IsValid() {
			return nil, fmt.Errorf("unknown config key: %s", key)
		}

		v = field
	}

	return v.Interface(), nil
}

func setConfigValue(cfg *workspace.V2Config, key, valueStr string) error {
	parts := strings.Split(key, ".")
	v := reflect.ValueOf(cfg).Elem()

	// Navigate to the parent of the field we want to set
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return fmt.Errorf("invalid key path at '%s'", strings.Join(parts[:i+1], "."))
		}

		field := findField(v, part)
		if !field.IsValid() {
			return fmt.Errorf("unknown config key: %s", strings.Join(parts[:i+1], "."))
		}

		v = field
	}

	// Set the final field
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	lastPart := parts[len(parts)-1]
	field := findField(v, lastPart)

	if !field.IsValid() {
		return fmt.Errorf("unknown config key: %s", key)
	}

	if !field.CanSet() {
		return fmt.Errorf("cannot set config key: %s", key)
	}

	// Convert value string to appropriate type
	switch field.Kind() {
	case reflect.String:
		field.SetString(valueStr)
	case reflect.Int, reflect.Int64:
		val, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", valueStr)
		}
		field.SetInt(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(valueStr)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", valueStr)
		}
		field.SetBool(val)
	case reflect.Float64:
		val, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", valueStr)
		}
		field.SetFloat(val)
	default:
		return fmt.Errorf("unsupported type for key %s: %s", key, field.Kind())
	}

	return nil
}

func findField(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		// Check TOML tag first
		if tomlTag := field.Tag.Get("toml"); tomlTag != "" {
			tagName := strings.Split(tomlTag, ",")[0]
			if strings.EqualFold(tagName, name) {
				return v.Field(i)
			}
		}
		// Fall back to field name
		if strings.EqualFold(field.Name, name) {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func listConfigKeys(cfg *workspace.V2Config, prefix string) []string {
	var keys []string
	v := reflect.ValueOf(*cfg)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Get TOML name
		tomlName := field.Name
		if tomlTag := field.Tag.Get("toml"); tomlTag != "" {
			tomlName = strings.Split(tomlTag, ",")[0]
		}
		tomlName = strings.ToLower(tomlName)

		fullKey := tomlName
		if prefix != "" {
			fullKey = prefix + "." + tomlName
		}

		// If it's a struct, recurse
		if fieldValue.Kind() == reflect.Struct {
			subKeys := listStructKeys(fieldValue, fullKey)
			keys = append(keys, subKeys...)
		} else {
			keys = append(keys, fullKey)
		}
	}

	return keys
}

func listStructKeys(v reflect.Value, prefix string) []string {
	var keys []string
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Get TOML name
		tomlName := field.Name
		if tomlTag := field.Tag.Get("toml"); tomlTag != "" {
			tomlName = strings.Split(tomlTag, ",")[0]
		}
		tomlName = strings.ToLower(tomlName)

		fullKey := prefix + "." + tomlName

		// If it's a struct, recurse
		if fieldValue.Kind() == reflect.Struct {
			subKeys := listStructKeys(fieldValue, fullKey)
			keys = append(keys, subKeys...)
		} else if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
			elem := fieldValue.Elem()
			if elem.Kind() == reflect.Struct {
				subKeys := listStructKeys(elem, fullKey)
				keys = append(keys, subKeys...)
			} else {
				keys = append(keys, fullKey)
			}
		} else {
			keys = append(keys, fullKey)
		}
	}

	return keys
}

func printConfig(cfg *workspace.V2Config) {
	fmt.Println("Workspace Configuration")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	fmt.Println("[workspace]")
	fmt.Printf("  name: %s\n", cfg.Workspace.Name)
	fmt.Printf("  version: %d\n", cfg.Workspace.Version)
	fmt.Println()

	fmt.Println("[worktrees]")
	fmt.Printf("  path: %s\n", cfg.Worktrees.Path)
	fmt.Printf("  auto_cleanup: %v\n", cfg.Worktrees.AutoCleanup)
	fmt.Println()

	fmt.Println("[tools]")
	fmt.Printf("  default: %s\n", cfg.Tools.Default)
	if cfg.Tools.Claude != nil {
		fmt.Printf("  claude.command: %s\n", cfg.Tools.Claude.Command)
		fmt.Printf("  claude.enabled: %v\n", cfg.Tools.Claude.Enabled)
	}
	if cfg.Tools.Cursor != nil {
		fmt.Printf("  cursor.command: %s\n", cfg.Tools.Cursor.Command)
		fmt.Printf("  cursor.enabled: %v\n", cfg.Tools.Cursor.Enabled)
	}
	if cfg.Tools.Codex != nil {
		fmt.Printf("  codex.command: %s\n", cfg.Tools.Codex.Command)
		fmt.Printf("  codex.enabled: %v\n", cfg.Tools.Codex.Enabled)
	}
	fmt.Println()

	fmt.Println("[memory]")
	fmt.Printf("  backend: %s\n", cfg.Memory.Backend)
	fmt.Printf("  path: %s\n", cfg.Memory.Path)
	fmt.Println()

	fmt.Println("[channels]")
	fmt.Printf("  default: %v\n", cfg.Channels.Default)
	fmt.Println()

	fmt.Println("[roster]")
	fmt.Printf("  engineers: %d\n", cfg.Roster.Engineers)
	fmt.Printf("  tech_leads: %d\n", cfg.Roster.TechLeads)
	fmt.Printf("  qa: %d\n", cfg.Roster.QA)
	fmt.Println()

	fmt.Println("[performance]")
	fmt.Printf("  poll_interval_agents: %d\n", cfg.Performance.PollIntervalAgents)
	fmt.Printf("  poll_interval_channels: %d\n", cfg.Performance.PollIntervalChannels)
	fmt.Printf("  poll_interval_costs: %d\n", cfg.Performance.PollIntervalCosts)
	fmt.Printf("  poll_interval_status: %d\n", cfg.Performance.PollIntervalStatus)
	fmt.Printf("  poll_interval_logs: %d\n", cfg.Performance.PollIntervalLogs)
	fmt.Printf("  poll_interval_teams: %d\n", cfg.Performance.PollIntervalTeams)
	fmt.Printf("  poll_interval_demons: %d\n", cfg.Performance.PollIntervalDemons)
	fmt.Printf("  cache_ttl_tmux: %d\n", cfg.Performance.CacheTTLTmux)
	fmt.Printf("  cache_ttl_commands: %d\n", cfg.Performance.CacheTTLCommands)
	fmt.Printf("  adaptive_fast_interval: %d\n", cfg.Performance.AdaptiveFastInterval)
	fmt.Printf("  adaptive_normal_interval: %d\n", cfg.Performance.AdaptiveNormalInterval)
	fmt.Printf("  adaptive_slow_interval: %d\n", cfg.Performance.AdaptiveSlowInterval)
	fmt.Printf("  adaptive_max_interval: %d\n", cfg.Performance.AdaptiveMaxInterval)
}

func printValue(key string, value any, indent int) error {
	prefix := strings.Repeat("  ", indent)

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Struct {
		fmt.Printf("%s[%s]\n", prefix, key)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			tomlName := field.Name
			if tomlTag := field.Tag.Get("toml"); tomlTag != "" {
				tomlName = strings.Split(tomlTag, ",")[0]
			}

			if err := printValue(tomlName, fieldValue.Interface(), indent+1); err != nil {
				return err
			}
		}
		return nil
	}

	fmt.Printf("%s%s: %s\n", prefix, key, formatValue(value))
	return nil
}

func formatValue(value any) string {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Slice:
		var items []string
		for i := 0; i < v.Len(); i++ {
			items = append(items, formatValue(v.Index(i).Interface()))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case reflect.Ptr:
		if v.IsNil() {
			return "<nil>"
		}
		return formatValue(v.Elem().Interface())
	default:
		return fmt.Sprintf("%v", value)
	}
}

// #1160: User config command implementations

func runConfigUserInit(cmd *cobra.Command, _ []string) error {
	quick, _ := cmd.Flags().GetBool("quick")

	path := workspace.UserRCConfigPath()
	if path == "" {
		return fmt.Errorf("could not determine home directory")
	}

	// Check if already exists
	if workspace.UserRCExists() {
		fmt.Printf("⚠️  User config already exists at %s\n", path)
		fmt.Print("Overwrite? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Handle scan error or empty input
			fmt.Println("Canceled")
			return nil
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
	}

	var cfg workspace.UserRCConfig

	if quick {
		// Quick mode: use defaults
		cfg = workspace.DefaultUserRCConfig()
	} else {
		// Interactive mode
		var err error
		cfg, err = runConfigUserInitWizard()
		if err != nil {
			return err
		}
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("✓ Created %s\n", path)
	fmt.Println()
	fmt.Println("Your user config:")
	printUserRCConfig(&cfg)

	return nil
}

func runConfigUserInitWizard() (workspace.UserRCConfig, error) {
	cfg := workspace.DefaultUserRCConfig()

	fmt.Println()
	fmt.Println("bc - User Configuration Setup")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println()

	// Nickname
	fmt.Printf("Your nickname [%s]: ", workspace.DefaultNickname)
	var input string
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		nickname, err := workspace.NormalizeNickname(input)
		if err != nil {
			fmt.Printf("⚠️  %s, using default\n", err)
		} else {
			cfg.User.Nickname = nickname
		}
	}

	// Default role
	fmt.Printf("Default role for new agents [engineer]: ")
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.Defaults.DefaultRole = input
	}

	// Auto-start root
	fmt.Printf("Auto-start root agent on 'bc up'? [Y/n]: ")
	if _, err := fmt.Scanln(&input); err == nil {
		input = strings.ToLower(strings.TrimSpace(input))
		cfg.Defaults.AutoStartRoot = input != "n" && input != "no"
	}

	// Preferred tools
	fmt.Printf("Preferred tools (comma-separated) [claude-code, gemini]: ")
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		tools := strings.Split(input, ",")
		for i := range tools {
			tools[i] = strings.TrimSpace(tools[i])
		}
		cfg.Tools.Preferred = tools
	}

	return cfg, nil
}

func runConfigUserShow(_ *cobra.Command, _ []string) error {
	cfg, err := workspace.LoadUserRCConfig()
	if err != nil {
		return err
	}

	if cfg == nil {
		path := workspace.UserRCConfigPath()
		fmt.Printf("No user config found at %s\n", path)
		fmt.Println("Run 'bc config user init' to create one.")
		return nil
	}

	fmt.Println("# User Config (~/.bcrc)")
	fmt.Println()
	printUserRCConfig(cfg)

	return nil
}

func runConfigUserPath(_ *cobra.Command, _ []string) error {
	path := workspace.UserRCConfigPath()
	exists := workspace.UserRCExists()

	fmt.Printf("User config: %s", path)
	if exists {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (not found)")
	}

	return nil
}

func printUserRCConfig(cfg *workspace.UserRCConfig) {
	fmt.Println("[user]")
	fmt.Printf("  nickname = %q\n", cfg.User.Nickname)
	fmt.Println()
	fmt.Println("[defaults]")
	fmt.Printf("  default_role = %q\n", cfg.Defaults.DefaultRole)
	fmt.Printf("  auto_start_root = %v\n", cfg.Defaults.AutoStartRoot)
	fmt.Println()
	fmt.Println("[tools]")
	fmt.Printf("  preferred = %v\n", cfg.Tools.Preferred)
}
