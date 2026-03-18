package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ServeStdio runs the MCP server over stdin/stdout using newline-delimited JSON.
// It blocks until ctx is cancelled or stdin is closed.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.serveStdio(ctx, os.Stdin, os.Stdout)
}

// ServeStdioRW runs the MCP server using the provided reader/writer instead of
// os.Stdin/os.Stdout. Useful for testing.
func (s *Server) ServeStdioRW(ctx context.Context, r io.Reader, w io.Writer) error {
	return s.serveStdio(ctx, r, w)
}

// serveStdio is the inner implementation.
func (s *Server) serveStdio(ctx context.Context, r io.Reader, w io.Writer) error {
	enc := json.NewEncoder(w)
	scanner := bufio.NewScanner(r)
	// Allow lines up to 4 MB (large prompts in tool args)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("stdio read error: %w", err)
			}
			return nil // EOF — client closed the connection
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := errResponse(nil, ErrParse, "parse error: "+err.Error())
			_ = enc.Encode(resp)
			continue
		}

		resp := s.Handle(ctx, req)

		// Don't send a response for notifications (no ID, no method result/error)
		if req.ID == nil {
			continue
		}

		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("stdio write error: %w", err)
		}
	}
}
