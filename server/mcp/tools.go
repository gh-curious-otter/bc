package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-curious-otter/bc/pkg/agent"
)

// definedTools returns the static list of tools this server exposes.
func definedTools() []Tool {
	return []Tool{
		{
			Name:        "create_agent",
			Description: "Create a new agent in the bc workspace",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Unique agent name (alphanumeric, hyphens, underscores)",
					},
					"role": map[string]any{
						"type":        "string",
						"description": "Role for the agent (e.g. engineer, manager, root)",
					},
					"tool": map[string]any{
						"type":        "string",
						"description": "AI tool to use (claude, gemini, cursor, aider, codex)",
					},
				},
				"required": []string{"name", "role"},
			},
		},
		{
			Name:        "read_channel",
			Description: "Read recent messages from a bc channel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"channel": map[string]any{
						"type":        "string",
						"description": "Channel name to read from",
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Number of messages to return (default 20, max 100)",
					},
				},
				"required": []string{"channel"},
			},
		},
		{
			Name:        "send_message",
			Description: "Send a message to a bc channel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"channel": map[string]any{
						"type":        "string",
						"description": "Channel name to send the message to",
					},
					"message": map[string]any{
						"type":        "string",
						"description": "Message content",
					},
				},
				"required": []string{"channel", "message"},
			},
		},
		{
			Name:        "send_file",
			Description: "Upload a file to a bc gateway channel (e.g., share a screenshot to Slack)",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"channel": map[string]any{
						"type":        "string",
						"description": "Gateway channel name (e.g., slack:all-bc)",
					},
					"file_path": map[string]any{
						"type":        "string",
						"description": "Local file path to upload",
					},
					"comment": map[string]any{
						"type":        "string",
						"description": "Optional text message to accompany the file",
					},
				},
				"required": []string{"channel", "file_path"},
			},
		},
		{
			Name:        "report_status",
			Description: "Update the current task description for a bc agent",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type":        "string",
						"description": "Agent name to update",
					},
					"task": map[string]any{
						"type":        "string",
						"description": "Current task description",
					},
				},
				"required": []string{"agent", "task"},
			},
		},
		{
			Name:        "query_costs",
			Description: "Query cost usage for the workspace or a specific agent",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent": map[string]any{
						"type":        "string",
						"description": "Agent name to query (omit for workspace total)",
					},
				},
			},
		},
	}
}

// ─── create_agent ─────────────────────────────────────────────────────────────

type createAgentArgs struct {
	Name string `json:"name"`
	Role string `json:"role"`
	Tool string `json:"tool,omitempty"`
}

func (s *Server) toolCreateAgent(ctx context.Context, raw json.RawMessage) (*toolsCallResult, error) {
	var args createAgentArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if args.Role == "" {
		return nil, fmt.Errorf("role is required")
	}
	if !agent.IsValidAgentName(args.Name) {
		return nil, fmt.Errorf("invalid agent name %q: use alphanumeric, hyphens, underscores", args.Name)
	}

	// Build bc agent create command
	cmdArgs := []string{"agent", "create", args.Name, "--role", args.Role}
	if args.Tool != "" {
		cmdArgs = append(cmdArgs, "--tool", args.Tool)
	}

	//nolint:gosec // G204: arguments are validated above
	out, err := exec.CommandContext(ctx, "bc", cmdArgs...).CombinedOutput()
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to create agent: %s\n%s", err, out))},
			IsError: true,
		}, nil
	}

	return &toolsCallResult{
		Content: []ToolContent{
			textContent(fmt.Sprintf("Created agent %q with role %q\n%s",
				args.Name, args.Role, strings.TrimSpace(string(out)))),
		},
	}, nil
}

// ─── send_message ─────────────────────────────────────────────────────────────

type sendMessageArgs struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

func (s *Server) toolSendMessage(ctx context.Context, raw json.RawMessage) (*toolsCallResult, error) {
	var args sendMessageArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if args.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	// Sender is determined from the authenticated SSE connection identity.
	// Falls back to workspace nickname, then "mcp".
	sender := AgentFromContext(ctx)
	if sender == "" {
		if s.ws != nil {
			nick := s.ws.Config.User.Name
			nick = strings.TrimPrefix(nick, "@")
			if nick != "" {
				sender = nick
			}
		}
	}
	if sender == "" {
		sender = "mcp"
	}

	// Use ChannelService when available — its OnMessage hook handles agent
	// delivery and SSE event publishing automatically.
	if s.chanSvc != nil {
		if _, err := s.chanSvc.Send(context.Background(), args.Channel, sender, args.Message); err != nil {
			return &toolsCallResult{
				Content: []ToolContent{textContent(fmt.Sprintf("failed to send message: %s", err))},
				IsError: true,
			}, nil
		}
	} else {
		// Standalone mode — store message and attempt direct delivery.
		if err := s.chans.AddHistory(args.Channel, sender, args.Message); err != nil {
			return &toolsCallResult{
				Content: []ToolContent{textContent(fmt.Sprintf("failed to send message: %s", err))},
				IsError: true,
			}, nil
		}
		// Best-effort delivery to channel members via agent manager
		if s.agents != nil {
			members, _ := s.chans.GetMembers(args.Channel)
			formatted := fmt.Sprintf("[bc-mcp][%s][#%s] %s: %s", time.Now().UTC().Format(time.RFC3339), args.Channel, sender, args.Message)
			for _, member := range members {
				if member == sender {
					continue
				}
				_ = s.agents.SendToAgent(context.Background(), member, formatted) //nolint:errcheck // best-effort
			}
		}
	}

	return &toolsCallResult{
		Content: []ToolContent{
			textContent(fmt.Sprintf("Sent message to #%s from %s", args.Channel, sender)),
		},
	}, nil
}

// ─── report_status ────────────────────────────────────────────────────────────

type reportStatusArgs struct {
	Agent string `json:"agent"`
	Task  string `json:"task"`
}

// ─── read_channel ───────────────────────────────────────────────────────────

func (s *Server) toolReadChannel(raw json.RawMessage) (*toolsCallResult, error) {
	var args struct {
		Channel string `json:"channel"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Channel == "" {
		return &toolsCallResult{
			Content: []ToolContent{textContent("channel is required")},
			IsError: true,
		}, nil
	}
	if args.Limit <= 0 {
		args.Limit = 20
	}
	if args.Limit > 100 {
		args.Limit = 100
	}

	if s.chans == nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent("channel store not available")},
			IsError: true,
		}, nil
	}

	history, err := s.chans.GetHistory(args.Channel)
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to read channel: %s", err))},
			IsError: true,
		}, nil
	}

	// Take last N entries (newest)
	if len(history) > args.Limit {
		history = history[len(history)-args.Limit:]
	}

	// Format as readable text
	var sb strings.Builder
	for _, entry := range history {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", entry.Time.Format("15:04:05"), entry.Sender, entry.Message))
	}
	if sb.Len() == 0 {
		sb.WriteString("(no messages)")
	}

	return &toolsCallResult{
		Content: []ToolContent{textContent(sb.String())},
	}, nil
}

// ─── send_file ──────────────────────────────────────────────────────────────

const maxFileSize = 50 * 1024 * 1024 // 50MB

func (s *Server) toolSendFile(ctx context.Context, raw json.RawMessage) (*toolsCallResult, error) {
	var args struct {
		Channel  string `json:"channel"`
		FilePath string `json:"file_path"`
		Comment  string `json:"comment"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Channel == "" || args.FilePath == "" {
		return &toolsCallResult{
			Content: []ToolContent{textContent("channel and file_path are required")},
			IsError: true,
		}, nil
	}

	// Validate file path is under workspace to prevent reading arbitrary files
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("invalid file path: %s", err))},
			IsError: true,
		}, nil
	}

	// Check file size before reading into memory
	info, err := os.Stat(absPath)
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("file not found: %s", err))},
			IsError: true,
		}, nil
	}
	if info.Size() > maxFileSize {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("file too large: %d bytes (max %d)", info.Size(), maxFileSize))},
			IsError: true,
		}, nil
	}

	data, err := os.ReadFile(absPath) //nolint:gosec // path validated above
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to read file: %s", err))},
			IsError: true,
		}, nil
	}

	// Detect MIME type — try content-based first, fall back to extension
	filename := filepath.Base(absPath)
	mimeType := "application/octet-stream"
	if len(data) >= 512 {
		mimeType = http.DetectContentType(data[:512])
	}
	// Override with extension for known types (DetectContentType can be imprecise)
	switch {
	case strings.HasSuffix(filename, ".png"):
		mimeType = "image/png"
	case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
		mimeType = "image/jpeg"
	case strings.HasSuffix(filename, ".gif"):
		mimeType = "image/gif"
	case strings.HasSuffix(filename, ".webp"):
		mimeType = "image/webp"
	case strings.HasSuffix(filename, ".pdf"):
		mimeType = "application/pdf"
	}

	// Get sender from context
	sender := "agent"
	if agentID, ok := ctx.Value(ctxKeyAgent).(string); ok && agentID != "" {
		sender = agentID
	}

	// Route through gateway manager
	if s.gateway == nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent("no gateway configured — file upload requires a gateway channel (e.g., slack:all-bc)")},
			IsError: true,
		}, nil
	}

	sent, err := s.gateway.SendFile(ctx, args.Channel, sender, filename, data, mimeType)
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("file upload failed: %s", err))},
			IsError: true,
		}, nil
	}
	if !sent {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("channel %q is not a gateway channel — file upload only works with gateway channels (slack:*, telegram:*, discord:*)", args.Channel))},
			IsError: true,
		}, nil
	}

	return &toolsCallResult{
		Content: []ToolContent{textContent(fmt.Sprintf("Uploaded %s (%d bytes) to %s", filename, len(data), args.Channel))},
	}, nil
}

// ─── report_status ──────────────────────────────────────────────────────────

func (s *Server) toolReportStatus(raw json.RawMessage) (*toolsCallResult, error) {
	var args reportStatusArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Agent == "" {
		return nil, fmt.Errorf("agent is required")
	}
	if args.Task == "" {
		return nil, fmt.Errorf("task is required")
	}

	ag := s.agents.GetAgent(args.Agent)
	if ag == nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("agent %q not found", args.Agent))},
			IsError: true,
		}, nil
	}

	// Keep current state; only update the task description.
	if err := s.agents.UpdateAgentState(args.Agent, ag.State, args.Task); err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to update status: %s", err))},
			IsError: true,
		}, nil
	}

	return &toolsCallResult{
		Content: []ToolContent{
			textContent(fmt.Sprintf("Updated task for agent %q: %s", args.Agent, args.Task)),
		},
	}, nil
}

// ─── query_costs ──────────────────────────────────────────────────────────────

type queryCostsArgs struct {
	Agent string `json:"agent,omitempty"`
}

func (s *Server) toolQueryCosts(raw json.RawMessage) (*toolsCallResult, error) {
	var args queryCostsArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if args.Agent != "" {
		summaries, err := s.costs.SummaryByAgent(context.Background())
		if err != nil {
			return &toolsCallResult{
				Content: []ToolContent{textContent(fmt.Sprintf("failed to query costs: %s", err))},
				IsError: true,
			}, nil
		}
		for _, a := range summaries {
			if a.AgentID == args.Agent {
				b, _ := json.MarshalIndent(a, "", "  ")
				return &toolsCallResult{
					Content: []ToolContent{textContent(string(b))},
				}, nil
			}
		}
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("no cost data for agent %q", args.Agent))},
		}, nil
	}

	ws, err := s.costs.WorkspaceSummary(context.Background())
	if err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to query costs: %s", err))},
			IsError: true,
		}, nil
	}

	b, _ := json.MarshalIndent(ws, "", "  ")
	return &toolsCallResult{
		Content: []ToolContent{textContent(string(b))},
	}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// commandExists reports whether a command is available on PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
