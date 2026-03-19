package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rpuneet/bc/pkg/agent"
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
					"sender": map[string]any{
						"type":        "string",
						"description": "Sender name (defaults to 'mcp')",
					},
				},
				"required": []string{"channel", "message"},
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
	Sender  string `json:"sender,omitempty"`
}

func (s *Server) toolSendMessage(raw json.RawMessage) (*toolsCallResult, error) {
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

	sender := args.Sender
	if sender == "" {
		sender = "mcp"
	}

	if err := s.chans.AddHistory(args.Channel, sender, args.Message); err != nil {
		return &toolsCallResult{
			Content: []ToolContent{textContent(fmt.Sprintf("failed to send message: %s", err))},
			IsError: true,
		}, nil
	}

	// Deliver to agent tmux/docker sessions
	if s.agents != nil {
		if ch, ok := s.chans.Get(args.Channel); ok {
			for _, member := range ch.Members {
				if member == sender {
					continue
				}
				_ = s.agents.SendToAgent(member, args.Message)
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
		summaries, err := s.costs.SummaryByAgent()
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

	ws, err := s.costs.WorkspaceSummary()
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
