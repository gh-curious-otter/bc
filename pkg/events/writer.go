package events

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// DefaultMaxLines is the max number of lines kept in the JSONL file.
	DefaultMaxLines = 10000
	// rotationTrimLines is how many lines we keep after rotation (trim oldest).
	rotationTrimLines = 5000
)

// SSEEvent is a single persisted SSE event with its broadcast timestamp.
type SSEEvent struct {
	Data      any       `json:"data"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
}

// JSONLWriter appends SSE events to a JSONL file with line-count rotation.
// It is safe for concurrent use.
type JSONLWriter struct {
	path     string
	maxLines int
	mu       sync.Mutex
}

// NewJSONLWriter creates a writer that appends to the given path.
// maxLines controls when rotation triggers (0 = DefaultMaxLines).
func NewJSONLWriter(path string, maxLines int) *JSONLWriter {
	if maxLines <= 0 {
		maxLines = DefaultMaxLines
	}
	return &JSONLWriter{
		path:     path,
		maxLines: maxLines,
	}
}

// Write appends a single SSE event to the JSONL file.
func (w *JSONLWriter) Write(eventType string, data any) error {
	evt := SSEEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
	line, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal SSE event: %w", err)
	}
	line = append(line, '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open JSONL file: %w", err)
	}
	if _, err := f.Write(line); err != nil {
		_ = f.Close() //nolint:errcheck // best-effort close on write error
		return fmt.Errorf("write JSONL line: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close JSONL file: %w", err)
	}

	// Check if rotation is needed (count lines)
	count, countErr := w.lineCount()
	if countErr == nil && count > w.maxLines {
		_ = w.rotate() //nolint:errcheck // best-effort rotation
	}

	return nil
}

// ReadLast returns the last n events from the JSONL file, oldest first.
func (w *JSONLWriter) ReadLast(n int) ([]SSEEvent, int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	lines, err := w.readAllLines()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	total := len(lines)
	if n <= 0 || n > total {
		n = total
	}

	// Take the last n lines
	start := total - n
	result := make([]SSEEvent, 0, n)
	for _, line := range lines[start:] {
		var evt SSEEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // skip malformed lines
		}
		result = append(result, evt)
	}
	return result, total, nil
}

// ReadPage returns a page of events with offset/limit, oldest first.
// Returns events, total count, and any error.
func (w *JSONLWriter) ReadPage(limit, offset int) ([]SSEEvent, int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	lines, err := w.readAllLines()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	total := len(lines)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	result := make([]SSEEvent, 0, end-offset)
	for _, line := range lines[offset:end] {
		var evt SSEEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		result = append(result, evt)
	}
	return result, total, nil
}

// lineCount returns the number of lines in the file.
// Caller must hold w.mu.
func (w *JSONLWriter) lineCount() (int, error) {
	f, err := os.Open(w.path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// readAllLines reads all non-empty lines from the file.
// Caller must hold w.mu.
func (w *JSONLWriter) readAllLines() ([][]byte, error) {
	f, err := os.Open(w.path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	// Allow large lines (1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		cp := make([]byte, len(b))
		copy(cp, b)
		lines = append(lines, cp)
	}
	return lines, scanner.Err()
}

// rotate keeps only the newest rotationTrimLines lines.
// Caller must hold w.mu.
func (w *JSONLWriter) rotate() error {
	lines, err := w.readAllLines()
	if err != nil {
		return err
	}
	if len(lines) <= rotationTrimLines {
		return nil
	}

	// Keep only the last rotationTrimLines lines
	keep := lines[len(lines)-rotationTrimLines:]

	tmp := w.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create tmp for rotation: %w", err)
	}

	bw := bufio.NewWriter(f)
	for _, line := range keep {
		if _, err := bw.Write(line); err != nil {
			_ = f.Close() //nolint:errcheck
			_ = os.Remove(tmp)
			return fmt.Errorf("write rotated line: %w", err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			_ = f.Close() //nolint:errcheck
			_ = os.Remove(tmp)
			return fmt.Errorf("write newline: %w", err)
		}
	}
	if err := bw.Flush(); err != nil {
		_ = f.Close() //nolint:errcheck
		_ = os.Remove(tmp)
		return fmt.Errorf("flush rotated file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close rotated file: %w", err)
	}

	return os.Rename(tmp, w.path)
}
