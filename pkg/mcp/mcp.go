// Package mcp implements Model Context Protocol (MCP) integration for bc.
//
// MCP is an open protocol that standardizes how AI applications connect to
// external data and capabilities. This package provides both server and client
// implementations for bc workspaces.
//
// Features:
// - MCP server exposing bc resources (agents, channels, costs, memory)
// - MCP client for connecting to external MCP servers
// - Tool discovery and invocation
// - JSON-RPC 2.0 transport
//
// Issue #1212: MCP integration for Phase 4 Ecosystem
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Protocol version
const ProtocolVersion = "2024-11-05"

// Transport types
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
	TransportSSE   = "sse"
)

// Message types for JSON-RPC 2.0
const (
	MethodInitialize       = "initialize"
	MethodInitialized      = "notifications/initialized"
	MethodToolsList        = "tools/list"
	MethodToolsCall        = "tools/call"
	MethodResourcesList    = "resources/list"
	MethodResourcesRead    = "resources/read"
	MethodPromptsList      = "prompts/list"
	MethodPromptsGet       = "prompts/get"
	MethodLoggingSetLevel  = "logging/setLevel"
	MethodSamplingCreate   = "sampling/createMessage"
	MethodRootsListChanged = "notifications/roots/list_changed"
	MethodCancelled        = "notifications/cancelled" //nolint:misspell // MCP protocol uses British spelling
)

// JSONRPCVersion is the JSON-RPC version used by MCP
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request
type Request struct {
	ID      any             `json:"id,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	ID      any             `json:"id,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Error codes
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// ServerInfo describes an MCP server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientInfo describes an MCP client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams contains initialization parameters
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// InitializeResult contains initialization result
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// ServerCapabilities describes what the server supports
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// RootsCapability indicates root listing support
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates sampling support
type SamplingCapability struct{}

// ToolsCapability indicates tool support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resource support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompt support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// Tool represents an MCP tool
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolCallParams contains tool call parameters
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult contains tool call result
type ToolCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in a result
type Content struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"` // base64 for images
	URI      string `json:"uri,omitempty"`  // for resources
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContents represents resource contents
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string    `json:"role"` // "user" or "assistant"
	Content []Content `json:"content"`
}

// Handler defines the interface for handling MCP requests
type Handler interface {
	// Initialize handles the initialize request
	Initialize(ctx context.Context, params InitializeParams) (*InitializeResult, error)

	// ListTools returns available tools
	ListTools(ctx context.Context) ([]Tool, error)

	// CallTool invokes a tool
	CallTool(ctx context.Context, params ToolCallParams) (*ToolCallResult, error)

	// ListResources returns available resources
	ListResources(ctx context.Context) ([]Resource, error)

	// ReadResource reads a resource
	ReadResource(ctx context.Context, uri string) (*ResourceContents, error)

	// ListPrompts returns available prompts
	ListPrompts(ctx context.Context) ([]Prompt, error)

	// GetPrompt retrieves a prompt with arguments applied
	GetPrompt(ctx context.Context, name string, args map[string]string) ([]PromptMessage, error)
}

// ConnectionState represents the state of an MCP connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Connection represents an MCP connection
type Connection struct {
	ServerInfo  *ServerInfo     `json:"serverInfo,omitempty"`
	ConnectedAt *time.Time      `json:"connectedAt,omitempty"`
	ServerURL   string          `json:"serverUrl,omitempty"`
	Error       string          `json:"error,omitempty"`
	State       ConnectionState `json:"state"`
}

// NewTextContent creates a text content item
func NewTextContent(text string) Content {
	return Content{Type: "text", Text: text}
}

// NewErrorContent creates an error content item
func NewErrorContent(err error) Content {
	return Content{Type: "text", Text: fmt.Sprintf("Error: %v", err)}
}

// NewRequest creates a new JSON-RPC request
func NewRequest(id any, method string, params any) (*Request, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = b
	}

	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewResponse creates a successful response
func NewResponse(id any, result any) (*Response, error) {
	var resultJSON json.RawMessage
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resultJSON = b
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}, nil
}

// NewErrorResponse creates an error response
func NewErrorResponse(id any, code int, message string, data any) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
