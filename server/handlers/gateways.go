package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gh-curious-otter/bc/pkg/channel"
	"github.com/gh-curious-otter/bc/pkg/gateway"
	"github.com/gh-curious-otter/bc/pkg/notify"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// GatewayHandler handles /api/gateways routes.
type GatewayHandler struct {
	gw        *gateway.Manager
	ws        *workspace.Workspace
	chanSvc   *channel.ChannelService
	notifySvc *notify.Service
}

// NewGatewayHandler creates a GatewayHandler.
func NewGatewayHandler(gw *gateway.Manager, ws *workspace.Workspace) *GatewayHandler {
	return &GatewayHandler{gw: gw, ws: ws}
}

// SetChannelService sets the channel service for activity queries.
func (h *GatewayHandler) SetChannelService(svc *channel.ChannelService) {
	h.chanSvc = svc
}

// SetNotifyService sets the notification service for subscription management.
func (h *GatewayHandler) SetNotifyService(svc *notify.Service) {
	h.notifySvc = svc
}

// Register mounts gateway routes.
func (h *GatewayHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/gateways/activity", h.activity)
	mux.HandleFunc("/api/gateways", h.list)
	// New notify-powered endpoints for agent subscriptions
	mux.HandleFunc("/api/notify/subscriptions", h.notifySubscriptions)
	mux.HandleFunc("/api/notify/subscriptions/", h.notifySubscriptionByChannel)
	mux.HandleFunc("/api/notify/activity/", h.notifyActivity)
	mux.HandleFunc("/api/gateways/", h.byPlatform)
}

// gatewayStatus represents a gateway platform's config and runtime state.
type gatewayStatus struct {
	Platform string   `json:"platform"`
	Enabled  bool     `json:"enabled"`
	Channels []string `json:"channels"`
	Config   any      `json:"config,omitempty"` // platform-specific config (tokens redacted)
}

func (h *GatewayHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	platforms := []gatewayStatus{}

	if h.ws != nil && h.ws.Config != nil {
		gw := h.ws.Config.Gateways

		if gw.Telegram != nil {
			platforms = append(platforms, gatewayStatus{
				Platform: "telegram",
				Enabled:  gw.Telegram.Enabled,
				Config: map[string]any{
					"mode":      gw.Telegram.Mode,
					"has_token": gw.Telegram.BotToken != "",
				},
			})
		}
		if gw.Discord != nil {
			platforms = append(platforms, gatewayStatus{
				Platform: "discord",
				Enabled:  gw.Discord.Enabled,
				Config: map[string]any{
					"has_token": gw.Discord.BotToken != "",
				},
			})
		}
		if gw.Slack != nil {
			platforms = append(platforms, gatewayStatus{
				Platform: "slack",
				Enabled:  gw.Slack.Enabled,
				Config: map[string]any{
					"mode":          gw.Slack.Mode,
					"has_bot_token": gw.Slack.BotToken != "",
					"has_app_token": gw.Slack.AppToken != "",
				},
			})
		}
	}

	// Enrich with discovered channels
	if h.gw != nil {
		extChannels := h.gw.ExternalChannels()
		for i := range platforms {
			prefix := platforms[i].Platform + ":"
			for _, ch := range extChannels {
				if strings.HasPrefix(ch, prefix) {
					platforms[i].Channels = append(platforms[i].Channels, ch)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, platforms)
}

func (h *GatewayHandler) byPlatform(w http.ResponseWriter, r *http.Request) {
	platform := strings.TrimPrefix(r.URL.Path, "/api/gateways/")
	if platform == "" {
		httpError(w, "platform name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.updatePlatform(w, r, platform)
	default:
		methodNotAllowed(w)
	}
}

func (h *GatewayHandler) updatePlatform(w http.ResponseWriter, r *http.Request, platform string) {
	if h.ws == nil || h.ws.Config == nil {
		httpError(w, "workspace not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	cfg := h.ws.Config
	switch platform {
	case "telegram":
		if cfg.Gateways.Telegram == nil {
			cfg.Gateways.Telegram = &workspace.TelegramGatewayConfig{}
		}
		if err := json.Unmarshal(body, cfg.Gateways.Telegram); err != nil {
			httpError(w, "invalid telegram config: "+err.Error(), http.StatusBadRequest)
			return
		}
	case "discord":
		if cfg.Gateways.Discord == nil {
			cfg.Gateways.Discord = &workspace.DiscordGatewayConfig{}
		}
		if err := json.Unmarshal(body, cfg.Gateways.Discord); err != nil {
			httpError(w, "invalid discord config: "+err.Error(), http.StatusBadRequest)
			return
		}
	case "slack":
		if cfg.Gateways.Slack == nil {
			cfg.Gateways.Slack = &workspace.SlackGatewayConfig{}
		}
		if err := json.Unmarshal(body, cfg.Gateways.Slack); err != nil {
			httpError(w, "invalid slack config: "+err.Error(), http.StatusBadRequest)
			return
		}
	default:
		httpError(w, "unknown platform: "+platform, http.StatusBadRequest)
		return
	}

	if err := cfg.Save(workspace.ConfigPath(h.ws.RootDir)); err != nil {
		httpInternalError(w, "save config", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "platform": platform})
}

// activityEntry is a message from a gateway channel in the unified activity feed.
type activityEntry struct {
	Time     time.Time `json:"time"`
	Channel  string    `json:"channel"`
	Platform string    `json:"platform"`
	Sender   string    `json:"sender"`
	Content  string    `json:"content"`
}

// activity returns recent messages across all gateway channels as a unified feed.
// GET /api/gateways/activity?limit=50
func (h *GatewayHandler) activity(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	limit = clampInt(limit, 1, 200)

	if h.chanSvc == nil {
		writeJSON(w, http.StatusOK, []activityEntry{})
		return
	}

	// Get all gateway channels
	var gwChannelNames []string
	if h.gw != nil {
		gwChannelNames = h.gw.ExternalChannels()
	}
	if len(gwChannelNames) == 0 {
		writeJSON(w, http.StatusOK, []activityEntry{})
		return
	}

	// Collect recent messages from all gateway channels
	var entries []activityEntry
	for _, chName := range gwChannelNames {
		msgs, err := h.chanSvc.History(r.Context(), chName, channel.HistoryOpts{Limit: limit})
		if err != nil {
			continue
		}
		platform := "unknown"
		if idx := strings.Index(chName, ":"); idx > 0 {
			platform = chName[:idx]
		}
		for _, msg := range msgs {
			entries = append(entries, activityEntry{
				Time:     msg.CreatedAt,
				Channel:  chName,
				Platform: platform,
				Sender:   msg.Sender,
				Content:  msg.Content,
			})
		}
	}

	// Sort by time descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.After(entries[j].Time)
	})

	// Apply limit
	if len(entries) > limit {
		entries = entries[:limit]
	}

	writeJSON(w, http.StatusOK, entries)
}

// --- Notify-powered subscription endpoints ---

// notifySubscriptions handles GET/POST /api/notify/subscriptions
func (h *GatewayHandler) notifySubscriptions(w http.ResponseWriter, r *http.Request) {
	if h.notifySvc == nil {
		httpError(w, "notify service not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List all subscriptions
		subs, err := h.notifySvc.AllSubscriptions(r.Context())
		if err != nil {
			httpInternalError(w, "list subscriptions", err)
			return
		}
		if subs == nil {
			subs = []notify.Subscription{}
		}
		writeJSON(w, http.StatusOK, subs)

	case http.MethodPost:
		// Subscribe: {"channel": "slack:eng", "agent": "eng-01", "mention_only": false}
		var req struct {
			Channel     string `json:"channel"`
			Agent       string `json:"agent"`
			MentionOnly bool   `json:"mention_only"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Channel == "" || req.Agent == "" {
			httpError(w, "channel and agent are required", http.StatusBadRequest)
			return
		}
		if err := h.notifySvc.Subscribe(r.Context(), req.Channel, req.Agent, req.MentionOnly); err != nil {
			httpInternalError(w, "subscribe", err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "subscribed", "channel": req.Channel, "agent": req.Agent})

	default:
		methodNotAllowed(w)
	}
}

// notifySubscriptionByChannel handles operations on /api/notify/subscriptions/{channel}
// GET    — list subscribers for a channel
// DELETE — unsubscribe: ?agent=eng-01
// PATCH  — update: {"agent": "eng-01", "mention_only": true}
func (h *GatewayHandler) notifySubscriptionByChannel(w http.ResponseWriter, r *http.Request) {
	if h.notifySvc == nil {
		httpError(w, "notify service not available", http.StatusServiceUnavailable)
		return
	}

	// Extract channel from path: /api/notify/subscriptions/slack:eng
	channel := strings.TrimPrefix(r.URL.Path, "/api/notify/subscriptions/")
	if channel == "" {
		httpError(w, "channel name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		subs, err := h.notifySvc.ChannelSubscriptions(r.Context(), channel)
		if err != nil {
			httpInternalError(w, "list channel subscriptions", err)
			return
		}
		if subs == nil {
			subs = []notify.Subscription{}
		}
		writeJSON(w, http.StatusOK, subs)

	case http.MethodDelete:
		agent := r.URL.Query().Get("agent")
		if agent == "" {
			httpError(w, "agent query param required", http.StatusBadRequest)
			return
		}
		if err := h.notifySvc.Unsubscribe(r.Context(), channel, agent); err != nil {
			httpInternalError(w, "unsubscribe", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "unsubscribed", "channel": channel, "agent": agent})

	case http.MethodPatch:
		var req struct {
			Agent       string `json:"agent"`
			MentionOnly *bool  `json:"mention_only"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Agent == "" {
			httpError(w, "agent is required", http.StatusBadRequest)
			return
		}
		if req.MentionOnly != nil {
			if err := h.notifySvc.SetMentionOnly(r.Context(), channel, req.Agent, *req.MentionOnly); err != nil {
				httpInternalError(w, "set mention_only", err)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "channel": channel, "agent": req.Agent})

	default:
		methodNotAllowed(w)
	}
}

// notifyActivity handles GET /api/notify/activity/{channel}
func (h *GatewayHandler) notifyActivity(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.notifySvc == nil {
		httpError(w, "notify service not available", http.StatusServiceUnavailable)
		return
	}

	channel := strings.TrimPrefix(r.URL.Path, "/api/notify/activity/")
	if channel == "" {
		httpError(w, "channel name required", http.StatusBadRequest)
		return
	}

	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	limit = clampInt(limit, 1, 200)

	entries, err := h.notifySvc.ChannelActivity(r.Context(), channel, limit)
	if err != nil {
		httpInternalError(w, "channel activity", err)
		return
	}
	if entries == nil {
		entries = []notify.DeliveryEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}
