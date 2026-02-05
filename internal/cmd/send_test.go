package cmd

import (
	"strings"
	"testing"
)

func TestSendRejectsEmptyMessage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"empty string", []string{"send", "worker-01", ""}},
		{"whitespace only", []string{"send", "worker-01", "   "}},
		{"tabs only", []string{"send", "worker-01", "\t\t"}},
		{"mixed whitespace", []string{"send", "worker-01", " \t \n "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCmd(tt.args...)
			if err == nil {
				t.Fatal("expected error for empty/whitespace message, got nil")
			}
			if !strings.Contains(err.Error(), "message cannot be empty") {
				t.Errorf("expected 'message cannot be empty' error, got: %v", err)
			}
		})
	}
}

func TestSendAcceptsValidMessage(t *testing.T) {
	// A valid message should pass the empty check but fail at workspace lookup
	// (since we're not in a workspace). This confirms the validation doesn't
	// reject valid messages.
	_, err := executeCmd("send", "worker-01", "hello world")
	if err == nil {
		t.Fatal("expected error (no workspace), got nil")
	}
	// Should fail at workspace lookup, NOT at message validation
	if strings.Contains(err.Error(), "message cannot be empty") {
		t.Error("valid message was incorrectly rejected as empty")
	}
}
