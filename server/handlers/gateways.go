package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gh-curious-otter/bc/pkg/gateway"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// GatewayHandler handles /api/gateways routes.
type GatewayHandler struct {
	gw *gateway.Manager
	ws *workspace.Workspace
}

// NewGatewayHandler creates a GatewayHandler.
func NewGatewayHandler(gw *gateway.Manager, ws *workspace.Workspace) *GatewayHandler {
	return &GatewayHandler{gw: gw, ws: ws}
}

// Register mounts gateway routes.
func (h *GatewayHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/gateways", h.list)
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
