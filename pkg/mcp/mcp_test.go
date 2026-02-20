package mcp

import (
	"encoding/json"
	"testing"
)

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		want  string
		state ConnectionState
	}{
		{"disconnected", StateDisconnected},
		{"connecting", StateConnecting},
		{"connected", StateConnected},
		{"error", StateError},
		{"unknown", ConnectionState(99)},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("ConnectionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewTextContent(t *testing.T) {
	content := NewTextContent("hello world")
	if content.Type != "text" {
		t.Errorf("NewTextContent().Type = %q, want %q", content.Type, "text")
	}
	if content.Text != "hello world" {
		t.Errorf("NewTextContent().Text = %q, want %q", content.Text, "hello world")
	}
}

func TestNewErrorContent(t *testing.T) {
	err := &testError{msg: "test error"}
	content := NewErrorContent(err)
	if content.Type != "text" {
		t.Errorf("NewErrorContent().Type = %q, want %q", content.Type, "text")
	}
	if content.Text != "Error: test error" {
		t.Errorf("NewErrorContent().Text = %q, want %q", content.Text, "Error: test error")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewRequest(t *testing.T) {
	params := map[string]string{"key": "value"}
	req, err := NewRequest(1, "test/method", params)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if req.JSONRPC != JSONRPCVersion {
		t.Errorf("NewRequest().JSONRPC = %q, want %q", req.JSONRPC, JSONRPCVersion)
	}
	if req.ID != 1 {
		t.Errorf("NewRequest().ID = %v, want %v", req.ID, 1)
	}
	if req.Method != "test/method" {
		t.Errorf("NewRequest().Method = %q, want %q", req.Method, "test/method")
	}

	var decodedParams map[string]string
	if err := json.Unmarshal(req.Params, &decodedParams); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decodedParams["key"] != "value" {
		t.Errorf("params[key] = %q, want %q", decodedParams["key"], "value")
	}
}

func TestNewRequestNilParams(t *testing.T) {
	req, err := NewRequest("abc", "test/method", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if req.Params != nil {
		t.Errorf("NewRequest().Params = %v, want nil", req.Params)
	}
}

func TestNewResponse(t *testing.T) {
	result := map[string]int{"count": 42}
	resp, err := NewResponse(1, result)
	if err != nil {
		t.Fatalf("NewResponse() error = %v", err)
	}

	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("NewResponse().JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
	}
	if resp.ID != 1 {
		t.Errorf("NewResponse().ID = %v, want %v", resp.ID, 1)
	}
	if resp.Error != nil {
		t.Errorf("NewResponse().Error = %v, want nil", resp.Error)
	}

	var decodedResult map[string]int
	if err := json.Unmarshal(resp.Result, &decodedResult); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decodedResult["count"] != 42 {
		t.Errorf("result[count] = %d, want %d", decodedResult["count"], 42)
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(1, ErrCodeInvalidParams, "invalid params", nil)

	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("NewErrorResponse().JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
	}
	if resp.ID != 1 {
		t.Errorf("NewErrorResponse().ID = %v, want %v", resp.ID, 1)
	}
	if resp.Result != nil {
		t.Errorf("NewErrorResponse().Result = %v, want nil", resp.Result)
	}
	if resp.Error == nil {
		t.Fatal("NewErrorResponse().Error = nil, want non-nil")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("NewErrorResponse().Error.Code = %d, want %d", resp.Error.Code, ErrCodeInvalidParams)
	}
	if resp.Error.Message != "invalid params" {
		t.Errorf("NewErrorResponse().Error.Message = %q, want %q", resp.Error.Message, "invalid params")
	}
}

func TestRequestJSONMarshaling(t *testing.T) {
	req, _ := NewRequest(1, MethodToolsList, nil)
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.JSONRPC != JSONRPCVersion {
		t.Errorf("decoded.JSONRPC = %q, want %q", decoded.JSONRPC, JSONRPCVersion)
	}
	if decoded.Method != MethodToolsList {
		t.Errorf("decoded.Method = %q, want %q", decoded.Method, MethodToolsList)
	}
}

func TestToolJSONMarshaling(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != tool.Name {
		t.Errorf("decoded.Name = %q, want %q", decoded.Name, tool.Name)
	}
	if decoded.Description != tool.Description {
		t.Errorf("decoded.Description = %q, want %q", decoded.Description, tool.Description)
	}
}

func TestResourceJSONMarshaling(t *testing.T) {
	resource := Resource{
		URI:         "agents://eng-01",
		Name:        "Agent eng-01",
		Description: "Engineer agent",
		MimeType:    "application/json",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Resource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.URI != resource.URI {
		t.Errorf("decoded.URI = %q, want %q", decoded.URI, resource.URI)
	}
}

func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, "2024-11-05")
	}
}

func TestTransportConstants(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{TransportStdio, "stdio"},
		{TransportHTTP, "http"},
		{TransportSSE, "sse"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("transport constant = %q, want %q", tt.got, tt.want)
		}
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		got  int
		want int
	}{
		{ErrCodeParse, -32700},
		{ErrCodeInvalidRequest, -32600},
		{ErrCodeMethodNotFound, -32601},
		{ErrCodeInvalidParams, -32602},
		{ErrCodeInternal, -32603},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("error code = %d, want %d", tt.got, tt.want)
		}
	}
}
