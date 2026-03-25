package handlers

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"github.com/gh-curious-otter/bc/pkg/agent"
	"github.com/gh-curious-otter/bc/pkg/log"
)

// TerminalHandler handles /api/agents/:name/terminal WebSocket connections.
// It bridges the browser to a tmux session via a PTY.
type TerminalHandler struct {
	svc      *agent.AgentService
	upgrader websocket.Upgrader
}

// NewTerminalHandler creates a TerminalHandler.
func NewTerminalHandler(svc *agent.AgentService) *TerminalHandler {
	return &TerminalHandler{
		svc: svc,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// Register mounts terminal routes on the agent sub-router.
// This is called from AgentHandler.byName for the "terminal" action.
// No separate mux registration needed — it's wired through the agent handler.

// HandleTerminal upgrades an HTTP request to a WebSocket and bridges it to a tmux session.
func (h *TerminalHandler) HandleTerminal(w http.ResponseWriter, r *http.Request, agentName string) {
	mgr := h.svc.Manager()

	// Verify agent exists and is running
	ag := mgr.GetAgent(agentName)
	if ag == nil {
		httpError(w, "agent not found", http.StatusNotFound)
		return
	}
	if ag.State == agent.StateStopped || ag.State == agent.StateError {
		httpError(w, "agent is not active", http.StatusConflict)
		return
	}

	// Get the runtime backend and build the attach command
	rt := mgr.RuntimeForAgent(agentName)
	if rt == nil {
		httpError(w, "no runtime backend for agent", http.StatusInternalServerError)
		return
	}
	if !rt.HasSession(r.Context(), agentName) {
		httpError(w, "no active session for agent", http.StatusConflict)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn("terminal: websocket upgrade failed", "agent", agentName, "error", err)
		return // Upgrade already sent error response
	}
	defer conn.Close()

	// Start tmux attach in a PTY
	cmd := rt.AttachCmd(r.Context(), agentName)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Warn("terminal: pty start failed", "agent", agentName, "error", err)
		writeWSError(conn, "failed to attach to session")
		return
	}
	defer func() {
		_ = ptmx.Close()  //nolint:errcheck // best-effort cleanup
		_ = cmd.Wait()     //nolint:errcheck // process cleanup
	}()

	// Set initial terminal size
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}) //nolint:errcheck

	log.Info("terminal: attached", "agent", agentName)

	var wg sync.WaitGroup

	// PTY → WebSocket (read from tmux, send to browser)
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// WebSocket → PTY (read from browser, send to tmux)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msgType, msg, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			switch msgType {
			case websocket.TextMessage:
				// Check for resize control messages: {"type":"resize","cols":80,"rows":24}
				if len(msg) > 0 && msg[0] == '{' && strings.Contains(string(msg), "resize") {
					handleResize(ptmx, msg)
					continue
				}
				// Regular text input
				if _, writeErr := ptmx.Write(msg); writeErr != nil {
					return
				}
			case websocket.BinaryMessage:
				if _, writeErr := ptmx.Write(msg); writeErr != nil {
					return
				}
			}
		}
	}()

	// Wait for either goroutine to exit (connection closed or process ended)
	wg.Wait()
	log.Info("terminal: detached", "agent", agentName)
}

// handleResize parses a resize message and updates the PTY size.
func handleResize(ptmx *os.File, msg []byte) {
	// Simple JSON parse without encoding/json to avoid import bloat.
	// Format: {"type":"resize","cols":80,"rows":24}
	s := string(msg)

	cols := parseJSONInt(s, "cols")
	rows := parseJSONInt(s, "rows")

	if cols > 0 && rows > 0 {
		_ = pty.Setsize(ptmx, &pty.Winsize{ //nolint:errcheck
			Rows: uint16(rows),
			Cols: uint16(cols),
		})
	}
}

// parseJSONInt extracts an integer value for a key from a simple JSON string.
func parseJSONInt(s, key string) int {
	idx := strings.Index(s, "\""+key+"\"")
	if idx < 0 {
		return 0
	}
	rest := s[idx+len(key)+3:] // skip past "key":
	rest = strings.TrimLeft(rest, " :")
	n := 0
	for _, c := range rest {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// writeWSError sends an error message over WebSocket before closing.
func writeWSError(conn *websocket.Conn, msg string) {
	deadline := time.Now().Add(5 * time.Second)
	_ = conn.WriteControl( //nolint:errcheck
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseInternalServerErr, msg),
		deadline,
	)
}
