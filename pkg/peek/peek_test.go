package peek

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_Basic(t *testing.T) {
	b := NewRingBuffer(5)

	// Test empty buffer
	if b.Count() != 0 {
		t.Errorf("Count() = %d, want 0", b.Count())
	}
	if b.Last() != "" {
		t.Errorf("Last() = %q, want empty", b.Last())
	}

	// Add some lines
	b.Add("line1")
	b.Add("line2")
	b.Add("line3")

	if b.Count() != 3 {
		t.Errorf("Count() = %d, want 3", b.Count())
	}
	if b.Last() != "line3" {
		t.Errorf("Last() = %q, want line3", b.Last())
	}

	// Get last 2 lines
	lines := b.Lines(2)
	if len(lines) != 2 {
		t.Errorf("Lines(2) len = %d, want 2", len(lines))
	}
	if lines[0] != "line2" || lines[1] != "line3" {
		t.Errorf("Lines(2) = %v, want [line2 line3]", lines)
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	b := NewRingBuffer(3)

	// Fill buffer
	b.Add("a")
	b.Add("b")
	b.Add("c")

	// Wrap around
	b.Add("d")
	b.Add("e")

	// Should have b, c, d, e with oldest (a) overwritten
	// Actually with size 3, we should have c, d, e
	if b.Count() != 3 {
		t.Errorf("Count() = %d, want 3", b.Count())
	}

	lines := b.All()
	if len(lines) != 3 {
		t.Errorf("All() len = %d, want 3", len(lines))
	}
	// Check order: c, d, e
	expected := []string{"c", "d", "e"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want)
		}
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	b := NewRingBuffer(5)
	b.Add("line1")
	b.Add("line2")

	b.Clear()

	if b.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", b.Count())
	}
	if b.Last() != "" {
		t.Errorf("Last() after Clear() = %q, want empty", b.Last())
	}
}

func TestRingBuffer_LinesEdgeCases(t *testing.T) {
	b := NewRingBuffer(5)
	b.Add("a")
	b.Add("b")

	// Request more lines than available
	lines := b.Lines(10)
	if len(lines) != 2 {
		t.Errorf("Lines(10) len = %d, want 2", len(lines))
	}

	// Request zero or negative
	lines = b.Lines(0)
	if len(lines) != 2 {
		t.Errorf("Lines(0) len = %d, want 2", len(lines))
	}

	lines = b.Lines(-1)
	if len(lines) != 2 {
		t.Errorf("Lines(-1) len = %d, want 2", len(lines))
	}
}

func TestDetectState(t *testing.T) {
	tests := []struct { //nolint:govet // test table
		name  string
		lines []string
		want  State
	}{
		{
			name:  "working spinner",
			lines: []string{"✻ Reading files..."},
			want:  StateWorking,
		},
		{
			name:  "working tool",
			lines: []string{"⏺ Running bash command"},
			want:  StateWorking,
		},
		{
			name:  "working thinking",
			lines: []string{"I'm thinking about this..."},
			want:  StateWorking,
		},
		{
			name:  "done checkmark",
			lines: []string{"✓ Task complete"},
			want:  StateDone,
		},
		{
			name:  "done completed",
			lines: []string{"Task completed successfully"},
			want:  StateDone,
		},
		{
			name:  "idle prompt",
			lines: []string{"❯ "},
			want:  StateIdle,
		},
		{
			name:  "error",
			lines: []string{"Error: file not found"},
			want:  StateError,
		},
		{
			name:  "error cross",
			lines: []string{"❌ Build failed"},
			want:  StateError,
		},
		{
			name:  "stuck rate limit",
			lines: []string{"Rate limit exceeded"},
			want:  StateStuck,
		},
		{
			name:  "stuck timeout",
			lines: []string{"Request timeout"},
			want:  StateStuck,
		},
		{
			name:  "unknown",
			lines: []string{"some random text"},
			want:  StateUnknown,
		},
		{
			name:  "empty",
			lines: []string{},
			want:  StateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectState(tt.lines)
			if got != tt.want {
				t.Errorf("detectState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTask(t *testing.T) {
	tests := []struct { //nolint:govet // test table
		name  string
		lines []string
		want  string
	}{
		{
			name:  "spinner with task",
			lines: []string{"✻ Reading pkg/agent/agent.go"},
			want:  "✻ Reading pkg/agent/agent.go",
		},
		{
			name:  "spinner with timing",
			lines: []string{"✻ Writing comprehensive test suite (3.2s elapsed)"},
			want:  "✻ Writing comprehensive test suite",
		},
		{
			name:  "no task",
			lines: []string{"some random output"},
			want:  "",
		},
		{
			name:  "skip status bar",
			lines: []string{"shift+Tab to cycle"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTask(tt.lines)
			if got != tt.want {
				t.Errorf("extractTask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTokens(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   TokenUsage
	}{
		{
			name:   "with tokens",
			output: "Used 1,234 input tokens and 567 output tokens",
			want:   TokenUsage{InputTokens: 1234, OutputTokens: 567, TotalTokens: 1801},
		},
		{
			name:   "with cost",
			output: "Cost: $0.05",
			want:   TokenUsage{CostUSD: 0.05},
		},
		{
			name:   "no tokens",
			output: "some random output",
			want:   TokenUsage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTokens(tt.output)
			if got.InputTokens != tt.want.InputTokens {
				t.Errorf("InputTokens = %d, want %d", got.InputTokens, tt.want.InputTokens)
			}
			if got.OutputTokens != tt.want.OutputTokens {
				t.Errorf("OutputTokens = %d, want %d", got.OutputTokens, tt.want.OutputTokens)
			}
			if got.TotalTokens != tt.want.TotalTokens {
				t.Errorf("TotalTokens = %d, want %d", got.TotalTokens, tt.want.TotalTokens)
			}
			if got.CostUSD != tt.want.CostUSD {
				t.Errorf("CostUSD = %f, want %f", got.CostUSD, tt.want.CostUSD)
			}
		})
	}
}

func TestStreamer_StartStop(t *testing.T) {
	var mu sync.Mutex
	captureCount := 0

	capture := func(name string, lines int) (string, error) {
		mu.Lock()
		captureCount++
		mu.Unlock()
		return "✻ Working...\nline2\n", nil
	}

	s := NewStreamer(capture)
	s.SetPollInterval(50 * time.Millisecond)

	ctx := context.Background()
	s.Start(ctx, "test-agent")

	if !s.IsStreaming("test-agent") {
		t.Error("expected agent to be streaming")
	}

	// Wait for a few captures
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := captureCount
	mu.Unlock()

	if count < 2 {
		t.Errorf("expected at least 2 captures, got %d", count)
	}

	// Get status
	status := s.GetStatus("test-agent")
	if status.State != StateWorking {
		t.Errorf("expected state Working, got %v", status.State)
	}

	// Stop
	s.Stop("test-agent")
	if s.IsStreaming("test-agent") {
		t.Error("expected agent to not be streaming after stop")
	}
}

func TestStreamer_GetLines(t *testing.T) {
	capture := func(name string, lines int) (string, error) {
		return "line1\nline2\nline3\n", nil
	}

	s := NewStreamer(capture)
	s.SetPollInterval(50 * time.Millisecond)

	ctx := context.Background()
	s.Start(ctx, "test-agent")

	// Wait for capture
	time.Sleep(150 * time.Millisecond)

	lines := s.GetLines("test-agent", 2)
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines, got %d", len(lines))
	}

	s.StopAll()
}

func TestStreamer_NotStreaming(t *testing.T) {
	s := NewStreamer(nil)

	// Get status for non-streaming agent
	status := s.GetStatus("nonexistent")
	if status.State != StateUnknown {
		t.Errorf("expected StateUnknown for non-streaming agent, got %v", status.State)
	}

	// Get lines for non-streaming agent
	lines := s.GetLines("nonexistent", 10)
	if lines != nil {
		t.Errorf("expected nil lines for non-streaming agent, got %v", lines)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0.05", 0.05},
		{"1.23", 1.23},
		{"10", 10.0},
		{"0.001", 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseFloat(tt.input)
			if err != nil {
				t.Errorf("parseFloat(%q) error = %v", tt.input, err)
			}
			// Allow small floating point difference
			if diff := got - tt.want; diff > 0.0001 || diff < -0.0001 {
				t.Errorf("parseFloat(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}
