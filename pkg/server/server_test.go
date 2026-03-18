package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/workspace"
)

// buildTestServer creates a server wired to minimal test doubles.
func buildTestServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()

	// Agent service with empty manager
	agentMgr := agent.NewWorkspaceManager(dir+"/agents", dir)
	agentSvc := agent.NewAgentService(agentMgr, nil, nil)

	// Channel service with empty JSON store
	chStore := channel.NewStore(dir)
	channelSvc := channel.NewChannelService(chStore)

	// Daemon manager
	daemonMgr, err := daemon.NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = daemonMgr.Close() })

	// Minimal workspace
	ws := &workspace.Workspace{RootDir: dir}

	return New(Config{Addr: ":0"}, agentSvc, channelSvc, daemonMgr, ws)
}

func TestHandleHealth(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}
}

func TestHandleHealthMethodNotAllowed(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleAgentsList(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()
	srv.handleAgents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var agents []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestHandleChannelsList(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()
	srv.handleChannels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestHandleWorkspaceStatus(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workspace/status", nil)
	w := httptest.NewRecorder()
	srv.handleWorkspaceStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var status map[string]any
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := status["agent_count"]; !ok {
		t.Error("response missing agent_count")
	}
}

func TestHandleDaemons(t *testing.T) {
	srv := buildTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/daemons", nil)
	w := httptest.NewRecorder()
	srv.handleDaemons(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var daemons []any
	if err := json.NewDecoder(w.Body).Decode(&daemons); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(daemons) != 0 {
		t.Errorf("expected 0 daemons, got %d", len(daemons))
	}
}

func TestServerStartShutdown(t *testing.T) {
	srv := buildTestServer(t)
	srv.addr = "127.0.0.1:0" // random port

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown via context
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("server returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("server did not shut down within 3s")
	}
}
