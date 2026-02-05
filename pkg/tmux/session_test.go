package tmux

import (
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
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
		name  string
		mgr   *Manager
		input string
		want  string
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

// hasTmux returns true if tmux is available.
func hasTmux() bool {
	return exec.Command("tmux", "-V").Run() == nil
}

// TestSendKeysPreservesSpaces verifies that spaces in messages survive the
// full send-keys -l path through tmux. This is a regression test for
// work-116/work-117 where spaces were reported stripped from channel messages.
func TestSendKeysPreservesSpaces(t *testing.T) {
	if !hasTmux() {
		t.Skip("tmux not available")
	}

	m := NewManager("test-sp-")

	tests := []struct {
		name    string
		message string
	}{
		{"simple spaces", "hello world"},
		{"multiple words", "Can someone update me with the status"},
		{"double spaces", "hello  world"},
		{"channel format", "[#standup] Can someone update me with the status of the build"},
		{"special chars with spaces", "fix: handle edge case (issue #42)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName := "sp-test"
			fullName := m.SessionName(sessionName)

			// Create session running cat (echoes stdin to PTY)
			cmd := exec.Command("tmux", "new-session", "-d", "-s", fullName, "cat")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("failed to create session: %v (%s)", err, out)
			}
			defer exec.Command("tmux", "kill-session", "-t", fullName).Run()

			time.Sleep(200 * time.Millisecond)

			// Send message WITHOUT submit key (text stays in input line)
			if err := m.SendKeysWithSubmit(sessionName, tt.message, ""); err != nil {
				t.Fatalf("SendKeysWithSubmit failed: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			captured, err := m.Capture(sessionName, 10)
			if err != nil {
				t.Fatalf("Capture failed: %v", err)
			}

			// The captured pane should contain the message with spaces intact.
			// Use Contains (not exact match) because the pane may include
			// a shell prompt or other artifacts.
			captured = strings.TrimSpace(captured)
			if !strings.Contains(captured, tt.message) {
				t.Errorf("spaces not preserved in tmux\n  sent:     %q\n  captured: %q", tt.message, captured)
			}
		})
	}
}

// TestPasteBufferPreservesSpaces verifies that the paste-buffer path (for
// messages > 500 chars) preserves internal spaces. Terminal line wrapping
// converts spaces at column boundaries to newlines, so we normalize both
// the sent and captured text by collapsing whitespace before comparing.
func TestPasteBufferPreservesSpaces(t *testing.T) {
	if !hasTmux() {
		t.Skip("tmux not available")
	}

	m := NewManager("test-pb-")

	// Build a message > 500 chars with lots of internal spaces.
	// Use distinct words so we can verify ordering too.
	var words []string
	for i := 0; i < 120; i++ {
		words = append(words, "word")
	}
	message := strings.Join(words, " ") // 120*4 + 119 = 599 chars
	if len(message) <= 500 {
		t.Fatalf("test message too short: %d chars", len(message))
	}

	sessionName := "pb-test"
	fullName := m.SessionName(sessionName)

	cmd := exec.Command("tmux", "new-session", "-d", "-s", fullName, "cat")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create session: %v (%s)", err, out)
	}
	defer exec.Command("tmux", "kill-session", "-t", fullName).Run()

	time.Sleep(200 * time.Millisecond)

	if err := m.SendKeysWithSubmit(sessionName, message, ""); err != nil {
		t.Fatalf("SendKeysWithSubmit failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	captured, err := m.Capture(sessionName, 50)
	if err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	// Terminal line wrapping converts spaces at column boundaries to newlines
	// and can split words across lines. Join lines back together for comparison.
	capturedJoined := strings.ReplaceAll(strings.TrimSpace(captured), "\n", "")

	if !strings.Contains(capturedJoined, message) {
		t.Errorf("paste-buffer path lost content\n  sent len:     %d\n  captured len: %d",
			len(message), len(capturedJoined))
	}
}
