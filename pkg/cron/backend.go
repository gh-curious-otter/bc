package cron

import "context"

// Backend is the storage interface for cron job persistence.
// Store is the default SQLite implementation.
type Backend interface {
	// AddJob inserts a new cron job.
	AddJob(ctx context.Context, job *Job) error
	// GetJob returns a job by name. Returns nil, nil if not found.
	GetJob(ctx context.Context, name string) (*Job, error)
	// ListJobs returns all cron jobs ordered by name.
	ListJobs(ctx context.Context) ([]*Job, error)
	// DeleteJob removes a cron job and its logs by name.
	DeleteJob(ctx context.Context, name string) error
	// SetEnabled enables or disables a job.
	SetEnabled(ctx context.Context, name string, enabled bool) error
	// RecordRun records a job execution result and updates run stats.
	RecordRun(ctx context.Context, entry *LogEntry) error
	// RecordManualTrigger records a manual trigger for a job.
	RecordManualTrigger(ctx context.Context, name string) error
	// GetLogs returns recent log entries for a job.
	GetLogs(ctx context.Context, jobName string, last int) ([]*LogEntry, error)
	// Close releases database resources.
	Close() error
}
