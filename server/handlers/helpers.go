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

// CORS returns a middleware that adds permissive CORS headers.
// This is safe because bcd only binds to loopback by default.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
