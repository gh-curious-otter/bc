// Package mcp implements the Model Context Protocol server for bc workspaces.
// It exposes workspace state as MCP resources and provides tools for controlling agents.
//
// Supports two transports:
//   - stdio: newline-delimited JSON-RPC on stdin/stdout (for direct Claude Code integration)
//   - SSE:   HTTP server with /sse for server→client events, /message for client→server
package mcp

import "encoding/json"

// Protocol version this server speaks.
const ProtocolVersion = "2024-11-05"

// JSON-RPC 2.0 error codes.
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// ─── JSON-RPC 2.0 envelope types ─────────────────────────────────────────────

// Request is an incoming JSON-RPC 2.0 message (may be a request or notification).
type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"` // nil for notifications
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

// Response is an outgoing JSON-RPC 2.0 response.
type Response struct {
	Result  any              `json:"result,omitempty"`
	ID      *json.RawMessage `json:"id"`
	Error   *RPCError        `json:"error,omitempty"`
	JSONRPC string           `json:"jsonrpc"`
}

// Notification is an outgoing JSON-RPC 2.0 notification (no ID).
type Notification struct {
	Params  any    `json:"params,omitempty"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *RPCError) Error() string { return e.Message }

func errResponse(id *json.RawMessage, code int, msg string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
}

func okResponse(id *json.RawMessage, result any) Response {
	return Response{JSONRPC: "2.0", ID: id, Result: result}
}

// ─── MCP initialize ───────────────────────────────────────────────────────────

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      serverInfo         `json:"serverInfo"`
}

type serverCapabilities struct {
	Resources *resourcesCapability `json:"resources,omitempty"`
	Tools     *toolsCapability     `json:"tools,omitempty"`
}

type resourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ─── MCP resources ────────────────────────────────────────────────────────────

type resourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// Resource describes a queryable MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type resourcesReadParams struct {
	URI string `json:"uri"`
}

type resourcesReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent holds the data returned for a resource read.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// ─── MCP tools ────────────────────────────────────────────────────────────────

type toolsListResult struct {
	Tools []Tool `json:"tools"`
}

// Tool describes a callable MCP tool.
type Tool struct {
	InputSchema map[string]any `json:"inputSchema"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolsCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is a piece of content returned by a tool call.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func textContent(text string) ToolContent { return ToolContent{Type: "text", Text: text} }
