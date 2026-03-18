package ws

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sseGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestHub_Publish_NoClients(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	// Should not block
	h.Publish("test.event", map[string]any{"key": "val"})
}

func TestHub_ClientCount(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	if h.ClientCount() != 0 {
		t.Fatalf("want 0 clients, got %d", h.ClientCount())
	}
}

func TestHub_Stop_IdempotentNoPanic(t *testing.T) {
	h := NewHub()
	go h.Run()
	h.Stop()
	h.Stop() // must not panic
}

func TestHub_ServeHTTP_ConnectedEvent(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	go func() {
		resp := sseGet(t, srv.URL)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				payload := strings.TrimPrefix(line, "data: ")
				var evt Event
				if err := json.Unmarshal([]byte(payload), &evt); err == nil && evt.Type == "connected" {
					close(done)
					return
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for connected event")
	}
}

func TestHub_EventDelivery(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	connected := make(chan struct{})

	go func() {
		resp := sseGet(t, srv.URL)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		firstSeen := false
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			var evt Event
			if err := json.Unmarshal([]byte(payload), &evt); err != nil {
				continue
			}
			if !firstSeen {
				firstSeen = true
				close(connected)
				continue
			}
			received <- evt
			return
		}
	}()

	select {
	case <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("client did not connect in time")
	}

	time.Sleep(10 * time.Millisecond)
	h.Publish("agent.started", map[string]any{"name": "foo"})

	select {
	case evt := <-received:
		if evt.Type != "agent.started" {
			t.Fatalf("want type agent.started, got %s", evt.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}
}
