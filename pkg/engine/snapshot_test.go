package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestSnapshotWorker_TakeAndLoad(t *testing.T) {
	// 1. Setup Store
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	// 2. Setup Projections with data
	idProj := NewIdentityProjection()
	// Simulate applying an event to advance ID
	idProj.LoadState("evt-1", time.Now(), []Identity{{ID: "u1"}})

	usageProj := NewUsageProjection()
	usageProj.LoadState("evt-1", time.Now(), []PoolState{{PoolID: "p1"}})

	provProj := NewProviderProjection()
	provProj.LoadState(map[string][]byte{"pr1": []byte("st1")})

	foreProj := forecast.NewForecastProjection(100)
	// Add forecast data if possible, or leave empty

	// 3. We need the event "evt-1" to exist before taking snapshot due to foreign key
	err = st.AppendEvent(context.Background(), &store.Event{
		EventID:   "evt-1",
		TsIngest:  time.Now(),
		EventType: store.EventTypeGrantIssued, // Dummy type
		Payload:   json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Failed to write checkpoint event: %v", err)
	}

	// 4. Create Worker
	worker := NewSnapshotWorker(st, idProj, usageProj, provProj, foreProj, time.Hour)

	// 5. Take Snapshot
	ctx := context.Background()
	if err := worker.TakeSnapshot(ctx); err != nil {
		t.Fatalf("TakeSnapshot failed: %v", err)
	}

	// 6. Verify snapshot in store
	snap, err := st.GetLatestSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}
	if snap == nil {
		t.Fatal("Snapshot not found in store")
	}
	if snap.LastEventID != "evt-1" {
		t.Errorf("Expected LastEventID evt-1, got %s", snap.LastEventID)
	}

	// 7. Test LoadLatestSnapshot
	// Reset projections
	newIdProj := NewIdentityProjection()
	newUsageProj := NewUsageProjection()
	newProvProj := NewProviderProjection()
	newForeProj := forecast.NewForecastProjection(100)

	ts, err := LoadLatestSnapshot(ctx, st, newIdProj, newUsageProj, newProvProj, newForeProj)
	if err != nil {
		t.Fatalf("LoadLatestSnapshot failed: %v", err)
	}
	if ts.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Verify restored state
	if _, ok := newIdProj.Get("u1"); !ok {
		t.Error("Identity u1 not restored")
	}
	if string(newProvProj.GetState("pr1")) != "st1" {
		t.Error("Provider state pr1 not restored")
	}
}

func TestSnapshotWorker_Run(t *testing.T) {
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	// Projections must have state for TakeSnapshot to succeed
	idProj := NewIdentityProjection()
	idProj.LoadState("evt-1", time.Now(), []Identity{{ID: "u1"}})
	usageProj := NewUsageProjection()
	usageProj.LoadState("evt-1", time.Now(), []PoolState{{PoolID: "p1"}})
	provProj := NewProviderProjection()
	foreProj := forecast.NewForecastProjection(10)

	// Add event for FK
	st.AppendEvent(context.Background(), &store.Event{
		EventID:   "evt-1",
		TsIngest:  time.Now(),
		EventType: "init",
		Payload:   json.RawMessage("{}"),
	})

	worker := NewSnapshotWorker(st, idProj, usageProj, provProj, foreProj, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // enough for a few ticks
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop")
	}

	// Verify snapshots were taken
	snap, err := st.GetLatestSnapshot(context.Background())
	if err != nil {
		t.Errorf("failed to get snapshot: %v", err)
	}
	if snap == nil {
		t.Error("expected snapshot to be taken by worker")
	}
}

func TestNewSnapshotWorker_Defaults(t *testing.T) {
	w := NewSnapshotWorker(nil, nil, nil, nil, nil, 0)
	if w.interval != 5*time.Minute {
		t.Errorf("expected default interval 5m, got %v", w.interval)
	}
}
