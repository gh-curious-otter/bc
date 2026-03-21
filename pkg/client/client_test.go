package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockServer creates a test server that returns the given response for any request.
func mockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

// jsonHandler returns an HTTP handler that responds with the given JSON body and status.
func jsonHandler(status int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(body) //nolint:errcheck
	}
}

func TestNew(t *testing.T) {
	c := New("http://localhost:9374")
	if c.BaseURL != "http://localhost:9374" {
		t.Errorf("BaseURL = %q, want http://localhost:9374", c.BaseURL)
	}
	if c.Agents == nil {
		t.Error("Agents client is nil")
	}
	if c.Channels == nil {
		t.Error("Channels client is nil")
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
}

func TestPing_Success(t *testing.T) {
	ts := mockServer(t, jsonHandler(200, map[string]string{"status": "ok"}))
	c := New(ts.URL)

	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestPing_Unhealthy(t *testing.T) {
	ts := mockServer(t, jsonHandler(503, map[string]string{"status": "error"}))
	c := New(ts.URL)

	if err := c.Ping(context.Background()); err == nil {
		t.Error("Ping() expected error for 503, got nil")
	}
}

func TestPing_ConnectionRefused(t *testing.T) {
	c := New("http://127.0.0.1:1") // port 1 — connection refused

	if err := c.Ping(context.Background()); err == nil {
		t.Error("Ping() expected error for connection refused, got nil")
	}
}

func TestAgents_List(t *testing.T) {
	agents := []map[string]string{
		{"name": "alice", "role": "engineer", "state": "idle"},
		{"name": "bob", "role": "engineer", "state": "working"},
	}
	ts := mockServer(t, jsonHandler(200, agents))
	c := New(ts.URL)

	var result []map[string]any
	err := c.get(context.Background(), "/api/agents", &result)
	if err != nil {
		t.Fatalf("get agents: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d agents, want 2", len(result))
	}
}

func TestAgents_Create(t *testing.T) {
	var receivedBody map[string]string
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		json.NewDecoder(r.Body).Decode(&receivedBody) //nolint:errcheck
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"name": "alice", "role": "engineer"}) //nolint:errcheck
	})
	c := New(ts.URL)

	var result map[string]any
	err := c.post(context.Background(), "/api/agents", map[string]string{
		"name": "alice",
		"role": "engineer",
	}, &result)
	if err != nil {
		t.Fatalf("post agents: %v", err)
	}
	if receivedBody["name"] != "alice" {
		t.Errorf("request body name = %q, want alice", receivedBody["name"])
	}
}

func TestChannels_List(t *testing.T) {
	channels := []map[string]string{
		{"name": "general"},
		{"name": "eng"},
	}
	ts := mockServer(t, jsonHandler(200, channels))
	c := New(ts.URL)

	var result []map[string]any
	err := c.get(context.Background(), "/api/channels", &result)
	if err != nil {
		t.Fatalf("get channels: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d channels, want 2", len(result))
	}
}

func TestErrorResponse(t *testing.T) {
	ts := mockServer(t, jsonHandler(400, map[string]string{"error": "bad request"}))
	c := New(ts.URL)

	var result map[string]any
	err := c.get(context.Background(), "/api/agents/missing", &result)
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}

func TestDelete(t *testing.T) {
	var method string
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(204)
	})
	c := New(ts.URL)

	err := c.delete(context.Background(), "/api/agents/alice")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", method)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	ts := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("not json")) //nolint:errcheck
	})
	c := New(ts.URL)

	var result map[string]any
	err := c.get(context.Background(), "/api/test", &result)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
