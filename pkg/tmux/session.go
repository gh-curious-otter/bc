// Package tmux provides tmux session management for agent orchestration.
package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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
}

// NewManager creates a new tmux manager with the given prefix.
func NewManager(prefix string) *Manager {
	return &Manager{
		SessionPrefix: prefix,
	}
}

// NewDefaultManager creates a new tmux manager with default prefix "bc-".
func NewDefaultManager() *Manager {
	return &Manager{
		SessionPrefix: "bc-",
	}
}

// SessionName returns the full session name with prefix.
func (m *Manager) SessionName(name string) string {
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
	fullName := m.SessionName(name)
	
	args := []string{"new-session", "-d", "-s", fullName}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, command)
	
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
func (m *Manager) SendKeys(name, keys string) error {
	fullName := m.SessionName(name)
	cmd := exec.Command("tmux", "send-keys", "-t", fullName, keys, "Enter")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to send keys to %s: %w (%s)", fullName, err, string(output))
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
		// Only include sessions with our prefix
		if !strings.HasPrefix(name, m.SessionPrefix) {
			continue
		}
		
		sessions = append(sessions, Session{
			Name:      strings.TrimPrefix(name, m.SessionPrefix),
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
