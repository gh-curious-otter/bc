package client

import (
	"context"
	"fmt"
	"time"
)

// NotifyClient provides notification subscription operations via the daemon.
type NotifyClient struct {
	client *Client
}

// Subscription represents an agent's subscription to a channel.
type Subscription struct {
	CreatedAt   time.Time `json:"created_at"`
	Channel     string    `json:"channel"`
	Agent       string    `json:"agent"`
	ID          int64     `json:"id"`
	MentionOnly bool      `json:"mention_only"`
}

// DeliveryEntry represents a delivery log entry.
type DeliveryEntry struct {
	LoggedAt time.Time `json:"logged_at"`
	Channel  string    `json:"channel"`
	Agent    string    `json:"agent"`
	Status   string    `json:"status"`
	Error    string    `json:"error,omitempty"`
	Preview  string    `json:"preview,omitempty"`
	ID       int64     `json:"id"`
}

// ListSubscriptions returns all subscriptions.
func (n *NotifyClient) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	var subs []Subscription
	err := n.client.get(ctx, "/api/notify/subscriptions", &subs)
	return subs, err
}

// ChannelSubscriptions returns subscriptions for a specific channel.
func (n *NotifyClient) ChannelSubscriptions(ctx context.Context, channel string) ([]Subscription, error) {
	var subs []Subscription
	err := n.client.get(ctx, fmt.Sprintf("/api/notify/subscriptions/%s", channel), &subs)
	return subs, err
}

// Subscribe adds an agent to a channel.
func (n *NotifyClient) Subscribe(ctx context.Context, channel, agent string, mentionOnly bool) error {
	body := map[string]any{"channel": channel, "agent": agent, "mention_only": mentionOnly}
	return n.client.post(ctx, "/api/notify/subscriptions", body, nil)
}

// Unsubscribe removes an agent from a channel.
func (n *NotifyClient) Unsubscribe(ctx context.Context, channel, agent string) error {
	return n.client.delete(ctx, fmt.Sprintf("/api/notify/subscriptions/%s?agent=%s", channel, agent))
}

// Activity returns recent delivery log entries for a channel.
func (n *NotifyClient) Activity(ctx context.Context, channel string, limit int) ([]DeliveryEntry, error) {
	var entries []DeliveryEntry
	err := n.client.get(ctx, fmt.Sprintf("/api/notify/activity/%s?limit=%d", channel, limit), &entries)
	return entries, err
}
