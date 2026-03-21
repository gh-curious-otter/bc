package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Server is a bc MCP server. It owns handles to workspace state and dispatches
// JSON-RPC 2.0 requests from either stdio or SSE transports.
type Server struct {
	ws         *workspace.Workspace
	agents     *agent.Manager
	chans      *channel.Store
	chanSvc    *channel.ChannelService
	costs      *cost.Store
	broker     *SSEBroker
	cancelPoll context.CancelFunc
	version    string
	pollWg     sync.WaitGroup
	ownChans   bool
	ownCosts   bool
}

// Config holds the dependencies needed to build a Server.
// When Channels or Costs are provided, the server reuses them (e.g. from bcd)
// instead of opening its own connections.
type Config struct {
	Workspace      *workspace.Workspace
	Agents         *agent.Manager          // optional: pre-built agent manager
	Channels       *channel.Store          // optional: pre-built channel store (SQLite/Postgres)
	ChannelService *channel.ChannelService // optional: service with OnMessage hook for delivery
	Costs          *cost.Store             // optional: pre-built cost store
	Version        string                  // bc binary version, e.g. "1.2.3"
}

// New creates a Server. Call Close when done.
// When stores are provided via Config, the caller owns their lifecycle and
// Close will not close them.
func New(cfg Config) (*Server, error) {
	if cfg.Workspace == nil {
		return nil, fmt.Errorf("workspace is required")
	}

	// Track whether we created stores ourselves (so Close knows what to clean up).
	var ownChans, ownCosts bool

	// Agent manager
	mgr := cfg.Agents
	if mgr == nil {
		mgr = agent.NewWorkspaceManager(cfg.Workspace.AgentsDir(), cfg.Workspace.RootDir)
		if err := mgr.LoadState(); err != nil {
			_ = err // Non-fatal
		}
	}

	// Channel store
	cs := cfg.Channels
	if cs == nil {
		var err error
		cs, err = channel.OpenStore(cfg.Workspace.RootDir)
		if err != nil {
			cs = channel.NewStore(cfg.Workspace.RootDir)
		}
		ownChans = true
	}

	// Cost store
	costStore := cfg.Costs
	if costStore == nil {
		costStore = cost.NewStore(cfg.Workspace.RootDir)
		if err := costStore.Open(); err != nil {
			_ = err // Non-fatal
		}
		ownCosts = true
	}

	v := cfg.Version
	if v == "" {
		v = "dev"
	}

	return &Server{
		ws:       cfg.Workspace,
		agents:   mgr,
		chans:    cs,
		chanSvc:  cfg.ChannelService,
		costs:    costStore,
		version:  v,
		ownChans: ownChans,
		ownCosts: ownCosts,
	}, nil
}

// Close releases resources held by the server.
// Only closes stores that the server created itself (not injected ones).
func (s *Server) Close() error {
	if s.cancelPoll != nil {
		s.cancelPoll()
		s.pollWg.Wait()
	}
	if s.ownChans && s.chans != nil {
		if err := s.chans.Close(); err != nil {
			return err
		}
	}
	if s.ownCosts && s.costs != nil {
		return s.costs.Close()
	}
	return nil
}

// SetBroker attaches an SSE broker and starts polling channels for new messages.
// New messages are pushed as notifications/message to all connected clients.
func (s *Server) SetBroker(broker *SSEBroker) {
	s.broker = broker
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelPoll = cancel
	s.pollWg.Add(1)
	go s.pollChannelMessages(ctx)
}

// channelMessagePayload is the notification payload for new channel messages.
type channelMessagePayload struct {
	Time    time.Time `json:"time"`
	Channel string    `json:"channel"`
	Sender  string    `json:"sender"`
	Message string    `json:"message"`
}

// pollChannelMessages watches all channels for new messages and pushes them
// as MCP notifications to connected SSE clients.
func (s *Server) pollChannelMessages(ctx context.Context) {
	defer s.pollWg.Done()

	// Track the latest message timestamp per channel to detect new messages.
	// Using timestamps instead of array length avoids breaking when history
	// is capped (e.g., GetHistory returns last 100 messages).
	lastSeen := make(map[string]time.Time)

	// Seed with current latest timestamp so we don't replay old history.
	for _, ch := range s.chans.List() {
		history, err := s.chans.GetHistory(ch.Name)
		if err == nil && len(history) > 0 {
			lastSeen[ch.Name] = history[len(history)-1].Time
		}
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkNewMessages(lastSeen)
		}
	}
}

// checkNewMessages checks each channel for messages newer than lastSeen and pushes them.
func (s *Server) checkNewMessages(lastSeen map[string]time.Time) {
	for _, ch := range s.chans.List() {
		history, err := s.chans.GetHistory(ch.Name)
		if err != nil || len(history) == 0 {
			continue
		}

		cutoff := lastSeen[ch.Name]

		// Find new messages (those with timestamp after cutoff)
		for _, entry := range history {
			if !entry.Time.After(cutoff) {
				continue
			}
			s.pushMessageNotification(ch.Name, entry.Sender, entry.Message, entry.Time)
		}

		// Update watermark to latest message
		lastSeen[ch.Name] = history[len(history)-1].Time
	}
}

// pushMessageNotification sends a notifications/message event to all SSE clients.
func (s *Server) pushMessageNotification(ch, sender, message string, t time.Time) {
	if s.broker == nil {
		return
	}
	s.broker.send(Notification{
		JSONRPC: "2.0",
		Method:  "notifications/message",
		Params: channelMessagePayload{
			Channel: ch,
			Sender:  sender,
			Message: message,
			Time:    t,
		},
	})
}

// Handle processes a single JSON-RPC request and returns the response.
// For notifications (no ID), the returned Response has a nil ID and no result/error set.
func (s *Server) Handle(ctx context.Context, req Request) Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification — no response needed; caller should discard
		return Response{}
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return errResponse(req.ID, ErrMethodNotFound,
			fmt.Sprintf("method not found: %s", req.Method))
	}
}

// ─── initialize ──────────────────────────────────────────────────────────────

func (s *Server) handleInitialize(req Request) Response {
	// Accept any client capabilities — we don't require specific ones.
	return okResponse(req.ID, initializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: serverCapabilities{
			Resources: &resourcesCapability{},
			Tools:     &toolsCapability{},
		},
		ServerInfo: serverInfo{
			Name:    "bc",
			Version: s.version,
		},
	})
}

// ─── resources/list ──────────────────────────────────────────────────────────

func (s *Server) handleResourcesList(req Request) Response {
	return okResponse(req.ID, resourcesListResult{
		Resources: definedResources(),
	})
}

// definedResources returns the static list of resources this server exposes.
func definedResources() []Resource {
	return []Resource{
		{
			URI:         "bc://workspace/status",
			Name:        "Workspace Status",
			Description: "Workspace name, path, version, and configuration summary",
			MIMEType:    "application/json",
		},
		{
			URI:         "bc://agents",
			Name:        "Agents",
			Description: "All agents with their state, role, tool, and worktree info",
			MIMEType:    "application/json",
		},
		{
			URI:         "bc://channels",
			Name:        "Channels",
			Description: "All channels with members and recent message counts",
			MIMEType:    "application/json",
		},
		{
			URI:         "bc://costs",
			Name:        "Costs",
			Description: "Workspace cost summary and per-agent breakdown",
			MIMEType:    "application/json",
		},
		{
			URI:         "bc://roles",
			Name:        "Roles",
			Description: "Role definitions with capabilities, permissions, and MCP server associations",
			MIMEType:    "application/json",
		},
		{
			URI:         "bc://tools",
			Name:        "Tools",
			Description: "AI agent tools available in this workspace",
			MIMEType:    "application/json",
		},
	}
}

// ─── resources/read ──────────────────────────────────────────────────────────

func (s *Server) handleResourcesRead(req Request) Response {
	var p resourcesReadParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return errResponse(req.ID, ErrInvalidParams, "invalid params: "+err.Error())
	}

	var (
		content string
		err     error
	)

	switch p.URI {
	case "bc://workspace/status":
		content, err = s.readWorkspaceStatus()
	case "bc://agents":
		content, err = s.readAgents()
	case "bc://channels":
		content, err = s.readChannels()
	case "bc://costs":
		content, err = s.readCosts()
	case "bc://roles":
		content, err = s.readRoles()
	case "bc://tools":
		content, err = s.readTools()
	default:
		return errResponse(req.ID, ErrInvalidParams, fmt.Sprintf("unknown resource URI: %s", p.URI))
	}

	if err != nil {
		return errResponse(req.ID, ErrInternal, err.Error())
	}

	return okResponse(req.ID, resourcesReadResult{
		Contents: []ResourceContent{
			{URI: p.URI, MIMEType: "application/json", Text: content},
		},
	})
}

// ─── tools/list ──────────────────────────────────────────────────────────────

func (s *Server) handleToolsList(req Request) Response {
	return okResponse(req.ID, toolsListResult{Tools: definedTools()})
}

// ─── tools/call ──────────────────────────────────────────────────────────────

func (s *Server) handleToolsCall(ctx context.Context, req Request) Response {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return errResponse(req.ID, ErrInvalidParams, "invalid params: "+err.Error())
	}

	var (
		result *toolsCallResult
		err    error
	)

	switch p.Name {
	case "create_agent":
		result, err = s.toolCreateAgent(ctx, p.Arguments)
	case "send_message":
		result, err = s.toolSendMessage(p.Arguments)
	case "report_status":
		result, err = s.toolReportStatus(p.Arguments)
	case "query_costs":
		result, err = s.toolQueryCosts(p.Arguments)
	default:
		return errResponse(req.ID, ErrInvalidParams, fmt.Sprintf("unknown tool: %s", p.Name))
	}

	if err != nil {
		return okResponse(req.ID, toolsCallResult{
			Content: []ToolContent{textContent(err.Error())},
			IsError: true,
		})
	}

	return okResponse(req.ID, result)
}

// ─── JSON marshaling helper ──────────────────────────────────────────────────

func marshalJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
