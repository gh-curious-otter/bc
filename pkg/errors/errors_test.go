package errors

import (
	"errors"
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
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
		{"ErrAgentStopped", ErrAgentStopped, "agent stopped"},
		{"ErrChannelNotFound", ErrChannelNotFound, "channel not found"},
		{"ErrRoleNotFound", ErrRoleNotFound, "role not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestAgentError(t *testing.T) {
	t.Run("with agent name", func(t *testing.T) {
		err := NewAgentError("eng-01", "start", ErrTimeout)
		want := `agent "eng-01": start: operation timeout`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without agent name", func(t *testing.T) {
		err := NewAgentError("", "list", ErrNotFound)
		want := "agent: list: not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		err := NewAgentError("test", "op", ErrTimeout)
		if !errors.Is(err, ErrTimeout) {
			t.Error("errors.Is() should return true for wrapped error")
		}
	})

	t.Run("errors.As", func(t *testing.T) {
		err := NewAgentError("eng-01", "stop", ErrAgentStopped)
		var agentErr *AgentError
		if !errors.As(err, &agentErr) {
			t.Error("errors.As() should return true for AgentError")
		}
		if agentErr.Agent != "eng-01" {
			t.Errorf("Agent = %q, want %q", agentErr.Agent, "eng-01")
		}
	})
}

func TestWorkspaceError(t *testing.T) {
	t.Run("with path", func(t *testing.T) {
		err := NewWorkspaceError("/path/to/ws", "load", ErrNotFound)
		want := `workspace "/path/to/ws": load: not found`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without path", func(t *testing.T) {
		err := NewWorkspaceError("", "init", ErrAlreadyExists)
		want := "workspace: init: already exists"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		err := NewWorkspaceError("/ws", "op", ErrNotInWorkspace)
		if !errors.Is(err, ErrNotInWorkspace) {
			t.Error("errors.Is() should return true for wrapped error")
		}
	})
}

func TestChannelError(t *testing.T) {
	t.Run("with channel name", func(t *testing.T) {
		err := NewChannelError("#general", "send", ErrPermission)
		want := `channel "#general": send: permission denied`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without channel name", func(t *testing.T) {
		err := NewChannelError("", "list", ErrTimeout)
		want := "channel: list: operation timeout"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestTmuxError(t *testing.T) {
	t.Run("with session name", func(t *testing.T) {
		err := NewTmuxError("bc-eng-01", "create", ErrAlreadyExists)
		want := `tmux session "bc-eng-01": create: already exists`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without session name", func(t *testing.T) {
		err := NewTmuxError("", "list-sessions", ErrTimeout)
		want := "tmux: list-sessions: operation timeout"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestRoleError(t *testing.T) {
	t.Run("with role name", func(t *testing.T) {
		err := NewRoleError("engineer", "validate", ErrInvalidInput)
		want := `role "engineer": validate: invalid input`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without role name", func(t *testing.T) {
		err := NewRoleError("", "list", ErrNotFound)
		want := "role: list: not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestIs(t *testing.T) {
	err := Wrap(ErrNotFound, "agent lookup failed")
	if !Is(err, ErrNotFound) {
		t.Error("Is() should return true for wrapped error")
	}
	if Is(err, ErrTimeout) {
		t.Error("Is() should return false for different error")
	}
}

func TestAs(t *testing.T) {
	err := NewAgentError("test", "op", ErrNotFound)
	var agentErr *AgentError
	if !As(err, &agentErr) {
		t.Error("As() should return true for AgentError")
	}

	var channelErr *ChannelError
	if As(err, &channelErr) {
		t.Error("As() should return false for wrong error type")
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps error with message", func(t *testing.T) {
		err := Wrap(ErrNotFound, "agent lookup failed")
		want := "agent lookup failed: not found"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		if err := Wrap(nil, "message"); err != nil {
			t.Errorf("Wrap(nil, ...) = %v, want nil", err)
		}
	})
}

func TestWrapf(t *testing.T) {
	t.Run("wraps error with formatted message", func(t *testing.T) {
		err := Wrapf(ErrNotFound, "agent %q lookup failed", "eng-01")
		want := `agent "eng-01" lookup failed: not found`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		if err := Wrapf(nil, "message %d", 42); err != nil {
			t.Errorf("Wrapf(nil, ...) = %v, want nil", err)
		}
	})
}
