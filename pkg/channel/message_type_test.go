package channel

import (
	"strings"
	"testing"
)

func TestAllMessageTypes(t *testing.T) {
	types := AllMessageTypes()
	if len(types) != 6 {
		t.Errorf("expected 6 message types, got %d", len(types))
	}

	// Verify all expected types are present
	expected := map[MessageType]bool{
		TypeText:     true,
		TypeTask:     true,
		TypeReview:   true,
		TypeApproval: true,
		TypeMerge:    true,
		TypeStatus:   true,
	}
	for _, typ := range types {
		if !expected[typ] {
			t.Errorf("unexpected type: %s", typ)
		}
		delete(expected, typ)
	}
	if len(expected) > 0 {
		t.Errorf("missing types: %v", expected)
	}
}

func TestValidMessageTypes(t *testing.T) {
	result := ValidMessageTypes()
	if result == "" {
		t.Error("ValidMessageTypes returned empty string")
	}
	// Should contain all type names
	for _, typ := range AllMessageTypes() {
		if !strings.Contains(result, string(typ)) {
			t.Errorf("ValidMessageTypes missing %s", typ)
		}
	}
}

func TestIsValidMessageType(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"text", true},
		{"task", true},
		{"review", true},
		{"approval", true},
		{"merge", true},
		{"status", true},
		{"TEXT", true},   // case insensitive
		{"TASK", true},   // case insensitive
		{"Review", true}, // case insensitive
		{"invalid", false},
		{"", false},
		{"chat", false},
		{"message", false},
	}

	for _, tt := range tests {
		result := IsValidMessageType(tt.input)
		if result != tt.valid {
			t.Errorf("IsValidMessageType(%q) = %v, want %v", tt.input, result, tt.valid)
		}
	}
}

func TestParseMessageType(t *testing.T) {
	tests := []struct {
		input    string
		expected MessageType
		wantErr  bool
	}{
		{"", TypeText, false},         // empty defaults to text
		{"text", TypeText, false},     // explicit text
		{"task", TypeTask, false},     // task
		{"review", TypeReview, false}, // review
		{"approval", TypeApproval, false},
		{"merge", TypeMerge, false},
		{"status", TypeStatus, false},
		{"TASK", TypeTask, false},     // case insensitive
		{"Review", TypeReview, false}, // case insensitive
		{"invalid", "", true},         // invalid type
		{"chat", "", true},            // not a valid type
	}

	for _, tt := range tests {
		result, err := ParseMessageType(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseMessageType(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseMessageType(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseMessageType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	}
}

func TestMessageType_Emoji(t *testing.T) {
	tests := []struct {
		typ   MessageType
		emoji string
	}{
		{TypeText, "💬"},
		{TypeTask, "📋"},
		{TypeReview, "👀"},
		{TypeApproval, "✅"},
		{TypeMerge, "🔀"},
		{TypeStatus, "📊"},
	}

	for _, tt := range tests {
		result := tt.typ.Emoji()
		if result != tt.emoji {
			t.Errorf("%s.Emoji() = %q, want %q", tt.typ, result, tt.emoji)
		}
	}
}

func TestMessageType_Description(t *testing.T) {
	for _, typ := range AllMessageTypes() {
		desc := typ.Description()
		if desc == "" {
			t.Errorf("%s.Description() returned empty string", typ)
		}
		if desc == "Unknown type" {
			t.Errorf("%s.Description() returned 'Unknown type'", typ)
		}
	}
}

func TestMessageType_IsWorkItem(t *testing.T) {
	workItems := map[MessageType]bool{
		TypeTask:   true,
		TypeReview: true,
		TypeMerge:  true,
	}

	for _, typ := range AllMessageTypes() {
		result := typ.IsWorkItem()
		expected := workItems[typ]
		if result != expected {
			t.Errorf("%s.IsWorkItem() = %v, want %v", typ, result, expected)
		}
	}
}

func TestMessageType_TargetRole(t *testing.T) {
	tests := []struct {
		typ  MessageType
		role string
	}{
		{TypeTask, "engineer"},
		{TypeReview, "tech-lead"},
		{TypeApproval, "manager"},
		{TypeMerge, "manager"},
		{TypeText, ""},
		{TypeStatus, ""},
	}

	for _, tt := range tests {
		result := tt.typ.TargetRole()
		if result != tt.role {
			t.Errorf("%s.TargetRole() = %q, want %q", tt.typ, result, tt.role)
		}
	}
}

func TestNewTypedMessage(t *testing.T) {
	msg := NewTypedMessage("Hello world", TypeTask, "engineer-01")

	if msg.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello world")
	}
	if msg.Type != TypeTask {
		t.Errorf("Type = %q, want %q", msg.Type, TypeTask)
	}
	if msg.Sender != "engineer-01" {
		t.Errorf("Sender = %q, want %q", msg.Sender, "engineer-01")
	}
	if msg.Metadata != nil {
		t.Errorf("Metadata should be nil initially")
	}
}

func TestTypedMessage_WithMetadata(t *testing.T) {
	msg := NewTypedMessage("PR ready", TypeReview, "engineer-01").
		WithMetadata("pr", "123").
		WithMetadata("branch", "feature/test")

	if msg.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if msg.Metadata["pr"] != "123" {
		t.Errorf("Metadata[pr] = %q, want %q", msg.Metadata["pr"], "123")
	}
	if msg.Metadata["branch"] != "feature/test" {
		t.Errorf("Metadata[branch] = %q, want %q", msg.Metadata["branch"], "feature/test")
	}
}

func TestTypedMessage_FormatForDisplay(t *testing.T) {
	msg := NewTypedMessage("Please review PR #123", TypeReview, "engineer-01")
	result := msg.FormatForDisplay()

	// Should contain emoji and type
	if !strings.Contains(result, "👀") {
		t.Error("FormatForDisplay should contain review emoji")
	}
	if !strings.Contains(result, "review") {
		t.Error("FormatForDisplay should contain type name")
	}
	if !strings.Contains(result, "Please review PR #123") {
		t.Error("FormatForDisplay should contain message content")
	}
}

func TestInferMessageType(t *testing.T) {
	tests := []struct {
		content  string
		expected MessageType
	}{
		// Review patterns
		{"Please review PR #123", TypeReview},
		{"PR #45 is ready for review", TypeReview},
		{"@tech-lead-01 please review this", TypeReview},

		// Approval patterns
		{"LGTM, approved!", TypeApproval},
		{"Looks good to me", TypeApproval},
		{"PR #123 approved", TypeApproval},

		// Merge patterns
		{"PR #123 merged to main", TypeMerge},
		{"Ready to merge", TypeMerge},
		{"Just merged the feature branch", TypeMerge},

		// Task patterns
		{"@engineer-01 please implement the login feature", TypeTask},
		{"@qa-01 please fix the failing tests", TypeTask},
		{"@engineer-02 add unit tests for auth module", TypeTask},

		// Status patterns
		{"status: working on auth feature", TypeStatus},
		{"Running bc agent report done", TypeStatus},

		// Default to text
		{"Hello everyone!", TypeText},
		{"Great work on the project", TypeText},
		{"", TypeText},
	}

	for _, tt := range tests {
		result := InferMessageType(tt.content)
		if result != tt.expected {
			t.Errorf("InferMessageType(%q) = %q, want %q", tt.content, result, tt.expected)
		}
	}
}
