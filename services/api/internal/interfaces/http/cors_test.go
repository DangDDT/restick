package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiServerCORS(t *testing.T) {
	s := &Server{
		webOrigins: []string{"https://a.com", "https://b.com"},
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := s.cors(dummyHandler)

	t.Run("allowed origin https://a.com", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		req.Header.Set("Origin", "https://a.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com" {
			t.Errorf("expected Access-Control-Allow-Origin https://a.com, got %q", got)
		}
		if got := rec.Header().Get("Vary"); got != "Origin" {
			t.Errorf("expected Vary: Origin, got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
			t.Errorf("expected Access-Control-Max-Age 86400, got %q", got)
		}
		if got := rec.Header().Get("Content-Security-Policy"); got != "frame-ancestors https://a.com https://b.com" {
			t.Errorf("expected CSP frame-ancestors, got %q", got)
		}
	})

	t.Run("allowed origin https://b.com", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		req.Header.Set("Origin", "https://b.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://b.com" {
			t.Errorf("expected Access-Control-Allow-Origin https://b.com, got %q", got)
		}
		if got := rec.Header().Get("Vary"); got != "Origin" {
			t.Errorf("expected Vary: Origin, got %q", got)
		}
	})

	t.Run("disallowed origin https://evil.com", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("expected empty Access-Control-Allow-Origin, got %q", got)
		}
	})

	t.Run("OPTIONS preflight for allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/healthz", nil)
		req.Header.Set("Origin", "https://a.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204 No Content, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com" {
			t.Errorf("expected Access-Control-Allow-Origin https://a.com, got %q", got)
		}
	})
}
