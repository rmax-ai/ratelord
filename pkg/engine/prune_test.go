package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPruneWorker(t *testing.T) {
	// 1. Setup Store
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_prune.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	// 2. Create Events
	ctx := context.Background()

	// Create a snapshot first (required for safety)
	// We need an event to be the boundary
	boundaryEvt := &store.Event{
		EventID:       "evt_boundary",
		EventType:     "system_started",
		SchemaVersion: 1,
		TsEvent:       time.Now(),
		TsIngest:      time.Now(),
		Payload:       []byte("{}"),
		Source:        store.EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    store.EventDimensions{AgentID: "sys", IdentityID: "sys", WorkloadID: "sys", ScopeID: "sys"},
		Correlation:   store.EventCorrelation{CorrelationID: "1", CausationID: "0"},
	}
	if err := st.AppendEvent(ctx, boundaryEvt); err != nil {
		t.Fatalf("failed to append boundary event: %v", err)
	}

	snap := &store.Snapshot{
		SnapshotID:    "snap_1",
		SchemaVersion: 1,
		TsSnapshot:    time.Now(),
		LastEventID:   "evt_boundary",
		Payload:       []byte("{}"),
	}
	if err := st.SaveSnapshot(ctx, snap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	// Create old events (older than boundary)
	// Note: TsIngest is what matters. We need to cheat or sleep.
	// Since we can't easily manipulate TsIngest in AppendEvent (it defaults to now, or takes struct value),
	// let's check AppendEvent implementation.
	// It respects TsIngest if set!

	oldTime := time.Now().Add(-24 * time.Hour)

	// Event 1: Type A, Old
	evt1 := &store.Event{
		EventID:       "evt_old_a",
		EventType:     "type_a",
		SchemaVersion: 1,
		TsEvent:       oldTime,
		TsIngest:      oldTime,
		Payload:       []byte("{}"),
		Source:        store.EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    store.EventDimensions{AgentID: "sys", IdentityID: "sys", WorkloadID: "sys", ScopeID: "sys"},
		Correlation:   store.EventCorrelation{CorrelationID: "1", CausationID: "0"},
	}
	st.AppendEvent(ctx, evt1)

	// Event 2: Type B, Old
	evt2 := &store.Event{
		EventID:       "evt_old_b",
		EventType:     "type_b",
		SchemaVersion: 1,
		TsEvent:       oldTime,
		TsIngest:      oldTime,
		Payload:       []byte("{}"),
		Source:        store.EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    store.EventDimensions{AgentID: "sys", IdentityID: "sys", WorkloadID: "sys", ScopeID: "sys"},
		Correlation:   store.EventCorrelation{CorrelationID: "1", CausationID: "0"},
	}
	st.AppendEvent(ctx, evt2)

	// Event 3: Type A, New (but older than boundary? No, boundary is NOW)
	// We need boundary to be NEWER than oldTime. Boundary is NOW. Correct.

	// 3. Configure PruneWorker
	// Default TTL: 10h (Should delete both)
	// Type A TTL: 5h (Should delete A)
	// Type B TTL: 30h (Should KEEP B)

	cfg := &RetentionConfig{
		Enabled:    true,
		DefaultTTL: "10h",
		ByType: map[string]string{
			"type_b": "30h",
		},
	}

	worker := NewPruneWorker(st, cfg)
	worker.Prune(ctx)

	// 4. Verify
	// Event A should be gone (Old=24h, Default=10h)
	// Event B should be present (Old=24h, TypeB=30h)

	if evt, _ := st.GetEvent(ctx, "evt_old_a"); evt != nil {
		t.Errorf("expected evt_old_a to be pruned, but it exists")
	}

	if evt, _ := st.GetEvent(ctx, "evt_old_b"); evt == nil {
		t.Errorf("expected evt_old_b to exist, but it was pruned")
	}
}
