package handlers

import (
	"net/http"

	"github.com/gh-curious-otter/bc/pkg/gateway"
)

// GatewayHandler handles /api/gateways routes.
type GatewayHandler struct {
	gw *gateway.Manager
}

// NewGatewayHandler creates a GatewayHandler.
func NewGatewayHandler(gw *gateway.Manager) *GatewayHandler {
	return &GatewayHandler{gw: gw}
}

// Register mounts gateway routes.
func (h *GatewayHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/gateways", h.list)
}

func (h *GatewayHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels := h.gw.ExternalChannels()
	writeJSON(w, http.StatusOK, map[string]any{
		"channels": channels,
	})
}
