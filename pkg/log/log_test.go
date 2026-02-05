package log

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentSetVerbose tests that concurrent SetVerbose calls don't race.
func TestConcurrentSetVerbose(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			SetVerbose(n%2 == 0)
		}(i)
	}
	wg.Wait()
}

// TestConcurrentLogging tests that concurrent log calls don't race.
func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetVerbose(true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Debug("debug message", "n", n)
			Info("info message", "n", n)
			Warn("warn message", "n", n)
			Error("error message", "n", n)
		}(i)
	}
	wg.Wait()
}

// TestConcurrentSetOutputAndLog tests concurrent SetOutput and logging.
func TestConcurrentSetOutputAndLog(t *testing.T) {
	var wg sync.WaitGroup

	// Loggers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Info("message", "n", n)
		}(i)
	}

	// Output setters
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			SetOutput(&buf)
		}()
	}

	wg.Wait()
}

// TestDebugOnlyWithVerbose tests that Debug only outputs when verbose is enabled.
func TestDebugOnlyWithVerbose(t *testing.T) {
	var buf bytes.Buffer

	// Verbose off - Debug should not appear
	SetVerbose(false)
	SetOutput(&buf)
	Debug("test debug")
	if strings.Contains(buf.String(), "test debug") {
		t.Error("Debug message appeared when verbose was false")
	}

	// Verbose on - Debug should appear
	buf.Reset()
	SetVerbose(true)
	SetOutput(&buf)
	Debug("test debug 2")
	if !strings.Contains(buf.String(), "test debug 2") {
		t.Error("Debug message did not appear when verbose was true")
	}
}

// TestInfoAlwaysLogs tests that Info logs regardless of verbose setting.
func TestInfoAlwaysLogs(t *testing.T) {
	var buf bytes.Buffer

	SetVerbose(false)
	SetOutput(&buf)
	Info("test info")
	if !strings.Contains(buf.String(), "test info") {
		t.Error("Info message did not appear")
	}
}

// TestLoggerReturnsLogger tests that Logger() returns a valid logger.
func TestLoggerReturnsLogger(t *testing.T) {
	l := Logger()
	if l == nil {
		t.Error("Logger() returned nil")
	}
}

// TestWithReturnsLogger tests that With() returns a valid logger.
func TestWithReturnsLogger(t *testing.T) {
	l := With("key", "value")
	if l == nil {
		t.Error("With() returned nil")
	}
}
