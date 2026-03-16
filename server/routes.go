package server

import "net/http"

// routes builds the HTTP handler with all API routes and middleware.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", s.handleHealth)

	// Agents
	mux.HandleFunc("GET /api/v1/agents", s.handleAgentList)
	mux.HandleFunc("POST /api/v1/agents", s.handleAgentCreate)
	mux.HandleFunc("GET /api/v1/agents/{name}", s.handleAgentGet)
	mux.HandleFunc("DELETE /api/v1/agents/{name}", s.handleAgentDelete)
	mux.HandleFunc("POST /api/v1/agents/{name}/start", s.handleAgentStart)
	mux.HandleFunc("POST /api/v1/agents/{name}/stop", s.handleAgentStop)
	mux.HandleFunc("GET /api/v1/agents/{name}/peek", s.handleAgentPeek)
	mux.HandleFunc("POST /api/v1/agents/{name}/send", s.handleAgentSend)

	// Channels
	mux.HandleFunc("GET /api/v1/channels", s.handleChannelList)
	mux.HandleFunc("POST /api/v1/channels", s.handleChannelCreate)
	mux.HandleFunc("GET /api/v1/channels/{name}", s.handleChannelGet)
	mux.HandleFunc("DELETE /api/v1/channels/{name}", s.handleChannelDelete)
	mux.HandleFunc("POST /api/v1/channels/{name}/send", s.handleChannelSend)
	mux.HandleFunc("GET /api/v1/channels/{name}/history", s.handleChannelHistory)

	// Costs
	mux.HandleFunc("GET /api/v1/costs", s.handleCostSummary)
	mux.HandleFunc("GET /api/v1/costs/agents/{name}", s.handleCostByAgent)
	mux.HandleFunc("GET /api/v1/costs/budget", s.handleCostBudget)

	// Workspace
	mux.HandleFunc("GET /api/v1/workspace", s.handleWorkspaceStatus)
	mux.HandleFunc("GET /api/v1/workspace/config", s.handleWorkspaceConfig)

	// Events
	mux.HandleFunc("GET /api/v1/events", s.handleEventList)
	mux.HandleFunc("GET /api/v1/events/stream", s.handleEventStream)

	// Roles
	mux.HandleFunc("GET /api/v1/roles", s.handleRoleList)
	mux.HandleFunc("GET /api/v1/roles/{name}", s.handleRoleGet)

	return requestLogger(recovery(cors(mux)))
}
