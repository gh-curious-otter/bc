package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	ws, err := workspace.Init(dir)
	if err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	srv, err := New(Config{Addr: "127.0.0.1:0", Dir: ws.RootDir})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(func() {
		_ = srv.channels.Close()
		_ = srv.costs.Close()
		_ = srv.events.Close()
	})
	return srv
}

func newReq(method, target string) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, target, nil)
}

func TestHealthEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/health")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	body, _ := io.ReadAll(w.Body)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp["status"])
	}
}

func TestAgentListEmpty(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/agents")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestChannelList(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/channels")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWorkspaceStatus(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/workspace")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	body, _ := io.ReadAll(w.Body)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["name"] == nil {
		t.Fatal("expected workspace name in response")
	}
}

func TestRoleList(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/roles")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCostSummary(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/costs")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestEventList(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/events?limit=10")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAgentNotFound(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodGet, "/api/v1/agents/nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	srv := setupTestServer(t)
	handler := srv.routes()

	req := newReq(http.MethodOptions, "/api/v1/agents")
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("expected CORS origin header")
	}
}
