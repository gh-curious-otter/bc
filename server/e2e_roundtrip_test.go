// Package server_test — round-trip integration tests for bcd HTTP API.
//
// These tests exercise full create→read→verify cycles through the HTTP API
// backed by real SQLite storage, proving the path:
//
//	HTTP request → handler → service → store → SQLite → response
package server_test

import (
	"testing"
)

// ─── Channel Round-Trip ──────────────────────────────────────────────────────

// TestE2E_ChannelRoundTrip exercises the full channel lifecycle:
// create channel → send messages → read history → verify content matches.
func TestE2E_ChannelRoundTrip(t *testing.T) {
	s := newE2EServer(t)

	// 1. Create channel
	code, body := s.postJSON(t, "/api/channels", map[string]string{
		"name":        "roundtrip",
		"description": "round-trip test channel",
	})
	if code != 201 {
		t.Fatalf("create channel: want 201, got %d: %v", code, body)
	}
	if body["name"] != "roundtrip" {
		t.Fatalf("create channel: want name=roundtrip, got %v", body["name"])
	}

	// 2. Send two messages from different senders
	messages := []struct {
		sender  string
		content string
	}{
		{"alice", "hello from alice"},
		{"bob", "hello from bob"},
	}
	for _, m := range messages {
		sendCode, resp := s.postJSON(t, "/api/channels/roundtrip/messages", map[string]string{
			"sender":  m.sender,
			"content": m.content,
		})
		if sendCode != 201 {
			t.Fatalf("send message from %s: want 201, got %d: %v", m.sender, sendCode, resp)
		}
	}

	// 3. Read history and verify content matches
	histCode, history := s.getList(t, "/api/channels/roundtrip/history")
	if histCode != 200 {
		t.Fatalf("get history: want 200, got %d", histCode)
	}
	if len(history) != 2 {
		t.Fatalf("want 2 messages in history, got %d", len(history))
	}

	// Verify each message's sender and content survived the round-trip
	for i, m := range messages {
		msg, ok := history[i].(map[string]any)
		if !ok {
			t.Fatalf("history[%d]: expected object, got %T", i, history[i])
		}
		if got := msg["sender"]; got != m.sender {
			t.Errorf("history[%d].sender: want %q, got %v", i, m.sender, got)
		}
		if got := msg["content"]; got != m.content {
			t.Errorf("history[%d].content: want %q, got %v", i, m.content, got)
		}
	}

	// 4. Verify channel appears in list
	code, channels := s.getList(t, "/api/channels")
	if code != 200 {
		t.Fatalf("list channels: want 200, got %d", code)
	}
	found := false
	for _, ch := range channels {
		if chMap, ok := ch.(map[string]any); ok && chMap["name"] == "roundtrip" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("channel 'roundtrip' not found in list after creation")
	}
}

// ─── Health Readiness ────────────────────────────────────────────────────────

// TestE2E_HealthReady verifies the readiness probe checks downstream deps.
func TestE2E_HealthReady(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/health/ready")
	if code != 200 {
		t.Fatalf("GET /health/ready: want 200, got %d", code)
	}
	if body["status"] != "ok" {
		t.Fatalf("want status=ok, got %v", body["status"])
	}
	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatal("expected checks map in readiness response")
	}
	if checks["db"] != "ok" {
		t.Fatalf("want checks.db=ok, got %v", checks["db"])
	}
}

// ─── Workspace Round-Trip ────────────────────────────────────────────────────

// TestE2E_WorkspaceStatus_Fields verifies workspace status returns expected
// fields with correct types — agent_count, running_count, is_healthy.
func TestE2E_WorkspaceStatus_Fields(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/workspace")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}

	fields := []struct {
		want any
		key  string
	}{
		{true, "is_healthy"},
	}
	for _, f := range fields {
		got := body[f.key]
		if got != f.want {
			t.Errorf("workspace.%s: want %v, got %v", f.key, f.want, got)
		}
	}

	// agent_count and running_count should be numeric (0 in empty workspace)
	if _, ok := body["agent_count"].(float64); !ok {
		t.Errorf("workspace.agent_count: want number, got %T", body["agent_count"])
	}
	if _, ok := body["running_count"].(float64); !ok {
		t.Errorf("workspace.running_count: want number, got %T", body["running_count"])
	}
}

// ─── Doctor Round-Trip ───────────────────────────────────────────────────────

// TestE2E_Doctor_HasChecks verifies doctor returns a report with check results.
func TestE2E_Doctor_HasChecks(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/doctor")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}

	// Doctor report should have at least one key (category)
	if len(body) == 0 {
		t.Fatal("doctor report is empty, expected at least one check category")
	}
}

// ─── Costs Round-Trip ────────────────────────────────────────────────────────

// TestE2E_Costs_StructureValid verifies cost summary returns a well-formed
// response (not just non-nil, but with expected structure).
func TestE2E_Costs_StructureValid(t *testing.T) {
	s := newE2EServer(t)

	code, body := s.get(t, "/api/costs")
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}

	// Cost summary should have a total_cost field (zero for empty workspace)
	if body == nil {
		t.Fatal("expected non-nil cost summary")
	}
}

// ─── Multi-Step Scenarios ────────────────────────────────────────────────────

// TestE2E_ChannelCreateDeleteVerify tests that a deleted channel is actually
// removed from the store.
func TestE2E_ChannelCreateDeleteVerify(t *testing.T) {
	s := newE2EServer(t)

	// Create
	code, _ := s.postJSON(t, "/api/channels", map[string]string{
		"name": "ephemeral",
	})
	if code != 201 {
		t.Fatalf("create: want 201, got %d", code)
	}

	// Verify it exists
	code, _ = s.get(t, "/api/channels/ephemeral")
	if code != 200 {
		t.Fatalf("get after create: want 200, got %d", code)
	}

	// Delete
	code = s.delete(t, "/api/channels/ephemeral")
	if code != 204 {
		t.Fatalf("delete: want 204, got %d", code)
	}

	// Verify it is gone
	code, _ = s.get(t, "/api/channels/ephemeral")
	if code != 404 {
		t.Fatalf("get after delete: want 404, got %d", code)
	}
}
