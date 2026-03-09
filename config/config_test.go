package config

import (
	"testing"
)

// --- TmuxConfig ---

func TestTmuxDefaults(t *testing.T) {
	if Tmux.SessionPrefix != "bc-" {
		t.Errorf("Tmux.SessionPrefix = %q, want %q", Tmux.SessionPrefix, "bc-")
	}
}

// --- ProvidersConfig ---

func TestProvidersDefault(t *testing.T) {
	if Providers.Default == "" {
		t.Error("Providers.Default should not be empty")
	}
}

func TestProvidersClaudeEnabled(t *testing.T) {
	if !Providers.Claude.Enabled {
		t.Error("Providers.Claude should be enabled by default")
	}
	if Providers.Claude.Command == "" {
		t.Error("Providers.Claude.Command should not be empty")
	}
}
