package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/api"
	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestTrendsIntegration(t *testing.T) {
	// Setup: Create temporary SQLite DB
	tmpDir, err := os.MkdirTemp("", "ratelord-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "trends_test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	// Initialize RollupWorker
	rollupWorker := engine.NewRollupWorker(st)

	ctx := context.Background()

	// Data Injection: Create 3 usage_observed events
	baseTime := time.Date(2023, 10, 1, 10, 0, 0, 0, time.UTC) // 2023-10-01 10:00:00

	events := []*store.Event{
		{
			EventID:       "usage_1",
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       baseTime,
			TsIngest:      baseTime,
			Source: store.EventSource{
				OriginKind: "test",
				OriginID:   "test",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: "test_identity",
				WorkloadID: store.SentinelSystem,
				ScopeID:    "test_scope",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: "corr_1",
				CausationID:   store.SentinelUnknown,
			},
			Payload: json.RawMessage(`{"provider_id": "A", "pool_id": "X", "delta": 10}`),
		},
		{
			EventID:       "usage_2",
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       baseTime.Add(15 * time.Minute), // 10:15
			TsIngest:      baseTime.Add(15 * time.Minute),
			Source: store.EventSource{
				OriginKind: "test",
				OriginID:   "test",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: "test_identity",
				WorkloadID: store.SentinelSystem,
				ScopeID:    "test_scope",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: "corr_2",
				CausationID:   store.SentinelUnknown,
			},
			Payload: json.RawMessage(`{"provider_id": "A", "pool_id": "X", "delta": 5}`),
		},
		{
			EventID:       "usage_3",
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       baseTime.Add(time.Hour), // 11:00
			TsIngest:      baseTime.Add(time.Hour),
			Source: store.EventSource{
				OriginKind: "test",
				OriginID:   "test",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: "test_identity",
				WorkloadID: store.SentinelSystem,
				ScopeID:    "test_scope",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: "corr_3",
				CausationID:   store.SentinelUnknown,
			},
			Payload: json.RawMessage(`{"provider_id": "A", "pool_id": "X", "delta": 20}`),
		},
	}

	// Append events to store
	for _, evt := range events {
		if err := st.AppendEvent(ctx, evt); err != nil {
			t.Fatalf("failed to append event %s: %v", evt.EventID, err)
		}
	}

	// Execution: Run rollup
	if err := rollupWorker.ProcessBatch(ctx); err != nil {
		t.Fatalf("rollup failed: %v", err)
	}

	// Verification (Store): GetUsageStats
	from := baseTime
	to := baseTime.Add(2 * time.Hour)
	filter := store.UsageFilter{
		From:       from,
		To:         to,
		Bucket:     "hour",
		ProviderID: "A",
		PoolID:     "X",
		IdentityID: "test_identity",
		ScopeID:    "test_scope",
	}

	stats, err := st.GetUsageStats(ctx, filter)
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}

	// Check hour 10: total 15
	hour10 := baseTime.Truncate(time.Hour)
	found := false
	for _, stat := range stats {
		if stat.BucketTs.Equal(hour10) {
			if stat.TotalUsage != 15 {
				t.Errorf("expected total usage 15 for hour 10, got %d", stat.TotalUsage)
			}
			if stat.EventCount != 2 {
				t.Errorf("expected event count 2 for hour 10, got %d", stat.EventCount)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("hour 10 stat not found")
	}

	// Check hour 11: total 20
	hour11 := baseTime.Add(time.Hour).Truncate(time.Hour)
	found = false
	for _, stat := range stats {
		if stat.BucketTs.Equal(hour11) {
			if stat.TotalUsage != 20 {
				t.Errorf("expected total usage 20 for hour 11, got %d", stat.TotalUsage)
			}
			if stat.EventCount != 1 {
				t.Errorf("expected event count 1 for hour 11, got %d", stat.EventCount)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("hour 11 stat not found")
	}

	// Verification (API): Initialize server and test HTTP endpoint
	// For simplicity, since handleTrends only uses store, we can pass nil for others
	server := api.NewServer(st, nil, nil, nil, ":0") // :0 for auto port

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("server start failed: %v", err)
		}
	}()
	defer server.Stop(ctx)

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual port (since :0 assigns random)
	// But NewServer doesn't expose the listener, so we need to use httptest or refactor.
	// Since the server is started, but to test, we can use httptest.NewServer with a handler.
	// But the server has middleware, so it's tricky.
	// For integration test, let's assume the port is known or use httptest.

	// Actually, since the server is http.Server, and we set Addr to ":0", but to get the port, we need access to the listener.
	// For simplicity, let's use a fixed port for test.
	// Change to ":8099"

	// Recreate server with fixed port
	server.Stop(ctx) // stop the previous
	server = api.NewServer(st, nil, nil, nil, ":8099")

	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("server start failed: %v", err)
		}
	}()
	defer server.Stop(ctx)

	time.Sleep(100 * time.Millisecond)

	// Make HTTP request
	resp, err := http.Get("http://localhost:8099/v1/trends?bucket=hour&from=" + from.Format(time.RFC3339) + "&to=" + to.Format(time.RFC3339) + "&provider_id=A&pool_id=X&identity_id=test_identity&scope_id=test_scope")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var apiStats []store.UsageStat
	if err := json.NewDecoder(resp.Body).Decode(&apiStats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(apiStats) != 2 {
		t.Fatalf("expected 2 stats from API, got %d", len(apiStats))
	}

	// Similar checks as above
	found10 := false
	found11 := false
	for _, stat := range apiStats {
		if stat.BucketTs.Equal(hour10) {
			if stat.TotalUsage != 15 {
				t.Errorf("API: expected total usage 15 for hour 10, got %d", stat.TotalUsage)
			}
			found10 = true
		}
		if stat.BucketTs.Equal(hour11) {
			if stat.TotalUsage != 20 {
				t.Errorf("API: expected total usage 20 for hour 11, got %d", stat.TotalUsage)
			}
			found11 = true
		}
	}
	if !found10 {
		t.Errorf("API: hour 10 stat not found")
	}
	if !found11 {
		t.Errorf("API: hour 11 stat not found")
	}
}
