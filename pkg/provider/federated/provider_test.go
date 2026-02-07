package federated

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/protocol"
	"github.com/rmax-ai/ratelord/pkg/provider"
)

func TestFederatedProvider_Poll(t *testing.T) {
	// Verify interface compliance
	var _ provider.Provider = (*FederatedProvider)(nil)

	// 1. Mock Leader
	leader := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/federation/grant" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req protocol.GrantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.PoolID == "fail" {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		resp := protocol.GrantResponse{
			Granted:    1000,
			ValidUntil: time.Now().Add(1 * time.Minute),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer leader.Close()

	// 2. Create Provider
	fp := NewFederatedProvider("test-fed", leader.URL, "follower-1")
	fp.RegisterPool("default")

	// 3. Track Usage locally
	fp.TrackUsage("default", 50)

	// 4. Poll
	res, err := fp.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll failed: %v", err)
	}

	// 5. Verify Grant was requested (since we started with 0)
	if len(res.Usage) != 1 {
		t.Fatalf("Expected 1 usage observation, got %d", len(res.Usage))
	}
	obs := res.Usage[0]
	if obs.PoolID != "default" {
		t.Errorf("Expected pool default, got %s", obs.PoolID)
	}
	// We tracked 50 usage. Grant should be 1000.
	// UsedLocal = 50.
	// Remaining = 1000 - 50 = 950.
	if obs.Limit != 1000 {
		t.Errorf("Expected limit 1000, got %d", obs.Limit)
	}
	if obs.Used != 50 {
		t.Errorf("Expected used 50, got %d", obs.Used)
	}
	if obs.Remaining != 950 {
		t.Errorf("Expected remaining 950, got %d", obs.Remaining)
	}
}

func TestUsageRouter(t *testing.T) {
	router := NewUsageRouter()
	fp := NewFederatedProvider("p1", "http://localhost", "f1")
	router.Register(fp)

	fp.RegisterPool("pool1")

	// Track via Router
	router.TrackUsage("p1", "pool1", 10)

	// Verify
	state := fp.pools["pool1"]
	if state.UsedLocal != 10 {
		t.Errorf("Expected usage 10, got %d", state.UsedLocal)
	}

	// Track unknown provider
	router.TrackUsage("p2", "pool1", 10)
	// Should not panic
}
