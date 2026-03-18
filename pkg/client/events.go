package client

import (
	"context"
	"fmt"
	"time"
)

// EventsClient provides event log operations via the daemon.
type EventsClient struct {
	client *Client
}

// EventInfo represents an event log entry returned by the daemon.
type EventInfo struct {
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"ts"`
	Type      string         `json:"type"`
	Agent     string         `json:"agent,omitempty"`
	Message   string         `json:"message,omitempty"`
}

// List returns all events from the daemon.
func (e *EventsClient) List(ctx context.Context) ([]EventInfo, error) {
	var evts []EventInfo
	if err := e.client.get(ctx, "/api/logs", &evts); err != nil {
		return nil, err
	}
	return evts, nil
}

// ListByAgent returns events for a specific agent.
func (e *EventsClient) ListByAgent(ctx context.Context, agentName string) ([]EventInfo, error) {
	var evts []EventInfo
	if err := e.client.get(ctx, "/api/logs/"+agentName, &evts); err != nil {
		return nil, err
	}
	return evts, nil
}

// Tail returns the last N events.
func (e *EventsClient) Tail(ctx context.Context, n int) ([]EventInfo, error) {
	var evts []EventInfo
	if err := e.client.get(ctx, fmt.Sprintf("/api/logs?tail=%d", n), &evts); err != nil {
		return nil, err
	}
	return evts, nil
}
