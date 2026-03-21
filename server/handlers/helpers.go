// Package handlers implements HTTP handlers for the bcd REST API.
// Each handler file covers one resource (agents, channels, workspace, etc.).
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rpuneet/bc/pkg/log"
)

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Debug("json encode failed", "error", err)
	}
}

// httpError writes a JSON error response.
func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck // best-effort
}

// methodNotAllowed writes a 405 response.
func methodNotAllowed(w http.ResponseWriter) {
	httpError(w, "method not allowed", http.StatusMethodNotAllowed)
}

// requireMethod returns false and writes 405 if the request method is not one of the allowed methods.
func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	methodNotAllowed(w)
	return false
}

// Recovery returns a middleware that recovers from panics, logs the error,
// and returns a 500 JSON response instead of crashing the server.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered", "error", err, "method", r.Method, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"}) //nolint:errcheck
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// clampInt clamps n to the range [min, max].
func clampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// CORSWithOrigin returns a middleware that adds CORS headers with the specified
// allowed origin. Use "*" for permissive (safe on loopback) or a specific
// origin like "http://localhost:9374" when exposed beyond loopback.
func CORSWithOrigin(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CORS returns a middleware with permissive CORS headers (Allow-Origin: *).
// Safe because bcd only binds to loopback by default.
func CORS(next http.Handler) http.Handler {
	return CORSWithOrigin("*", next)
}
