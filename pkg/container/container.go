// Package container implements a Docker-based runtime backend for agent sessions.
//
// Each agent runs in an isolated Docker container with tmux inside for session
// management. This provides process isolation and resource limits while maintaining
// the same interactive experience as the native tmux backend.
//
// Communication uses `docker exec ... tmux send-keys` — no persistent connections
// or FIFOs needed.
package container

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/runtime"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Ensure Backend implements runtime.Backend.
var _ runtime.Backend = (*Backend)(nil)

// Config holds Docker runtime configuration.
type Config struct {
	Image       string
	Network     string
	ExtraMounts []string
	CPUs        float64
	MemoryMB    int64
}

// ConfigFromWorkspace converts workspace DockerRuntimeConfig to container Config.
func ConfigFromWorkspace(dcfg workspace.DockerRuntimeConfig) Config {
	cfg := Config{
		Image:       dcfg.Image,
		Network:     dcfg.Network,
		ExtraMounts: dcfg.ExtraMounts,
		CPUs:        dcfg.CPUs,
		MemoryMB:    dcfg.MemoryMB,
	}
	if cfg.Image == "" {
		cfg.Image = "bc-agent-claude:latest"
	}
	if cfg.CPUs == 0 {
		cfg.CPUs = 2.0
	}
	if cfg.MemoryMB == 0 {
		cfg.MemoryMB = 2048
	}
	if cfg.Network == "" {
		cfg.Network = "host"
	}
	return cfg
}

// Backend manages Docker containers as agent sessions.
// Each container runs tmux internally for interactive session management.
type Backend struct {
	logCancels       map[string]context.CancelFunc
	providerRegistry *provider.Registry
	prefix           string
	workspaceHash    string
	workspacePath    string
	cfg              Config
	mu               sync.RWMutex
}

// NewBackend creates a Docker runtime backend.
// Returns an error if the Docker daemon is not reachable.
func NewBackend(cfg Config, prefix, workspacePath string, registry *provider.Registry) (*Backend, error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "docker", "info") //nolint:gosec // trusted binary
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker daemon not available: %w", err)
	}

	h := sha256.Sum256([]byte(workspacePath))
	return &Backend{
		cfg:              cfg,
		prefix:           prefix,
		workspaceHash:    fmt.Sprintf("%x", h[:3]),
		workspacePath:    workspacePath,
		providerRegistry: registry,
		logCancels:       make(map[string]context.CancelFunc),
	}, nil
}

// containerName returns the Docker container name for an agent.
func (b *Backend) containerName(name string) string {
	return b.prefix + b.workspaceHash + "-" + name
}

// imageForTool returns the Docker image for a given agent tool name.
func (b *Backend) imageForTool(toolName string) string {
	if toolName == "" {
		return b.cfg.Image
	}
	if b.providerRegistry != nil {
		if p, ok := b.providerRegistry.Get(toolName); ok {
			if cc, ccOk := p.(provider.ContainerCustomizer); ccOk {
				if img := cc.DockerImage(); img != "" {
					return img
				}
			}
		}
	}
	return "bc-agent-" + toolName + ":latest"
}

// passthroughEnvKeys are environment variables passed from host to container.
var passthroughEnvKeys = []string{
	"GITHUB_TOKEN",
	"GH_TOKEN",
}

// SessionName returns the full session name with prefix.
func (b *Backend) SessionName(name string) string {
	return b.containerName(name)
}

// tmuxTarget returns the tmux session target inside the container.
// Providers like claude --tmux create their own session name (e.g., "workspace_worktree-root"),
// so we discover the first available session rather than assuming a fixed name.
func (b *Backend) tmuxTarget(ctx context.Context, name string) string {
	cn := b.containerName(name)
	//nolint:gosec // trusted
	cmd := exec.CommandContext(ctx, "docker", "exec", cn,
		"tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return name // fallback to agent name
	}
	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessions) > 0 && sessions[0] != "" {
		return sessions[0]
	}
	return name
}

// HasSession checks if a container exists, is running, AND has a live tmux
// session inside. A container with only zombie processes is treated as dead
// so the caller will respawn it rather than reusing a broken session.
func (b *Backend) HasSession(ctx context.Context, name string) bool {
	cn := b.containerName(name)

	// Check container is running
	//nolint:gosec // trusted
	inspect := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", cn)
	out, err := inspect.Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return false
	}

	// Also verify at least one tmux session is alive inside the container.
	// Claude --tmux creates sessions with varying names (workspace0, workspace_worktree-*).
	// If all tmux sessions are dead (zombie), treat the container as gone.
	//nolint:gosec // trusted
	tmuxCheck := exec.CommandContext(ctx, "docker", "exec", cn, "tmux", "list-sessions")
	return tmuxCheck.Run() == nil
}

// CreateSession creates a new container session.
func (b *Backend) CreateSession(ctx context.Context, name, dir string) error {
	return b.CreateSessionWithEnv(ctx, name, dir, "bash", nil)
}

// CreateSessionWithCommand creates a container with a command.
func (b *Backend) CreateSessionWithCommand(ctx context.Context, name, dir, command string) error {
	return b.CreateSessionWithEnv(ctx, name, dir, command, nil)
}

// CreateSessionWithEnv creates a container running the agent inside tmux.
// The container entrypoint:
//  1. Starts a tmux session named "agent" with the given command
//  2. Polls until the tmux session exits, then the container stops
//
// All interaction happens via `docker exec ... tmux send-keys/capture-pane/attach`.
func (b *Backend) CreateSessionWithEnv(ctx context.Context, name, dir, command string, env map[string]string) error {
	cn := b.containerName(name)

	// Remove any stale container with the same name before creating a new one.
	// This prevents "container name already in use" errors on agent restart.
	//nolint:gosec // trusted
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", cn).Run() //nolint:errcheck // best-effort cleanup

	args := []string{
		"run", "-d", "-t",
		"--name", cn,
		"--label", "bc.managed=true",
		"--label", "bc.workspace=" + b.workspaceHash,
		"--label", "bc.agent=" + name,
	}

	// Resource limits
	if b.cfg.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.1f", b.cfg.CPUs))
	}
	if b.cfg.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", b.cfg.MemoryMB))
	}

	// Network
	if b.cfg.Network != "" {
		args = append(args, "--network", b.cfg.Network)
	}

	// Volume mounts
	if dir != "" {
		args = append(args, "-v", dir+":/workspace")
	}

	// Per-agent auth — seeded from host credentials on first use so agents start
	// pre-authenticated without a separate login. Two mounts:
	//   1. auth/.claude/      → /home/agent/.claude/     (settings + cache)
	//   2. auth/.claude.json  → /home/agent/.claude.json (primary auth token)
	// Mounting the token file individually avoids replacing all of /home/agent.
	authDir, authErr := EnsureAuthDir(b.workspacePath, name)
	if authErr != nil {
		log.Debug("failed to create agent auth dir", "agent", name, "error", authErr)
	}
	if authErr == nil {
		args = append(args, "-v", authDir+":/home/agent/.claude")
		tokenFile := AgentAuthTokenFile(b.workspacePath, name)
		if _, statErr := os.Stat(tokenFile); statErr == nil {
			args = append(args, "-v", tokenFile+":/home/agent/.claude.json")
		}
	}

	// Read-only host config mounts (git, ssh)
	home, _ := os.UserHomeDir()
	if home != "" {
		roMounts := []struct{ src, dst string }{
			{home + "/.ssh", "/home/agent/.ssh"},
			{home + "/.gitconfig", "/home/agent/.gitconfig"},
		}
		for _, m := range roMounts {
			if _, err := os.Stat(m.src); err == nil {
				args = append(args, "-v", m.src+":"+m.dst+":ro")
			}
		}
	}

	// Extra mounts from config
	for _, mount := range b.cfg.ExtraMounts {
		args = append(args, "-v", mount)
	}

	// Environment variables from env map
	for k, v := range env {
		args = append(args, "-e", k+"="+v)
	}

	// Passthrough host env vars
	for _, key := range passthroughEnvKeys {
		if val := os.Getenv(key); val != "" {
			args = append(args, "-e", key+"="+val)
		}
	}

	// BC_* env vars
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "BC_") {
			args = append(args, "-e", e)
		}
	}

	// Select image based on agent tool
	image := b.cfg.Image
	if toolName, ok := env["BC_AGENT_TOOL"]; ok && toolName != "" {
		image = b.imageForTool(toolName)
	}

	// NOTE: Provider session customization (e.g., --tmux) is now applied
	// in agent.Manager.SpawnAgentWithOptions for ALL backends.

	// Run the agent command directly. claude --tmux handles its own tmux session.
	// Override entrypoint to avoid conflict with Dockerfile ENTRYPOINT.
	args = append(args, "--entrypoint", "bash", image, "-c", command)

	log.Debug("creating docker container", "name", cn, "image", image)
	//nolint:gosec // args are constructed from trusted internal values
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w (%s)", cn, err, strings.TrimSpace(string(output)))
	}

	return nil
}

// KillSession stops and removes a container.
func (b *Backend) KillSession(ctx context.Context, name string) error {
	cn := b.containerName(name)
	log.Debug("killing docker container", "name", cn)

	// Cancel any log stream
	b.mu.Lock()
	if cancel, ok := b.logCancels[cn]; ok {
		cancel()
		delete(b.logCancels, cn)
	}
	b.mu.Unlock()

	// Stop container (10s timeout)
	//nolint:gosec // trusted
	stopCmd := exec.CommandContext(ctx, "docker", "stop", "-t", "10", cn)
	_ = stopCmd.Run() //nolint:errcheck // may already be stopped

	// Remove container
	//nolint:gosec // trusted
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", cn)
	output, err := rmCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w (%s)", cn, err, strings.TrimSpace(string(output)))
	}

	return nil
}

// RenameSession renames a container.
func (b *Backend) RenameSession(ctx context.Context, oldName, newName string) error {
	oldCN := b.containerName(oldName)
	newCN := b.containerName(newName)

	//nolint:gosec // trusted
	cmd := exec.CommandContext(ctx, "docker", "rename", oldCN, newCN)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename container %s to %s: %w (%s)", oldCN, newCN, err, strings.TrimSpace(string(output)))
	}

	// Update log cancel mapping
	b.mu.Lock()
	if cancel, ok := b.logCancels[oldCN]; ok {
		b.logCancels[newCN] = cancel
		delete(b.logCancels, oldCN)
	}
	b.mu.Unlock()

	return nil
}

// SendKeys sends text to the agent's tmux session with Enter.
func (b *Backend) SendKeys(ctx context.Context, name, keys string) error {
	return b.SendKeysWithSubmit(ctx, name, keys, "Enter")
}

// SendKeysWithSubmit sends text to the agent's tmux session inside the container
// via `docker exec ... tmux send-keys`. Stateless — no persistent connections.
// Text is sent literally (-l flag), then the submit key is sent separately.
func (b *Backend) SendKeysWithSubmit(ctx context.Context, name, keys, submitKey string) error {
	cn := b.containerName(name)
	target := b.tmuxTarget(ctx, name)
	keys = strings.TrimRight(keys, "\n")

	// Send text literally (not as key names)
	//nolint:gosec // all args are trusted internal values
	sendCmd := exec.CommandContext(ctx, "docker", "exec", cn,
		"tmux", "send-keys", "-t", target, "-l", "--", keys)
	if output, err := sendCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send keys to %s: %w (%s)", cn, err, strings.TrimSpace(string(output)))
	}

	// Send submit key separately (as a tmux key name, not literal)
	if submitKey != "" {
		//nolint:gosec // trusted
		keyCmd := exec.CommandContext(ctx, "docker", "exec", cn,
			"tmux", "send-keys", "-t", target, submitKey)
		if output, err := keyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to send submit key to %s: %w (%s)", cn, err, strings.TrimSpace(string(output)))
		}
	}

	return nil
}

// Capture returns recent output from the agent's tmux pane.
func (b *Backend) Capture(ctx context.Context, name string, lines int) (string, error) {
	cn := b.containerName(name)

	startLine := "-100"
	if lines > 0 {
		startLine = fmt.Sprintf("-%d", lines)
	}

	target := b.tmuxTarget(ctx, name)

	//nolint:gosec // all args are trusted
	cmd := exec.CommandContext(ctx, "docker", "exec", cn,
		"tmux", "capture-pane", "-t", target, "-p", "-S", startLine)
	output, err := cmd.Output()
	if err != nil {
		// Fall back to docker logs if tmux capture fails
		//nolint:gosec // trusted
		fallback := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", lines), cn)
		fbOut, fbErr := fallback.Output()
		if fbErr != nil {
			return "", fmt.Errorf("failed to capture output from %s: %w", cn, err)
		}
		return string(fbOut), nil
	}
	return string(output), nil
}

// ListSessions lists all BC-managed containers for this workspace.
func (b *Backend) ListSessions(ctx context.Context) ([]runtime.Session, error) {
	//nolint:gosec // all args are trusted internal values
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=bc.managed=true",
		"--filter", "label=bc.workspace="+b.workspaceHash,
		"--format", "{{.Names}}|{{.CreatedAt}}|{{.Status}}")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var sessions []runtime.Session
	fullPrefix := b.prefix + b.workspaceHash + "-"
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		n := parts[0]
		if !strings.HasPrefix(n, fullPrefix) {
			continue
		}
		sessions = append(sessions, runtime.Session{
			Name:    strings.TrimPrefix(n, fullPrefix),
			Created: parts[1],
		})
	}

	return sessions, nil
}

// AttachCmd returns an exec.Cmd to attach to the agent's tmux session inside the container.
func (b *Backend) AttachCmd(ctx context.Context, name string) *exec.Cmd {
	cn := b.containerName(name)
	target := b.tmuxTarget(ctx, name)
	//nolint:gosec // trusted
	return exec.CommandContext(ctx, "docker", "exec", "-it", cn, "tmux", "attach", "-t", target)
}

// IsRunning checks if the Docker daemon is running.
func (b *Backend) IsRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info") //nolint:gosec // trusted binary
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// KillServer stops and removes all BC containers for this workspace.
func (b *Backend) KillServer(ctx context.Context) error {
	//nolint:gosec // all args are trusted internal values
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq",
		"--filter", "label=bc.managed=true",
		"--filter", "label=bc.workspace="+b.workspaceHash)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	ids := strings.Fields(strings.TrimSpace(string(output)))
	if len(ids) == 0 {
		return nil
	}

	// Cancel all log streams
	b.mu.Lock()
	for n, cancel := range b.logCancels {
		cancel()
		delete(b.logCancels, n)
	}
	b.mu.Unlock()

	// Remove all containers
	args := append([]string{"rm", "-f"}, ids...)
	//nolint:gosec // trusted
	rmCmd := exec.CommandContext(ctx, "docker", args...)
	rmOutput, rmErr := rmCmd.CombinedOutput()
	if rmErr != nil {
		return fmt.Errorf("failed to remove containers: %w (%s)", rmErr, strings.TrimSpace(string(rmOutput)))
	}

	return nil
}

// SetEnvironment is a no-op for Docker containers.
func (b *Backend) SetEnvironment(_ context.Context, name, key, value string) error {
	log.Debug("SetEnvironment is a no-op for docker containers", "name", name, "key", key, "value", value)
	return nil
}

// PipePane streams container logs to a file.
func (b *Backend) PipePane(ctx context.Context, name, logPath string) error {
	cn := b.containerName(name)

	b.mu.Lock()
	if cancel, ok := b.logCancels[cn]; ok {
		cancel()
		delete(b.logCancels, cn)
	}
	b.mu.Unlock()

	if logPath == "" {
		return nil
	}

	logCtx, cancel := context.WithCancel(ctx)

	b.mu.Lock()
	b.logCancels[cn] = cancel
	b.mu.Unlock()

	go func() {
		defer cancel()
		//nolint:gosec // trusted
		cmd := exec.CommandContext(logCtx, "docker", "logs", "-f", cn)

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		if err := cmd.Start(); err != nil {
			log.Warn("failed to start log streaming", "container", cn, "error", err)
			return
		}

		_ = cmd.Wait() //nolint:errcheck // expected when context canceled

		if buf.Len() > 0 {
			//nolint:gosec // logPath is from trusted internal sources
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				log.Warn("failed to open log file", "path", logPath, "error", err)
				return
			}
			_, _ = f.Write(buf.Bytes()) //nolint:errcheck // best-effort
			_ = f.Close()               //nolint:errcheck // best-effort
		}
	}()

	return nil
}
