package client

import (
	"context"
	"time"
)

// TeamsClient provides team operations via the daemon.
type TeamsClient struct {
	client *Client
}

// TeamInfo represents team data returned by the daemon.
type TeamInfo struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Lead        string    `json:"lead,omitempty"`
	Members     []string  `json:"members,omitempty"`
}

// List returns all teams from the daemon.
func (t *TeamsClient) List(ctx context.Context) ([]TeamInfo, error) {
	var teams []TeamInfo
	if err := t.client.get(ctx, "/api/teams", &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

// Get returns a single team by name.
func (t *TeamsClient) Get(ctx context.Context, name string) (*TeamInfo, error) {
	var info TeamInfo
	if err := t.client.get(ctx, "/api/teams/"+name, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
