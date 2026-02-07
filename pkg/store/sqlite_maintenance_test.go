package store

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestSnapshots(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Get Latest (Empty)
	snap, err := store.GetLatestSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}
	if snap != nil {
		t.Errorf("expected nil snapshot, got %v", snap)
	}

	ts, err := store.GetLatestSnapshotTime(context.Background())
	if err != nil {
		t.Fatalf("GetLatestSnapshotTime failed: %v", err)
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time, got %v", ts)
	}

	// Create a dummy event for referential integrity
	evt := &Event{
		EventID:       "evt_last",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       time.Now().UTC(),
		TsIngest:      time.Now().UTC(),
		Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:       json.RawMessage(`{}`),
	}
	if err := store.AppendEvent(context.Background(), evt); err != nil {
		t.Fatalf("failed to append dependency event: %v", err)
	}

	// 2. Save Snapshot
	newSnap := &Snapshot{
		SnapshotID:    "snap_1",
		SchemaVersion: 1,
		TsSnapshot:    time.Now().UTC(),
		LastEventID:   "evt_last",
		Payload:       json.RawMessage(`{"state": "captured"}`),
	}

	if err := store.SaveSnapshot(context.Background(), newSnap); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// 3. Get Latest (Found)
	snap, err = store.GetLatestSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}
	if snap == nil {
		t.Fatalf("expected snapshot, got nil")
	}
	if snap.SnapshotID != "snap_1" {
		t.Errorf("expected snap_1, got %s", snap.SnapshotID)
	}

	ts, err = store.GetLatestSnapshotTime(context.Background())
	if err != nil {
		t.Fatalf("GetLatestSnapshotTime failed: %v", err)
	}
	// SQLite stores time with some precision loss depending on how it's stored (string vs int),
	// but here we just check if it's close or equal.
	// The implementation likely returns the stored time.
	if ts.Unix() != newSnap.TsSnapshot.Unix() {
		t.Errorf("expected time %v, got %v", newSnap.TsSnapshot, ts)
	}
}

func TestDeleteIdentityData(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Seed data for identity "user_del"
	identity := "user_del"

	// Event
	evt := &Event{
		EventID:       "evt_del_1",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       time.Now().UTC(),
		TsIngest:      time.Now().UTC(),
		Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    EventDimensions{AgentID: "a", IdentityID: identity, WorkloadID: "w", ScopeID: "s"},
		Payload:       json.RawMessage(`{}`),
	}
	store.AppendEvent(context.Background(), evt)

	// Usage
	store.UpsertUsageStats(context.Background(), []UsageStat{{
		BucketTs:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		ProviderID: "p", PoolID: "p", IdentityID: identity, ScopeID: "s",
		TotalUsage: 100,
	}})

	// Verify data exists
	events, _ := store.QueryEvents(context.Background(), EventFilter{IdentityID: identity})
	if len(events) == 0 {
		t.Fatalf("setup failed: events not found")
	}

	// Delete
	if err := store.DeleteIdentityData(context.Background(), identity); err != nil {
		t.Fatalf("DeleteIdentityData failed: %v", err)
	}

	// Verify deletion
	events, _ = store.QueryEvents(context.Background(), EventFilter{IdentityID: identity})
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}

	stats, _ := store.GetUsageStats(context.Background(), UsageFilter{
		Bucket: "day", From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		IdentityID: identity,
	})
	if len(stats) != 0 {
		t.Errorf("expected 0 stats, got %d", len(stats))
	}
}

func TestDeleteEvents(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Seed 3 events
	ids := []string{"e1", "e2", "e3"}
	for _, id := range ids {
		evt := &Event{
			EventID:       EventID(id),
			EventType:     EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       time.Now().UTC(),
			TsIngest:      time.Now().UTC(),
			Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
			Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
			Payload:       json.RawMessage(`{}`),
		}
		store.AppendEvent(context.Background(), evt)
	}

	// Delete e1 and e3
	if err := store.DeleteEvents(context.Background(), []string{"e1", "e3"}); err != nil {
		t.Fatalf("DeleteEvents failed: %v", err)
	}

	// Check e1 (gone)
	e1, _ := store.GetEvent(context.Background(), "e1")
	if e1 != nil {
		t.Errorf("expected e1 to be deleted")
	}

	// Check e2 (exists)
	e2, _ := store.GetEvent(context.Background(), "e2")
	if e2 == nil {
		t.Errorf("expected e2 to exist")
	}

	// Check e3 (gone)
	e3, _ := store.GetEvent(context.Background(), "e3")
	if e3 != nil {
		t.Errorf("expected e3 to be deleted")
	}
}

func TestPruneEvents(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Pruning requires a snapshot
	// 1. Create older events (older than retention)
	// 2. Create newer events (within retention)
	// 3. Create a snapshot that covers the older events

	now := time.Now().UTC()
	retention := 24 * time.Hour
	oldTime := now.Add(-48 * time.Hour)
	newTime := now

	// Old event
	evtOld := &Event{
		EventID:       "evt_old",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       oldTime,
		TsIngest:      oldTime,
		Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:       json.RawMessage(`{}`),
	}
	store.AppendEvent(context.Background(), evtOld)

	// Boundary event (for snapshot)
	// Must be AFTER the old event but ideally BEFORE retention cutoff for this test?
	// Actually, PruneEvents logic:
	// cutoffTime = now - retention
	// if snapshot.LastEvent.TsIngest < cutoffTime, then cutoffTime = snapshot.LastEvent.TsIngest (safety: don't prune beyond snapshot)
	// Wait, code says:
	// if snapEvent.TsIngest.Before(cutoffTime) { cutoffTime = snapEvent.TsIngest }
	// This means we prune UP TO the snapshot OR the retention period, whichever is OLDER (safer).
	// So if snapshot is very old, we only prune up to snapshot.
	// If snapshot is recent (newer than retention limit), we prune up to retention limit.

	// Let's make snapshot recent so retention limit applies.
	snapEvt := &Event{
		EventID:       "evt_snap",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       newTime,
		TsIngest:      newTime,
		Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:       json.RawMessage(`{}`),
	}
	store.AppendEvent(context.Background(), snapEvt)

	store.SaveSnapshot(context.Background(), &Snapshot{
		SnapshotID:  "snap_1",
		LastEventID: "evt_snap",
		TsSnapshot:  newTime,
		Payload:     json.RawMessage(`{}`),
	})

	// Prune
	deleted, err := store.PruneEvents(context.Background(), retention, "", nil)
	if err != nil {
		t.Fatalf("PruneEvents failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted event, got %d", deleted)
	}

	// Verify old is gone
	gone, _ := store.GetEvent(context.Background(), "evt_old")
	if gone != nil {
		t.Errorf("expected evt_old to be deleted")
	}

	// Verify snapshot event remains
	remains, _ := store.GetEvent(context.Background(), "evt_snap")
	if remains == nil {
		t.Errorf("expected evt_snap to remain")
	}
}

func TestReadCandidateEvents(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now().UTC()

	// Seed 3 events
	for i := 1; i <= 3; i++ {
		evt := &Event{
			EventID:    EventID(fmt.Sprintf("evt_%d", i)),
			EventType:  EventTypeUsageObserved,
			TsEvent:    now.Add(time.Duration(i) * time.Second),
			TsIngest:   now.Add(time.Duration(i) * time.Second),
			Source:     EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
			Payload:    json.RawMessage(`{}`),
		}
		store.AppendEvent(context.Background(), evt)
	}

	// Read candidates before evt_3
	candidates, err := store.ReadCandidateEvents(context.Background(), now.Add(3*time.Second), 10)
	if err != nil {
		t.Fatalf("ReadCandidateEvents failed: %v", err)
	}

	// Should match evt_1 and evt_2 (ingest < evt_3 time)
	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}
}

func TestPruneEventsWithFilter(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	now := time.Now().UTC()
	retention := 24 * time.Hour
	oldTime := now.Add(-48 * time.Hour)

	// Old event A (should be pruned if type matches)
	// Must be older than snapshot boundary event (evt_B) for pruning to work safely
	evtA := &Event{
		EventID:    "evt_A",
		EventType:  EventTypeUsageObserved,
		TsEvent:    oldTime.Add(-1 * time.Second),
		TsIngest:   oldTime.Add(-1 * time.Second),
		Source:     EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:    json.RawMessage(`{}`),
	}
	store.AppendEvent(context.Background(), evtA)

	// Old event B (should NOT be pruned if type doesn't match)
	evtB := &Event{
		EventID:    "evt_B",
		EventType:  EventTypePolicyTriggered,
		TsEvent:    oldTime,
		TsIngest:   oldTime,
		Source:     EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:    json.RawMessage(`{}`),
	}
	store.AppendEvent(context.Background(), evtB)

	// Snapshot required
	store.SaveSnapshot(context.Background(), &Snapshot{
		SnapshotID:  "snap_1",
		LastEventID: "evt_B",
		TsSnapshot:  now,
		Payload:     json.RawMessage(`{}`),
	})

	// Prune only UsageObserved
	deleted, err := store.PruneEvents(context.Background(), retention, string(EventTypeUsageObserved), nil)
	if err != nil {
		t.Fatalf("PruneEvents failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify A gone
	a, _ := store.GetEvent(context.Background(), "evt_A")
	if a != nil {
		t.Errorf("evt_A should be deleted")
	}

	// Verify B remains
	b, _ := store.GetEvent(context.Background(), "evt_B")
	if b == nil {
		t.Errorf("evt_B should remain")
	}
}

func TestEmptyOps(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	if err := store.DeleteEvents(context.Background(), nil); err != nil {
		t.Errorf("DeleteEvents(nil) failed: %v", err)
	}

	if err := store.UpsertUsageStats(context.Background(), nil); err != nil {
		t.Errorf("UpsertUsageStats(nil) failed: %v", err)
	}
}
