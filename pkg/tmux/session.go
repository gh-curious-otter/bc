// Package tmux provides tmux session management for agent orchestration.
package tmux

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// Session represents a tmux session.
type Session struct {
	Name      string
	Created   string
	Attached  bool
	Windows   int
	Directory string
}

// Manager handles tmux session operations.
type Manager struct {
	// SessionPrefix is prepended to all session names (e.g., "bc-")
	SessionPrefix string
	// workspaceHash is included in session names for workspace isolation.
	workspaceHash string

	// sessionMu protects per-session SendKeys serialization.
	// Concurrent sends to the same session are serialized to prevent interleaving.
	sessionMu    sync.Mutex
	sessionLocks map[string]*sync.Mutex

	// execCommand creates exec.Cmd objects. Defaults to exec.Command.
	// Override in tests for mocking.
	execCommand func(name string, arg ...string) *exec.Cmd
}

// command returns an exec.Cmd using the configured executor.
func (m *Manager) command(name string, args ...string) *exec.Cmd {
	if m.execCommand != nil {
		return m.execCommand(name, args...)
	}
	return exec.Command(name, args...)
}

// NewManager creates a new tmux manager with the given prefix.
func NewManager(prefix string) *Manager {
	return &Manager{
		SessionPrefix: prefix,
		execCommand:   exec.Command,
	}
}

// NewWorkspaceManager creates a tmux manager scoped to a workspace.
// Session names include a short hash of the workspace path for isolation.
func NewWorkspaceManager(prefix, workspacePath string) *Manager {
	h := sha256.Sum256([]byte(workspacePath))
	return &Manager{
		SessionPrefix: prefix,
		workspaceHash: fmt.Sprintf("%x", h[:3]),
		execCommand:   exec.Command,
	}
}

// NewDefaultManager creates a new tmux manager with default prefix "bc-".
func NewDefaultManager() *Manager {
	return &Manager{
		SessionPrefix: "bc-",
		execCommand:   exec.Command,
	}
}

// SessionName returns the full session name with prefix (and workspace hash if set).
func (m *Manager) SessionName(name string) string {
	if m.workspaceHash != "" {
		return m.SessionPrefix + m.workspaceHash + "-" + name
	}
	return m.SessionPrefix + name
}

// HasSession checks if a session exists.
func (m *Manager) HasSession(name string) bool {
	fullName := m.SessionName(name)
	cmd := m.command("tmux", "has-session", "-t", fullName)
	return cmd.Run() == nil
}

// CreateSession creates a new tmux session.
func (m *Manager) CreateSession(name, dir string) error {
	fullName := m.SessionName(name)
	log.Debug("creating tmux session", "name", fullName, "dir", dir)

	args := []string{"new-session", "-d", "-s", fullName}
	if dir != "" {
		args = append(args, "-c", dir)
	}

	cmd := m.command("tmux", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create session %s: %w (%s)", fullName, err, string(output))
	}
	return nil
}

// CreateSessionWithCommand creates a session and runs a command.
func (m *Manager) CreateSessionWithCommand(name, dir, command string) error {
	return m.CreateSessionWithEnv(name, dir, command, nil)
}

// CreateSessionWithEnv creates a session with env vars baked into the shell command.
func (m *Manager) CreateSessionWithEnv(name, dir, command string, env map[string]string) error {
	fullName := m.SessionName(name)

	// Build shell command with env vars prefixed
	var parts []string
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s=%q;", k, v))
	}
	parts = append(parts, command)
	shellCmd := strings.Join(parts, " ")

	args := []string{"new-session", "-d", "-s", fullName}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, "bash", "-c", shellCmd)

	cmd := m.command("tmux", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create session %s: %w (%s)", fullName, err, string(output))
	}
	return nil
}

// KillSession kills a tmux session.
func (m *Manager) KillSession(name string) error {
	fullName := m.SessionName(name)
	log.Debug("killing tmux session", "name", fullName)
	cmd := m.command("tmux", "kill-session", "-t", fullName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill session %s: %w (%s)", fullName, err, string(output))
	}
	return nil
}

// getSessionLock returns a mutex for the given session name, creating one if needed.
// This serializes concurrent SendKeys calls to the same session.
func (m *Manager) getSessionLock(sessionName string) *sync.Mutex {
	m.sessionMu.Lock()
	defer m.sessionMu.Unlock()
	if m.sessionLocks == nil {
		m.sessionLocks = make(map[string]*sync.Mutex)
	}
	mu, ok := m.sessionLocks[sessionName]
	if !ok {
		mu = &sync.Mutex{}
		m.sessionLocks[sessionName] = mu
	}
	return mu
}

// SendKeys sends keys to a session with Enter as submit key.
// This is a convenience wrapper around SendKeysWithSubmit.
func (m *Manager) SendKeys(name, keys string) error {
	return m.SendKeysWithSubmit(name, keys, "Enter")
}

// SendKeysWithSubmit sends keys to a session with a specified submit key.
// For messages longer than 500 chars, uses tmux load-buffer/paste-buffer to avoid truncation.
// Trailing newlines are trimmed. submitKey specifies what to send after the message:
// - "Enter" sends the Enter key as a tmux key-name event
// - "" sends nothing (message is left in input buffer)
// - Other values are sent as tmux key names (e.g., "C-m" for Ctrl+M)
func (m *Manager) SendKeysWithSubmit(name, keys, submitKey string) error {
	keys = strings.TrimRight(keys, "\n")
	fullName := m.SessionName(name)

	// Serialize sends to the same session to prevent interleaving
	mu := m.getSessionLock(fullName)
	mu.Lock()
	defer mu.Unlock()

	if len(keys) <= 500 {
		// Send text literally (no key-name lookup)
		cmd := m.command("tmux", "send-keys", "-t", fullName, "-l", keys)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to send keys to %s: %w (%s)", fullName, err, string(output))
		}
	} else {
		// Long message: use temp file + load-buffer + paste-buffer with named buffer
		// Use a unique buffer name to avoid race conditions with concurrent sends
		bufferName := generateBufferName()

		tmpDir := filepath.Join(os.TempDir(), "bc-tmux")
		os.MkdirAll(tmpDir, 0700)
		tmpFile, err := os.CreateTemp(tmpDir, "send-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := tmpFile.WriteString(keys); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()

		// Load into named buffer
		loadCmd := m.command("tmux", "load-buffer", "-b", bufferName, tmpPath)
		if output, err := loadCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to load buffer: %w (%s)", err, string(output))
		}

		// Paste from named buffer and delete it afterward
		pasteCmd := m.command("tmux", "paste-buffer", "-b", bufferName, "-d", "-t", fullName)
		if output, err := pasteCmd.CombinedOutput(); err != nil {
			// Clean up buffer on error
			m.command("tmux", "delete-buffer", "-b", bufferName).Run()
			return fmt.Errorf("failed to paste buffer to %s: %w (%s)", fullName, err, string(output))
		}
	}

	if submitKey == "" {
		return nil
	}

	// Send the submit key as a separate operation.
	// Use -H 0D (raw hex carriage return byte) for Enter instead of the key name.
	// In tmux 3.5+, "send-keys Enter" (key name resolution) is unreliable after
	// paste-buffer operations — the key is silently dropped after bracketed paste.
	// The -H flag sends the raw byte directly, bypassing key resolution, and works
	// reliably in all scenarios including after paste-buffer.
	delay := 100 * time.Millisecond
	if len(keys) > 500 {
		delay = 500 * time.Millisecond
	}
	time.Sleep(delay)

	var submitCmd *exec.Cmd
	if submitKey == "Enter" {
		// Send raw CR byte (0x0D) via -H flag — reliable after paste-buffer.
		submitCmd = m.command("tmux", "send-keys", "-t", fullName, "-H", "0D")
	} else {
		submitCmd = m.command("tmux", "send-keys", "-t", fullName, submitKey)
	}
	if output, err := submitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send submit key to %s: %w (%s)", fullName, err, string(output))
	}

	return nil
}

// Capture captures the current pane content.
func (m *Manager) Capture(name string, lines int) (string, error) {
	fullName := m.SessionName(name)

	args := []string{"capture-pane", "-t", fullName, "-p"}
	if lines > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}

	cmd := m.command("tmux", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane %s: %w", fullName, err)
	}
	return string(output), nil
}

// ListSessions lists all sessions with our prefix.
func (m *Manager) ListSessions() ([]Session, error) {
	cmd := m.command("tmux", "list-sessions", "-F",
		"#{session_name}|#{session_created_string}|#{session_attached}|#{session_windows}|#{session_path}")

	output, err := cmd.Output()
	if err != nil {
		// tmux list-sessions fails with various messages when no server/sessions
		// exist: "no server running", "error connecting to ...", etc.
		// If it's an exit error, there are simply no sessions available.
		if _, ok := err.(*exec.ExitError); ok {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		name := parts[0]
		// Build full prefix including workspace hash
		fullPrefix := m.SessionPrefix
		if m.workspaceHash != "" {
			fullPrefix = m.SessionPrefix + m.workspaceHash + "-"
		}
		// Only include sessions with our prefix
		if !strings.HasPrefix(name, fullPrefix) {
			continue
		}

		sessions = append(sessions, Session{
			Name:      strings.TrimPrefix(name, fullPrefix),
			Created:   parts[1],
			Attached:  parts[2] == "1",
			Directory: parts[4],
		})
	}

	return sessions, nil
}

// AttachCmd returns an exec.Cmd to attach to a session.
// The caller should set Stdin/Stdout/Stderr and run it.
func (m *Manager) AttachCmd(name string) *exec.Cmd {
	fullName := m.SessionName(name)
	return m.command("tmux", "attach-session", "-t", fullName)
}

// IsRunning checks if tmux server is running.
func (m *Manager) IsRunning() bool {
	cmd := m.command("tmux", "list-sessions")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// "no server running" means tmux is available but no sessions
		if strings.Contains(stderr.String(), "no server running") {
			return false
		}
	}
	return err == nil
}

// KillServer kills the tmux server (all sessions).
func (m *Manager) KillServer() error {
	cmd := m.command("tmux", "kill-server")
	return cmd.Run()
}

// SetEnvironment sets an environment variable in a session.
func (m *Manager) SetEnvironment(name, key, value string) error {
	fullName := m.SessionName(name)
	cmd := m.command("tmux", "set-environment", "-t", fullName, key, value)
	return cmd.Run()
}

// generateBufferName creates a unique buffer name for tmux operations.
// This prevents race conditions when multiple goroutines send keys concurrently.
func generateBufferName() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "bc-" + hex.EncodeToString(b)
}
