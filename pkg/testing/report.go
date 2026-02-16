// Package testing provides utilities for test result parsing, reporting, and issue creation.
package testing

import (
	"fmt"
	"strings"
	"time"
)

// TestReport represents a complete test report.
type TestReport struct {
	Timestamp    time.Time
	Failures     []TestFailure
	Packages     []PackageResult
	Duration     time.Duration
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
}

// PackageResult represents test results for a single package.
type PackageResult struct {
	Name     string
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
}

// Summary returns a one-line summary of the test report.
func (r *TestReport) Summary() string {
	if r.FailedTests == 0 {
		return fmt.Sprintf("✅ All %d tests passed in %s", r.TotalTests, r.Duration.Round(time.Millisecond))
	}
	return fmt.Sprintf("❌ %d/%d tests failed in %s", r.FailedTests, r.TotalTests, r.Duration.Round(time.Millisecond))
}

// IsSuccess returns true if all tests passed.
func (r *TestReport) IsSuccess() bool {
	return r.FailedTests == 0
}

// PassRate returns the percentage of passed tests.
func (r *TestReport) PassRate() float64 {
	if r.TotalTests == 0 {
		return 100.0
	}
	return float64(r.PassedTests) / float64(r.TotalTests) * 100
}

// FormatMarkdown generates a Markdown report.
func (r *TestReport) FormatMarkdown() string {
	var b strings.Builder

	// Header
	b.WriteString("# Test Report\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.Timestamp.Format(time.RFC3339)))

	// Summary
	b.WriteString("## Summary\n\n")
	if r.IsSuccess() {
		b.WriteString("✅ **All tests passed**\n\n")
	} else {
		b.WriteString("❌ **Tests failed**\n\n")
	}

	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Total Tests | %d |\n", r.TotalTests))
	b.WriteString(fmt.Sprintf("| Passed | %d |\n", r.PassedTests))
	b.WriteString(fmt.Sprintf("| Failed | %d |\n", r.FailedTests))
	b.WriteString(fmt.Sprintf("| Skipped | %d |\n", r.SkippedTests))
	b.WriteString(fmt.Sprintf("| Pass Rate | %.1f%% |\n", r.PassRate()))
	b.WriteString(fmt.Sprintf("| Duration | %s |\n", r.Duration.Round(time.Millisecond)))
	b.WriteString("\n")

	// Package breakdown
	if len(r.Packages) > 0 {
		b.WriteString("## Packages\n\n")
		b.WriteString("| Package | Passed | Failed | Skipped | Duration |\n")
		b.WriteString("|---------|--------|--------|---------|----------|\n")
		for _, pkg := range r.Packages {
			status := "✅"
			if pkg.Failed > 0 {
				status = "❌"
			}
			b.WriteString(fmt.Sprintf("| %s %s | %d | %d | %d | %s |\n",
				status, pkg.Name, pkg.Passed, pkg.Failed, pkg.Skipped,
				pkg.Duration.Round(time.Millisecond)))
		}
		b.WriteString("\n")
	}

	// Failures
	if len(r.Failures) > 0 {
		b.WriteString("## Failures\n\n")
		for i, f := range r.Failures {
			b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, f.TestName))
			b.WriteString(fmt.Sprintf("**Package:** `%s`\n\n", f.Package))
			b.WriteString(fmt.Sprintf("**Message:**\n```\n%s\n```\n\n", f.Message))
			if len(f.Output) > 0 && len(f.Output) <= 20 {
				b.WriteString("**Output:**\n```\n")
				for _, line := range f.Output {
					b.WriteString(line + "\n")
				}
				b.WriteString("```\n\n")
			}
		}
	}

	return b.String()
}

// FormatText generates a plain text report.
func (r *TestReport) FormatText() string {
	var b strings.Builder

	// Header
	b.WriteString("TEST REPORT\n")
	b.WriteString(strings.Repeat("=", 50) + "\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.Timestamp.Format(time.RFC3339)))

	// Summary
	b.WriteString(r.Summary() + "\n\n")

	b.WriteString(fmt.Sprintf("Total:   %d\n", r.TotalTests))
	b.WriteString(fmt.Sprintf("Passed:  %d\n", r.PassedTests))
	b.WriteString(fmt.Sprintf("Failed:  %d\n", r.FailedTests))
	b.WriteString(fmt.Sprintf("Skipped: %d\n", r.SkippedTests))
	b.WriteString(fmt.Sprintf("Rate:    %.1f%%\n", r.PassRate()))
	b.WriteString(fmt.Sprintf("Time:    %s\n\n", r.Duration.Round(time.Millisecond)))

	// Failures
	if len(r.Failures) > 0 {
		b.WriteString("FAILURES\n")
		b.WriteString(strings.Repeat("-", 50) + "\n")
		for i, f := range r.Failures {
			b.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, f.FullName))
			b.WriteString(fmt.Sprintf("   %s\n", f.Message))
		}
	}

	return b.String()
}

// FormatJSON generates a JSON report structure.
func (r *TestReport) FormatJSON() map[string]any {
	return map[string]any{
		"timestamp":     r.Timestamp.Format(time.RFC3339),
		"total_tests":   r.TotalTests,
		"passed_tests":  r.PassedTests,
		"failed_tests":  r.FailedTests,
		"skipped_tests": r.SkippedTests,
		"pass_rate":     r.PassRate(),
		"duration_ms":   r.Duration.Milliseconds(),
		"success":       r.IsSuccess(),
		"failures":      r.Failures,
	}
}

// NewReport creates a new empty test report.
func NewReport() *TestReport {
	return &TestReport{
		Timestamp: time.Now(),
		Failures:  []TestFailure{},
		Packages:  []PackageResult{},
	}
}

// AddFailure adds a test failure to the report.
func (r *TestReport) AddFailure(f TestFailure) {
	r.Failures = append(r.Failures, f)
	r.FailedTests++
	r.TotalTests++
}

// AddPass increments the passed test count.
func (r *TestReport) AddPass() {
	r.PassedTests++
	r.TotalTests++
}

// AddSkip increments the skipped test count.
func (r *TestReport) AddSkip() {
	r.SkippedTests++
	r.TotalTests++
}

// SetDuration sets the total test duration.
func (r *TestReport) SetDuration(d time.Duration) {
	r.Duration = d
}
