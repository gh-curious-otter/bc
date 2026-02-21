package testing

import (
	"strings"
	"testing"
	"time"
)

func TestNewReport(t *testing.T) {
	r := NewReport()
	if r == nil {
		t.Fatal("NewReport returned nil")
	}
	if r.TotalTests != 0 {
		t.Errorf("expected 0 total tests, got %d", r.TotalTests)
	}
	if r.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestReportAddFailure(t *testing.T) {
	r := NewReport()
	r.AddFailure(TestFailure{
		Package:  "github.com/test/pkg",
		TestName: "TestSomething",
		Message:  "assertion failed",
	})

	if r.TotalTests != 1 {
		t.Errorf("expected 1 total test, got %d", r.TotalTests)
	}
	if r.FailedTests != 1 {
		t.Errorf("expected 1 failed test, got %d", r.FailedTests)
	}
	if len(r.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(r.Failures))
	}
}

func TestReportAddPass(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.AddPass()
	r.AddPass()

	if r.TotalTests != 3 {
		t.Errorf("expected 3 total tests, got %d", r.TotalTests)
	}
	if r.PassedTests != 3 {
		t.Errorf("expected 3 passed tests, got %d", r.PassedTests)
	}
}

func TestReportAddSkip(t *testing.T) {
	r := NewReport()
	r.AddSkip()

	if r.TotalTests != 1 {
		t.Errorf("expected 1 total test, got %d", r.TotalTests)
	}
	if r.SkippedTests != 1 {
		t.Errorf("expected 1 skipped test, got %d", r.SkippedTests)
	}
}

func TestReportIsSuccess(t *testing.T) {
	tests := []struct {
		setup    func(*TestReport)
		name     string
		expected bool
	}{
		{
			name:     "empty report",
			setup:    func(r *TestReport) {},
			expected: true,
		},
		{
			name: "all passed",
			setup: func(r *TestReport) {
				r.AddPass()
				r.AddPass()
			},
			expected: true,
		},
		{
			name: "with failures",
			setup: func(r *TestReport) {
				r.AddPass()
				r.AddFailure(TestFailure{TestName: "TestFail"})
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReport()
			tt.setup(r)
			if r.IsSuccess() != tt.expected {
				t.Errorf("expected IsSuccess=%v, got %v", tt.expected, r.IsSuccess())
			}
		})
	}
}

func TestReportPassRate(t *testing.T) {
	tests := []struct {
		name     string
		passed   int
		failed   int
		expected float64
	}{
		{"empty", 0, 0, 100.0},
		{"all passed", 10, 0, 100.0},
		{"all failed", 0, 10, 0.0},
		{"half", 5, 5, 50.0},
		{"quarter failed", 3, 1, 75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReport()
			for i := 0; i < tt.passed; i++ {
				r.AddPass()
			}
			for i := 0; i < tt.failed; i++ {
				r.AddFailure(TestFailure{TestName: "TestFail"})
			}

			rate := r.PassRate()
			if rate != tt.expected {
				t.Errorf("expected %.1f%%, got %.1f%%", tt.expected, rate)
			}
		})
	}
}

func TestReportSummary(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*TestReport)
		contains string
	}{
		{
			name: "success",
			setup: func(r *TestReport) {
				r.AddPass()
				r.AddPass()
				r.SetDuration(100 * time.Millisecond)
			},
			contains: "✅",
		},
		{
			name: "failure",
			setup: func(r *TestReport) {
				r.AddPass()
				r.AddFailure(TestFailure{TestName: "TestFail"})
				r.SetDuration(100 * time.Millisecond)
			},
			contains: "❌",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReport()
			tt.setup(r)
			summary := r.Summary()
			if !strings.Contains(summary, tt.contains) {
				t.Errorf("expected summary to contain %q, got %q", tt.contains, summary)
			}
		})
	}
}

func TestReportFormatMarkdown(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.AddPass()
	r.AddFailure(TestFailure{
		Package:  "github.com/test/pkg",
		TestName: "TestFail",
		FullName: "github.com/test/pkg.TestFail",
		Message:  "assertion failed",
	})
	r.SetDuration(150 * time.Millisecond)

	md := r.FormatMarkdown()

	// Check required sections
	if !strings.Contains(md, "# Test Report") {
		t.Error("missing header")
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("missing summary section")
	}
	if !strings.Contains(md, "## Failures") {
		t.Error("missing failures section")
	}
	if !strings.Contains(md, "TestFail") {
		t.Error("missing failure test name")
	}
	if !strings.Contains(md, "66.7%") {
		t.Error("missing pass rate")
	}
}

func TestReportFormatText(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.SetDuration(100 * time.Millisecond)

	text := r.FormatText()

	if !strings.Contains(text, "TEST REPORT") {
		t.Error("missing header")
	}
	if !strings.Contains(text, "Total:") {
		t.Error("missing total line")
	}
}

func TestReportFormatJSON(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.AddFailure(TestFailure{TestName: "TestFail"})
	r.SetDuration(100 * time.Millisecond)

	data := r.FormatJSON()

	totalTests, ok := data["total_tests"].(int)
	if !ok || totalTests != 2 {
		t.Errorf("expected total_tests=2, got %v", data["total_tests"])
	}
	passedTests, ok := data["passed_tests"].(int)
	if !ok || passedTests != 1 {
		t.Errorf("expected passed_tests=1, got %v", data["passed_tests"])
	}
	failedTests, ok := data["failed_tests"].(int)
	if !ok || failedTests != 1 {
		t.Errorf("expected failed_tests=1, got %v", data["failed_tests"])
	}
	success, ok := data["success"].(bool)
	if !ok || success {
		t.Error("expected success=false")
	}
}

func TestReportSetDuration(t *testing.T) {
	r := NewReport()
	d := 5 * time.Second
	r.SetDuration(d)

	if r.Duration != d {
		t.Errorf("expected duration=%v, got %v", d, r.Duration)
	}
}

func TestReportFormatMarkdownWithPackages(t *testing.T) {
	r := NewReport()
	r.Packages = []PackageResult{
		{
			Name:     "github.com/test/pkg1",
			Passed:   5,
			Failed:   0,
			Skipped:  1,
			Duration: 100 * time.Millisecond,
		},
		{
			Name:     "github.com/test/pkg2",
			Passed:   3,
			Failed:   2,
			Skipped:  0,
			Duration: 200 * time.Millisecond,
		},
	}
	r.PassedTests = 8
	r.FailedTests = 2
	r.SkippedTests = 1
	r.TotalTests = 11
	r.SetDuration(300 * time.Millisecond)

	md := r.FormatMarkdown()

	if !strings.Contains(md, "## Packages") {
		t.Error("missing packages section")
	}
	if !strings.Contains(md, "github.com/test/pkg1") {
		t.Error("missing pkg1")
	}
	if !strings.Contains(md, "github.com/test/pkg2") {
		t.Error("missing pkg2")
	}
}

func TestReportFormatMarkdownAllPassed(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.AddPass()
	r.AddPass()
	r.SetDuration(50 * time.Millisecond)

	md := r.FormatMarkdown()

	if !strings.Contains(md, "✅ **All tests passed**") {
		t.Error("missing success message for all-passed report")
	}
	if strings.Contains(md, "## Failures") {
		t.Error("should not have failures section when all passed")
	}
}

func TestReportFormatMarkdownWithOutput(t *testing.T) {
	r := NewReport()
	r.AddFailure(TestFailure{
		Package:  "github.com/test/pkg",
		TestName: "TestWithOutput",
		FullName: "github.com/test/pkg.TestWithOutput",
		Message:  "assertion failed",
		Output:   []string{"line 1", "line 2", "line 3"},
	})
	r.SetDuration(100 * time.Millisecond)

	md := r.FormatMarkdown()

	if !strings.Contains(md, "**Output:**") {
		t.Error("missing output section for failure with output")
	}
	if !strings.Contains(md, "line 1") {
		t.Error("missing output line 1")
	}
}

func TestReportFormatTextWithFailures(t *testing.T) {
	r := NewReport()
	r.AddPass()
	r.AddFailure(TestFailure{
		Package:  "github.com/test/pkg",
		TestName: "TestFail",
		FullName: "github.com/test/pkg.TestFail",
		Message:  "expected true, got false",
	})
	r.SetDuration(100 * time.Millisecond)

	text := r.FormatText()

	if !strings.Contains(text, "FAILURES") {
		t.Error("missing failures section")
	}
	if !strings.Contains(text, "TestFail") {
		t.Error("missing failure test name")
	}
	if !strings.Contains(text, "expected true, got false") {
		t.Error("missing failure message")
	}
}
