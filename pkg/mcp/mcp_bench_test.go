package mcp

import (
	"encoding/json"
	"errors"
	"testing"
)

// --- ConnectionState benchmarks ---

func BenchmarkConnectionState_String(b *testing.B) {
	states := []ConnectionState{StateDisconnected, StateConnecting, StateConnected, StateError}

	b.ResetTimer()
	for i := range b.N {
		_ = states[i%len(states)].String()
	}
}

// --- Content creation benchmarks ---

func BenchmarkNewTextContent(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = NewTextContent("hello world")
	}
}

func BenchmarkNewTextContent_Long(b *testing.B) {
	longText := "This is a longer piece of text that simulates a more realistic content payload with multiple sentences and details."

	b.ResetTimer()
	for range b.N {
		_ = NewTextContent(longText)
	}
}

func BenchmarkNewErrorContent(b *testing.B) {
	err := errors.New("test error message")

	b.ResetTimer()
	for range b.N {
		_ = NewErrorContent(err)
	}
}

// --- Request creation benchmarks ---

func BenchmarkNewRequest_NilParams(b *testing.B) {
	b.ResetTimer()
	for i := range b.N {
		_, _ = NewRequest(i, MethodToolsList, nil)
	}
}

func BenchmarkNewRequest_SimpleParams(b *testing.B) {
	params := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewRequest(i, MethodToolsCall, params)
	}
}

func BenchmarkNewRequest_ComplexParams(b *testing.B) {
	params := ToolCallParams{
		Name:      "test_tool",
		Arguments: json.RawMessage(`{"arg1":"value1","arg2":42,"arg3":true}`),
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewRequest(i, MethodToolsCall, params)
	}
}

func BenchmarkNewRequest_InitializeParams(b *testing.B) {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo: ClientInfo{
			Name:    "bc",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{
			Roots:    &RootsCapability{ListChanged: true},
			Sampling: &SamplingCapability{},
		},
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewRequest(i, MethodInitialize, params)
	}
}

// --- Response creation benchmarks ---

func BenchmarkNewResponse_NilResult(b *testing.B) {
	b.ResetTimer()
	for i := range b.N {
		_, _ = NewResponse(i, nil)
	}
}

func BenchmarkNewResponse_SimpleResult(b *testing.B) {
	result := map[string]int{"count": 42}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewResponse(i, result)
	}
}

func BenchmarkNewResponse_ToolsList(b *testing.B) {
	result := struct {
		Tools []Tool `json:"tools"`
	}{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "tool2", Description: "Second tool", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "tool3", Description: "Third tool", InputSchema: json.RawMessage(`{"type":"object"}`)},
		},
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewResponse(i, result)
	}
}

func BenchmarkNewResponse_InitializeResult(b *testing.B) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: ServerInfo{
			Name:    "bc-mcp",
			Version: "1.0.0",
		},
		Capabilities: ServerCapabilities{
			Tools:     &ToolsCapability{ListChanged: true},
			Resources: &ResourcesCapability{Subscribe: true, ListChanged: true},
			Prompts:   &PromptsCapability{ListChanged: true},
			Logging:   &LoggingCapability{},
		},
	}

	b.ResetTimer()
	for i := range b.N {
		_, _ = NewResponse(i, result)
	}
}

// --- Error response benchmarks ---

func BenchmarkNewErrorResponse(b *testing.B) {
	b.ResetTimer()
	for i := range b.N {
		_ = NewErrorResponse(i, ErrCodeInvalidParams, "invalid params", nil)
	}
}

func BenchmarkNewErrorResponse_WithData(b *testing.B) {
	data := map[string]string{"field": "name", "issue": "required"}

	b.ResetTimer()
	for i := range b.N {
		_ = NewErrorResponse(i, ErrCodeInvalidParams, "invalid params", data)
	}
}

// --- JSON marshaling benchmarks ---

func BenchmarkRequest_Marshal(b *testing.B) {
	req, _ := NewRequest(1, MethodToolsList, nil)

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkRequest_Unmarshal(b *testing.B) {
	req, _ := NewRequest(1, MethodToolsList, nil)
	data, _ := json.Marshal(req)

	b.ResetTimer()
	for range b.N {
		var decoded Request
		_ = json.Unmarshal(data, &decoded)
	}
}

func BenchmarkResponse_Marshal(b *testing.B) {
	resp, _ := NewResponse(1, map[string]int{"count": 42})

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(resp)
	}
}

func BenchmarkResponse_Unmarshal(b *testing.B) {
	resp, _ := NewResponse(1, map[string]int{"count": 42})
	data, _ := json.Marshal(resp)

	b.ResetTimer()
	for range b.N {
		var decoded Response
		_ = json.Unmarshal(data, &decoded)
	}
}

func BenchmarkTool_Marshal(b *testing.B) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool for benchmarking",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"arg1":{"type":"string"}}}`),
	}

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(tool)
	}
}

func BenchmarkTool_Unmarshal(b *testing.B) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool for benchmarking",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"arg1":{"type":"string"}}}`),
	}
	data, _ := json.Marshal(tool)

	b.ResetTimer()
	for range b.N {
		var decoded Tool
		_ = json.Unmarshal(data, &decoded)
	}
}

func BenchmarkResource_Marshal(b *testing.B) {
	resource := Resource{
		URI:         "agents://eng-01",
		Name:        "Agent eng-01",
		Description: "Engineer agent",
		MimeType:    "application/json",
	}

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(resource)
	}
}

func BenchmarkResource_Unmarshal(b *testing.B) {
	resource := Resource{
		URI:         "agents://eng-01",
		Name:        "Agent eng-01",
		Description: "Engineer agent",
		MimeType:    "application/json",
	}
	data, _ := json.Marshal(resource)

	b.ResetTimer()
	for range b.N {
		var decoded Resource
		_ = json.Unmarshal(data, &decoded)
	}
}

func BenchmarkToolCallResult_Marshal(b *testing.B) {
	result := ToolCallResult{
		Content: []Content{
			NewTextContent("Result line 1"),
			NewTextContent("Result line 2"),
			NewTextContent("Result line 3"),
		},
		IsError: false,
	}

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(result)
	}
}

func BenchmarkPromptMessage_Marshal(b *testing.B) {
	msg := PromptMessage{
		Role: "user",
		Content: []Content{
			NewTextContent("Please analyze this code and provide feedback."),
		},
	}

	b.ResetTimer()
	for range b.N {
		_, _ = json.Marshal(msg)
	}
}

// --- Parallel benchmarks ---

func BenchmarkNewTextContent_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewTextContent("parallel text content")
		}
	})
}

func BenchmarkNewRequest_Parallel(b *testing.B) {
	params := map[string]string{"key": "value"}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = NewRequest(i, MethodToolsCall, params)
			i++
		}
	})
}

func BenchmarkNewResponse_Parallel(b *testing.B) {
	result := map[string]int{"count": 42}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = NewResponse(i, result)
			i++
		}
	})
}

func BenchmarkConnectionState_String_Parallel(b *testing.B) {
	states := []ConnectionState{StateDisconnected, StateConnecting, StateConnected, StateError}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = states[i%len(states)].String()
			i++
		}
	})
}
