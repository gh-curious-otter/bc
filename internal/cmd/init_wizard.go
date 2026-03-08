package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
	"github.com/rpuneet/bc/pkg/workspace"
)

// WizardPreset represents a preconfigured workspace setup.
type WizardPreset string

const (
	PresetSolo      WizardPreset = "solo"
	PresetSmallTeam WizardPreset = "small-team"
	PresetFullTeam  WizardPreset = "full-team"
	PresetCustom    WizardPreset = "custom"
)

// WizardState tracks the wizard's progress and collected data.
type WizardState struct {
	Dir       string
	Nickname  string
	Tool      string
	Preset    WizardPreset
	Channels  []string
	Roster    workspace.RosterConfig
	Step      int
	TotalStep int
}

// NewWizardState creates a new wizard state for the given directory.
func NewWizardState(dir string) *WizardState {
	return &WizardState{
		Step:      1,
		TotalStep: 4,
		Dir:       dir,
		Nickname:  workspace.DefaultNickname,
		Preset:    PresetSolo,
		Tool:      "claude",
		Channels:  []string{"general", "eng"},
	}
}

// RunWizard runs the interactive workspace initialization wizard.
func RunWizard(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Check for existing workspaces
	if isV2Workspace(absDir) {
		return fmt.Errorf("workspace already initialized in %s", absDir)
	}
	if isV1Workspace(absDir) {
		fmt.Fprintln(os.Stderr, "Warning: Existing v1 workspace detected.")
		fmt.Fprintln(os.Stderr, "Run 'bc init' after removing .bc/ directory to migrate.")
		return fmt.Errorf("cannot initialize: v1 workspace exists")
	}

	state := NewWizardState(absDir)
	reader := bufio.NewReader(os.Stdin)

	// Print welcome banner
	printWelcome()

	// Step 1: Workspace basics
	if err := wizardStepBasics(state, reader); err != nil {
		return err
	}

	// Step 2: Preset selection
	if err := wizardStepPreset(state, reader); err != nil {
		return err
	}

	// Step 3: Tool selection (only for custom preset)
	if state.Preset == PresetCustom {
		if err := wizardStepTools(state, reader); err != nil {
			return err
		}
	}

	// Step 4: Confirmation
	if err := wizardStepConfirm(state, reader); err != nil {
		return err
	}

	// Create workspace with wizard settings
	return createWorkspaceFromWizard(state)
}

// printWelcome prints the wizard welcome banner.
func printWelcome() {
	fmt.Println()
	fmt.Println("  " + ui.CyanText("Welcome to bc!"))
	fmt.Println("  " + ui.GrayText("AI Agent Orchestration CLI"))
	fmt.Println()
}

// printStepHeader prints the current step header.
func printStepHeader(step, total int, title string) {
	fmt.Printf("  [%d/%d] %s\n", step, total, ui.BoldText(title))
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Println()
}

// wizardStepBasics handles step 1: workspace basics (nickname).
func wizardStepBasics(state *WizardState, reader *bufio.Reader) error {
	printStepHeader(1, state.TotalStep, "Workspace Setup")

	// Prompt for nickname
	fmt.Printf("  Your nickname [%s]: ", workspace.DefaultNickname)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input != "" {
		nickname, err := workspace.NormalizeNickname(input)
		if err != nil {
			fmt.Printf("  %s Using default: %s\n", ui.YellowText("!"), workspace.DefaultNickname)
		} else {
			state.Nickname = nickname
			if !strings.HasPrefix(input, "@") {
				fmt.Printf("  %s Auto-corrected to %s\n", ui.GreenText("✓"), nickname)
			}
		}
	}

	fmt.Println()
	return nil
}

// wizardStepPreset handles step 2: preset selection.
func wizardStepPreset(state *WizardState, reader *bufio.Reader) error {
	printStepHeader(2, state.TotalStep, "Team Configuration")

	fmt.Println("  Select a configuration preset:")
	fmt.Println()
	fmt.Printf("    %s Solo Developer %s\n", ui.GreenText("[1]"), ui.GrayText("(recommended for starting out)"))
	fmt.Println("        Root agent only - ideal for personal projects")
	fmt.Println()
	fmt.Printf("    %s Small Team %s\n", ui.CyanText("[2]"), ui.GrayText("(for small teams)"))
	fmt.Println("        Root + 1 PM + 1 Manager + 2 Engineers")
	fmt.Println()
	fmt.Printf("    %s Full Team %s\n", ui.CyanText("[3]"), ui.GrayText("(for larger projects)"))
	fmt.Println("        Root + 1 PM + 1 Manager + 4 Engineers + 2 QA")
	fmt.Println()
	fmt.Printf("    %s Custom %s\n", ui.CyanText("[4]"), ui.GrayText("(configure everything)"))
	fmt.Println("        Choose your own roster and tools")
	fmt.Println()

	fmt.Print("  Enter choice [1]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)
	switch choice {
	case "", "1":
		state.Preset = PresetSolo
		state.Roster = workspace.RosterConfig{} // All zeros = root only
		fmt.Printf("  %s Solo Developer preset selected\n", ui.GreenText("✓"))
	case "2":
		state.Preset = PresetSmallTeam
		state.Roster = workspace.RosterConfig{
			ProductManager: 1,
			Manager:        1,
			Engineers:      2,
		}
		fmt.Printf("  %s Small Team preset selected\n", ui.GreenText("✓"))
	case "3":
		state.Preset = PresetFullTeam
		state.Roster = workspace.RosterConfig{
			ProductManager: 1,
			Manager:        1,
			Engineers:      4,
			TechLeads:      2,
			QA:             2,
		}
		fmt.Printf("  %s Full Team preset selected\n", ui.GreenText("✓"))
	case "4":
		state.Preset = PresetCustom
		if err := promptCustomRoster(state, reader); err != nil {
			return err
		}
	default:
		fmt.Printf("  %s Invalid choice, using Solo Developer\n", ui.YellowText("!"))
		state.Preset = PresetSolo
	}

	fmt.Println()
	return nil
}

// promptCustomRoster prompts for custom roster configuration.
func promptCustomRoster(state *WizardState, reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  Configure agent roster (enter 0-10 for each):")
	fmt.Println()

	state.Roster.ProductManager = promptInt(reader, "  Product Managers", 0)
	state.Roster.Manager = promptInt(reader, "  Managers", 0)
	state.Roster.Engineers = promptInt(reader, "  Engineers", 2)
	state.Roster.TechLeads = promptInt(reader, "  Tech Leads", 0)
	state.Roster.QA = promptInt(reader, "  QA Engineers", 0)

	fmt.Printf("  %s Custom roster configured\n", ui.GreenText("✓"))
	return nil
}

// promptInt prompts for an integer value with a default.
func promptInt(reader *bufio.Reader, label string, defaultVal int) int {
	fmt.Printf("%s [%d]: ", label, defaultVal)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(input)
	if err != nil || val < 0 || val > 10 {
		fmt.Printf("  %s Invalid value, using %d\n", ui.YellowText("!"), defaultVal)
		return defaultVal
	}

	return val
}

// wizardStepTools handles step 3: tool selection (custom preset only).
func wizardStepTools(state *WizardState, reader *bufio.Reader) error {
	printStepHeader(3, state.TotalStep, "AI Tool Selection")

	fmt.Println("  Select your default AI tool:")
	fmt.Println()
	fmt.Printf("    %s Claude %s\n", ui.CyanText("[1]"), ui.GrayText("(Anthropic Claude Code)"))
	fmt.Printf("    %s Gemini %s\n", ui.CyanText("[2]"), ui.GrayText("(Google Gemini CLI)"))
	fmt.Printf("    %s Cursor %s\n", ui.CyanText("[3]"), ui.GrayText("(Cursor AI Editor)"))
	fmt.Println()

	fmt.Print("  Enter choice [1]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)
	switch choice {
	case "", "1":
		state.Tool = "claude"
		fmt.Printf("  %s Claude selected\n", ui.GreenText("✓"))
	case "2":
		state.Tool = "gemini"
		fmt.Printf("  %s Gemini selected\n", ui.GreenText("✓"))
	case "3":
		state.Tool = "cursor"
		fmt.Printf("  %s Cursor selected\n", ui.GreenText("✓"))
	default:
		fmt.Printf("  %s Invalid choice, using Claude\n", ui.YellowText("!"))
		state.Tool = "claude"
	}

	fmt.Println()
	return nil
}

// wizardStepConfirm handles the final confirmation step.
func wizardStepConfirm(state *WizardState, reader *bufio.Reader) error {
	step := state.TotalStep
	if state.Preset != PresetCustom {
		step = 3 // Skip tool step for non-custom
	}
	printStepHeader(step, step, "Confirmation")

	fmt.Println("  Review your configuration:")
	fmt.Println()
	fmt.Printf("    Directory:  %s\n", state.Dir)
	fmt.Printf("    Nickname:   %s\n", state.Nickname)
	fmt.Printf("    Preset:     %s\n", presetLabel(state.Preset))
	fmt.Printf("    AI Tool:    %s\n", state.Tool)
	fmt.Println()
	fmt.Println("  Agents to create:")
	fmt.Println("    - root (always created)")
	if state.Roster.ProductManager > 0 {
		fmt.Printf("    - %d product manager(s)\n", state.Roster.ProductManager)
	}
	if state.Roster.Manager > 0 {
		fmt.Printf("    - %d manager(s)\n", state.Roster.Manager)
	}
	if state.Roster.Engineers > 0 {
		fmt.Printf("    - %d engineer(s)\n", state.Roster.Engineers)
	}
	if state.Roster.TechLeads > 0 {
		fmt.Printf("    - %d tech lead(s)\n", state.Roster.TechLeads)
	}
	if state.Roster.QA > 0 {
		fmt.Printf("    - %d QA engineer(s)\n", state.Roster.QA)
	}
	fmt.Println()

	fmt.Print("  Proceed? [Y/n]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.ToLower(strings.TrimSpace(input))
	if choice == "n" || choice == "no" {
		return fmt.Errorf("initialization canceled")
	}

	fmt.Println()
	return nil
}

// presetLabel returns a human-readable label for a preset.
func presetLabel(preset WizardPreset) string {
	switch preset {
	case PresetSolo:
		return "Solo Developer"
	case PresetSmallTeam:
		return "Small Team"
	case PresetFullTeam:
		return "Full Team"
	case PresetCustom:
		return "Custom"
	default:
		return string(preset)
	}
}

// createWorkspaceFromWizard creates a workspace using wizard settings.
func createWorkspaceFromWizard(state *WizardState) error {
	stateDir := filepath.Join(state.Dir, ".bc")

	// Create state directory
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create .bc directory: %w", err)
	}

	// Create agents directory
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Create config
	name := filepath.Base(state.Dir)
	cfg := workspace.DefaultConfig(name)
	cfg.User.Nickname = state.Nickname
	cfg.Roster = state.Roster
	cfg.Tools.Default = state.Tool
	cfg.Channels.Default = state.Channels

	configPath := workspace.ConfigPath(state.Dir)
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create roles directory and default root.md
	roleMgr := workspace.NewRoleManager(stateDir)
	_, err := roleMgr.EnsureDefaultRoot()
	if err != nil {
		return fmt.Errorf("failed to create role files: %w", err)
	}

	// Initialize channel database
	channelStore := channel.NewSQLiteStore(state.Dir)
	if openErr := channelStore.Open(); openErr != nil {
		return fmt.Errorf("failed to initialize channel database: %w", openErr)
	}
	_ = channelStore.Close()

	// Register in global registry
	reg, regErr := workspace.LoadRegistry()
	if regErr == nil {
		reg.Register(state.Dir, name)
		_ = reg.Save()
	}

	// Print success
	printWizardSuccess(state)
	return nil
}

// printWizardSuccess prints the success message after wizard completion.
func printWizardSuccess(state *WizardState) {
	fmt.Println("  " + ui.GreenText("✓") + " Workspace initialized!")
	fmt.Println()
	fmt.Println("  Created:")
	fmt.Println("    .bc/config.toml     # Workspace configuration")
	fmt.Println("    .bc/agents/         # Agent state directory")
	fmt.Println("    .bc/roles/          # Role definitions")
	fmt.Println("    .bc/roles/root.md   # Root agent role")
	fmt.Println("    .bc/channels.db     # Channel database")
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    bc          # Open the dashboard")
	fmt.Println("    bc up       # Start agents")
	fmt.Println("    bc status   # Check agent status")
	fmt.Println()
}

// InitWithPreset initializes a workspace with a specific preset (non-interactive).
func InitWithPreset(dir string, preset WizardPreset) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	log.Debug("init with preset", "dir", absDir, "preset", preset)

	// Check for existing workspaces
	if isV2Workspace(absDir) {
		return fmt.Errorf("workspace already initialized in %s", absDir)
	}
	if isV1Workspace(absDir) {
		return fmt.Errorf("cannot initialize: v1 workspace exists")
	}

	state := NewWizardState(absDir)
	state.Preset = preset

	// Apply preset roster
	switch preset {
	case PresetSolo:
		state.Roster = workspace.RosterConfig{}
	case PresetSmallTeam:
		state.Roster = workspace.RosterConfig{
			ProductManager: 1,
			Manager:        1,
			Engineers:      2,
		}
	case PresetFullTeam:
		state.Roster = workspace.RosterConfig{
			ProductManager: 1,
			Manager:        1,
			Engineers:      4,
			TechLeads:      2,
			QA:             2,
		}
	default:
		return fmt.Errorf("unknown preset: %s", preset)
	}

	return createWorkspaceFromWizard(state)
}
