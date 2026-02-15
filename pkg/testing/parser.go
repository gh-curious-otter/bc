// Package testing provides utilities for test result parsing and issue creation.
package testing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TestEvent represents a single event from go test -json output.
type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"` // run, pause, cont, pass, fail, skip, output, bench
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// TestFailure represents a parsed test failure.
type TestFailure struct {
	Package   string
	TestName  string
	FullName  string
	Message   string
	Output    []string
	Timestamp string
}

// ParseTestJSON parses go test -json output from an io.Reader and returns list of failures.
func ParseTestJSON(reader io.Reader) ([]TestFailure, error) {
	var failures []TestFailure
	var outputLines []string

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Action == "output" && event.Output != "" {
			outputLines = append(outputLines, strings.TrimSuffix(event.Output, "\n"))
		}

		if event.Action == "fail" && event.Test != "" {
			failure := TestFailure{
				Package:   event.Package,
				TestName:  event.Test,
				FullName:  fmt.Sprintf("%s.%s", event.Package, event.Test),
				Message:   extractFailureMessage(outputLines),
				Output:    outputLines,
				Timestamp: event.Time,
			}
			failures = append(failures, failure)
			outputLines = []string{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return failures, nil
}

// ParseTestResults parses go test -json output and returns list of failures.
func ParseTestResults(jsonLines string) ([]TestFailure, error) {
	var failures []TestFailure
	var outputLines []string

	lines := strings.Split(jsonLines, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Action == "output" && event.Output != "" {
			outputLines = append(outputLines, strings.TrimSuffix(event.Output, "\n"))
		}

		if event.Action == "fail" && event.Test != "" {
			failure := TestFailure{
				Package:   event.Package,
				TestName:  event.Test,
				FullName:  fmt.Sprintf("%s.%s", event.Package, event.Test),
				Message:   extractFailureMessage(outputLines),
				Output:    outputLines,
				Timestamp: event.Time,
			}
			failures = append(failures, failure)
			outputLines = []string{}
		}
	}

	return failures, nil
}

func extractFailureMessage(lines []string) string {
	// First pass: look for Error or panic
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Error") || strings.Contains(line, "panic") {
			return line
		}
	}
	// Second pass: look for FAIL
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "FAIL") {
			return line
		}
	}
	// Fallback to last line or default
	if len(lines) > 0 {
		return lines[len(lines)-1]
	}
	return "test failed"
}

// FormatFailureForIssue formats a test failure into GitHub issue body.
func FormatFailureForIssue(failure TestFailure) string {
	var body strings.Builder
	body.WriteString(fmt.Sprintf("**Test:** `%s`\n\n", failure.FullName))
	body.WriteString(fmt.Sprintf("**Package:** `%s`\n\n", failure.Package))
	body.WriteString(fmt.Sprintf("**Time:** %s\n\n", failure.Timestamp))
	body.WriteString(fmt.Sprintf("**Message:**\n```\n%s\n```\n\n", failure.Message))

	if len(failure.Output) > 0 {
		body.WriteString("**Full Output:**\n```\n")
		for _, line := range failure.Output {
			if len(line) > 200 {
				body.WriteString(line[:200] + "...\n")
			} else {
				body.WriteString(line + "\n")
			}
		}
		body.WriteString("```\n")
	}

	body.WriteString("\n---\n*Created by automated testing demon*")
	return body.String()
}

// IssueTitle returns a GitHub issue title for the test failure.
func (f TestFailure) IssueTitle() string {
	return fmt.Sprintf("[TEST FAILURE] %s", f.TestName)
}

// FormatIssueBody returns a GitHub issue body for the test failure.
func (f TestFailure) FormatIssueBody() string {
	return FormatFailureForIssue(f)
}
