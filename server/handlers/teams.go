package handlers

import (
	"net/http"
	"strings"

	"github.com/rpuneet/bc/pkg/team"
)

// TeamHandler handles /api/teams routes.
type TeamHandler struct {
	store *team.Store
}

// NewTeamHandler creates a TeamHandler.
func NewTeamHandler(store *team.Store) *TeamHandler {
	return &TeamHandler{store: store}
}

// Register mounts team routes on mux.
func (h *TeamHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/teams", h.list)
	mux.HandleFunc("/api/teams/", h.byName)
}

// list handles GET /api/teams — returns all teams.
func (h *TeamHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	teams, err := h.store.List()
	if err != nil {
		httpInternalError(w, "list teams", err)
		return
	}
	if teams == nil {
		teams = []*team.Team{}
	}
	writeJSON(w, http.StatusOK, teams)
}

// byName handles GET /api/teams/{name} — returns a single team.
func (h *TeamHandler) byName(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/teams/")
	if name == "" {
		httpError(w, "team name required", http.StatusBadRequest)
		return
	}

	t, err := h.store.Get(name)
	if err != nil {
		httpInternalError(w, "get team", err)
		return
	}
	if t == nil {
		httpError(w, "team not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
