package runtime_test

import (
	"testing"

	"github.com/rpuneet/bc/pkg/runtime"
	"github.com/rpuneet/bc/pkg/tmux"
)

func TestTmuxBackendImplementsBackend(t *testing.T) {
	mgr := tmux.NewManager("test-")
	backend := runtime.NewTmuxBackend(mgr)

	// Verify it satisfies the interface
	var _ runtime.Backend = backend

	// Verify TmuxManager accessor
	if backend.TmuxManager() != mgr {
		t.Error("TmuxManager() should return the underlying tmux manager")
	}
}

func TestTmuxBackendSessionName(t *testing.T) {
	mgr := tmux.NewManager("bc-")
	backend := runtime.NewTmuxBackend(mgr)

	name := backend.SessionName("test-agent")
	if name != "bc-test-agent" {
		t.Errorf("SessionName() = %q, want %q", name, "bc-test-agent")
	}
}

func TestSessionStruct(t *testing.T) {
	s := runtime.Session{
		Name:      "test",
		Created:   "2024-01-01",
		Directory: "/tmp",
		Attached:  true,
	}
	if s.Name != "test" || s.Created != "2024-01-01" || s.Directory != "/tmp" || !s.Attached {
		t.Error("Session fields should be set correctly")
	}
}
