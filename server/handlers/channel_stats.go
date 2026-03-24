package handlers

import (
	"net/http"

	"github.com/gh-curious-otter/bc/pkg/channel"
)

// ChannelStatsHandler handles /api/stats/channels routes.
type ChannelStatsHandler struct {
	svc *channel.ChannelService
}

// NewChannelStatsHandler creates a ChannelStatsHandler.
func NewChannelStatsHandler(svc *channel.ChannelService) *ChannelStatsHandler {
	return &ChannelStatsHandler{svc: svc}
}

// Register mounts channel stats routes on mux.
func (h *ChannelStatsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/stats/channels", h.stats)
}

func (h *ChannelStatsHandler) stats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		httpInternalError(w, "channel stats", err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
