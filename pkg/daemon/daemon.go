// Package daemon provides workspace process management for bc.
//
// A daemon is a named long-lived process (database, API server, etc.)
// running in either a tmux bash session or a Docker container, scoped
// to the current workspace.
//
// State is persisted in .bc/bc.db (SQLite) so daemons survive
// bc invocations. Each daemon runs under a unique name and can be
// stopped, restarted, and removed independently.
package daemon

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/db"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/tmux"
)

// Status represents the runtime status of a daemon.
type Status string

const (
	// StatusRunning indicates the process is active.
	StatusRunning Status = "running"
	// StatusStopped indicates the process has been stopped.
	StatusStopped Status = "stopped"
	// StatusFailed indicates the process exited with an error.
	StatusFailed Status = "failed"
)

// Runtime selects the execution backend.
const (
	RuntimeBash   = "bash"   // tmux session on the host
	RuntimeDocker = "docker" // Docker container
)

// Daemon represents a managed workspace process.
//
//nolint:govet // fieldalignment: logical grouping preferred
type Daemon struct {
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   time.Time  `json:"started_at,omitempty"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
	Name        string     `json:"name"`
	Runtime     string     `json:"runtime"`
	Cmd         string     `json:"cmd,omitempty"`
	Image       string     `json:"image,omitempty"`
	ContainerID string     `json:"container_id,omitempty"`
	Restart     string     `json:"restart"`
	Status      Status     `json:"status"`
	Ports       []string   `json:"ports,omitempty"`
	Volumes     []string   `json:"volumes,omitempty"`
	EnvVars     []string   `json:"env,omitempty"`
	PID         int64      `json:"pid,omitempty"`
}

// RunOptions configures how a daemon is started.
type RunOptions struct {
	Name    string
	Runtime string
	Cmd     string
	Image   string
	EnvFile string
	Restart string
	Ports   []string
	Volumes []string
	Env     []string
	Detach  bool
}

// Manager manages workspace daemons using SQLite state and tmux/Docker.
type Manager struct {
	tmuxMgr       *tmux.Manager
	db            *db.DB
	workspacePath string
	workspaceHash string
	logsDir       string
}

// NewManager creates a daemon manager for the given workspace.
// The db is at workspaceDir/.bc/bc.db.
func NewManager(workspaceDir string) (*Manager, error) {
	dbPath := filepath.Join(workspaceDir, ".bc", "daemons.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open daemons db: %w", err)
	}

	h := sha256.Sum256([]byte(workspaceDir))
	hash := fmt.Sprintf("%x", h[:3])

	logsDir := filepath.Join(workspaceDir, ".bc", "logs")
	if mkErr := os.MkdirAll(logsDir, 0750); mkErr != nil {
		log.Debug("failed to create logs dir", "error", mkErr)
	}

	mgr := &Manager{
		db:            database,
		workspacePath: workspaceDir,
		workspaceHash: hash,
		logsDir:       logsDir,
		tmuxMgr:       tmux.NewManager("bc-daemon-"),
	}

	if err := mgr.initSchema(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("init daemon schema: %w", err)
	}

	return mgr, nil
}

// Close releases database resources.
func (m *Manager) Close() error {
	return m.db.Close()
}

// initSchema creates the daemons table if it does not exist.
func (m *Manager) initSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS daemons (
		name         TEXT PRIMARY KEY,
		runtime      TEXT NOT NULL,
		cmd          TEXT,
		image        TEXT,
		status       TEXT NOT NULL DEFAULT 'stopped',
		pid          INTEGER,
		container_id TEXT,
		ports        TEXT,
		env          TEXT,
		restart      TEXT NOT NULL DEFAULT 'no',
		created_at   TEXT NOT NULL,
		started_at   TEXT,
		stopped_at   TEXT
	)`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.db.ExecContext(ctx, schema)
	return err
}

// isValidDaemonName returns true if name contains only safe characters.
// Allows lowercase/uppercase letters, digits, hyphens, and underscores.
// This prevents shell injection via tmux session names and Docker container names.
func isValidDaemonName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for _, c := range name {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		if !isLower && !isUpper && !isDigit && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// Run starts a new daemon or restarts an existing stopped one.
func (m *Manager) Run(ctx context.Context, opts RunOptions) (*Daemon, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("daemon name is required")
	}
	if !isValidDaemonName(opts.Name) {
		return nil, fmt.Errorf("invalid daemon name %q: use only letters, digits, hyphens, underscores (max 63 chars)", opts.Name)
	}
	if opts.Runtime != RuntimeBash && opts.Runtime != RuntimeDocker {
		return nil, fmt.Errorf("runtime must be %q or %q", RuntimeBash, RuntimeDocker)
	}
	if opts.Runtime == RuntimeBash && opts.Cmd == "" {
		return nil, fmt.Errorf("--cmd is required for bash runtime")
	}
	if opts.Runtime == RuntimeDocker && opts.Image == "" {
		return nil, fmt.Errorf("--image is required for docker runtime")
	}
	if opts.Restart == "" {
		opts.Restart = "no"
	}

	existing, err := m.Get(ctx, opts.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == StatusRunning {
		return nil, fmt.Errorf("daemon %q is already running", opts.Name)
	}

	// Build env from opts
	env := opts.Env
	if opts.EnvFile != "" {
		fileEnv, readErr := readEnvFile(opts.EnvFile)
		if readErr != nil {
			return nil, fmt.Errorf("read env file: %w", readErr)
		}
		env = append(env, fileEnv...)
	}

	now := time.Now()
	d := &Daemon{
		Name:      opts.Name,
		Runtime:   opts.Runtime,
		Cmd:       opts.Cmd,
		Image:     opts.Image,
		Ports:     opts.Ports,
		Volumes:   opts.Volumes,
		EnvVars:   env,
		Restart:   opts.Restart,
		Status:    StatusRunning,
		CreatedAt: now,
		StartedAt: now,
	}

	if opts.Runtime == RuntimeBash {
		if err := m.startBash(ctx, d); err != nil {
			d.Status = StatusFailed
			_ = m.save(ctx, d) //nolint:errcheck // best-effort on failure path
			return nil, fmt.Errorf("start bash daemon: %w", err)
		}
	} else {
		if err := m.startDocker(ctx, d); err != nil {
			d.Status = StatusFailed
			_ = m.save(ctx, d) //nolint:errcheck // best-effort on failure path
			return nil, fmt.Errorf("start docker daemon: %w", err)
		}
	}

	if err := m.save(ctx, d); err != nil {
		return nil, fmt.Errorf("save daemon state: %w", err)
	}

	return d, nil
}

// startBash launches the daemon command in a named tmux session.
func (m *Manager) startBash(ctx context.Context, d *Daemon) error {
	logFile := m.logFile(d.Name)

	env := map[string]string{}
	for _, kv := range d.EnvVars {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	if err := m.tmuxMgr.CreateSessionWithEnv(ctx, d.Name, m.workspacePath, d.Cmd, env); err != nil {
		return err
	}

	// Pipe session output to log file
	if err := m.tmuxMgr.PipePane(ctx, d.Name, logFile); err != nil {
		log.Debug("failed to pipe daemon log", "daemon", d.Name, "error", err)
	}

	return nil
}

// startDocker launches the daemon in a Docker container.
func (m *Manager) startDocker(ctx context.Context, d *Daemon) error {
	cn := m.containerName(d.Name)

	// Remove stale container if present
	//nolint:gosec // trusted binary
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", cn).Run() //nolint:errcheck // best-effort cleanup

	args := []string{"run", "-d"}
	args = append(args,
		"--name", cn,
		"--label", "bc.daemon=true",
		"--label", "bc.workspace="+m.workspaceHash,
		"--label", "bc.daemon.name="+d.Name,
	)

	for _, p := range d.Ports {
		args = append(args, "-p", p)
	}
	for _, v := range d.Volumes {
		args = append(args, "-v", v)
	}
	for _, e := range d.EnvVars {
		args = append(args, "-e", e)
	}

	// Restart policy
	if d.Restart != "no" && d.Restart != "" {
		args = append(args, "--restart", d.Restart)
	}

	args = append(args, d.Image)

	//nolint:gosec // args constructed from internal values
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	d.ContainerID = strings.TrimSpace(string(out))

	return nil
}

// Stop stops a running daemon.
func (m *Manager) Stop(ctx context.Context, name string) error {
	d, err := m.Get(ctx, name)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("daemon %q not found", name)
	}
	if d.Status != StatusRunning {
		return fmt.Errorf("daemon %q is not running (status: %s)", name, d.Status)
	}

	if d.Runtime == RuntimeBash {
		if err := m.tmuxMgr.KillSession(ctx, name); err != nil {
			log.Debug("failed to kill daemon tmux session", "daemon", name, "error", err)
		}
	} else {
		cn := m.containerName(name)
		//nolint:gosec // trusted binary + container name from internal state
		if out, stopErr := exec.CommandContext(ctx, "docker", "stop", cn).CombinedOutput(); stopErr != nil {
			log.Debug("failed to stop docker daemon", "daemon", name, "output", string(out), "error", stopErr)
		}
	}

	now := time.Now()
	d.Status = StatusStopped
	d.StoppedAt = &now
	return m.save(ctx, d)
}

// Restart stops and restarts a daemon using its saved configuration.
func (m *Manager) Restart(ctx context.Context, name string) (*Daemon, error) {
	d, err := m.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("daemon %q not found", name)
	}

	if d.Status == StatusRunning {
		if err := m.Stop(ctx, name); err != nil {
			return nil, fmt.Errorf("stop daemon: %w", err)
		}
	}

	return m.Run(ctx, RunOptions{
		Name:    d.Name,
		Runtime: d.Runtime,
		Cmd:     d.Cmd,
		Image:   d.Image,
		Ports:   d.Ports,
		Volumes: d.Volumes,
		Env:     d.EnvVars,
		Restart: d.Restart,
		Detach:  true,
	})
}

// Remove permanently deletes a daemon record. The daemon must be stopped first.
func (m *Manager) Remove(ctx context.Context, name string) error {
	d, err := m.Get(ctx, name)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("daemon %q not found", name)
	}
	if d.Status == StatusRunning {
		return fmt.Errorf("daemon %q is running — stop it first with: bc daemon stop %s", name, name)
	}

	_, err = m.db.ExecContext(ctx, `DELETE FROM daemons WHERE name = ?`, name)
	return err
}

// List returns all daemons.
// syncStatus writes are deferred until after rows are fully consumed to
// avoid a deadlock on SQLite's single-writer connection.
func (m *Manager) List(ctx context.Context) ([]*Daemon, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT name, runtime, cmd, image, status, pid, container_id,
		       ports, env, restart, created_at, started_at, stopped_at
		FROM daemons ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query daemons: %w", err)
	}

	// Collect all rows before closing — syncStatus does DB writes and must
	// not run while the read query is still open on SQLite's single connection.
	var daemons []*Daemon
	for rows.Next() {
		d, err := scanDaemon(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		daemons = append(daemons, d)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Now safe to write: reconcile live process state with DB records.
	for _, d := range daemons {
		m.syncStatus(ctx, d)
	}
	return daemons, nil
}

// Get returns a daemon by name or nil if not found.
func (m *Manager) Get(ctx context.Context, name string) (*Daemon, error) {
	row := m.db.QueryRowContext(ctx, `
		SELECT name, runtime, cmd, image, status, pid, container_id,
		       ports, env, restart, created_at, started_at, stopped_at
		FROM daemons WHERE name = ?`, name)

	d, err := scanDaemon(row)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	m.syncStatus(ctx, d)
	return d, nil
}

// Logs returns recent log lines for a daemon.
// For docker runtimes, reads from docker logs. For bash, reads the log file.
func (m *Manager) Logs(ctx context.Context, name string, lines int) (string, error) {
	d, err := m.Get(ctx, name)
	if err != nil {
		return "", err
	}
	if d == nil {
		return "", fmt.Errorf("daemon %q not found", name)
	}

	if d.Runtime == RuntimeDocker {
		cn := m.containerName(name)
		linesStr := fmt.Sprintf("%d", lines)
		//nolint:gosec // trusted binary
		out, cmdErr := exec.CommandContext(ctx, "docker", "logs", "--tail", linesStr, cn).CombinedOutput()
		if cmdErr != nil {
			return string(out), nil // Return whatever we got
		}
		return string(out), nil
	}

	// Bash runtime: tail the log file
	logFile := m.logFile(name)
	data, readErr := os.ReadFile(logFile) //nolint:gosec // path from trusted internal logFile()
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "(no logs yet)", nil
		}
		return "", readErr
	}

	// Return last N lines
	all := strings.Split(string(data), "\n")
	if lines > 0 && len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n"), nil
}

// syncStatus updates the daemon's status field by checking if the process is actually running.
// This keeps status accurate after crashes or external stops.
func (m *Manager) syncStatus(ctx context.Context, d *Daemon) {
	if d.Status != StatusRunning {
		return
	}
	var alive bool
	if d.Runtime == RuntimeBash {
		alive = m.tmuxMgr.HasSession(ctx, d.Name)
	} else {
		cn := m.containerName(d.Name)
		//nolint:gosec // trusted binary
		out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", cn).Output()
		alive = err == nil && strings.TrimSpace(string(out)) == "true"
	}
	if !alive {
		now := time.Now()
		d.Status = StatusFailed
		d.StoppedAt = &now
		if err := m.save(ctx, d); err != nil {
			log.Debug("failed to sync daemon status", "daemon", d.Name, "error", err)
		}
	}
}

// save persists daemon state to SQLite.
func (m *Manager) save(ctx context.Context, d *Daemon) error {
	portsJSON, err := json.Marshal(d.Ports)
	if err != nil {
		return fmt.Errorf("marshal ports: %w", err)
	}
	envJSON, err := json.Marshal(d.EnvVars)
	if err != nil {
		return fmt.Errorf("marshal env: %w", err)
	}

	var stoppedAt *string
	if d.StoppedAt != nil {
		s := d.StoppedAt.UTC().Format(time.RFC3339)
		stoppedAt = &s
	}

	startedAt := d.StartedAt.UTC().Format(time.RFC3339)
	if d.StartedAt.IsZero() {
		startedAt = ""
	}

	_, err = m.db.ExecContext(ctx, `
		INSERT INTO daemons (name, runtime, cmd, image, status, pid, container_id,
		                     ports, env, restart, created_at, started_at, stopped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			runtime=excluded.runtime, cmd=excluded.cmd, image=excluded.image,
			status=excluded.status, pid=excluded.pid, container_id=excluded.container_id,
			ports=excluded.ports, env=excluded.env, restart=excluded.restart,
			started_at=excluded.started_at, stopped_at=excluded.stopped_at`,
		d.Name, d.Runtime, nullStr(d.Cmd), nullStr(d.Image),
		string(d.Status), d.PID, nullStr(d.ContainerID),
		string(portsJSON), string(envJSON), d.Restart,
		d.CreatedAt.UTC().Format(time.RFC3339),
		nullStr(startedAt), stoppedAt,
	)
	return err
}

// rowScanner abstracts *sql.Row and *sql.Rows for scanDaemon.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanDaemon reads a daemon record from a DB row.
func scanDaemon(row rowScanner) (*Daemon, error) {
	var d Daemon
	var cmd, image, containerID, startedAtStr sql.NullString
	var stoppedAtStr sql.NullString
	var createdAtStr string
	var pid sql.NullInt64
	var portsJSON, envJSON string

	err := row.Scan(
		&d.Name, &d.Runtime, &cmd, &image, &d.Status, &pid, &containerID,
		&portsJSON, &envJSON, &d.Restart, &createdAtStr, &startedAtStr, &stoppedAtStr,
	)
	if err != nil {
		return nil, err
	}

	d.Cmd = cmd.String
	d.Image = image.String
	d.ContainerID = containerID.String
	d.PID = pid.Int64

	if createdAtStr != "" {
		if t, parseErr := time.Parse(time.RFC3339, createdAtStr); parseErr == nil {
			d.CreatedAt = t
		}
	}
	if startedAtStr.Valid && startedAtStr.String != "" {
		if t, parseErr := time.Parse(time.RFC3339, startedAtStr.String); parseErr == nil {
			d.StartedAt = t
		}
	}
	if stoppedAtStr.Valid && stoppedAtStr.String != "" {
		if t, parseErr := time.Parse(time.RFC3339, stoppedAtStr.String); parseErr == nil {
			d.StoppedAt = &t
		}
	}

	if portsJSON != "" && portsJSON != "null" {
		_ = json.Unmarshal([]byte(portsJSON), &d.Ports) //nolint:errcheck // best-effort
	}
	if envJSON != "" && envJSON != "null" {
		_ = json.Unmarshal([]byte(envJSON), &d.EnvVars) //nolint:errcheck // best-effort
	}

	return &d, nil
}

// containerName returns the Docker container name for a daemon.
func (m *Manager) containerName(name string) string {
	return "bc-" + m.workspaceHash + "-" + name
}

// logFile returns the log file path for a bash daemon.
func (m *Manager) logFile(name string) string {
	return filepath.Join(m.logsDir, "daemon-"+name+".log")
}

// nullStr returns a *string — nil for empty strings, value otherwise.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// readEnvFile reads a file of KEY=VALUE pairs, one per line.
func readEnvFile(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from user input, validated by caller
	if err != nil {
		return nil, err
	}
	var env []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		env = append(env, line)
	}
	return env, nil
}
