// Package tmux provides tmux session management for agent orchestration.
package tmux

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
}

// NewManager creates a new tmux manager with the given prefix.
func NewManager(prefix string) *Manager {
	return &Manager{
		SessionPrefix: prefix,
	}
}

// NewWorkspaceManager creates a tmux manager scoped to a workspace.
// Session names include a short hash of the workspace path for isolation.
func NewWorkspaceManager(prefix, workspacePath string) *Manager {
	h := sha256.Sum256([]byte(workspacePath))
	return &Manager{
		SessionPrefix: prefix,
		workspaceHash: fmt.Sprintf("%x", h[:3]),
	}
}

// NewDefaultManager creates a new tmux manager with default prefix "bc-".
func NewDefaultManager() *Manager {
	return &Manager{
		SessionPrefix: "bc-",
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
	cmd := exec.Command("tmux", "has-session", "-t", fullName)
	return cmd.Run() == nil
}

// CreateSession creates a new tmux session.
func (m *Manager) CreateSession(name, dir string) error {
	fullName := m.SessionName(name)

	args := []string{"new-session", "-d", "-s", fullName}
	if dir != "" {
		args = append(args, "-c", dir)
	}

	cmd := exec.Command("tmux", args...)
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

	cmd := exec.Command("tmux", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create session %s: %w (%s)", fullName, err, string(output))
	}
	return nil
}

// KillSession kills a tmux session.
func (m *Manager) KillSession(name string) error {
	fullName := m.SessionName(name)
	cmd := exec.Command("tmux", "kill-session", "-t", fullName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill session %s: %w (%s)", fullName, err, string(output))
	}
	return nil
}

// SendKeys sends keys to a session.
// For messages longer than 500 chars, uses tmux load-buffer/paste-buffer to avoid truncation.
func (m *Manager) SendKeys(name, keys string) error {
	fullName := m.SessionName(name)

	if len(keys) <= 500 {
		cmd := exec.Command("tmux", "send-keys", "-t", fullName, keys, "Enter")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to send keys to %s: %w (%s)", fullName, err, string(output))
		}
		return nil
	}

	// Long message: use temp file + load-buffer + paste-buffer
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

	loadCmd := exec.Command("tmux", "load-buffer", tmpPath)
	if output, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load buffer: %w (%s)", err, string(output))
	}

	pasteCmd := exec.Command("tmux", "paste-buffer", "-t", fullName)
	if output, err := pasteCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to paste buffer to %s: %w (%s)", fullName, err, string(output))
	}

	// Wait for paste to complete before sending Enter
	time.Sleep(500 * time.Millisecond)

	enterCmd := exec.Command("tmux", "send-keys", "-t", fullName, "Enter")
	if output, err := enterCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send Enter to %s: %w (%s)", fullName, err, string(output))
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

	cmd := exec.Command("tmux", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane %s: %w", fullName, err)
	}
	return string(output), nil
}

// ListSessions lists all sessions with our prefix.
func (m *Manager) ListSessions() ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F",
		"#{session_name}|#{session_created_string}|#{session_attached}|#{session_windows}|#{session_path}")

	output, err := cmd.Output()
	if err != nil {
		// No sessions might return error
		if strings.Contains(err.Error(), "no server running") {
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
	return exec.Command("tmux", "attach-session", "-t", fullName)
}

// IsRunning checks if tmux server is running.
func (m *Manager) IsRunning() bool {
	cmd := exec.Command("tmux", "list-sessions")
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
	cmd := exec.Command("tmux", "kill-server")
	return cmd.Run()
}

// SetEnvironment sets an environment variable in a session.
func (m *Manager) SetEnvironment(name, key, value string) error {
	fullName := m.SessionName(name)
	cmd := exec.Command("tmux", "set-environment", "-t", fullName, key, value)
	return cmd.Run()
}
