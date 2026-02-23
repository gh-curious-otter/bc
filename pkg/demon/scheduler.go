// Package demon provides scheduled task management for bc.
package demon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// SchedulerStatus represents the current state of the scheduler.
type SchedulerStatus struct {
	StartedAt     time.Time `json:"started_at,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
	Uptime        string    `json:"uptime,omitempty"`
	PID           int       `json:"pid,omitempty"`
	Running       bool      `json:"running"`
	Healthy       bool      `json:"healthy"`
}

// Scheduler manages the demon scheduler process.
type Scheduler struct {
	rootDir       string
	demonsDir     string
	pidFile       string
	logFile       string
	statusFile    string
	heartbeatFile string
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(rootDir string) *Scheduler {
	demonsDir := filepath.Join(rootDir, ".bc", "demons")
	return &Scheduler{
		rootDir:       rootDir,
		demonsDir:     demonsDir,
		pidFile:       filepath.Join(demonsDir, "scheduler.pid"),
		logFile:       filepath.Join(demonsDir, "scheduler.log"),
		statusFile:    filepath.Join(demonsDir, "scheduler.json"),
		heartbeatFile: filepath.Join(demonsDir, "scheduler.heartbeat"),
	}
}

// Start starts the scheduler daemon.
// Returns an error if the scheduler is already running.
func (s *Scheduler) Start() error {
	// Check if already running
	if s.IsRunning() {
		return fmt.Errorf("scheduler is already running")
	}

	// Ensure demons directory exists
	if err := os.MkdirAll(s.demonsDir, 0750); err != nil {
		return fmt.Errorf("failed to create demons directory: %w", err)
	}

	// Start the scheduler process in background
	// Use the bc binary itself with a special internal flag
	bcPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create log file for scheduler output
	logFile, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) //nolint:gosec // path from trusted dir
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Start background process
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, bcPath, "demon", "scheduler-loop", "--root", s.rootDir) //nolint:gosec // bcPath from os.Executable
	cmd.Dir = s.rootDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session (detach from terminal)
	}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Write PID file
	pid := cmd.Process.Pid
	if err := os.WriteFile(s.pidFile, []byte(strconv.Itoa(pid)), 0600); err != nil {
		// Try to kill the started process
		_ = cmd.Process.Kill()
		_ = logFile.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Write status file
	status := SchedulerStatus{
		Running:   true,
		PID:       pid,
		StartedAt: time.Now().UTC(),
	}
	if err := s.saveStatus(status); err != nil {
		// Non-fatal, PID file is the primary tracking mechanism
		_ = logFile.Close()
		return nil
	}

	_ = logFile.Close()
	return nil
}

// Stop stops the scheduler daemon.
func (s *Scheduler) Stop() error {
	pid, err := s.readPID()
	if err != nil {
		return fmt.Errorf("scheduler is not running: %w", err)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process not found, clean up PID file
		_ = os.Remove(s.pidFile)
		_ = os.Remove(s.statusFile)
		return fmt.Errorf("scheduler process not found (pid %d)", pid)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Try SIGKILL if SIGTERM fails
		if killErr := process.Kill(); killErr != nil {
			return fmt.Errorf("failed to stop scheduler: %w", err)
		}
	}

	// Wait briefly for process to exit
	time.Sleep(100 * time.Millisecond)

	// Clean up files
	_ = os.Remove(s.pidFile)
	_ = os.Remove(s.statusFile)
	_ = os.Remove(s.heartbeatFile)

	return nil
}

// Status returns the current scheduler status.
func (s *Scheduler) Status() (*SchedulerStatus, error) {
	pid, err := s.readPID()
	if err != nil {
		return &SchedulerStatus{Running: false, Healthy: false}, nil
	}

	// Check if process is actually running
	if !s.processRunning(pid) {
		// Clean up stale PID file
		_ = os.Remove(s.pidFile)
		_ = os.Remove(s.statusFile)
		_ = os.Remove(s.heartbeatFile)
		return &SchedulerStatus{Running: false, Healthy: false}, nil
	}

	// Read status file for start time
	status, err := s.loadStatus()
	if err != nil {
		// Fallback: process is running but no status file
		status = &SchedulerStatus{}
	}

	status.Running = true
	status.PID = pid
	if !status.StartedAt.IsZero() {
		status.Uptime = time.Since(status.StartedAt).Truncate(time.Second).String()
	}

	// Check heartbeat for health status
	status.LastHeartbeat, status.Healthy = s.readHeartbeat()

	return status, nil
}

// readHeartbeat reads the heartbeat file and returns the timestamp and health status.
// The scheduler is considered healthy if the heartbeat is less than 60 seconds old.
func (s *Scheduler) readHeartbeat() (time.Time, bool) {
	data, err := os.ReadFile(s.heartbeatFile) //nolint:gosec // path from trusted dir
	if err != nil {
		return time.Time{}, false
	}

	timestamp, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}, false
	}

	// Healthy if heartbeat is less than 60 seconds old (2x the 30-second tick)
	healthy := time.Since(timestamp) < 60*time.Second
	return timestamp, healthy
}

// IsRunning checks if the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	pid, err := s.readPID()
	if err != nil {
		return false
	}
	return s.processRunning(pid)
}

// RunLoop runs the scheduler loop (called by scheduler-loop command).
// This should be called in the background process started by Start().
func (s *Scheduler) RunLoop(ctx context.Context) error {
	store := NewStore(s.rootDir)

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	s.log("Scheduler started")

	// Run immediately on start, then on ticker
	s.writeHeartbeat()
	s.checkAndRunDemons(store)

	for {
		select {
		case <-ctx.Done():
			s.log("Scheduler stopping")
			// Clean up heartbeat file on graceful shutdown
			_ = os.Remove(s.heartbeatFile)
			return ctx.Err()
		case <-ticker.C:
			s.writeHeartbeat()
			s.checkAndRunDemons(store)
		}
	}
}

// writeHeartbeat writes the current timestamp to the heartbeat file.
// This allows external tools to verify the scheduler is actively running.
func (s *Scheduler) writeHeartbeat() {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	_ = os.WriteFile(s.heartbeatFile, []byte(timestamp), 0600) //nolint:errcheck,gosec // best-effort heartbeat
}

// checkAndRunDemons checks all enabled demons and runs any that are due.
func (s *Scheduler) checkAndRunDemons(store *Store) {
	demons, err := store.ListEnabled()
	if err != nil {
		s.log("Failed to list demons: %v", err)
		return
	}

	now := time.Now()
	for _, d := range demons {
		// Skip if NextRun is zero or in the future
		if d.NextRun.IsZero() || d.NextRun.After(now) {
			continue
		}

		// Run the demon
		s.log("Running demon %q: %s", d.Name, d.Command)
		s.runDemon(store, d)
	}
}

// runDemon executes a demon's command.
func (s *Scheduler) runDemon(store *Store, d *Demon) {
	startTime := time.Now()
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "sh", "-c", d.Command) //nolint:gosec // command from trusted demon config
	cmd.Dir = s.rootDir
	output, err := cmd.CombinedOutput()

	duration := time.Since(startTime)
	exitCode := 0
	success := true

	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		s.log("Demon %q failed: %v", d.Name, err)
	} else {
		s.log("Demon %q completed successfully in %s", d.Name, duration)
	}

	// Record the run
	if recordErr := store.RecordRun(d.Name); recordErr != nil {
		s.log("Failed to record run for %q: %v", d.Name, recordErr)
	}

	// Record run log
	runLog := RunLog{
		Timestamp: startTime.UTC(),
		Duration:  duration.Milliseconds(),
		ExitCode:  exitCode,
		Success:   success,
	}
	if logErr := store.RecordRunLog(d.Name, runLog); logErr != nil {
		s.log("Failed to record log for %q: %v", d.Name, logErr)
	}

	// Log output if any
	if len(output) > 0 && len(output) < 1000 {
		s.log("Demon %q output: %s", d.Name, string(output))
	}
}

// log writes a timestamped message to the scheduler log.
func (s *Scheduler) log(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] %s\n", timestamp, msg)
}

// readPID reads the PID from the PID file.
func (s *Scheduler) readPID() (int, error) {
	data, err := os.ReadFile(s.pidFile) //nolint:gosec // path from trusted dir
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// processRunning checks if a process with the given PID is running.
func (s *Scheduler) processRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// saveStatus saves the scheduler status to disk.
func (s *Scheduler) saveStatus(status SchedulerStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statusFile, data, 0600) //nolint:gosec // path from trusted dir
}

// loadStatus loads the scheduler status from disk.
func (s *Scheduler) loadStatus() (*SchedulerStatus, error) {
	data, err := os.ReadFile(s.statusFile) //nolint:gosec // path from trusted dir
	if err != nil {
		return nil, err
	}
	var status SchedulerStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// GetNextRuns returns the next scheduled runs for all enabled demons.
func (s *Scheduler) GetNextRuns() ([]DemonNextRun, error) {
	store := NewStore(s.rootDir)
	demons, err := store.ListEnabled()
	if err != nil {
		return nil, err
	}

	var runs []DemonNextRun
	for _, d := range demons {
		runs = append(runs, DemonNextRun{
			Name:    d.Name,
			NextRun: d.NextRun,
			Command: d.Command,
		})
	}

	return runs, nil
}

// DemonNextRun represents a demon's next scheduled run.
type DemonNextRun struct {
	NextRun time.Time `json:"next_run"`
	Name    string    `json:"name"`
	Command string    `json:"command"`
}
