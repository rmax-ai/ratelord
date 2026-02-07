package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestRollupWorker_ProcessBatch(t *testing.T) {
	// 1. Setup Store
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	// 2. Insert usage events
	ctx := context.Background()
	used := 100
	payload := struct {
		ProviderID string `json:"provider_id"`
		PoolID     string `json:"pool_id"`
		Used       *int   `json:"used,omitempty"`
	}{
		ProviderID: "prov-1",
		PoolID:     "pool-1",
		Used:       &used,
	}
	payloadBytes, _ := json.Marshal(payload)

	// Event 1
	st.AppendEvent(ctx, &store.Event{
		EventID:   "evt-1",
		EventType: store.EventTypeUsageObserved,
		TsEvent:   time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		TsIngest:  time.Now(),
		Payload:   payloadBytes,
		Dimensions: store.EventDimensions{
			IdentityID: "user-1",
			ScopeID:    "scope-1",
		},
	})

	// Event 2 (Same bucket)
	used2 := 200
	payload.Used = &used2
	payloadBytes2, _ := json.Marshal(payload)
	st.AppendEvent(ctx, &store.Event{
		EventID:   "evt-2",
		EventType: store.EventTypeUsageObserved,
		TsEvent:   time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
		TsIngest:  time.Now(),
		Payload:   payloadBytes2,
		Dimensions: store.EventDimensions{
			IdentityID: "user-1",
			ScopeID:    "scope-1",
		},
	})

	// 3. Run Rollup
	worker := NewRollupWorker(st)
	if err := worker.ProcessBatch(ctx); err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	// 4. Verify UsageStats
	stats, err := st.GetUsageStats(ctx, store.UsageFilter{
		From:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		Bucket: "hour",
	})
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats) == 0 {
		t.Fatal("No usage stats found")
	}

	// Should have aggregated 2 events in 10:00 bucket
	stat := stats[0]
	if stat.BucketTs.Hour() != 10 {
		t.Errorf("Expected bucket hour 10, got %d", stat.BucketTs.Hour())
	}

	if stat.TotalUsage != 300 {
		t.Errorf("Expected TotalUsage 300, got %d", stat.TotalUsage)
	}
	if stat.MinUsage != 100 {
		t.Errorf("Expected MinUsage 100, got %d", stat.MinUsage)
	}
	if stat.MaxUsage != 200 {
		t.Errorf("Expected MaxUsage 200, got %d", stat.MaxUsage)
	}
	if stat.EventCount != 2 {
		t.Errorf("Expected EventCount 2, got %d", stat.EventCount)
	}
}

func TestRollupWorker_Run(t *testing.T) {
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	worker := NewRollupWorker(st)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop")
	}
}
