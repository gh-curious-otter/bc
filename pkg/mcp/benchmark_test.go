package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

// BenchmarkNewRequest measures request creation performance.
func BenchmarkNewRequest(b *testing.B) {
	params := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRequest(1, MethodToolsCall, params)
	}
}

// BenchmarkNewRequestNilParams measures request creation without params.
func BenchmarkNewRequestNilParams(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRequest(1, MethodToolsList, nil)
	}
}

// BenchmarkNewRequestComplex measures request with complex params.
func BenchmarkNewRequestComplex(b *testing.B) {
	params := ToolCallParams{
		Name:      "test-tool",
		Arguments: json.RawMessage(`{"arg1": "value1", "arg2": 42, "nested": {"key": "val"}}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRequest(i, MethodToolsCall, params)
	}
}

// BenchmarkNewResponse measures response creation performance.
func BenchmarkNewResponse(b *testing.B) {
	result := map[string]any{"status": "ok", "count": 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewResponse(1, result)
	}
}

// BenchmarkNewResponseNilResult measures response creation without result.
func BenchmarkNewResponseNilResult(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewResponse(1, nil)
	}
}

// BenchmarkNewErrorResponse measures error response creation.
func BenchmarkNewErrorResponse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewErrorResponse(1, ErrCodeInvalidParams, "invalid parameters", nil)
	}
}

// BenchmarkNewErrorResponseWithData measures error response with data.
func BenchmarkNewErrorResponseWithData(b *testing.B) {
	data := map[string]string{"field": "name", "reason": "required"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewErrorResponse(1, ErrCodeInvalidParams, "validation failed", data)
	}
}

// BenchmarkNewTextContent measures text content creation.
func BenchmarkNewTextContent(b *testing.B) {
	text := "This is a sample response text from the tool execution."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewTextContent(text)
	}
}

// BenchmarkNewErrorContent measures error content creation.
func BenchmarkNewErrorContent(b *testing.B) {
	err := errors.New("something went wrong")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewErrorContent(err)
	}
}

// BenchmarkConnectionStateString measures state string conversion.
func BenchmarkConnectionStateString(b *testing.B) {
	states := []ConnectionState{StateDisconnected, StateConnecting, StateConnected, StateError}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := states[i%len(states)] //nolint:gosec // index bounded
		_ = state.String()
	}
}

// BenchmarkRequestMarshal measures request JSON marshaling.
func BenchmarkRequestMarshal(b *testing.B) {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      1,
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name": "test"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkRequestUnmarshal measures request JSON unmarshaling.
func BenchmarkRequestUnmarshal(b *testing.B) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req Request
		_ = json.Unmarshal(data, &req)
	}
}

// BenchmarkResponseMarshal measures response JSON marshaling.
func BenchmarkResponseMarshal(b *testing.B) {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      1,
		Result:  json.RawMessage(`{"content":[{"type":"text","text":"result"}]}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}

// BenchmarkResponseUnmarshal measures response JSON unmarshaling.
func BenchmarkResponseUnmarshal(b *testing.B) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"result"}]}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp Response
		_ = json.Unmarshal(data, &resp)
	}
}

// BenchmarkToolMarshal measures Tool struct marshaling.
func BenchmarkToolMarshal(b *testing.B) {
	tool := Tool{
		Name:        "bc_agent_list",
		Description: "List all agents in the workspace",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(tool)
	}
}

// BenchmarkToolCallResultMarshal measures ToolCallResult marshaling.
func BenchmarkToolCallResultMarshal(b *testing.B) {
	result := ToolCallResult{
		Content: []Content{
			{Type: "text", Text: "Agent eng-01 is idle"},
			{Type: "text", Text: "Agent eng-02 is working"},
		},
		IsError: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(result)
	}
}

// BenchmarkInitializeResultMarshal measures InitializeResult marshaling.
func BenchmarkInitializeResultMarshal(b *testing.B) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo:      ServerInfo{Name: "bc", Version: "1.0.0"},
		Capabilities: ServerCapabilities{
			Tools:     &ToolsCapability{ListChanged: true},
			Resources: &ResourcesCapability{Subscribe: true, ListChanged: true},
			Prompts:   &PromptsCapability{ListChanged: true},
			Logging:   &LoggingCapability{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(result)
	}
}

// BenchmarkContentSliceCreation measures creating content slices.
func BenchmarkContentSliceCreation(b *testing.B) {
	sizes := []int{1, 5, 10}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				contents := make([]Content, size)
				for j := 0; j < size; j++ {
					contents[j] = NewTextContent(fmt.Sprintf("content item %d", j))
				}
			}
		})
	}
}
