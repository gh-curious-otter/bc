package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"

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

Example:
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

Example:
  bc config validate`,
	RunE: runConfigValidate,
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long: `Reset the workspace configuration to default values.

WARNING: This will overwrite your current config. Back up first if needed.

Example:
  bc config reset
  bc config reset --force         # Skip confirmation`,
	RunE: runConfigReset,
}

// Flags
var (
	configForce bool
)

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configResetCmd)

	configResetCmd.Flags().BoolVar(&configForce, "force", false, "Skip confirmation prompt")

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
		return fmt.Errorf("not in a bc workspace: %w", err)
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

	fmt.Println("[beads]")
	fmt.Printf("  enabled: %v\n", cfg.Beads.Enabled)
	fmt.Printf("  issues_dir: %s\n", cfg.Beads.IssuesDir)
	fmt.Println()

	fmt.Println("[channels]")
	fmt.Printf("  default: %v\n", cfg.Channels.Default)
	fmt.Println()

	fmt.Println("[roster]")
	fmt.Printf("  engineers: %d\n", cfg.Roster.Engineers)
	fmt.Printf("  tech_leads: %d\n", cfg.Roster.TechLeads)
	fmt.Printf("  qa: %d\n", cfg.Roster.QA)
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
