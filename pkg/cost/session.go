package cost

import (
	"bufio"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionEntry is a parsed assistant message from a Claude Code JSONL session file.
type SessionEntry struct {
	Timestamp           time.Time
	SessionID           string
	Model               string
	CWD                 string
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// jsonlEvent is the minimal structure we decode from each JSONL line.
type jsonlEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	CWD       string          `json:"cwd"`
	Message   json.RawMessage `json:"message"`
}

type jsonlMessage struct {
	Model string     `json:"model"`
	Usage jsonlUsage `json:"usage"`
}

type jsonlUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// ParseSessionFile reads a Claude Code JSONL session file and returns all
// assistant message entries that contain token usage.
func ParseSessionFile(path string) ([]SessionEntry, error) {
	f, err := os.Open(path) //nolint:gosec // path constructed from workspace dir
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return parseSessionReader(f)
}

func parseSessionReader(r io.Reader) ([]SessionEntry, error) {
	var entries []SessionEntry
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1 MiB line buffer

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var evt jsonlEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // skip malformed lines
		}
		if evt.Type != "assistant" || len(evt.Message) == 0 {
			continue
		}

		var msg jsonlMessage
		if err := json.Unmarshal(evt.Message, &msg); err != nil {
			continue
		}
		// Only include entries that have actual token usage recorded.
		if msg.Usage.InputTokens == 0 && msg.Usage.OutputTokens == 0 &&
			msg.Usage.CacheCreationInputTokens == 0 && msg.Usage.CacheReadInputTokens == 0 {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, evt.Timestamp)
		if ts.IsZero() {
			ts = time.Now()
		}

		entries = append(entries, SessionEntry{
			Timestamp:           ts,
			SessionID:           evt.SessionID,
			Model:               msg.Model,
			CWD:                 evt.CWD,
			InputTokens:         msg.Usage.InputTokens,
			OutputTokens:        msg.Usage.OutputTokens,
			CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadTokens:     msg.Usage.CacheReadInputTokens,
		})
	}
	return entries, scanner.Err()
}

// FindSessionFiles returns all .jsonl session file paths under the given
// Claude projects root directory (~/.claude/projects/).
func FindSessionFiles(claudeProjectsDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(claudeProjectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible directories
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
