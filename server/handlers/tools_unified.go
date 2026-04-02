package handlers

import (
	"net/http"
	"os/exec"
	"strings"

	"github.com/gh-curious-otter/bc/pkg/agent"
	"github.com/gh-curious-otter/bc/pkg/mcp"
	"github.com/gh-curious-otter/bc/pkg/tool"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

// UnifiedTool represents a tool (MCP or CLI) with its status.
type UnifiedTool struct {
	Name       string `json:"name"`
	Type       string `json:"type"`                  // "mcp" or "cli"
	Status     string `json:"status"`                // "connected", "installed", "not_installed", "error", "unknown"
	Transport  string `json:"transport,omitempty"`    // for MCP: "sse" or "stdio"
	Command    string `json:"command,omitempty"`      // for CLI or stdio MCP
	URL        string `json:"url,omitempty"`          // for SSE MCP
	Version    string `json:"version,omitempty"`      // for CLI tools
	Error      string `json:"error,omitempty"`        // if status is "error"
	Required   bool   `json:"required"`
	InstallCmd string `json:"install_cmd,omitempty"`  // install command
	UpgradeCmd string `json:"upgrade_cmd,omitempty"`  // upgrade command
}

// UnifiedToolsHandler handles the merged /api/tools endpoint.
type UnifiedToolsHandler struct {
	mcpStore  *mcp.Store
	toolStore *tool.Store
	agents    *agent.AgentService
	ws        *workspace.Workspace
}

// NewUnifiedToolsHandler creates a UnifiedToolsHandler.
func NewUnifiedToolsHandler(mcpStore *mcp.Store, toolStore *tool.Store, agents *agent.AgentService, ws *workspace.Workspace) *UnifiedToolsHandler {
	return &UnifiedToolsHandler{mcpStore: mcpStore, toolStore: toolStore, agents: agents, ws: ws}
}

// Register mounts unified tools routes.
func (h *UnifiedToolsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools/unified", h.list)
	mux.HandleFunc("/api/tools/unified/check", h.checkAll)
}

// list returns all tools (MCP + CLI) with their current status.
func (h *UnifiedToolsHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	var tools []UnifiedTool

	// MCP servers from store
	if h.mcpStore != nil {
		servers, err := h.mcpStore.List()
		if err == nil {
			for _, s := range servers {
				status := "unknown"
				if s.Enabled {
					status = "configured"
				} else {
					status = "disabled"
				}
				tools = append(tools, UnifiedTool{
					Name:      s.Name,
					Type:      "mcp",
					Transport: string(s.Transport),
					Command:   s.Command,
					URL:       s.URL,
					Status:    status,
					Required:  true,
				})
			}
		}
	}

	// CLI tools from role configs
	if h.ws != nil && h.ws.RoleManager != nil {
		seen := make(map[string]bool)
		roles, _ := h.ws.RoleManager.LoadAllRoles()
		for _, role := range roles {
			for _, t := range role.Metadata.CLITools {
				if seen[t] {
					continue
				}
				seen[t] = true
				ut := UnifiedTool{
					Name:     t,
					Type:     "cli",
					Required: true,
				}
				// Check if installed
				if path, err := exec.LookPath(t); err == nil {
					ut.Status = "installed"
					ut.Command = path
					// Try to get version
					if out, verr := exec.Command(t, "--version").Output(); verr == nil {
						ver := strings.TrimSpace(string(out))
						if len(ver) > 80 {
							ver = ver[:80]
						}
						ut.Version = ver
					}
				} else {
					ut.Status = "not_installed"
				}
				tools = append(tools, ut)
			}
		}
	}

	// Built-in tools from tool store
	if h.toolStore != nil {
		builtins, err := h.toolStore.List(r.Context())
		if err == nil {
			for _, t := range builtins {
				if t.Builtin {
					toolType := "cli"
					if t.Type != "" {
						toolType = t.Type
					}
					status := "installed"
					if toolType == "mcp" {
						status = "configured"
						if !t.Enabled {
							status = "disabled"
						}
					} else {
						// Extract binary name (first word) from command — e.g. "claude --dangerously-skip-permissions" → "claude"
						bin := t.Command
						if i := strings.IndexByte(bin, ' '); i > 0 {
							bin = bin[:i]
						}
						if bin == "" {
							bin = t.Name // fallback to tool name
						}
						if _, lookErr := exec.LookPath(bin); lookErr != nil {
							status = "not_installed"
						}
					}
					ut := UnifiedTool{
						Name:       t.Name,
						Type:       toolType,
						Command:    t.Command,
						Transport:  t.Transport,
						URL:        t.URL,
						Status:     status,
						InstallCmd: t.InstallCmd,
						UpgradeCmd: t.UpgradeCmd,
					}
					// Try to get version for CLI tools
					if toolType == "cli" && status == "installed" && t.VersionCmd != "" {
						parts := strings.Fields(t.VersionCmd)
						if out, verr := exec.Command(parts[0], parts[1:]...).Output(); verr == nil {
							ver := strings.TrimSpace(string(out))
							if len(ver) > 80 {
								ver = ver[:80]
							}
							ut.Version = ver
						}
					}
					tools = append(tools, ut)
				}
			}
		}
	}

	if tools == nil {
		tools = []UnifiedTool{}
	}
	writeJSON(w, http.StatusOK, tools)
}

// checkAll runs health checks on all tools and returns results.
func (h *UnifiedToolsHandler) checkAll(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var results []UnifiedTool

	// Check MCP servers
	if h.mcpStore != nil {
		servers, err := h.mcpStore.List()
		if err == nil {
			for _, s := range servers {
				ut := UnifiedTool{
					Name:      s.Name,
					Type:      "mcp",
					Transport: string(s.Transport),
					Status:    "connected",
					Required:  true,
				}
				// For SSE servers, we could ping the URL
				// For stdio, we could check the command exists
				if s.Transport == "stdio" && s.Command != "" {
					cmd := strings.Fields(s.Command)[0]
					if _, err := exec.LookPath(cmd); err != nil {
						ut.Status = "error"
						ut.Error = "command not found: " + cmd
					}
				}
				results = append(results, ut)
			}
		}
	}

	// Check CLI tools
	if h.ws != nil && h.ws.RoleManager != nil {
		seen := make(map[string]bool)
		roles, _ := h.ws.RoleManager.LoadAllRoles()
		for _, role := range roles {
			for _, t := range role.Metadata.CLITools {
				if seen[t] {
					continue
				}
				seen[t] = true
				ut := UnifiedTool{
					Name:     t,
					Type:     "cli",
					Required: true,
				}
				if _, err := exec.LookPath(t); err == nil {
					ut.Status = "installed"
				} else {
					ut.Status = "not_installed"
					ut.Error = t + " not found in PATH"
				}
				results = append(results, ut)
			}
		}
	}

	if results == nil {
		results = []UnifiedTool{}
	}
	writeJSON(w, http.StatusOK, results)
}
