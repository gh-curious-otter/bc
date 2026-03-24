// Package server_test provides E2E tests for the bcd HTTP API.
//
// These tests spin up a full bcd server in-process using httptest,
// backed by real SQLite databases in a temp directory. No external
// services or running daemon required — suitable for CI.
package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-curious-otter/bc/pkg/agent"
	"github.com/gh-curious-otter/bc/pkg/channel"
	"github.com/gh-curious-otter/bc/pkg/cost"
	"github.com/gh-curious-otter/bc/pkg/cron"

	"github.com/gh-curious-otter/bc/pkg/events"
	pkgmcp "github.com/gh-curious-otter/bc/pkg/mcp"
	"github.com/gh-curious-otter/bc/pkg/tool"
	"github.com/gh-curious-otter/bc/pkg/workspace"
	"github.com/gh-curious-otter/bc/server"
	"github.com/gh-curious-otter/bc/server/ws"
)

// ─── Test Harness ────────────────────────────────────────────────────────────

// e2eServer is a fully wired bcd test server backed by real stores.
type e2eServer struct {
	*httptest.Server
	ws *workspace.Workspace
}

// newE2EServer creates a bcd server with all services wired to temp SQLite DBs.
func newE2EServer(t *testing.T) *e2eServer {
	t.Helper()

	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "roles"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatal(err)
	}

	// Write minimal valid workspace config
	cfg := `[workspace]
name = "e2e-test"
version = 2

[providers]
default = "gemini"

[providers.gemini]
command = "echo test"
enabled = true
`
	if err := os.WriteFile(filepath.Join(bcDir, "settings.toml"), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("workspace load: %v", err)
	}

	// SSE hub
	hub := ws_hub(t)

	// Agent service (no runtime backend — just state management)
	mgr := agent.NewWorkspaceManager(ws.StateDir(), ws.RootDir)
	_ = mgr.LoadState()
	agentSvc := agent.NewAgentService(mgr, hub, nil)

	// Channel service
	var channelSvc *channel.ChannelService
	if chStore, err := channel.OpenStore(ws.RootDir); err == nil {
		channelSvc = channel.NewChannelService(chStore)
		t.Cleanup(func() { _ = chStore.Close() })
	}

	// Cost store
	var costStore *cost.Store
	cs := cost.NewStore(ws.RootDir)
	if err := cs.Open(); err == nil {
		costStore = cs
		t.Cleanup(func() { _ = cs.Close() })
	}

	// Cron store
	var cronStore *cron.Store
	if cr, err := cron.Open(ws.RootDir); err == nil {
		cronStore = cr
		t.Cleanup(func() { _ = cr.Close() })
	}

	// MCP store
	var mcpStore *pkgmcp.Store
	if ms, err := pkgmcp.NewStore(ws.RootDir); err == nil {
		mcpStore = ms
		t.Cleanup(func() { _ = ms.Close() })
	}

	// Tool store
	var toolStore *tool.Store
	ts := tool.NewStore(ws.StateDir())
	if err := ts.Open(); err == nil {
		toolStore = ts
		t.Cleanup(func() { _ = ts.Close() })
	}

	// Event log
	var eventLog events.EventStore
	if el, err := events.NewSQLiteLog(filepath.Join(ws.StateDir(), "state.db")); err == nil {
		eventLog = el
		t.Cleanup(func() { _ = el.Close() })
	}

	svc := server.Services{
		Agents:   agentSvc,
		Channels: channelSvc,
		Costs:    costStore,
		Cron:     cronStore,
		MCP:      mcpStore,
		Tools:    toolStore,
		EventLog: eventLog,
		WS:       ws,
	}

	srvCfg := server.Config{Addr: "127.0.0.1:0", CORS: true}
	srv := server.New(srvCfg, svc, hub, nil)
	ts2 := httptest.NewServer(srv.Handler())
	t.Cleanup(ts2.Close)

	return &e2eServer{Server: ts2, ws: ws}
}

func ws_hub(t *testing.T) *ws.Hub {
	t.Helper()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)
	return hub
}

// ─── HTTP helpers ────────────────────────────────────────────────────────────

func (s *e2eServer) get(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", s.URL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	return resp.StatusCode, result
}

func (s *e2eServer) getList(t *testing.T, path string) (int, []any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", s.URL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result []any
	_ = json.Unmarshal(body, &result)
	return resp.StatusCode, result
}

func (s *e2eServer) postJSON(t *testing.T, path string, payload any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", s.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	return resp.StatusCode, result
}

func (s *e2eServer) patchJSON(t *testing.T, path string, payload any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), "PATCH", s.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	return resp.StatusCode, result
}

func (s *e2eServer) delete(t *testing.T, path string) int {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", s.URL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// ─── Health ──────────────────────────────────────────────────────────────────

func TestE2E_Health(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/health")
	if code != 200 {
		t.Fatalf("GET /health: want 200, got %d", code)
	}
	if body["status"] != "ok" {
		t.Fatalf("want status=ok, got %v", body["status"])
	}
}

// ─── Agents ──────────────────────────────────────────────────────────────────

func TestE2E_Agents_ListEmpty(t *testing.T) {
	s := newE2EServer(t)

	code, agents := s.getList(t, "/api/agents")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if len(agents) != 0 {
		t.Fatalf("want 0 agents, got %d", len(agents))
	}
}

func TestE2E_Agents_GetNotFound(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/agents/nonexistent")
	if code != 404 {
		t.Fatalf("want 404, got %d", code)
	}
	if body["error"] == nil {
		t.Fatal("expected error message")
	}
}

func TestE2E_Agents_DeleteNotFound(t *testing.T) {
	s := newE2EServer(t)

	code := s.delete(t, "/api/agents/nonexistent")
	if code != 400 {
		t.Fatalf("want 400, got %d", code)
	}
}

func TestE2E_Agents_GenerateName(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/agents/generate-name")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	name, ok := body["name"].(string)
	if !ok || name == "" {
		t.Fatalf("want non-empty name, got %v", body)
	}
}

func TestE2E_Agents_StopAll(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.postJSON(t, "/api/agents/stop-all", nil)
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if body["stopped"] == nil {
		t.Fatal("expected stopped count")
	}
}

// ─── Channels ────────────────────────────────────────────────────────────────

func TestE2E_Channels_ListDefault(t *testing.T) {
	s := newE2EServer(t)

	code, channels := s.getList(t, "/api/channels")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	// No default channels — starts empty
	if len(channels) != 0 {
		t.Fatalf("want 0 channels (no defaults), got %d", len(channels))
	}
}

func TestE2E_Channels_CreateAndGet(t *testing.T) {
	s := newE2EServer(t)

	// Create
	code, body := s.postJSON(t, "/api/channels", map[string]string{
		"name":        "test-channel",
		"description": "E2E test channel",
	})
	if code != 201 {
		t.Fatalf("want 201, got %d: %v", code, body)
	}

	// Get
	code, body = s.get(t, "/api/channels/test-channel")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if body["name"] != "test-channel" {
		t.Fatalf("want name=test-channel, got %v", body["name"])
	}
}

func TestE2E_Channels_SendMessage(t *testing.T) {
	s := newE2EServer(t)

	// Create channel first (no defaults exist)
	s.postJSON(t, "/api/channels", map[string]string{"name": "general"})

	// Send to channel
	code, _ := s.postJSON(t, "/api/channels/general/messages", map[string]string{
		"sender":  "test-agent",
		"content": "hello from e2e",
	})
	if code != 201 {
		t.Fatalf("want 201, got %d", code)
	}

	// Verify in history
	code, history := s.getList(t, "/api/channels/general/history")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if len(history) == 0 {
		t.Fatal("expected at least 1 message in history")
	}
}

func TestE2E_Channels_Delete(t *testing.T) {
	s := newE2EServer(t)

	// Create then delete
	s.postJSON(t, "/api/channels", map[string]string{"name": "to-delete"})

	code := s.delete(t, "/api/channels/to-delete")
	if code != 204 {
		t.Fatalf("want 204, got %d", code)
	}
}

// ─── Costs ───────────────────────────────────────────────────────────────────

func TestE2E_Costs_Summary(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/costs")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	// Empty workspace — should return valid structure with zero costs
	if body == nil {
		t.Fatal("expected cost summary body")
	}
}

// ─── Cron ────────────────────────────────────────────────────────────────────

func TestE2E_Cron_ListEmpty(t *testing.T) {
	s := newE2EServer(t)

	code, jobs := s.getList(t, "/api/cron")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if len(jobs) != 0 {
		t.Fatalf("want 0 cron jobs, got %d", len(jobs))
	}
}

// ─── Tools ───────────────────────────────────────────────────────────────────

func TestE2E_Tools_List(t *testing.T) {
	s := newE2EServer(t)

	code, tools := s.getList(t, "/api/tools")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	// Default workspace has provider tools registered
	_ = tools // may be empty or populated depending on config
}

// ─── MCP Servers ─────────────────────────────────────────────────────────────

func TestE2E_MCP_ListEmpty(t *testing.T) {
	s := newE2EServer(t)

	code, servers := s.getList(t, "/api/mcp")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if len(servers) != 0 {
		t.Fatalf("want 0 MCP servers, got %d", len(servers))
	}
}

// ─── Events ──────────────────────────────────────────────────────────────────

func TestE2E_Events_List(t *testing.T) {
	s := newE2EServer(t)

	code, events := s.getList(t, "/api/logs?tail=10")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	_ = events // empty is fine
}

// ─── Workspace ───────────────────────────────────────────────────────────────

func TestE2E_Workspace_Status(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/workspace")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if body["name"] != "e2e-test" {
		t.Fatalf("want name=e2e-test, got %v", body["name"])
	}
}

func TestE2E_Workspace_Roles(t *testing.T) {
	s := newE2EServer(t)

	code, _ := s.getList(t, "/api/workspace/roles")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
}

// ─── Doctor ──────────────────────────────────────────────────────────────────

func TestE2E_Doctor(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/doctor")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if body == nil {
		t.Fatal("expected doctor report")
	}
}

// ─── Error Cases ─────────────────────────────────────────────────────────────

func TestE2E_NotFound(t *testing.T) {
	s := newE2EServer(t)

	code, _ := s.get(t, "/api/nonexistent")
	// Should return 404 or fall through to SPA handler
	if code != 404 && code != 200 {
		t.Fatalf("want 404 or 200 (SPA fallback), got %d", code)
	}
}

func TestE2E_MethodNotAllowed(t *testing.T) {
	s := newE2EServer(t)

	code, _ := s.get(t, "/health")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}

	// POST to health should be 405
	code, body := s.postJSON(t, "/health", nil)
	if code != 405 {
		t.Fatalf("want 405, got %d: %v", code, body)
	}
}

// ─── MCP SSE ─────────────────────────────────────────────────────────────────

func TestE2E_MCP_SSE_ContentType(t *testing.T) {
	s := newE2EServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", s.URL+"/mcp/sse", nil)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /mcp/sse: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Fatalf("want Content-Type text/event-stream, got %q", ct)
	}

	// Read the first event — should be the endpoint event
	buf := make([]byte, 512)
	n, readErr := resp.Body.Read(buf)
	if readErr != nil {
		t.Fatalf("reading SSE body: %v", readErr)
	}
	data := string(buf[:n])
	if !bytes.Contains(buf[:n], []byte("event: endpoint")) {
		t.Fatalf("expected endpoint event, got: %s", data)
	}
}

// ─── Settings PATCH ──────────────────────────────────────────────────────────

func TestE2E_Settings_PatchUser(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.patchJSON(t, "/api/settings/user", map[string]string{
		"Nickname": "@test",
	})
	if code != 200 {
		t.Fatalf("want 200, got %d: %v", code, body)
	}

	// The response is the full config — check user section
	userRaw, ok := body["User"].(map[string]any)
	if !ok {
		t.Fatalf("expected User section in response, got %v", body)
	}
	if userRaw["Nickname"] != "@test" {
		t.Fatalf("want Nickname=@test, got %v", userRaw["Nickname"])
	}
}

func TestE2E_Settings_PatchUnknownSection(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.patchJSON(t, "/api/settings/nonexistent", map[string]string{
		"foo": "bar",
	})
	if code != 400 {
		t.Fatalf("want 400, got %d: %v", code, body)
	}
	if body["error"] == nil {
		t.Fatal("expected error message")
	}
}

// ─── CORS ────────────────────────────────────────────────────────────────────

func TestE2E_CORS_Headers(t *testing.T) {
	s := newE2EServer(t)

	req, _ := http.NewRequestWithContext(context.Background(), "OPTIONS", s.URL+"/api/agents", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Fatalf("OPTIONS want 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS allow-origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("missing CORS allow-methods header")
	}
}
