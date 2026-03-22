package client

import (
	"context"
	"time"
)

// DaemonsClient provides daemon process operations via the daemon.
type DaemonsClient struct {
	client *Client
}

// DaemonInfo represents a managed daemon process returned by the daemon.
type DaemonInfo struct {
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   time.Time  `json:"started_at,omitempty"`
	StoppedAt   *time.Time `json:"stopped_at,omitempty"`
	Name        string     `json:"name"`
	Runtime     string     `json:"runtime"`
	Cmd         string     `json:"cmd,omitempty"`
	Image       string     `json:"image,omitempty"`
	ContainerID string     `json:"container_id,omitempty"`
	Restart     string     `json:"restart"`
	Status      string     `json:"status"`
	Ports       []string   `json:"ports,omitempty"`
	Volumes     []string   `json:"volumes,omitempty"`
	EnvVars     []string   `json:"env,omitempty"`
	PID         int64      `json:"pid,omitempty"`
}

// List returns all managed daemons from the daemon.
func (d *DaemonsClient) List(ctx context.Context) ([]DaemonInfo, error) {
	var daemons []DaemonInfo
	if err := d.client.get(ctx, "/api/daemons", &daemons); err != nil {
		return nil, err
	}
	return daemons, nil
}

// Get returns a single daemon by name.
func (d *DaemonsClient) Get(ctx context.Context, name string) (*DaemonInfo, error) {
	var info DaemonInfo
	if err := d.client.get(ctx, "/api/daemons/"+name, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Stop stops a running daemon.
func (d *DaemonsClient) Stop(ctx context.Context, name string) error {
	return d.client.post(ctx, "/api/daemons/"+name+"/stop", nil, nil)
}

// Restart restarts a daemon.
func (d *DaemonsClient) Restart(ctx context.Context, name string) (*DaemonInfo, error) {
	var info DaemonInfo
	if err := d.client.post(ctx, "/api/daemons/"+name+"/restart", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Delete removes a daemon.
func (d *DaemonsClient) Delete(ctx context.Context, name string) error {
	return d.client.delete(ctx, "/api/daemons/"+name)
}
