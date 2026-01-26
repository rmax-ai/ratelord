package openai

import (
	"context"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	id := provider.ProviderID("test-openai")
	token := "fake-token"
	orgID := "org-123"
	baseURL := "https://api.openai.com/v1"

	p := NewOpenAIProvider(id, token, orgID, baseURL)

	if p.ID() != id {
		t.Errorf("Expected ID %s, got %s", id, p.ID())
	}
	if p.token != token {
		t.Errorf("Expected token %s, got %s", token, p.token)
	}
	if p.orgID != orgID {
		t.Errorf("Expected orgID %s, got %s", orgID, p.orgID)
	}
	if p.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, p.baseURL)
	}
}

func TestPoll_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fake-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("OpenAI-Organization") != "org-123" {
			t.Errorf("Expected OpenAI-Organization header, got %s", r.Header.Get("OpenAI-Organization"))
		}

		// Set rate limit headers
		w.Header().Set("x-ratelimit-limit-requests", "5000")
		w.Header().Set("x-ratelimit-remaining-requests", "4999")
		w.Header().Set("x-ratelimit-reset-requests", "100ms")

		w.Header().Set("x-ratelimit-limit-tokens", "160000")
		w.Header().Set("x-ratelimit-remaining-tokens", "159000")
		w.Header().Set("x-ratelimit-reset-tokens", "2s")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider("test", "fake-token", "org-123", server.URL)

	result, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Expected status success, got %s", result.Status)
	}
	if len(result.Usage) != 2 {
		t.Errorf("Expected 2 usages, got %d", len(result.Usage))
	}

	// Verify requests usage
	var reqObs *provider.UsageObservation
	for i := range result.Usage {
		if strings.HasSuffix(result.Usage[i].PoolID, ":requests") {
			reqObs = &result.Usage[i]
			break
		}
	}
	if reqObs == nil {
		t.Fatalf("Expected requests usage observation")
	}
	if reqObs.PoolID != "openai:requests" {
		t.Errorf("Expected pool openai:requests, got %s", reqObs.PoolID)
	}
	if reqObs.Limit != 5000 {
		t.Errorf("Expected limit 5000, got %d", reqObs.Limit)
	}
	if reqObs.Remaining != 4999 {
		t.Errorf("Expected remaining 4999, got %d", reqObs.Remaining)
	}
	if reqObs.Used != 1 {
		t.Errorf("Expected used 1, got %d", reqObs.Used)
	}

	// Verify tokens usage
	var tokObs *provider.UsageObservation
	for i := range result.Usage {
		if strings.HasSuffix(result.Usage[i].PoolID, ":tokens") {
			tokObs = &result.Usage[i]
			break
		}
	}
	if tokObs == nil {
		t.Fatalf("Expected tokens usage observation")
	}
	if tokObs.PoolID != "openai:tokens" {
		t.Errorf("Expected pool openai:tokens, got %s", tokObs.PoolID)
	}
	if tokObs.Limit != 160000 {
		t.Errorf("Expected limit 160000, got %d", tokObs.Limit)
	}
	if tokObs.Remaining != 159000 {
		t.Errorf("Expected remaining 159000, got %d", tokObs.Remaining)
	}
	if tokObs.Used != 1000 {
		t.Errorf("Expected used 1000, got %d", tokObs.Used)
	}
}

func TestPoll_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Use empty string for baseURL to default to real API, but since we mock client
	// we need to override the URL construction in Poll or pass a bad URL that the client uses.
	// Actually, NewOpenAIProvider defaults baseURL if empty.
	// We can't easily inject the mock server URL into the default client if we don't pass it.
	// So we pass a dummy URL.
	// BUT, p.client is internal. We can't set it easily without access or constructor.
	// The constructor sets client.
	// Wait, p.client is public? No, lowercase.
	// We have to use the constructor which takes baseURL.
	// But to hit the mock server, baseURL must match.

	// The issue: The code does `client.Do(req)`.
	// We can't swap the client easily in test if it's private.
	// In GitHub test we did `p.client = server.Client()`. Is client exported there?
	// Checking GitHub provider... `client *http.Client`. It is unexported.
	// How did GitHub test work?
	// Ah, I see `p.client = server.Client()` in `TestPoll_HTTPError`.
	// That implies `client` IS exported or test is in same package.
	// Yes, `package github`.
	// So here `package openai` is correct.

	p := NewOpenAIProvider("test", "bad-token", "", server.URL)
	// Override client to use server's transport if needed?
	// Actually, just passing server.URL as baseURL is enough because we construct the request with that URL.
	// The default client will hit that URL.

	// However, `TestPoll_HTTPError` in GitHub explicitly sets `p.client`.
	// The default client might not trust the test certs if HTTPS?
	// httptest.NewServer is usually HTTP.
	// Let's rely on baseURL injection.

	// One catch: NewOpenAIProvider defaults baseURL if empty.
	// We passed server.URL.

	// BUT, if the code in Poll does `strings.TrimRight(o.baseURL, "/") + "/models"`,
	// and we provide `http://127.0.0.1:12345`, it works.

	// Wait, is client unexported?
	// `type OpenAIProvider struct { ... client *http.Client }`
	// Yes, unexported.
	// But test is `package openai`. So it has access.

	result, err := p.Poll(context.Background())
	if err != nil {
		// It might not error if it just returns a result with Error set.
		t.Fatalf("Expected no error in result return, got %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status error, got %s", result.Status)
	}
	if result.Error == nil {
		t.Error("Expected error object, got nil")
	}
}
