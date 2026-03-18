package client

import (
	"context"
	"fmt"
	"time"
)

// CronClient provides cron job operations via the daemon.
type CronClient struct {
	client *Client
}

// CronJob represents a cron job returned by the daemon.
type CronJob struct {
	CreatedAt *time.Time `json:"created_at,omitempty"`
	LastRun   *time.Time `json:"last_run,omitempty"`
	NextRun   *time.Time `json:"next_run,omitempty"`
	Name      string     `json:"name"`
	Schedule  string     `json:"schedule"`
	AgentName string     `json:"agent_name,omitempty"`
	Prompt    string     `json:"prompt,omitempty"`
	Command   string     `json:"command,omitempty"`
	RunCount  int        `json:"run_count"`
	Enabled   bool       `json:"enabled"`
}

// CronLogEntry represents a cron job execution log entry.
type CronLogEntry struct {
	RunAt      time.Time `json:"run_at"`
	Status     string    `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	CostUSD    float64   `json:"cost_usd,omitempty"`
}

// List returns all cron jobs.
func (c *CronClient) List(ctx context.Context) ([]CronJob, error) {
	var jobs []CronJob
	if err := c.client.get(ctx, "/api/cron", &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// Get returns a specific cron job by name.
func (c *CronClient) Get(ctx context.Context, name string) (*CronJob, error) {
	var job CronJob
	if err := c.client.get(ctx, "/api/cron/"+name, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Add creates a new cron job.
func (c *CronClient) Add(ctx context.Context, job *CronJob) (*CronJob, error) {
	var created CronJob
	if err := c.client.post(ctx, "/api/cron", job, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// Delete removes a cron job.
func (c *CronClient) Delete(ctx context.Context, name string) error {
	return c.client.delete(ctx, "/api/cron/"+name)
}

// Enable enables a cron job.
func (c *CronClient) Enable(ctx context.Context, name string) error {
	return c.client.post(ctx, "/api/cron/"+name+"/enable", nil, nil)
}

// Disable disables a cron job.
func (c *CronClient) Disable(ctx context.Context, name string) error {
	return c.client.post(ctx, "/api/cron/"+name+"/disable", nil, nil)
}

// Run manually triggers a cron job.
func (c *CronClient) Run(ctx context.Context, name string) error {
	return c.client.post(ctx, "/api/cron/"+name+"/run", nil, nil)
}

// Logs returns the execution history for a cron job.
func (c *CronClient) Logs(ctx context.Context, name string, last int) ([]CronLogEntry, error) {
	var entries []CronLogEntry
	path := fmt.Sprintf("/api/cron/%s/logs?last=%d", name, last)
	if err := c.client.get(ctx, path, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
