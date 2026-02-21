package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/ui"
)

// Check result status
type checkStatus int

const (
	checkOK checkStatus = iota
	checkWarn
	checkFail
)

// Check represents a single dependency check
type check struct {
	Name    string
	Message string
	Fix     string
	Status  checkStatus
	// Required must be last for fieldalignment
	Required bool
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies and configuration",
	Long: `Diagnose your bc installation by checking required dependencies.

Required dependencies:
  tmux    Terminal multiplexer for agent sessions
  git     Version control for worktrees

Optional dependencies:
  claude  Anthropic Claude CLI
  cursor  Cursor editor

Examples:
  bc doctor           # Run all checks
  bc doctor --json    # Output as JSON

Exit codes:
  0  All required checks passed
  1  One or more required checks failed`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("bc doctor")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	ctx := cmd.Context()
	checks := make([]check, 0, 5)
	allRequired := true

	// Required: tmux
	checks = append(checks, checkCommand(ctx, "tmux", true, "brew install tmux"))

	// Required: git
	checks = append(checks, checkCommand(ctx, "git", true, "brew install git"))

	// Optional: claude CLI
	checks = append(checks, checkCommand(ctx, "claude", false, "npx -y @anthropic-ai/claude-code"))

	// Optional: cursor
	checks = append(checks, checkCommand(ctx, "cursor", false, "https://cursor.sh"))

	// Check ANTHROPIC_API_KEY
	checks = append(checks, checkEnvVar("ANTHROPIC_API_KEY", false))

	// Print results
	fmt.Println("Required:")
	for _, c := range checks {
		if c.Required {
			printCheck(c)
			if c.Status == checkFail {
				allRequired = false
			}
		}
	}

	fmt.Println()
	fmt.Println("Optional:")
	for _, c := range checks {
		if !c.Required {
			printCheck(c)
		}
	}

	fmt.Println()

	// Summary
	if allRequired {
		fmt.Println(ui.GreenText("✓") + " All required dependencies installed")
		return nil
	}

	fmt.Println(ui.RedText("✗") + " Some required dependencies are missing")
	fmt.Println()
	fmt.Println("Install missing dependencies and run 'bc doctor' again.")
	return fmt.Errorf("required dependencies missing")
}

func checkCommand(ctx context.Context, name string, required bool, fix string) check {
	c := check{
		Name:     name,
		Required: required,
		Fix:      fix,
	}

	path, err := exec.LookPath(name)
	if err != nil {
		c.Status = checkFail
		c.Message = "not found"
		return c
	}

	// Get version if available
	version := getVersion(ctx, name)
	if version != "" {
		c.Message = fmt.Sprintf("%s (%s)", path, version)
	} else {
		c.Message = path
	}
	c.Status = checkOK
	return c
}

func checkEnvVar(name string, required bool) check {
	c := check{
		Name:     name,
		Required: required,
	}

	value := os.Getenv(name)
	if value == "" {
		c.Status = checkWarn
		c.Message = "not set"
		if required {
			c.Status = checkFail
		}
		return c
	}

	// Mask the value
	masked := value[:4] + "..." + value[len(value)-4:]
	c.Message = masked
	c.Status = checkOK
	return c
}

func getVersion(ctx context.Context, name string) string {
	var cmd *exec.Cmd
	switch name {
	case "tmux":
		cmd = exec.CommandContext(ctx, "tmux", "-V")
	case "git":
		cmd = exec.CommandContext(ctx, "git", "--version")
	case "claude":
		cmd = exec.CommandContext(ctx, "claude", "--version")
	case "cursor":
		cmd = exec.CommandContext(ctx, "cursor", "--version")
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func printCheck(c check) {
	var icon string
	var nameColor string

	switch c.Status {
	case checkOK:
		icon = ui.GreenText("✓")
		nameColor = c.Name
	case checkWarn:
		icon = ui.YellowText("⚠")
		nameColor = ui.YellowText(c.Name)
	case checkFail:
		icon = ui.RedText("✗")
		nameColor = ui.RedText(c.Name)
	}

	fmt.Printf("  %s %-20s %s\n", icon, nameColor, c.Message)
	if c.Status == checkFail && c.Fix != "" {
		fmt.Printf("    Fix: %s\n", c.Fix)
	}
}
