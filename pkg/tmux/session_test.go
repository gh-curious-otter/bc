package tmux

import (
	"sync"
	"testing"
)

func TestGetSessionLock_ReturnsSameMutex(t *testing.T) {
	m := NewManager("test-")

	mu1 := m.getSessionLock("session-a")
	mu2 := m.getSessionLock("session-a")
	if mu1 != mu2 {
		t.Error("expected same mutex for same session name")
	}
}

func TestGetSessionLock_DifferentSessions(t *testing.T) {
	m := NewManager("test-")

	mu1 := m.getSessionLock("session-a")
	mu2 := m.getSessionLock("session-b")
	if mu1 == mu2 {
		t.Error("expected different mutexes for different session names")
	}
}

func TestGetSessionLock_ConcurrentAccess(t *testing.T) {
	m := NewManager("test-")
	var wg sync.WaitGroup
	results := make([]*sync.Mutex, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = m.getSessionLock("session-x")
		}(i)
	}
	wg.Wait()

	// All goroutines should have gotten the same mutex
	for i := 1; i < 100; i++ {
		if results[i] != results[0] {
			t.Errorf("goroutine %d got a different mutex", i)
		}
	}
}

func TestGetSessionLock_LazyInit(t *testing.T) {
	m := NewManager("test-")
	if m.sessionLocks != nil {
		t.Error("sessionLocks should be nil before first use")
	}
	m.getSessionLock("session-a")
	if m.sessionLocks == nil {
		t.Error("sessionLocks should be initialized after first use")
	}
}

func TestGenerateBufferName_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		name := generateBufferName()
		if seen[name] {
			t.Fatalf("duplicate buffer name: %s", name)
		}
		seen[name] = true
	}
}

func TestGenerateBufferName_Format(t *testing.T) {
	name := generateBufferName()
	if len(name) != 3+16 { // "bc-" + 16 hex chars
		t.Errorf("unexpected buffer name length: %d (%s)", len(name), name)
	}
	if name[:3] != "bc-" {
		t.Errorf("buffer name should start with 'bc-': %s", name)
	}
}

func TestSessionName(t *testing.T) {
	tests := []struct {
		name   string
		mgr    *Manager
		input  string
		want   string
	}{
		{
			name:  "simple prefix",
			mgr:   NewManager("bc-"),
			input: "agent1",
			want:  "bc-agent1",
		},
		{
			name:  "workspace manager",
			mgr:   NewWorkspaceManager("bc-", "/some/path"),
			input: "agent1",
			want:  "bc-" + NewWorkspaceManager("bc-", "/some/path").workspaceHash + "-agent1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mgr.SessionName(tt.input)
			if got != tt.want {
				t.Errorf("SessionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
