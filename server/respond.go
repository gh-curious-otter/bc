package server

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Error writes a JSON error response with the given status code and message.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]any{
		"error": msg,
		"code":  status,
	})
}

// decode reads a JSON request body into v.
func decode(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
