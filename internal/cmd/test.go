package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

// TestResult holds the result of a test run
type TestResult struct {
	Package string  `json:"package"`
	Test    string  `json:"test,omitempty"`
	Action  string  `json:"action"`
	Output  string  `json:"output,omitempty"`
	Time    string  `json:"time,omitempty"`
	Elapsed float64 `json:"elapsed,omitempty"`
}

// TestSummary holds aggregated test results
//
//nolint:govet // fieldalignment: JSON ordering preferred over memory layout
type TestSummary struct {
	TotalTests   int           `json:"total_tests"`
	PassedTests  int           `json:"passed_tests"`
	FailedTests  int           `json:"failed_tests"`
	SkippedTests int           `json:"skipped_tests"`
	Duration     time.Duration `json:"duration"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Failures     []string      `json:"failures,omitempty"`
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run bc tests",
	Long: `Run bc tests and generate reports.

The test command provides comprehensive testing capabilities:
- Run Go tests with race detector
- Run TUI tests with Bun
- Generate test reports and summaries

Examples:
  bc test run              # Run all Go tests
  bc test run --verbose    # Run tests with verbose output
  bc test tui              # Run TUI tests
  bc test report           # Generate test summary report`,
}

var testRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Go tests",
	Long: `Run Go tests with race detector.

Examples:
  bc test run                     # Run all tests
  bc test run --package ./pkg/... # Run specific package tests
  bc test run --verbose           # Verbose output
  bc test run --coverage          # Generate coverage report`,
	RunE: runTestRun,
}

var testTuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Run TUI tests",
	Long: `Run TUI tests using Bun test runner.

Examples:
  bc test tui           # Run all TUI tests
  bc test tui --watch   # Run in watch mode`,
	RunE: runTestTui,
}

var testReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate test report",
	Long: `Generate a summary report of the last test run.

Examples:
  bc test report         # Show last test summary
  bc test report --json  # Output as JSON`,
	RunE: runTestReport,
}

var (
	testPackage  string
	testVerbose  bool
	testCoverage bool
	testWatch    bool
)

func init() {
	// test run flags
	testRunCmd.Flags().StringVar(&testPackage, "package", "./...", "Package pattern to test")
	testRunCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")
	testRunCmd.Flags().BoolVar(&testCoverage, "coverage", false, "Generate coverage report")

	// test tui flags
	testTuiCmd.Flags().BoolVar(&testWatch, "watch", false, "Run in watch mode")

	// Add subcommands
	testCmd.AddCommand(testRunCmd)
	testCmd.AddCommand(testTuiCmd)
	testCmd.AddCommand(testReportCmd)

	rootCmd.AddCommand(testCmd)
}

func runTestRun(cmd *cobra.Command, args []string) error {
	log.Debug("test run command started", "package", testPackage, "verbose", testVerbose)

	// Find workspace root (where go.mod is)
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	fmt.Println("Running Go tests...")
	fmt.Println()

	// Build go test command
	goArgs := []string{"test", "-race"}
	if testVerbose {
		goArgs = append(goArgs, "-v")
	}
	if testCoverage {
		goArgs = append(goArgs, "-coverprofile=coverage.out")
	}
	goArgs = append(goArgs, testPackage)

	startTime := time.Now()

	// Run go test with context
	ctx := context.Background()
	goCmd := exec.CommandContext(ctx, "go", goArgs...)
	goCmd.Dir = ws.RootDir
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr

	err = goCmd.Run()
	duration := time.Since(startTime)

	fmt.Println()
	if err != nil {
		fmt.Printf("❌ Tests failed after %s\n", duration.Round(time.Millisecond))
		return fmt.Errorf("tests failed: %w", err)
	}

	fmt.Printf("✅ Tests passed in %s\n", duration.Round(time.Millisecond))

	if testCoverage {
		fmt.Println()
		fmt.Println("Coverage report saved to coverage.out")
		fmt.Println("View with: go tool cover -html=coverage.out")
	}

	return nil
}

func runTestTui(cmd *cobra.Command, args []string) error {
	log.Debug("test tui command started", "watch", testWatch)

	// Find workspace root
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	tuiDir := filepath.Join(ws.RootDir, "tui")
	if _, statErr := os.Stat(tuiDir); os.IsNotExist(statErr) {
		return fmt.Errorf("TUI directory not found at %s", tuiDir)
	}

	fmt.Println("Running TUI tests...")
	fmt.Println()

	// Build bun test command
	bunArgs := []string{"test"}
	if testWatch {
		bunArgs = append(bunArgs, "--watch")
	}

	startTime := time.Now()

	// Run bun test with context
	ctx := context.Background()
	bunCmd := exec.CommandContext(ctx, "bun", bunArgs...)
	bunCmd.Dir = tuiDir
	bunCmd.Stdout = os.Stdout
	bunCmd.Stderr = os.Stderr

	err = bunCmd.Run()
	duration := time.Since(startTime)

	fmt.Println()
	if err != nil {
		fmt.Printf("❌ TUI tests failed after %s\n", duration.Round(time.Millisecond))
		return fmt.Errorf("TUI tests failed: %w", err)
	}

	fmt.Printf("✅ TUI tests passed in %s\n", duration.Round(time.Millisecond))
	return nil
}

func runTestReport(cmd *cobra.Command, args []string) error {
	log.Debug("test report command started")

	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Find workspace root
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Run go test with JSON output to capture results
	goArgs := []string{"test", "-json", "-race", "./..."}

	ctx := context.Background()
	goCmd := exec.CommandContext(ctx, "go", goArgs...)
	goCmd.Dir = ws.RootDir

	output, err := goCmd.Output()
	if err != nil {
		// Tests may fail but we still want the report
		log.Debug("tests had failures", "error", err)
	}

	// Parse JSON output
	summary := TestSummary{
		StartTime: time.Now(),
	}
	var failures []string

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var result TestResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		switch result.Action {
		case "pass":
			if result.Test != "" {
				summary.PassedTests++
				summary.TotalTests++
			}
		case "fail":
			if result.Test != "" {
				summary.FailedTests++
				summary.TotalTests++
				failures = append(failures, fmt.Sprintf("%s.%s", result.Package, result.Test))
			}
		case "skip":
			if result.Test != "" {
				summary.SkippedTests++
				summary.TotalTests++
			}
		}
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	summary.Failures = failures

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}

	// Print human-readable report
	fmt.Println("Test Report")
	fmt.Println("===========")
	fmt.Println()
	fmt.Printf("Total:   %d tests\n", summary.TotalTests)
	fmt.Printf("Passed:  %d\n", summary.PassedTests)
	fmt.Printf("Failed:  %d\n", summary.FailedTests)
	fmt.Printf("Skipped: %d\n", summary.SkippedTests)
	fmt.Printf("Duration: %s\n", summary.Duration.Round(time.Millisecond))

	if len(failures) > 0 {
		fmt.Println()
		fmt.Println("Failed Tests:")
		for _, f := range failures {
			fmt.Printf("  ❌ %s\n", f)
		}
	}

	return nil
}
