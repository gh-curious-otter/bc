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
	result := GrayText("test")
	if !strings.Contains(result, "test") {
		t.Error("GrayText should contain text")
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
}

func TestKeyValueList(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	KeyValueList(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	result := buf.String()

	if !strings.Contains(result, "key1") {
		t.Error("KeyValueList should contain key1")
	}
	if !strings.Contains(result, "value1") {
		t.Error("KeyValueList should contain value1")
	}
}

func TestFormatFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)
	SetColorEnabled(false)

	tests := []struct { //nolint:govet // test struct alignment not critical
		name     string
		fn       func(string, ...any)
		format   string
		args     []any
		contains string
	}{
		{"Successf", Successf, "done %s", []any{"task"}, "done task"},
		{"Errorf", Errorf, "failed %s", []any{"op"}, "failed op"},
		{"Warningf", Warningf, "warn %d", []any{42}, "warn 42"},
		{"Infof", Infof, "info %v", []any{true}, "info true"},
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

	Debug("debug %s", "message")
	result := buf.String()
	if !strings.Contains(result, "debug message") {
		t.Errorf("Debug should output message, got %q", result)
	}

	buf.Reset()
	Debugf("formatted %s", "debug")
	result = buf.String()
	if !strings.Contains(result, "formatted debug") {
		t.Errorf("Debugf should output formatted message, got %q", result)
	}
}

func TestPrintFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(nil)

	Print("test print")
	result := buf.String()
	if !strings.Contains(result, "test print") {
		t.Errorf("Print should output text, got %q", result)
	}

	buf.Reset()
	Println("test println")
	result = buf.String()
	if !strings.Contains(result, "test println") {
		t.Errorf("Println should output text, got %q", result)
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("Println should end with newline")
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

	table := NewTable("Name", "Age")
	table.AddRow("Alice", "30")
	table.AddRow("Bob", "25")
	table.Print()

	result := buf.String()

	// Should contain headers
	if !strings.Contains(result, "Name") {
		t.Error("Table.Print should output header 'Name'")
	}
	if !strings.Contains(result, "Age") {
		t.Error("Table.Print should output header 'Age'")
	}

	// Should contain data
	if !strings.Contains(result, "Alice") {
		t.Error("Table.Print should output row data 'Alice'")
	}
	if !strings.Contains(result, "30") {
		t.Error("Table.Print should output row data '30'")
	}
}
