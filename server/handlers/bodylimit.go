package handlers

import (
	"net/http"
	"strings"
)

const (
	// DefaultMaxBodySize is the default request body limit (1MB).
	DefaultMaxBodySize = 1 << 20 // 1MB

	// MCPMaxBodySize is the body limit for MCP message endpoints (4MB).
	MCPMaxBodySize = 4 << 20 // 4MB
)

// MaxBodySize returns middleware that limits request body size.
// Returns 413 Payload Too Large if the body exceeds maxBytes.
// MCP endpoints (/mcp/message) get a larger limit per protocol spec.
func MaxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.ContentLength == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// MCP message endpoint gets larger limit
		limit := int64(DefaultMaxBodySize)
		if strings.HasPrefix(r.URL.Path, "/mcp/") {
			limit = MCPMaxBodySize
		}

		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
