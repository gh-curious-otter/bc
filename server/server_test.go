package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rpuneet/bc/server"
	"github.com/rpuneet/bc/server/ws"
)

func buildTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	cfg := server.Config{Addr: "127.0.0.1:0", CORS: true}
	srv := server.New(cfg, server.Services{}, hub, nil)
	return httptest.NewServer(srv.Handler())
}

func TestHandleHealth(t *testing.T) {
	ts := buildTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("want status ok, got %v", body["status"])
	}
}

func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	ts := buildTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/health", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", resp.StatusCode)
	}
}

func TestCORSHeaders(t *testing.T) {
	ts := buildTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("CORS header missing")
	}
}

func TestServerStartShutdown(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()

	cfg := server.Config{Addr: "127.0.0.1:0"}
	srv := server.New(cfg, server.Services{}, hub, nil)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start(ctx) }()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestWebDist_Placeholder(t *testing.T) {
	if server.WebDist() != nil {
		t.Fatal("want nil when only placeholder.txt present")
	}
}
