package cron

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rpuneet/bc/pkg/log"
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
	execFn     func(ctx context.Context, command string) (output string, err error)
	store      *Store
	interval   time.Duration
	jobTimeout time.Duration
}

// NewScheduler creates a Scheduler that polls at DefaultPollInterval.
func NewScheduler(store *Store) *Scheduler {
	s := &Scheduler{
		store:      store,
		interval:   DefaultPollInterval,
		jobTimeout: DefaultJobTimeout,
	}
	s.execFn = s.defaultExec
	return s
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

	jobCtx, cancel := context.WithTimeout(ctx, s.jobTimeout)
	defer cancel()

	start := time.Now()
	output, execErr := s.execFn(jobCtx, job.Command)
	elapsed := time.Since(start)

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

// defaultExec runs a shell command and captures combined stdout/stderr.
func (s *Scheduler) defaultExec(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command) //#nosec G204 -- commands are user-configured cron jobs
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := buf.String()
	// Truncate very large output to avoid bloating the database.
	const maxOutput = 64 * 1024
	if len(output) > maxOutput {
		output = output[:maxOutput] + fmt.Sprintf("\n... (truncated, total %d bytes)", len(output))
	}
	return output, err
}
