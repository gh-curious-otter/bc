// Package client provides an HTTP client for the bcd daemon.
//
// Commands use this client to communicate with the daemon instead of
// calling pkg/ packages directly. This enables the daemon architecture
// where the CLI is a thin client.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultSocketPath returns the default Unix socket path for bcd.
func DefaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/bcd.sock"
	}
	return filepath.Join(home, ".bc", "bcd.sock")
}

// DefaultHTTPAddr is the fallback HTTP address for bcd.
const DefaultHTTPAddr = "http://localhost:4880"

// Client is the HTTP client for the bcd daemon.
type Client struct {
	HTTPClient *http.Client
	Agents     *AgentsClient
	Channels   *ChannelsClient
	Workspaces *WorkspacesClient
	BaseURL    string
}

// New creates a new bcd client with the given base URL.
// If addr is empty, it tries to auto-discover the daemon.
func New(addr string) *Client {
	if addr == "" {
		addr = discoverDaemon()
	}

	c := &Client{
		BaseURL: addr,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	c.Agents = &AgentsClient{client: c}
	c.Channels = &ChannelsClient{client: c}
	c.Workspaces = &WorkspacesClient{client: c}

	return c
}

// Ping checks if the daemon is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running (connect to %s failed): %w", c.BaseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon unhealthy (status %d)", resp.StatusCode)
	}
	return nil
}

// get performs a GET request and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// discoverDaemon tries to find the daemon address.
// Priority: BC_DAEMON_ADDR env > default HTTP address.
func discoverDaemon() string {
	if addr := os.Getenv("BC_DAEMON_ADDR"); addr != "" {
		return addr
	}
	return DefaultHTTPAddr
}
