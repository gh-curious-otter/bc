package testing

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseTestJSON(t *testing.T) {
	// Sample go test -json output with one failure
	testOutput := `{"Time":"2024-01-15T10:30:00Z","Action":"run","Package":"github.com/user/bc/pkg/agent","Test":"TestStart"}
{"Time":"2024-01-15T10:30:01Z","Action":"output","Package":"github.com/user/bc/pkg/agent","Test":"TestStart","Output":"=== RUN   TestStart\n"}
{"Time":"2024-01-15T10:30:02Z","Action":"output","Package":"github.com/user/bc/pkg/agent","Test":"TestStart","Output":"    agent_test.go:45: expected foo, got bar\n"}
{"Time":"2024-01-15T10:30:03Z","Action":"fail","Package":"github.com/user/bc/pkg/agent","Test":"TestStart","Elapsed":3.0}
`

	failures, err := ParseTestJSON(strings.NewReader(testOutput))
	if err != nil {
		t.Fatalf("ParseTestJSON failed: %v", err)
	}

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	f := failures[0]
	if f.Package != "github.com/user/bc/pkg/agent" {
		t.Errorf("package: got %q, want %q", f.Package, "github.com/user/bc/pkg/agent")
	}
	if f.Test != "TestStart" {
		t.Errorf("test: got %q, want %q", f.Test, "TestStart")
	}
	if f.Duration != 3.0 {
		t.Errorf("duration: got %v, want 3.0", f.Duration)
	}
}

func TestParseTestJSONMultipleFailures(t *testing.T) {
	testOutput := `{"Time":"2024-01-15T10:30:00Z","Action":"run","Package":"pkg/a","Test":"TestA"}
{"Time":"2024-01-15T10:30:01Z","Action":"output","Package":"pkg/a","Test":"TestA","Output":"failure A\n"}
{"Time":"2024-01-15T10:30:02Z","Action":"fail","Package":"pkg/a","Test":"TestA","Elapsed":2.0}
{"Time":"2024-01-15T10:30:03Z","Action":"run","Package":"pkg/b","Test":"TestB"}
{"Time":"2024-01-15T10:30:04Z","Action":"output","Package":"pkg/b","Test":"TestB","Output":"failure B\n"}
{"Time":"2024-01-15T10:30:05Z","Action":"fail","Package":"pkg/b","Test":"TestB","Elapsed":2.0}
`

	failures, err := ParseTestJSON(strings.NewReader(testOutput))
	if err != nil {
		t.Fatalf("ParseTestJSON failed: %v", err)
	}

	if len(failures) != 2 {
		t.Fatalf("expected 2 failures, got %d", len(failures))
	}

	if failures[0].Test != "TestA" || failures[1].Test != "TestB" {
		t.Error("unexpected test names")
	}
}

func TestParseTestJSONIgnoresMalformed(t *testing.T) {
	testOutput := `{"Time":"2024-01-15T10:30:00Z","Action":"run","Package":"pkg/a","Test":"TestA"}
this is not valid json
{"Time":"2024-01-15T10:30:01Z","Action":"output","Package":"pkg/a","Test":"TestA","Output":"error\n"}
{"Time":"2024-01-15T10:30:02Z","Action":"fail","Package":"pkg/a","Test":"TestA","Elapsed":1.0}
`

	failures, err := ParseTestJSON(strings.NewReader(testOutput))
	if err != nil {
		t.Fatalf("ParseTestJSON failed: %v", err)
	}

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure (malformed line ignored), got %d", len(failures))
	}
}

func TestExtractFileLocation(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantFile  string
		wantLine  int
		wantFound bool
	}{
		{
			name:      "simple .go:line format",
			output:    "    agent_test.go:45: expected foo",
			wantFile:  "agent_test.go",
			wantLine:  45,
			wantFound: true,
		},
		{
			name:      "path with directories",
			output:    "    pkg/agent/agent_test.go:123: error occurred",
			wantFile:  "pkg/agent/agent_test.go",
			wantLine:  123,
			wantFound: true,
		},
		{
			name:      "no file location",
			output:    "some error message without file",
			wantFound: false,
		},
		{
			name:      "multiple go files in output",
			output:    "pkg/a.go:10: first error\n    pkg/b.go:20: second error",
			wantFile:  "pkg/a.go",
			wantLine:  10,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, line, found := extractFileLocation(tt.output)
			if found != tt.wantFound {
				t.Errorf("found: got %v, want %v", found, tt.wantFound)
			}
			if found {
				if file != tt.wantFile {
					t.Errorf("file: got %q, want %q", file, tt.wantFile)
				}
				if line != tt.wantLine {
					t.Errorf("line: got %d, want %d", line, tt.wantLine)
				}
			}
		})
	}
}

func TestFormatIssueBody(t *testing.T) {
	f := &TestFailure{
		Package:   "github.com/user/bc/pkg/agent",
		Test:      "TestStart",
		File:      "pkg/agent/agent.go",
		Line:      42,
		Duration:  2.5,
		Timestamp: "2024-01-15T10:30:00Z",
		Output:    []string{"error: connection failed\n", "stack trace here\n"},
	}

	body := f.FormatIssueBody()

	// Verify key sections exist
	if !strings.Contains(body, "## Test Failure") {
		t.Error("missing Test Failure header")
	}
	if !strings.Contains(body, "github.com/user/bc/pkg/agent") {
		t.Error("missing package name")
	}
	if !strings.Contains(body, "TestStart") {
		t.Error("missing test name")
	}
	if !strings.Contains(body, "2.500s") {
		t.Error("missing duration")
	}
	if !strings.Contains(body, "pkg/agent/agent.go:42") {
		t.Error("missing file location")
	}
	if !strings.Contains(body, "error: connection failed") {
		t.Error("missing error output")
	}
}

func TestIssueTitle(t *testing.T) {
	tests := []struct {
		name      string
		failure   *TestFailure
		wantTitle string
	}{
		{
			name: "standard test",
			failure: &TestFailure{
				Package: "github.com/user/bc/pkg/agent",
				Test:    "TestStart",
			},
			wantTitle: "[Test Failure] TestStart - agent",
		},
		{
			name: "test with simple package",
			failure: &TestFailure{
				Package: "mypackage",
				Test:    "TestFunction",
			},
			wantTitle: "[Test Failure] TestFunction - mypackage",
		},
		{
			name: "empty test name",
			failure: &TestFailure{
				Package: "pkg/sub",
				Test:    "",
			},
			wantTitle: "[Test Failure] package test - sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := tt.failure.IssueTitle()
			if title != tt.wantTitle {
				t.Errorf("got %q, want %q", title, tt.wantTitle)
			}
		})
	}
}

func TestPackageShortName(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		expected string
	}{
		{
			name:     "nested package",
			pkg:      "github.com/user/bc/pkg/agent",
			expected: "agent",
		},
		{
			name:     "single component",
			pkg:      "mypackage",
			expected: "mypackage",
		},
		{
			name:     "two components",
			pkg:      "pkg/subpkg",
			expected: "subpkg",
		},
		{
			name:     "empty package",
			pkg:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packageShortName(tt.pkg)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseTestJSONWithFileLocations(t *testing.T) {
	testOutput := `{"Time":"2024-01-15T10:30:00Z","Action":"run","Package":"pkg/agent","Test":"TestError"}
{"Time":"2024-01-15T10:30:01Z","Action":"output","Package":"pkg/agent","Test":"TestError","Output":"=== RUN   TestError\n"}
{"Time":"2024-01-15T10:30:02Z","Action":"output","Package":"pkg/agent","Test":"TestError","Output":"    pkg/agent/agent_test.go:87: assertion failed\n"}
{"Time":"2024-01-15T10:30:03Z","Action":"fail","Package":"pkg/agent","Test":"TestError","Elapsed":1.5}
`

	failures, err := ParseTestJSON(strings.NewReader(testOutput))
	if err != nil {
		t.Fatalf("ParseTestJSON failed: %v", err)
	}

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	f := failures[0]
	if f.File != "pkg/agent/agent_test.go" {
		t.Errorf("file: got %q, want %q", f.File, "pkg/agent/agent_test.go")
	}
	if f.Line != 87 {
		t.Errorf("line: got %d, want 87", f.Line)
	}
}

func BenchmarkParseTestJSON(b *testing.B) {
	testOutput := `{"Time":"2024-01-15T10:30:00Z","Action":"run","Package":"pkg/a","Test":"TestA"}
{"Time":"2024-01-15T10:30:01Z","Action":"output","Package":"pkg/a","Test":"TestA","Output":"error\n"}
{"Time":"2024-01-15T10:30:02Z","Action":"fail","Package":"pkg/a","Test":"TestA","Elapsed":1.0}
`

	buf := &bytes.Buffer{}

	// Write test output multiple times to benchmark
	for i := 0; i < 100; i++ {
		buf.WriteString(testOutput)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		for j := 0; j < 100; j++ {
			buf.WriteString(testOutput)
		}
		_ = bytes.NewReader(buf.Bytes())            // Reset reader for parsing
		ParseTestJSON(bytes.NewReader(buf.Bytes())) //nolint:errcheck
	}
}
