package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rpuneet/bc/pkg/channel"
)

// ChannelHandler handles /api/channels routes.
type ChannelHandler struct {
	svc *channel.ChannelService
}

// NewChannelHandler creates a ChannelHandler.
func NewChannelHandler(svc *channel.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

// Register mounts channel routes on mux.
func (h *ChannelHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/channels", h.list)
	mux.HandleFunc("/api/channels/", h.byName)
}

func (h *ChannelHandler) list(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		channels, err := h.svc.List(r.Context())
		if err != nil {
			httpError(w, "list channels: "+err.Error(), http.StatusInternalServerError)
			return
		}
		limit, offset := parsePagination(r, 50)
		if offset >= len(channels) {
			channels = channels[:0]
		} else {
			channels = channels[offset:]
			if len(channels) > limit {
				channels = channels[:limit]
			}
		}
		writeJSON(w, http.StatusOK, channels)

	case http.MethodPost:
		var req channel.CreateChannelReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		ch, err := h.svc.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, channel.ErrChannelExists) {
				httpError(w, err.Error(), http.StatusConflict)
				return
			}
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, ch)

	default:
		methodNotAllowed(w)
	}
}

// byName handles /api/channels/<name>[/<subresource>]
func (h *ChannelHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/channels/"), "/", 2)
	name := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if name == "" {
		httpError(w, "channel name required", http.StatusBadRequest)
		return
	}

	switch sub {
	case "":
		h.channel(w, r, name)
	case "history":
		h.history(w, r, name)
	case "messages":
		h.postMessage(w, r, name)
	case "members":
		h.members(w, r, name)
	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (h *ChannelHandler) channel(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		ch, err := h.svc.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, ch)

	case http.MethodPatch:
		var req channel.UpdateChannelReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		ch, err := h.svc.Update(r.Context(), name, req)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, ch)

	case http.MethodDelete:
		if err := h.svc.Delete(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w)
	}
}

func (h *ChannelHandler) history(w http.ResponseWriter, r *http.Request, name string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	q := r.URL.Query()
	opts := channel.HistoryOpts{Limit: 50}
	if s := q.Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	opts.Limit = clampInt(opts.Limit, 1, 1000)
	if s := q.Get("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			opts.Offset = n
		}
	}
	opts.Offset = clampInt(opts.Offset, 0, 100000)
	msgs, err := h.svc.History(r.Context(), name, opts)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *ChannelHandler) postMessage(w http.ResponseWriter, r *http.Request, name string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Sender  string `json:"sender"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	msg, err := h.svc.Send(r.Context(), name, req.Sender, req.Content)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (h *ChannelHandler) members(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			AgentID string `json:"agent_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := h.svc.AddMember(r.Context(), name, req.AgentID); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			httpError(w, "agent_id required", http.StatusBadRequest)
			return
		}
		if err := h.svc.RemoveMember(r.Context(), name, agentID); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w)
	}
}
