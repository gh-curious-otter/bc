// Package client provides an HTTP client for the bcd daemon.
//
// Commands use this client to communicate with the daemon instead of
// calling pkg/ packages directly. This enables the daemon architecture
// where the CLI is a thin client.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultHTTPAddr is the fallback HTTP address for bcd.
// Uses 127.0.0.1 (IPv4 loopback) to match the server's bind address and avoid
// localhost→::1 (IPv6) resolution failures on dual-stack systems.
const DefaultHTTPAddr = "http://127.0.0.1:9374"

// AddrFileName is the name of the file written by bcd containing its listen address.
const AddrFileName = "bcd.addr"

// Client is the HTTP client for the bcd daemon.
type Client struct {
	HTTPClient *http.Client
	Agents     *AgentsClient
	Channels   *ChannelsClient
	Workspaces *WorkspacesClient
	Events     *EventsClient
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
	c.Events = &EventsClient{client: c}

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
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("request failed (status %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// discoverDaemon tries to find the daemon address.
// Priority: BC_DAEMON_ADDR env > bcd.addr file in workspace .bc/ > default HTTP address.
func discoverDaemon() string {
	if addr := os.Getenv("BC_DAEMON_ADDR"); addr != "" {
		return addr
	}
	if addr := readAddrFile(); addr != "" {
		return addr
	}
	return DefaultHTTPAddr
}

// readAddrFile walks up from the current directory looking for a .bc/bcd.addr file
// written by bcd on startup, and returns the address it contains.
func readAddrFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		addrPath := filepath.Join(dir, ".bc", AddrFileName)
		if data, err := os.ReadFile(addrPath); err == nil {
			addr := strings.TrimSpace(string(data))
			if addr != "" {
				return addr
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// IsDaemonNotRunning returns true if the error indicates the daemon is not running.
func IsDaemonNotRunning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "daemon not running") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such file")
}

// post performs a POST request with JSON body and decodes the JSON response.
func (c *Client) post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// put performs a PUT request with JSON body and decodes the JSON response.
func (c *Client) put(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPut, path, body, result)
}

// delete performs a DELETE request (no body, no response body expected).
func (c *Client) delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// do performs an HTTP request with optional JSON body and response.
func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("request failed (status %d)", resp.StatusCode)
		}
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
