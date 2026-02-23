package peek

import (
	"sync"
)

// RingBuffer is a thread-safe circular buffer for storing log lines.
//
//nolint:govet // fieldalignment: logical field grouping preferred
type RingBuffer struct {
	mu    sync.RWMutex
	lines []string
	head  int // Next write position
	count int // Number of lines stored
	size  int // Maximum capacity
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &RingBuffer{
		lines: make([]string, size),
		size:  size,
	}
}

// Add appends a line to the buffer.
// If the buffer is full, the oldest line is overwritten.
func (b *RingBuffer) Add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines[b.head] = line
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}
}

// Lines returns the last n lines in chronological order.
// If n > count, returns all available lines.
// If n <= 0, returns all lines.
func (b *RingBuffer) Lines(n int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 || n > b.count {
		n = b.count
	}

	if n == 0 {
		return nil
	}

	result := make([]string, n)

	// Calculate starting position
	// head points to next write, so oldest line is at head (if full) or 0 (if not full)
	// We want the last n lines before head
	start := (b.head - n + b.size) % b.size

	for i := 0; i < n; i++ {
		result[i] = b.lines[(start+i)%b.size]
	}

	return result
}

// Count returns the number of lines currently stored.
func (b *RingBuffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// Clear removes all lines from the buffer.
func (b *RingBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.lines {
		b.lines[i] = ""
	}
	b.head = 0
	b.count = 0
}

// Last returns the most recent line, or empty string if buffer is empty.
func (b *RingBuffer) Last() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return ""
	}

	lastIdx := (b.head - 1 + b.size) % b.size
	return b.lines[lastIdx]
}

// All returns all lines in chronological order.
func (b *RingBuffer) All() []string {
	return b.Lines(0)
}
