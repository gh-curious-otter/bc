package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServeSSE starts an HTTP server that implements the MCP SSE transport.
//
// Endpoints:
//   - GET  /sse      — client connects; receives server→client events as SSE
//   - POST /message  — client sends JSON-RPC requests; response sent via SSE
//
// addr must be a host:port pair. If addr is a bare ":port" it is rewritten
// to "127.0.0.1:port" so the server only listens on localhost — never on all
// interfaces — which prevents accidental network exposure.
//
// The server shuts down cleanly when ctx is canceled.
func (s *Server) ServeSSE(ctx context.Context, addr string) error {
	addr = LocalhostAddr(addr)

	broker := NewSSEBroker()
	s.SetBroker(broker)

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", broker.handleSSE)
	mux.HandleFunc("/message", s.HandleSSEMessage(ctx, broker))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","server":"bc-mcp","version":%q}`, s.version) //nolint:errcheck // writing to response
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Shut down when ctx is canceled
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background()) //nolint:contextcheck
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("SSE server error: %w", err)
	}
	return nil
}

// HandleSSEMessage processes POST /message — client→server direction.
// Exported so tests can mount it on their own ServeMux.
func (s *Server) HandleSSEMessage(ctx context.Context, broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			resp := errResponse(nil, ErrParse, "parse error: "+err.Error())
			broker.send(resp)
			w.WriteHeader(http.StatusAccepted)
			return
		}

		resp := s.Handle(ctx, req)

		// Notifications have no ID — no response to send
		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		broker.send(resp)
		w.WriteHeader(http.StatusAccepted)
	}
}

// ─── SSE broker ───────────────────────────────────────────────────────────────

// SSEBroker fans out SSE messages to all connected clients.
type SSEBroker struct {
	clients         map[chan []byte]struct{}
	messageEndpoint string
	mu              sync.Mutex
}

// NewSSEBroker creates an SSEBroker ready to use.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{clients: make(map[chan []byte]struct{}), messageEndpoint: "/message"}
}

func (b *SSEBroker) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *SSEBroker) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *SSEBroker) send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	msg := append([]byte("data: "), data...)
	msg = append(msg, '\n', '\n')

	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default: // Drop if the client is slow
		}
	}
}

// SSEHandler returns an http.HandlerFunc for the SSE endpoint.
// Exported so tests in mcp_test can mount it on their own ServeMux.
func (b *SSEBroker) SSEHandler() http.HandlerFunc {
	return b.handleSSE
}

// handleSSE streams server→client events over SSE.
func (b *SSEBroker) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	// Send endpoint event so client knows where to POST
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", b.messageEndpoint) //nolint:errcheck // writing to response
	flusher.Flush()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg) //nolint:errcheck
			flusher.Flush()
		case <-keepalive.C:
			// SSE comment line — prevents idle timeout, ignored by clients
			fmt.Fprint(w, ": keepalive\n\n") //nolint:errcheck // writing to response
			flusher.Flush()
		}
	}
}

// MountOn registers MCP SSE endpoints on an existing ServeMux under the given prefix.
// This allows embedding the MCP server into bcd's HTTP server.
func MountOn(mux *http.ServeMux, srv *Server, prefix string) {
	broker := NewSSEBroker()
	broker.messageEndpoint = prefix + "/message"
	srv.SetBroker(broker)
	mux.HandleFunc(prefix+"/sse", broker.handleSSE)
	mux.HandleFunc(prefix+"/message", srv.HandleSSEMessage(context.Background(), broker))
}

// LocalhostAddr rewrites a bare ":port" address to "127.0.0.1:port".
// Explicit host addresses (e.g. "0.0.0.0:8811") are returned unchanged so
// callers that deliberately want to bind all interfaces can still do so.
func LocalhostAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}
