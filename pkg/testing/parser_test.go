package testing

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseTestResults(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFail int
		wantPass int
	}{
		{
			name: "parse single failure",
			input: `{"Time":"2026-02-15T10:00:00Z","Action":"run","Package":"github.com/rpuneet/bc/pkg/cost","Test":"TestParse"}
{"Time":"2026-02-15T10:00:00Z","Action":"output","Output":"--- FAIL: TestParse\n"}
{"Time":"2026-02-15T10:00:01Z","Action":"fail","Package":"github.com/rpuneet/bc/pkg/cost","Test":"TestParse","Elapsed":1.234}`,
			wantFail: 1,
			wantPass: 0,
		},
		{
			name: "parse multiple failures",
			input: `{"Time":"2026-02-15T10:00:00Z","Action":"run","Package":"github.com/rpuneet/bc/pkg/cost","Test":"Test1"}
{"Time":"2026-02-15T10:00:00Z","Action":"output","Output":"Error: expected 1, got 2\n"}
{"Time":"2026-02-15T10:00:01Z","Action":"fail","Package":"github.com/rpuneet/bc/pkg/cost","Test":"Test1","Elapsed":1.0}
{"Time":"2026-02-15T10:00:01Z","Action":"run","Package":"github.com/rpuneet/bc/pkg/cost","Test":"Test2"}
{"Time":"2026-02-15T10:00:01Z","Action":"output","Output":"panic: nil pointer\n"}
{"Time":"2026-02-15T10:00:02Z","Action":"fail","Package":"github.com/rpuneet/bc/pkg/cost","Test":"Test2","Elapsed":1.0}`,
			wantFail: 2,
			wantPass: 0,
		},
		{
			name: "empty output",
			input: "",
			wantFail: 0,
			wantPass: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failures, err := ParseTestResults(tt.input)
			if err != nil {
				t.Fatalf("ParseTestResults() error = %v", err)
			}

			if len(failures) != tt.wantFail {
				t.Errorf("got %d failures, want %d", len(failures), tt.wantFail)
			}

			for _, f := range failures {
				if f.FullName == "" {
					t.Error("FullName is empty")
				}
				if f.Package == "" {
					t.Error("Package is empty")
				}
				if f.TestName == "" {
					t.Error("TestName is empty")
				}
			}
		})
	}
}

func TestExtractFailureMessage(t *testing.T) {
	tests := []struct {
		name string
		lines []string
		want string
	}{
		{
			name: "error line",
			lines: []string{"--- FAIL: TestParse", "Error: expected 1, got 2"},
			want: "Error: expected 1, got 2",
		},
		{
			name: "panic line",
			lines: []string{"output", "panic: nil pointer"},
			want: "panic: nil pointer",
		},
		{
			name: "last line",
			lines: []string{"test failed"},
			want: "test failed",
		},
		{
			name: "empty",
			lines: []string{},
			want: "test failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFailureMessage(tt.lines)
			if !strings.Contains(got, strings.TrimSpace(tt.want)) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFailureForIssue(t *testing.T) {
	failure := TestFailure{
		Package:   "github.com/rpuneet/bc/pkg/cost",
		TestName:  "TestParse",
		FullName:  "github.com/rpuneet/bc/pkg/cost.TestParse",
		Message:   "Error: expected 1, got 2",
		Output:    []string{"line 1", "line 2"},
		Timestamp: "2026-02-15T10:00:00Z",
	}

	body := FormatFailureForIssue(failure)

	required := []string{
		"TestParse",
		"github.com/rpuneet/bc/pkg/cost",
		"Error: expected 1, got 2",
		"line 1",
		"line 2",
		"automated testing demon",
	}

	for _, req := range required {
		if !strings.Contains(body, req) {
			t.Errorf("FormatFailureForIssue missing %q", req)
		}
	}
}

func TestParseTestJSON(t *testing.T) {
	input := `{"Time":"2026-02-15T10:00:00Z","Action":"run","Package":"github.com/rpuneet/bc/pkg/cost","Test":"TestParse"}
{"Time":"2026-02-15T10:00:00Z","Action":"output","Output":"--- FAIL: TestParse\n"}
{"Time":"2026-02-15T10:00:01Z","Action":"fail","Package":"github.com/rpuneet/bc/pkg/cost","Test":"TestParse","Elapsed":1.234}
`

	reader := bytes.NewReader([]byte(input))
	failures, err := ParseTestJSON(reader)
	if err != nil {
		t.Fatalf("ParseTestJSON() error = %v", err)
	}

	if len(failures) != 1 {
		t.Errorf("got %d failures, want 1", len(failures))
		return
	}

	f := failures[0]
	if f.FullName != "github.com/rpuneet/bc/pkg/cost.TestParse" {
		t.Errorf("FullName = %q, want github.com/rpuneet/bc/pkg/cost.TestParse", f.FullName)
	}

	if !strings.Contains(f.IssueTitle(), "TestParse") {
		t.Errorf("IssueTitle() missing test name: %q", f.IssueTitle())
	}

	body := f.FormatIssueBody()
	if !strings.Contains(body, "TestParse") {
		t.Errorf("FormatIssueBody() missing test name")
	}
}
