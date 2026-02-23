// Package peek provides background log streaming and status parsing for agents.
// Issue #1687: Agent peek background log streaming
package peek

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DefaultBufferSize is the default number of lines to keep in the ring buffer.
const DefaultBufferSize = 1000

// DefaultPollInterval is the default interval between log captures.
const DefaultPollInterval = 500 * time.Millisecond

// State represents the detected state of an agent.
type State string

const (
	StateUnknown State = "unknown"
	StateIdle    State = "idle"
	StateWorking State = "working"
	StateDone    State = "done"
	StateError   State = "error"
	StateStuck   State = "stuck"
)

// TokenUsage tracks token consumption from agent output.
type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	CostUSD      float64
}

// Status represents the current status of an agent.
//
//nolint:govet // fieldalignment: logical field grouping preferred
type Status struct {
	State      State
	Task       string
	Tokens     TokenUsage
	LastUpdate time.Time
}

// CaptureFunc is a function that captures output from an agent's session.
type CaptureFunc func(name string, lines int) (string, error)

// Streamer manages background log streaming for agents.
//
//nolint:govet // fieldalignment: logical field grouping preferred
type Streamer struct {
	mu           sync.RWMutex
	agents       map[string]*agentStream
	captureFunc  CaptureFunc
	bufferSize   int
	pollInterval time.Duration
}

// agentStream holds streaming state for a single agent.
//
//nolint:govet // fieldalignment: logical field grouping preferred
type agentStream struct {
	mu           sync.RWMutex
	buffer       *RingBuffer
	status       Status
	cancel       context.CancelFunc
	lastCapture  string
	pollInterval time.Duration
}

// NewStreamer creates a new log streamer.
func NewStreamer(captureFunc CaptureFunc) *Streamer {
	return &Streamer{
		agents:       make(map[string]*agentStream),
		captureFunc:  captureFunc,
		bufferSize:   DefaultBufferSize,
		pollInterval: DefaultPollInterval,
	}
}

// SetBufferSize sets the ring buffer size for new streams.
func (s *Streamer) SetBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bufferSize = size
}

// SetPollInterval sets the polling interval for new streams.
func (s *Streamer) SetPollInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollInterval = interval
}

// Start begins streaming logs for an agent.
// If already streaming, this is a no-op.
func (s *Streamer) Start(ctx context.Context, agentName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agents[agentName]; exists {
		return // Already streaming
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream := &agentStream{
		buffer:       NewRingBuffer(s.bufferSize),
		status:       Status{State: StateUnknown, LastUpdate: time.Now()},
		cancel:       cancel,
		pollInterval: s.pollInterval,
	}
	s.agents[agentName] = stream

	go s.streamLoop(streamCtx, agentName, stream)
}

// Stop stops streaming logs for an agent.
func (s *Streamer) Stop(agentName string) {
	s.mu.Lock()
	stream, exists := s.agents[agentName]
	if exists {
		stream.cancel()
		delete(s.agents, agentName)
	}
	s.mu.Unlock()
}

// StopAll stops all streaming.
func (s *Streamer) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, stream := range s.agents {
		stream.cancel()
		delete(s.agents, name)
	}
}

// GetLines returns the last N lines from the agent's buffer.
func (s *Streamer) GetLines(agentName string, n int) []string {
	s.mu.RLock()
	stream, exists := s.agents[agentName]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()
	return stream.buffer.Lines(n)
}

// GetStatus returns the current status for an agent.
func (s *Streamer) GetStatus(agentName string) Status {
	s.mu.RLock()
	stream, exists := s.agents[agentName]
	s.mu.RUnlock()

	if !exists {
		return Status{State: StateUnknown}
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()
	return stream.status
}

// IsStreaming returns true if streaming is active for an agent.
func (s *Streamer) IsStreaming(agentName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.agents[agentName]
	return exists
}

// streamLoop runs the background streaming for an agent.
func (s *Streamer) streamLoop(ctx context.Context, agentName string, stream *agentStream) {
	ticker := time.NewTicker(stream.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.captureAndParse(agentName, stream)
		}
	}
}

// captureAndParse captures output and parses status.
func (s *Streamer) captureAndParse(agentName string, stream *agentStream) {
	output, err := s.captureFunc(agentName, 100) // Capture 100 lines at a time
	if err != nil {
		return // Ignore capture errors (session may be gone)
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Only process if output changed
	if output == stream.lastCapture {
		return
	}
	stream.lastCapture = output

	// Add new lines to buffer
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line != "" {
			stream.buffer.Add(line)
		}
	}

	// Parse status from output
	stream.status = parseStatus(output)
	stream.status.LastUpdate = time.Now()
}

// parseStatus extracts agent status from output.
func parseStatus(output string) Status {
	status := Status{State: StateUnknown}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return status
	}

	// Parse state from last few lines
	status.State = detectState(lines)

	// Extract task from recent activity
	status.Task = extractTask(lines)

	// Extract token usage if present
	status.Tokens = extractTokens(output)

	return status
}

// Claude Code spinner and state patterns
var (
	// Working patterns
	workingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^[✻✳✽·⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`), // Spinner characters
		regexp.MustCompile(`^⏺`),                            // Tool call indicator
		regexp.MustCompile(`(?i)thinking`),                  // Thinking indicator
		regexp.MustCompile(`(?i)reading|writing|analyzing`), // Action verbs
	}

	// Done patterns
	donePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^[✓✔]`),                   // Checkmarks
		regexp.MustCompile(`(?i)completed?|finished`), // Completion words
		regexp.MustCompile(`(?i)success`),             // Success indicator
	}

	// Idle patterns
	idlePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^❯`),               // Claude prompt
		regexp.MustCompile(`^>`),               // Generic prompt
		regexp.MustCompile(`(?i)waiting|idle`), // Waiting indicators
	}

	// Error patterns
	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^error:`),             // Error prefix
		regexp.MustCompile(`(?i)failed|failure`),      // Failure words
		regexp.MustCompile(`^[✗✘❌]`),                  // Error symbols
		regexp.MustCompile(`(?i)exception|traceback`), // Exception indicators
	}

	// Stuck patterns
	stuckPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)rate.?limit`), // Rate limiting
		regexp.MustCompile(`(?i)timeout`),     // Timeout
		regexp.MustCompile(`(?i)quota`),       // Quota exceeded
		regexp.MustCompile(`(?i)api.?key`),    // API key issues
	}

	// Token usage patterns - Claude Code format
	tokenPattern = regexp.MustCompile(`(?i)(\d+(?:,\d+)?)\s*(?:input|prompt)\s*(?:tokens?).*?(\d+(?:,\d+)?)\s*(?:output|completion)\s*(?:tokens?)`)
	costPattern  = regexp.MustCompile(`\$(\d+\.?\d*)`)
)

// detectState determines the state from output lines.
func detectState(lines []string) State {
	// Check last 5 lines for state indicators
	start := len(lines) - 5
	if start < 0 {
		start = 0
	}

	for i := len(lines) - 1; i >= start; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Check patterns in priority order
		for _, p := range errorPatterns {
			if p.MatchString(line) {
				return StateError
			}
		}

		for _, p := range stuckPatterns {
			if p.MatchString(line) {
				return StateStuck
			}
		}

		for _, p := range donePatterns {
			if p.MatchString(line) {
				return StateDone
			}
		}

		for _, p := range workingPatterns {
			if p.MatchString(line) {
				return StateWorking
			}
		}

		for _, p := range idlePatterns {
			if p.MatchString(line) {
				return StateIdle
			}
		}
	}

	return StateUnknown
}

// extractTask extracts the current task from output.
func extractTask(lines []string) string {
	// Look for spinner lines with task description
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-10; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Skip status bar noise
		if strings.Contains(line, "bypass permissions") ||
			strings.Contains(line, "shift+Tab") ||
			strings.Contains(line, "Update available") {
			continue
		}

		// Spinner with task description
		for _, p := range workingPatterns[:2] { // Only check spinner patterns
			if p.MatchString(line) {
				// Remove parenthesized timing info
				if idx := strings.LastIndex(line, "("); idx > 20 {
					return strings.TrimSpace(line[:idx])
				}
				return line
			}
		}
	}
	return ""
}

// extractTokens parses token usage from output.
func extractTokens(output string) TokenUsage {
	usage := TokenUsage{}

	// Look for token counts
	if matches := tokenPattern.FindStringSubmatch(output); len(matches) >= 3 {
		usage.InputTokens = parseTokenCount(matches[1])
		usage.OutputTokens = parseTokenCount(matches[2])
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}

	// Look for cost
	if matches := costPattern.FindStringSubmatch(output); len(matches) >= 2 {
		// Parse cost (simplified)
		if cost, err := parseFloat(matches[1]); err == nil {
			usage.CostUSD = cost
		}
	}

	return usage
}

// parseTokenCount parses a token count string (handles commas).
func parseTokenCount(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

// parseFloat parses a float string.
func parseFloat(s string) (float64, error) {
	var result float64
	var decimal float64 = 0
	var decimalPlaces float64 = 1

	for _, c := range s {
		if c == '.' {
			decimal = 1
		} else if c >= '0' && c <= '9' {
			if decimal > 0 {
				decimalPlaces *= 10
				result += float64(c-'0') / decimalPlaces
			} else {
				result = result*10 + float64(c-'0')
			}
		}
	}
	return result, nil
}
