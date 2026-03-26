package cron

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gh-curious-otter/bc/pkg/log"
)

const (
	// DefaultPollInterval is how often the scheduler checks for due jobs.
	DefaultPollInterval = 30 * time.Second

	// DefaultJobTimeout is the maximum time a single job execution may take.
	DefaultJobTimeout = 5 * time.Minute
)

// Scheduler polls the cron store and executes due jobs in the background.
type Scheduler struct {
	// execFn is the function used to run a command. Replaceable for testing.
	execFn     func(ctx context.Context, command string, logWriter io.Writer) (err error)
	store      *Store
	interval   time.Duration
	jobTimeout time.Duration
	logDir     string // directory for live log files

	running   map[string]bool // jobs currently executing
	runningMu sync.RWMutex
}

// NewScheduler creates a Scheduler that polls at DefaultPollInterval.
func NewScheduler(store *Store, logDir string) *Scheduler {
	if logDir != "" {
		_ = os.MkdirAll(logDir, 0o755)
	}
	s := &Scheduler{
		store:      store,
		interval:   DefaultPollInterval,
		jobTimeout: DefaultJobTimeout,
		logDir:     logDir,
		running:    make(map[string]bool),
	}
	s.execFn = s.defaultExec
	return s
}

// IsRunning returns true if the named job is currently executing.
func (s *Scheduler) IsRunning(name string) bool {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()
	return s.running[name]
}

// RunningJobs returns the names of all currently executing jobs.
func (s *Scheduler) RunningJobs() []string {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()
	names := make([]string, 0, len(s.running))
	for name := range s.running {
		names = append(names, name)
	}
	return names
}

// LogFilePath returns the live log file path for a job.
func (s *Scheduler) LogFilePath(name string) string {
	if s.logDir == "" {
		return ""
	}
	return filepath.Join(s.logDir, name+".log")
}

// Run blocks until ctx is canceled, polling for due jobs on each tick.
func (s *Scheduler) Run(ctx context.Context) {
	log.Info("cron scheduler started", "interval", s.interval.String())

	// Perform an initial tick immediately so jobs due at startup are caught.
	s.tick(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("cron scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick checks all enabled jobs and executes any that are due.
func (s *Scheduler) tick(ctx context.Context) {
	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		log.Error("cron scheduler: failed to list jobs", "error", err)
		return
	}

	now := time.Now()
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		if job.Command == "" {
			continue
		}
		if s.IsRunning(job.Name) {
			continue // don't overlap
		}
		if !isDue(job, now) {
			continue
		}
		s.executeJob(ctx, job)
	}
}

// isDue reports whether a job should run: it has a next_run time that is at or before now.
func isDue(job *Job, now time.Time) bool {
	if job.NextRun == nil {
		return false
	}
	return !job.NextRun.After(now)
}

// executeJob runs a single job's command and records the result.
func (s *Scheduler) executeJob(ctx context.Context, job *Job) {
	log.Info("cron scheduler: executing job", "job", job.Name, "command", job.Command)

	// Mark as running
	s.runningMu.Lock()
	s.running[job.Name] = true
	s.runningMu.Unlock()

	defer func() {
		s.runningMu.Lock()
		delete(s.running, job.Name)
		s.runningMu.Unlock()
	}()

	jobCtx, cancel := context.WithTimeout(ctx, s.jobTimeout)
	defer cancel()

	// Set up live log file
	var logWriter io.Writer
	var logFile *os.File
	logPath := s.LogFilePath(job.Name)
	if logPath != "" {
		var openErr error
		logFile, openErr = os.Create(logPath) //nolint:gosec // path from controlled logDir
		if openErr != nil {
			log.Warn("cron scheduler: failed to create log file", "job", job.Name, "error", openErr)
		} else {
			logWriter = logFile
		}
	}
	if logWriter == nil {
		logWriter = io.Discard
	}

	start := time.Now()
	execErr := s.execFn(jobCtx, job.Command, logWriter)
	elapsed := time.Since(start)

	if logFile != nil {
		_ = logFile.Close() //nolint:errcheck
	}

	// Read output from log file for the DB record
	var output string
	if logPath != "" {
		data, readErr := os.ReadFile(logPath) //nolint:gosec
		if readErr == nil {
			output = string(data)
			const maxOutput = 64 * 1024
			if len(output) > maxOutput {
				output = output[:maxOutput] + fmt.Sprintf("\n... (truncated, total %d bytes)", len(data))
			}
		}
	}

	status := "success"
	if execErr != nil {
		status = "failed"
		if jobCtx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
		log.Warn("cron scheduler: job failed", "job", job.Name, "error", execErr)
	}

	entry := &LogEntry{
		JobName:    job.Name,
		Status:     status,
		DurationMS: elapsed.Milliseconds(),
		Output:     output,
		RunAt:      start,
	}

	if err := s.store.RecordRun(ctx, entry); err != nil {
		log.Error("cron scheduler: failed to record run", "job", job.Name, "error", err)
	}
}

// defaultExec runs a shell command and streams output to the writer.
func (s *Scheduler) defaultExec(ctx context.Context, command string, logWriter io.Writer) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command) //#nosec G204 -- commands are user-configured cron jobs
	var buf bytes.Buffer
	w := io.MultiWriter(&buf, logWriter)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
