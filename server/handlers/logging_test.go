package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusRecorder(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		rec.WriteHeader(http.StatusNotFound)
		if rec.status != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.status)
		}
	})

	t.Run("defaults to 200", func(t *testing.T) {
		rec := &statusRecorder{status: 200}
		if rec.status != 200 {
			t.Errorf("expected default 200, got %d", rec.status)
		}
	})
}

func TestRequestLoggerMiddleware(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("normal request is logged", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("SSE events path is skipped", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("MCP sse path is skipped", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/mcp/sse", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestRequestLogger_ErrorStatus(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/channels", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
