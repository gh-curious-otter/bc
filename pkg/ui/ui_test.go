package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestColor(t *testing.T) {
	// Enable colors for testing
	SetColorEnabled(true)

	tests := []struct {
		name     string
		text     string
		color    string
		contains string
	}{
		{"red text", "error", Red, "\033[31m"},
		{"green text", "success", Green, "\033[32m"},
		{"yellow text", "warning", Yellow, "\033[33m"},
		{"blue text", "info", Blue, "\033[34m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Color(tc.text, tc.color)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("Color(%q, %q) = %q, want contains %q", tc.text, tc.color, result, tc.contains)
			}
			if !strings.Contains(result, tc.text) {
				t.Errorf("Color(%q, %q) = %q, want contains %q", tc.text, tc.color, result, tc.text)
			}
			if !strings.HasSuffix(result, Reset) {
				t.Errorf("Color(%q, %q) = %q, want suffix %q", tc.text, tc.color, result, Reset)
			}
		})
	}
}

func TestColorDisabled(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	result := Color("test", Red)
	if result != "test" {
		t.Errorf("Color with colors disabled = %q, want %q", result, "test")
	}
}

func TestColorize(t *testing.T) {
	SetColorEnabled(true)

	result := Colorize("test", Bold, Red)
	if !strings.Contains(result, Bold) {
		t.Errorf("Colorize should contain Bold code")
	}
	if !strings.Contains(result, Red) {
		t.Errorf("Colorize should contain Red code")
	}
}

func TestStyleHelpers(t *testing.T) {
	SetColorEnabled(true)

	tests := []struct {
		name     string
		fn       func(string) string
		contains string
	}{
		{"BoldText", BoldText, Bold},
		{"DimText", DimText, Dim},
		{"RedText", RedText, Red},
		{"GreenText", GreenText, Green},
		{"YellowText", YellowText, Yellow},
		{"BlueText", BlueText, Blue},
		{"CyanText", CyanText, Cyan},
		{"MagentaText", MagentaText, Magenta},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn("test")
			if !strings.Contains(result, tc.contains) {
				t.Errorf("%s should contain %q", tc.name, tc.contains)
			}
		})
	}
}

func TestMessages(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	tests := []struct { //nolint:govet // test struct alignment not critical
		args     []any
		fn       func(string, ...any)
		name     string
		prefix   string
		format   string
		contains string
	}{
		{[]any{"task"}, Success, "Success", "✓", "done %s", "done task"},
		{[]any{"op"}, Error, "Error", "✗", "failed %s", "failed op"},
		{[]any{42}, Warning, "Warning", "!", "warn %d", "warn 42"},
		{[]any{true}, Info, "Info", "→", "info %v", "info true"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.fn(tc.format, tc.args...)
			result := buf.String()
			if !strings.Contains(result, tc.prefix) {
				t.Errorf("%s output should contain prefix %q, got %q", tc.name, tc.prefix, result)
			}
			if !strings.Contains(result, tc.contains) {
				t.Errorf("%s output should contain %q, got %q", tc.name, tc.contains, result)
			}
		})
	}
}

func TestHeader(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	Header("Test Header")
	result := buf.String()
	if !strings.Contains(result, "Test Header") {
		t.Errorf("Header should contain text, got %q", result)
	}
}

func TestDivider(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	Divider(5)
	result := buf.String()
	if !strings.Contains(result, "─────") {
		t.Errorf("Divider(5) should have 5 dashes, got %q", result)
	}
}

func TestTable(t *testing.T) {
	SetColorEnabled(false)

	table := NewTable("Name", "Value")
	table.AddRow("foo", "bar")
	table.AddRow("longer", "x")

	result := table.String()

	// Should contain headers
	if !strings.Contains(result, "Name") {
		t.Error("Table should contain Name header")
	}
	if !strings.Contains(result, "Value") {
		t.Error("Table should contain Value header")
	}

	// Should contain rows
	if !strings.Contains(result, "foo") {
		t.Error("Table should contain foo")
	}
	if !strings.Contains(result, "bar") {
		t.Error("Table should contain bar")
	}

	// Should contain separator
	if !strings.Contains(result, "──") {
		t.Error("Table should contain separator")
	}
}

func TestSimpleTable(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	SimpleTable("Name", "Alice", "Age", "30")
	result := buf.String()

	if !strings.Contains(result, "Name:") {
		t.Error("SimpleTable should contain Name:")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("SimpleTable should contain Alice")
	}
	if !strings.Contains(result, "Age:") {
		t.Error("SimpleTable should contain Age:")
	}
}

func TestList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	List("item1", "item2", "item3")
	result := buf.String()

	if !strings.Contains(result, "•") {
		t.Error("List should contain bullet")
	}
	if !strings.Contains(result, "item1") {
		t.Error("List should contain item1")
	}
	if strings.Count(result, "•") != 3 {
		t.Errorf("List should have 3 bullets, got %d", strings.Count(result, "•"))
	}
}

func TestNumberedList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	NumberedList("first", "second")
	result := buf.String()

	if !strings.Contains(result, "1.") {
		t.Error("NumberedList should contain 1.")
	}
	if !strings.Contains(result, "2.") {
		t.Error("NumberedList should contain 2.")
	}
}

func TestCheckList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	CheckList(
		ListItem{Text: "task1"},
		ListItem{Text: "task2", Detail: "done"},
	)
	result := buf.String()

	if !strings.Contains(result, "✓") {
		t.Error("CheckList should contain checkmark")
	}
	if !strings.Contains(result, "task1") {
		t.Error("CheckList should contain task1")
	}
	if !strings.Contains(result, "(done)") {
		t.Error("CheckList should contain detail")
	}
}

func TestColorEnabled(t *testing.T) {
	SetColorEnabled(true)
	if !ColorEnabled() {
		t.Error("ColorEnabled should return true")
	}

	SetColorEnabled(false)
	if ColorEnabled() {
		t.Error("ColorEnabled should return false")
	}
}

func TestGrayText(t *testing.T) {
	SetColorEnabled(true)
	result := GrayText("gray")
	if !strings.Contains(result, "gray") {
		t.Errorf("GrayText should contain text, got %q", result)
	}
	// Gray uses bright black (90) code
	if !strings.Contains(result, "\x1b[90m") {
		t.Errorf("GrayText should contain gray code, got %q", result)
	}
}

func TestIndentedList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	IndentedList(2, "item1", "item2")
	result := buf.String()

	if !strings.Contains(result, "item1") {
		t.Error("IndentedList should contain item1")
	}
	if !strings.Contains(result, "item2") {
		t.Error("IndentedList should contain item2")
	}
	// Should be indented with spaces
	if !strings.Contains(result, "  ") {
		t.Error("IndentedList should be indented")
	}
}

func TestKeyValueList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	KeyValueList(map[string]string{
		"Name": "Alice",
		"Age":  "30",
	})
	result := buf.String()

	if !strings.Contains(result, "Name") {
		t.Error("KeyValueList should contain Name")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("KeyValueList should contain Alice")
	}
	if !strings.Contains(result, "Age") {
		t.Error("KeyValueList should contain Age")
	}
	if !strings.Contains(result, "30") {
		t.Error("KeyValueList should contain 30")
	}
}

func TestFormatFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	tests := []struct { //nolint:govet // test struct alignment not critical
		fn       func(string, ...any)
		args     []any
		name     string
		format   string
		contains string
	}{
		{Successf, []any{"task"}, "Successf", "done %s", "done task"},
		{Errorf, []any{"op"}, "Errorf", "failed %s", "failed op"},
		{Warningf, []any{42}, "Warningf", "warn %d", "warn 42"},
		{Infof, []any{true}, "Infof", "info %v", "info true"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.fn(tc.format, tc.args...)
			result := buf.String()
			if !strings.Contains(result, tc.contains) {
				t.Errorf("%s output should contain %q, got %q", tc.name, tc.contains, result)
			}
		})
	}
}

func TestDebugFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	buf.Reset()
	Debug("debug message")
	result := buf.String()
	if !strings.Contains(result, "debug message") {
		t.Errorf("Debug should contain message, got %q", result)
	}

	buf.Reset()
	Debugf("debug %d", 123)
	result = buf.String()
	if !strings.Contains(result, "debug 123") {
		t.Errorf("Debugf should contain formatted message, got %q", result)
	}
}

func TestPrintFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	buf.Reset()
	Print("hello")
	result := buf.String()
	if result != "hello" {
		t.Errorf("Print should output exact text, got %q", result)
	}

	buf.Reset()
	Println("world")
	result = buf.String()
	if result != "world\n" {
		t.Errorf("Println should output text with newline, got %q", result)
	}
}

func TestBlankLine(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	BlankLine()
	result := buf.String()
	if result != "\n" {
		t.Errorf("BlankLine should output single newline, got %q", result)
	}
}

func TestTablePrint(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	table := NewTable("Col1", "Col2")
	table.AddRow("a", "b")
	table.Print()

	result := buf.String()
	if !strings.Contains(result, "Col1") {
		t.Error("Table.Print should output headers")
	}
	if !strings.Contains(result, "a") {
		t.Error("Table.Print should output row data")
	}
}
