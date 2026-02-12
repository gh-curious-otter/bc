// Package runtime provides the Go-side bridge for the Ink TUI.
package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// InkBridge manages communication between Go and the Ink TUI process.
type InkBridge struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader

	tuiDir     string
	entryPoint string

	mu     sync.Mutex
	closed bool
}

// BridgeConfig holds configuration for the Ink bridge.
type BridgeConfig struct {
	// TUIDir is the directory containing the TUI package (default: "tui")
	TUIDir string
	// EntryPoint is the entry script (default: "src/index.tsx")
	EntryPoint string
	// WorkspaceRoot is the bc workspace root directory
	WorkspaceRoot string
}

// DefaultConfig returns the default bridge configuration.
func DefaultConfig() BridgeConfig {
	return BridgeConfig{
		TUIDir:     "tui",
		EntryPoint: "src/index.tsx",
	}
}

// NewInkBridge creates a new Ink bridge with the given configuration.
func NewInkBridge(cfg BridgeConfig) (*InkBridge, error) {
	if cfg.TUIDir == "" {
		cfg.TUIDir = "tui"
	}
	if cfg.EntryPoint == "" {
		cfg.EntryPoint = "src/index.tsx"
	}

	// Resolve TUI directory path
	tuiDir := cfg.TUIDir
	if cfg.WorkspaceRoot != "" {
		tuiDir = filepath.Join(cfg.WorkspaceRoot, cfg.TUIDir)
	}

	// Check if TUI directory exists
	if _, err := os.Stat(tuiDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("TUI directory not found: %s", tuiDir)
	}

	entryPath := filepath.Join(tuiDir, cfg.EntryPoint)
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("TUI entry point not found: %s", entryPath)
	}

	return &InkBridge{
		tuiDir:     tuiDir,
		entryPoint: cfg.EntryPoint,
	}, nil
}

// Start launches the Ink TUI process.
func (b *InkBridge) Start() error {
	return b.StartWithContext(context.Background())
}

// StartWithContext launches the Ink TUI process with the given context.
func (b *InkBridge) StartWithContext(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("bridge is closed")
	}

	// Create command to run Bun with the TUI entry point
	// #nosec G204 - entryPoint is validated in NewInkBridge
	b.cmd = exec.CommandContext(ctx, "bun", "run", b.entryPoint)
	b.cmd.Dir = b.tuiDir
	b.cmd.Env = append(os.Environ(), "BC_BRIDGE_MODE=1")

	var err error

	// Setup stdin pipe for sending specs to TUI
	b.stdin, err = b.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Setup stdout pipe for reading events from TUI
	b.stdout, err = b.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	b.reader = bufio.NewReader(b.stdout)

	// Setup stderr pipe for error output
	b.stderr, err = b.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start TUI process: %w", err)
	}

	return nil
}

// SendSpec sends a JSON spec to the TUI for rendering.
func (b *InkBridge) SendSpec(spec []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("bridge is closed")
	}

	if b.stdin == nil {
		return fmt.Errorf("bridge not started")
	}

	// Write spec followed by newline delimiter
	if _, err := b.stdin.Write(spec); err != nil {
		return fmt.Errorf("failed to write spec: %w", err)
	}
	if _, err := b.stdin.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write delimiter: %w", err)
	}

	return nil
}

// SendJSON marshals the given value to JSON and sends it to the TUI.
func (b *InkBridge) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}
	return b.SendSpec(data)
}

// ReadEvent reads the next event from the TUI.
// Events are newline-delimited JSON messages.
func (b *InkBridge) ReadEvent() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, fmt.Errorf("bridge is closed")
	}

	if b.reader == nil {
		return nil, fmt.Errorf("bridge not started")
	}

	line, err := b.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Trim newline
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	return line, nil
}

// ReadEventJSON reads and unmarshals the next event into the given value.
func (b *InkBridge) ReadEventJSON(v any) error {
	data, err := b.ReadEvent()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Close shuts down the Ink TUI process.
func (b *InkBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	var errs []error

	if b.stdin != nil {
		if err := b.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
		}
	}

	if b.cmd != nil && b.cmd.Process != nil {
		if err := b.cmd.Process.Kill(); err != nil {
			errs = append(errs, fmt.Errorf("failed to kill process: %w", err))
		}
		_ = b.cmd.Wait() // Clean up zombie process
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Wait waits for the TUI process to exit and returns its exit error.
func (b *InkBridge) Wait() error {
	if b.cmd == nil {
		return fmt.Errorf("bridge not started")
	}
	return b.cmd.Wait()
}

// IsRunning returns true if the TUI process is running.
func (b *InkBridge) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed || b.cmd == nil || b.cmd.Process == nil {
		return false
	}

	// Check if process has exited
	if b.cmd.ProcessState != nil {
		return false
	}

	return true
}
