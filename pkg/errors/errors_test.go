package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrAlreadyExists", ErrAlreadyExists, "already exists"},
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
		{"ErrPermission", ErrPermission, "permission denied"},
		{"ErrTimeout", ErrTimeout, "operation timeout"},
		{"ErrNotInWorkspace", ErrNotInWorkspace, "not in a bc workspace"},
		{"ErrInvalidState", ErrInvalidState, "invalid state"},
		{"ErrCanceled", ErrCanceled, "operation canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgentError(t *testing.T) {
	t.Run("with agent name", func(t *testing.T) {
		err := &AgentError{Agent: "eng-01", Op: "start", Err: ErrTimeout}
		want := `agent "eng-01": start: operation timeout`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without agent name", func(t *testing.T) {
		err := &AgentError{Op: "list", Err: ErrPermission}
		want := "agent: list: permission denied"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := &AgentError{Agent: "eng-01", Op: "start", Err: ErrNotFound}
		if !errors.Is(err, ErrNotFound) {
			t.Error("errors.Is should match ErrNotFound")
		}
	})

	t.Run("as", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", &AgentError{Agent: "eng-01", Op: "start", Err: ErrNotFound})
		var agentErr *AgentError
		if !errors.As(err, &agentErr) {
			t.Error("errors.As should match AgentError")
		}
		if agentErr.Agent != "eng-01" {
			t.Errorf("Agent = %q, want %q", agentErr.Agent, "eng-01")
		}
	})
}

func TestNewAgentError(t *testing.T) {
	err := NewAgentError("test-agent", "attach", ErrTimeout)
	if err.Agent != "test-agent" {
		t.Errorf("Agent = %q, want %q", err.Agent, "test-agent")
	}
	if err.Op != "attach" {
		t.Errorf("Op = %q, want %q", err.Op, "attach")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Error("should wrap ErrTimeout")
	}
}

func TestAgentNotFoundError(t *testing.T) {
	err := AgentNotFoundError("missing-agent")
	if err.Agent != "missing-agent" {
		t.Errorf("Agent = %q, want %q", err.Agent, "missing-agent")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Error("should wrap ErrNotFound")
	}
}

func TestChannelError(t *testing.T) {
	t.Run("with channel name", func(t *testing.T) {
		err := &ChannelError{Channel: "broadcast", Op: "send", Err: ErrPermission}
		want := `channel "broadcast": send: permission denied`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without channel name", func(t *testing.T) {
		err := &ChannelError{Op: "create", Err: ErrAlreadyExists}
		want := "channel: create: already exists"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := NewChannelError("test", "read", ErrTimeout)
		if !errors.Is(err, ErrTimeout) {
			t.Error("errors.Is should match ErrTimeout")
		}
	})
}

func TestWorkspaceError(t *testing.T) {
	t.Run("with path", func(t *testing.T) {
		err := &WorkspaceError{Path: "/home/user/project", Op: "init", Err: ErrAlreadyExists}
		want := `workspace "/home/user/project": init: already exists`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without path", func(t *testing.T) {
		err := &WorkspaceError{Op: "load", Err: ErrNotInWorkspace}
		want := "workspace: load: not in a bc workspace"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := NewWorkspaceError("/path", "save", ErrPermission)
		if !errors.Is(err, ErrPermission) {
			t.Error("errors.Is should match ErrPermission")
		}
	})
}

func TestConfigError(t *testing.T) {
	t.Run("with key", func(t *testing.T) {
		err := &ConfigError{Key: "tui.theme", Op: "parse", Err: ErrInvalidInput}
		want := `config "tui.theme": parse: invalid input`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without key", func(t *testing.T) {
		err := &ConfigError{Op: "load", Err: ErrNotFound}
		want := "config: load: not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := NewConfigError("key", "set", ErrInvalidInput)
		if !errors.Is(err, ErrInvalidInput) {
			t.Error("errors.Is should match ErrInvalidInput")
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := &ValidationError{Field: "name", Value: "", Message: "cannot be empty"}
		want := `validation error on "name": cannot be empty`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &ValidationError{Message: "invalid configuration"}
		want := "invalid configuration"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap to ErrInvalidInput", func(t *testing.T) {
		err := NewValidationError("age", -1, "must be positive")
		if !errors.Is(err, ErrInvalidInput) {
			t.Error("errors.Is should match ErrInvalidInput")
		}
	})

	t.Run("value preserved", func(t *testing.T) {
		err := NewValidationError("count", 42, "too large")
		if err.Value != 42 {
			t.Errorf("Value = %v, want 42", err.Value)
		}
	})
}

func TestIs(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrNotFound)
	if !Is(err, ErrNotFound) {
		t.Error("Is should return true for wrapped ErrNotFound")
	}
	if Is(err, ErrTimeout) {
		t.Error("Is should return false for non-matching error")
	}
}

func TestAs(t *testing.T) {
	original := &AgentError{Agent: "test", Op: "run", Err: ErrTimeout}
	err := fmt.Errorf("outer: %w", original)

	var agentErr *AgentError
	if !As(err, &agentErr) {
		t.Error("As should match AgentError")
	}
	if agentErr.Agent != "test" {
		t.Errorf("Agent = %q, want %q", agentErr.Agent, "test")
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		err := Wrap(ErrNotFound, "loading agent")
		want := "loading agent: not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Error("wrapped error should match ErrNotFound")
		}
	})

	t.Run("nil error returns nil", func(t *testing.T) {
		if err := Wrap(nil, "context"); err != nil {
			t.Errorf("Wrap(nil) = %v, want nil", err)
		}
	})
}

func TestWrapf(t *testing.T) {
	t.Run("wraps with format", func(t *testing.T) {
		err := Wrapf(ErrTimeout, "agent %s after %d seconds", "eng-01", 30)
		want := "agent eng-01 after 30 seconds: operation timeout"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
		if !errors.Is(err, ErrTimeout) {
			t.Error("wrapped error should match ErrTimeout")
		}
	})

	t.Run("nil error returns nil", func(t *testing.T) {
		if err := Wrapf(nil, "format %s", "arg"); err != nil {
			t.Errorf("Wrapf(nil) = %v, want nil", err)
		}
	})
}

func TestErrorChaining(t *testing.T) {
	// Test deep error chain: ValidationError -> AgentError -> ErrNotFound
	validationErr := &ValidationError{
		Field:   "agent",
		Message: "agent not found",
	}
	agentErr := &AgentError{
		Agent: "eng-01",
		Op:    "validate",
		Err:   validationErr,
	}
	wrapped := fmt.Errorf("command failed: %w", agentErr)

	// Should be able to extract AgentError
	var extractedAgent *AgentError
	if !errors.As(wrapped, &extractedAgent) {
		t.Error("should extract AgentError from chain")
	}

	// Should be able to extract ValidationError
	var extractedValidation *ValidationError
	if !errors.As(wrapped, &extractedValidation) {
		t.Error("should extract ValidationError from chain")
	}

	// ValidationError wraps ErrInvalidInput
	if !errors.Is(wrapped, ErrInvalidInput) {
		t.Error("chain should match ErrInvalidInput via ValidationError")
	}
}
