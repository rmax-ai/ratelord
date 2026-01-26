package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/provider"
)

func TestNewGitHubProvider(t *testing.T) {
	id := provider.ProviderID("test-github")
	token := "fake-token"
	enterpriseURL := "https://github.example.com/api/v3"

	p := NewGitHubProvider(id, token, enterpriseURL)

	if p.ID() != id {
		t.Errorf("Expected ID %s, got %s", id, p.ID())
	}
	if p.token != token {
		t.Errorf("Expected token %s, got %s", token, p.token)
	}
	if p.enterpriseURL != enterpriseURL {
		t.Errorf("Expected enterpriseURL %s, got %s", enterpriseURL, p.enterpriseURL)
	}
}

func TestPoll_Success(t *testing.T) {
	// Mock server
	mockResponse := map[string]interface{}{
		"resources": map[string]interface{}{
			"core": map[string]interface{}{
				"limit":     5000,
				"remaining": 4999,
				"reset":     1640995200, // Unix timestamp
			},
			"search": map[string]interface{}{
				"limit":     30,
				"remaining": 18,
				"reset":     1640995300,
			},
		},
	}
	responseJSON, _ := json.Marshal(mockResponse)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token fake-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseJSON)
	}))
	defer server.Close()

	p := NewGitHubProvider("test", "fake-token", server.URL)

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
	// Check core
	// Note: The order of iteration over map is random in Go, so we can't guarantee core is first.
	// We need to find "github:core".
	var core *provider.UsageObservation
	for _, u := range result.Usage {
		if u.PoolID == "github:core" {
			v := u // Create a copy to take address
			core = &v
			break
		}
	}

	if core == nil {
		t.Fatalf("Expected pool github:core not found")
	}

	if core.Used != 1 {
		t.Errorf("Expected used 1, got %d", core.Used)
	}
	if core.Remaining != 4999 {
		t.Errorf("Expected remaining 4999, got %d", core.Remaining)
	}
	if core.Limit != 5000 {
		t.Errorf("Expected limit 5000, got %d", core.Limit)
	}
	expectedReset := time.Unix(1640995200, 0)
	if !core.ResetAt.Equal(expectedReset) {
		t.Errorf("Expected reset %v, got %v", expectedReset, core.ResetAt)
	}
}

func TestPoll_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewGitHubProvider("test", "bad-token", "")
	p.client = server.Client()

	result, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("Expected no error in result, got %v", err)
	}
	if result.Status != "error" {
		t.Errorf("Expected status error, got %s", result.Status)
	}
	if result.Error == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRestore(t *testing.T) {
	p := NewGitHubProvider("test", "", "")
	err := p.Restore([]byte("some state"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
