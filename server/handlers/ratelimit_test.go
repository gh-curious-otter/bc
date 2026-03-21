package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allows burst requests", func(t *testing.T) {
		rl := NewRateLimiter(10, 5) // 10 rps, burst 5
		for i := 0; i < 5; i++ {
			if !rl.Allow() {
				t.Errorf("request %d should be allowed within burst", i)
			}
		}
	})

	t.Run("rejects after burst exhausted", func(t *testing.T) {
		rl := NewRateLimiter(10, 3) // 10 rps, burst 3
		for i := 0; i < 3; i++ {
			rl.Allow() // exhaust burst
		}
		if rl.Allow() {
			t.Error("request after burst should be rejected")
		}
	})

	t.Run("initial burst equals burst param", func(t *testing.T) {
		rl := NewRateLimiter(0, 3) // 0 rps (no refill), burst 3
		allowed := 0
		for i := 0; i < 10; i++ {
			if rl.Allow() {
				allowed++
			}
		}
		if allowed != 3 {
			t.Errorf("expected exactly 3 allowed from burst, got %d", allowed)
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("passes when under limit", func(t *testing.T) {
		rl := NewRateLimiter(100, 10)
		handler := RateLimit(rl)(okHandler)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 429 when over limit", func(t *testing.T) {
		rl := NewRateLimiter(100, 1) // burst of 1
		handler := RateLimit(rl)(okHandler)

		// First request uses the burst token
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))
		if w1.Code != http.StatusOK {
			t.Errorf("first request: expected 200, got %d", w1.Code)
		}

		// Second request should be rate limited
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/", nil))
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("second request: expected 429, got %d", w2.Code)
		}
		if got := w2.Header().Get("Retry-After"); got != "1" {
			t.Errorf("expected Retry-After: 1, got %q", got)
		}
	})
}
