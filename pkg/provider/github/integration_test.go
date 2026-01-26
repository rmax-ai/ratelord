//go:build integration

package github

import (
	"context"
	"testing"

	"github.com/rmax-ai/ratelord/pkg/provider"
)

func TestGitHubProvider_Integration(t *testing.T) {
	// Instantiate GitHubProvider with empty token for unauthenticated requests
	p := NewGitHubProvider(provider.ProviderID("test-github"), "", "")

	// Call Poll against real GitHub API
	result, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll failed with error: %v", err)
	}

	// Assert Status is success
	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
		if result.Error != nil {
			t.Errorf("Error: %v", result.Error)
		}
	}

	// Assert Usage contains github:core and values are sane
	foundCore := false
	for _, usage := range result.Usage {
		if usage.PoolID == "github:core" {
			foundCore = true
			if usage.Limit <= 0 {
				t.Errorf("Expected Limit > 0 for github:core, got %d", usage.Limit)
			}
			// For unauthenticated, limit should be 60
			if usage.Limit != 60 {
				t.Logf("Warning: Expected limit 60 for unauthenticated, got %d", usage.Limit)
			}
			break
		}
	}
	if !foundCore {
		t.Errorf("Expected to find github:core in usage, but it was not present")
	}
}
