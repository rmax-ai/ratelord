package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecureHeaders(t *testing.T) {
	// Create a handler that just returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap it with our middleware
	secureHandler := withSecureHeaders(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Serve
	secureHandler.ServeHTTP(w, req)

	// Check headers
	expectedHeaders := map[string]string{
		"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"X-XSS-Protection":          "1; mode=block",
	}

	for key, expected := range expectedHeaders {
		got := w.Header().Get(key)
		if got != expected {
			t.Errorf("Header %s: expected %q, got %q", key, expected, got)
		}
	}
}
