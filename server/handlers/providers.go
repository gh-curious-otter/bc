package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/gh-curious-otter/bc/pkg/agent"
	"github.com/gh-curious-otter/bc/pkg/cost"
	"github.com/gh-curious-otter/bc/pkg/provider"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// ProviderInfo represents a provider with usage stats.
type ProviderInfo struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Binary       string  `json:"binary"`
	Command      string  `json:"command"`
	InstallHint  string  `json:"install_hint"`
	Version      string  `json:"version"`
	Status       string  `json:"status"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	AgentCount   int     `json:"agent_count"`
	Installed    bool    `json:"installed"`
	Enabled      bool    `json:"enabled"`
}

// ProviderDetail extends ProviderInfo with per-model cost breakdown and agent list.
type ProviderDetail struct {
	ProviderInfo
	Config      map[string]string `json:"config"`
	Agents      []AgentSummary    `json:"agents"`
	CostByModel []ModelCost       `json:"cost_by_model"`
}

// AgentSummary is a lightweight agent reference.
type AgentSummary struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	State string `json:"state"`
}

// ModelCost holds per-model cost data.
type ModelCost struct {
	Model        string  `json:"model"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// ProviderHandler handles /api/providers routes.
type ProviderHandler struct {
	registry *provider.Registry
	agents   *agent.AgentService
	costs    *cost.Store
	ws       *workspace.Workspace
}

// NewProviderHandler creates a ProviderHandler.
func NewProviderHandler(registry *provider.Registry, agents *agent.AgentService, costs *cost.Store, ws *workspace.Workspace) *ProviderHandler {
	return &ProviderHandler{registry: registry, agents: agents, costs: costs, ws: ws}
}

// Register mounts provider routes on mux.
func (h *ProviderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/providers", h.list)
	mux.HandleFunc("/api/providers/", h.byName)
}

// list returns all providers with agent counts and cost stats.
func (h *ProviderHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	providers := h.registry.List()
	agentCounts, agentsByProvider := h.countAgentsByProvider(r.Context())
	costByProvider := h.aggregateCostsByProvider(r.Context())

	infos := make([]ProviderInfo, 0, len(providers))
	for _, p := range providers {
		info := h.buildProviderInfo(r.Context(), p, agentCounts, costByProvider)
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	_ = agentsByProvider // used only in detail endpoint
	writeJSON(w, http.StatusOK, infos)
}

// byName handles /api/providers/:name and sub-routes.
func (h *ProviderHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/providers/"), "/", 2)
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if name == "" {
		httpError(w, "provider name required", http.StatusBadRequest)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		h.detail(w, r, name)
	case r.Method == http.MethodPost && action == "install":
		h.install(w, r, name)
	case r.Method == http.MethodPost && action == "update":
		h.update(w, r, name)
	case r.Method == http.MethodPatch && action == "config":
		h.patchConfig(w, r, name)
	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

// detail returns a single provider with agents and per-model costs.
func (h *ProviderHandler) detail(w http.ResponseWriter, r *http.Request, name string) {
	p, ok := h.registry.Get(name)
	if !ok {
		httpError(w, "unknown provider: "+name, http.StatusNotFound)
		return
	}

	agentCounts, agentsByProvider := h.countAgentsByProvider(r.Context())
	costByProvider := h.aggregateCostsByProvider(r.Context())

	info := h.buildProviderInfo(r.Context(), p, agentCounts, costByProvider)

	detail := ProviderDetail{
		ProviderInfo: info,
		Config:       h.providerConfig(name),
		Agents:       agentsByProvider[name],
		CostByModel:  h.costByModelForProvider(r.Context(), name),
	}

	if detail.Agents == nil {
		detail.Agents = []AgentSummary{}
	}
	if detail.CostByModel == nil {
		detail.CostByModel = []ModelCost{}
	}

	writeJSON(w, http.StatusOK, detail)
}

// install runs the provider's install hint command.
func (h *ProviderHandler) install(w http.ResponseWriter, r *http.Request, name string) {
	p, ok := h.registry.Get(name)
	if !ok {
		httpError(w, "unknown provider: "+name, http.StatusNotFound)
		return
	}

	hint := p.InstallHint()
	if hint == "" {
		httpError(w, "no install command available for "+name, http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":      "install_hint",
		"provider":    name,
		"install_cmd": hint,
	})
}

// update returns the upgrade command for the provider.
func (h *ProviderHandler) update(w http.ResponseWriter, r *http.Request, name string) {
	p, ok := h.registry.Get(name)
	if !ok {
		httpError(w, "unknown provider: "+name, http.StatusNotFound)
		return
	}

	hint := p.InstallHint()
	if hint == "" {
		httpError(w, "no update command available for "+name, http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "update_hint",
		"provider":   name,
		"update_cmd": hint,
	})
}

// patchConfig updates the provider's command in workspace settings.
func (h *ProviderHandler) patchConfig(w http.ResponseWriter, r *http.Request, name string) {
	if h.ws == nil || h.ws.Config == nil {
		httpError(w, "workspace not available", http.StatusServiceUnavailable)
		return
	}

	_, ok := h.registry.Get(name)
	if !ok {
		httpError(w, "unknown provider: "+name, http.StatusNotFound)
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.ws.Config.Providers.Providers == nil {
		h.ws.Config.Providers.Providers = make(map[string]workspace.ProviderConfig)
	}
	h.ws.Config.Providers.Providers[name] = workspace.ProviderConfig{Command: req.Command}

	if err := h.ws.Save(); err != nil {
		httpInternalError(w, "save config", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "provider": name, "command": req.Command})
}

// buildProviderInfo builds a ProviderInfo from a provider and pre-computed maps.
func (h *ProviderHandler) buildProviderInfo(
	ctx context.Context,
	p provider.Provider,
	agentCounts map[string]int,
	costByProvider map[string]*costAgg,
) ProviderInfo {
	installed := p.IsInstalled(ctx)
	version := ""
	if installed {
		version = p.Version(ctx)
	}

	status := "not_installed"
	if installed {
		status = "healthy"
	}

	command := p.Command()
	if h.ws != nil && h.ws.Config != nil {
		if cfg := h.ws.Config.GetProvider(p.Name()); cfg != nil {
			command = cfg.Command
		}
	}

	enabled := installed
	if h.ws != nil && h.ws.Config != nil {
		_, enabled = h.ws.Config.Providers.Providers[p.Name()]
		if !enabled {
			enabled = installed
		}
	}

	info := ProviderInfo{
		Name:        p.Name(),
		Description: p.Description(),
		Binary:      p.Binary(),
		Command:     command,
		InstallHint: p.InstallHint(),
		Version:     version,
		Status:      status,
		AgentCount:  agentCounts[p.Name()],
		Installed:   installed,
		Enabled:     enabled,
	}

	if agg, ok := costByProvider[p.Name()]; ok {
		info.TotalTokens = agg.tokens
		info.TotalCostUSD = agg.cost
	}

	return info
}

// countAgentsByProvider counts agents per provider tool name.
func (h *ProviderHandler) countAgentsByProvider(ctx context.Context) (map[string]int, map[string][]AgentSummary) {
	counts := make(map[string]int)
	byProvider := make(map[string][]AgentSummary)

	if h.agents == nil {
		return counts, byProvider
	}

	agents, err := h.agents.List(ctx, agent.ListOptions{})
	if err != nil {
		return counts, byProvider
	}

	for _, a := range agents {
		tool := strings.ToLower(a.Tool)
		if tool == "" {
			continue
		}
		counts[tool]++
		byProvider[tool] = append(byProvider[tool], AgentSummary{
			Name:  a.Name,
			Role:  string(a.Role),
			State: string(a.State),
		})
	}

	return counts, byProvider
}

// costAgg holds aggregated cost data.
type costAgg struct {
	tokens int64
	cost   float64
}

// aggregateCostsByProvider groups model costs by provider name.
// A model belongs to a provider if the model name contains the provider name.
func (h *ProviderHandler) aggregateCostsByProvider(ctx context.Context) map[string]*costAgg {
	result := make(map[string]*costAgg)
	if h.costs == nil {
		return result
	}

	summaries, err := h.costs.SummaryByModel(ctx)
	if err != nil {
		return result
	}

	providers := h.registry.List()
	for _, s := range summaries {
		model := strings.ToLower(s.Model)
		for _, p := range providers {
			if strings.Contains(model, strings.ToLower(p.Name())) {
				agg, ok := result[p.Name()]
				if !ok {
					agg = &costAgg{}
					result[p.Name()] = agg
				}
				agg.tokens += s.TotalTokens
				agg.cost += s.TotalCostUSD
				break
			}
		}
	}

	return result
}

// costByModelForProvider returns per-model costs for a specific provider.
func (h *ProviderHandler) costByModelForProvider(ctx context.Context, name string) []ModelCost {
	if h.costs == nil {
		return nil
	}

	summaries, err := h.costs.SummaryByModel(ctx)
	if err != nil {
		return nil
	}

	var models []ModelCost
	lowerName := strings.ToLower(name)
	for _, s := range summaries {
		if strings.Contains(strings.ToLower(s.Model), lowerName) {
			models = append(models, ModelCost{
				Model:        s.Model,
				TotalTokens:  s.TotalTokens,
				TotalCostUSD: s.TotalCostUSD,
			})
		}
	}

	return models
}

// providerConfig returns the workspace config for a provider as a string map.
func (h *ProviderHandler) providerConfig(name string) map[string]string {
	cfg := make(map[string]string)
	if h.ws == nil || h.ws.Config == nil {
		return cfg
	}

	if p := h.ws.Config.GetProvider(name); p != nil {
		cfg["command"] = p.Command
	}

	if h.ws.Config.Providers.Default == name {
		cfg["default"] = "true"
	}

	return cfg
}
