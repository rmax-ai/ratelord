package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestStore creates a temporary database for testing
func setupTestStore(t *testing.T) (*Store, string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ratelord-store-test-ext")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "ratelord.db")
	store, err := NewStore(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("NewStore failed: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, dbPath, cleanup
}

func TestGetEvent(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Test GetEvent on empty DB
	val, err := store.GetEvent(context.Background(), "non_existent")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil for non-existent event, got %v", val)
	}

	// 2. Insert and Get
	evt := &Event{
		EventID:       "evt_get_1",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       time.Now().UTC(),
		TsIngest:      time.Now().UTC(),
		Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
		Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		Payload:       json.RawMessage(`{}`),
	}
	if err := store.AppendEvent(context.Background(), evt); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	got, err := store.GetEvent(context.Background(), "evt_get_1")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got == nil {
		t.Fatalf("expected event, got nil")
	}
	if got.EventID != evt.EventID {
		t.Errorf("expected ID %s, got %s", evt.EventID, got.EventID)
	}
}

func TestReadRecentEvents(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Seed 5 events
	for i := 1; i <= 5; i++ {
		evt := &Event{
			EventID:       EventID(fmt.Sprintf("evt_%d", i)),
			EventType:     EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       time.Now().UTC(),
			TsIngest:      time.Now().UTC().Add(time.Duration(i) * time.Second), // increasing ingest time
			Source:        EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"},
			Dimensions:    EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
			Payload:       json.RawMessage(`{}`),
		}
		if err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("AppendEvent failed: %v", err)
		}
	}

	// 1. Read last 3
	recent, err := store.ReadRecentEvents(context.Background(), 3)
	if err != nil {
		t.Fatalf("ReadRecentEvents failed: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("expected 3 events, got %d", len(recent))
	}
	// Should be reverse order (5, 4, 3)
	if recent[0].EventID != "evt_5" {
		t.Errorf("expected first to be evt_5, got %s", recent[0].EventID)
	}
	if recent[2].EventID != "evt_3" {
		t.Errorf("expected last to be evt_3, got %s", recent[2].EventID)
	}

	// 2. Default limit (pass 0/neg)
	// We only have 5, so it should return all 5
	recentAll, err := store.ReadRecentEvents(context.Background(), 0)
	if err != nil {
		t.Fatalf("ReadRecentEvents failed: %v", err)
	}
	if len(recentAll) != 5 {
		t.Errorf("expected 5 events, got %d", len(recentAll))
	}
}

func TestQueryEvents(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	baseTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	// Seed events with different dimensions
	events := []*Event{
		{
			EventID:    "evt_A_1",
			EventType:  EventTypeUsageObserved,
			TsEvent:    baseTime,
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "user_A", WorkloadID: "w", ScopeID: "scope_1"},
		},
		{
			EventID:    "evt_A_2",
			EventType:  EventTypePolicyTriggered, // Different type
			TsEvent:    baseTime.Add(1 * time.Hour),
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "user_A", WorkloadID: "w", ScopeID: "scope_2"},
		},
		{
			EventID:    "evt_B_1",
			EventType:  EventTypeUsageObserved,
			TsEvent:    baseTime.Add(2 * time.Hour),
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "user_B", WorkloadID: "w", ScopeID: "scope_1"},
		},
	}

	for _, e := range events {
		e.SchemaVersion = 1
		e.TsIngest = time.Now().UTC()
		e.Source = EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"}
		e.Payload = json.RawMessage(`{}`)
		if err := store.AppendEvent(context.Background(), e); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}

	// 1. Filter by Time Range
	res, err := store.QueryEvents(context.Background(), EventFilter{
		From: baseTime.Add(30 * time.Minute),
		To:   baseTime.Add(90 * time.Minute), // Should capture evt_A_2 (offset 1h)
	})
	if err != nil {
		t.Fatalf("QueryEvents time filter failed: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 event in time range, got %d", len(res))
	} else if res[0].EventID != "evt_A_2" {
		t.Errorf("expected evt_A_2, got %s", res[0].EventID)
	}

	// 2. Filter by Identity
	res, err = store.QueryEvents(context.Background(), EventFilter{
		IdentityID: "user_A",
	})
	if err != nil {
		t.Fatalf("QueryEvents identity filter failed: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 events for user_A, got %d", len(res))
	}

	// 3. Filter by Scope
	res, err = store.QueryEvents(context.Background(), EventFilter{
		ScopeID: "scope_1",
	})
	if err != nil {
		t.Fatalf("QueryEvents scope filter failed: %v", err)
	}
	if len(res) != 2 { // evt_A_1 and evt_B_1
		t.Errorf("expected 2 events for scope_1, got %d", len(res))
	}

	// 4. Filter by Event Type
	res, err = store.QueryEvents(context.Background(), EventFilter{
		EventTypes: []EventType{EventTypePolicyTriggered},
	})
	if err != nil {
		t.Fatalf("QueryEvents type filter failed: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 limit event, got %d", len(res))
	} else if res[0].EventID != "evt_A_2" {
		t.Errorf("expected evt_A_2, got %s", res[0].EventID)
	}

	// 5. Combined Filter
	res, err = store.QueryEvents(context.Background(), EventFilter{
		IdentityID: "user_A",
		ScopeID:    "scope_1",
	})
	if err != nil {
		t.Fatalf("QueryEvents combined filter failed: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 event (A_1), got %d", len(res))
	} else if res[0].EventID != "evt_A_1" {
		t.Errorf("expected evt_A_1, got %s", res[0].EventID)
	}
}

func TestSystemState(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Get non-existent
	val, err := store.GetSystemState(context.Background(), "missing_key")
	if err == nil {
		t.Errorf("expected error for missing key, got nil")
	}
	if val != "" {
		t.Errorf("expected empty string, got %s", val)
	}

	// 2. Set and Get
	key := "last_processed_id"
	value := "evt_100"
	if err := store.SetSystemState(context.Background(), key, value); err != nil {
		t.Fatalf("SetSystemState failed: %v", err)
	}

	got, err := store.GetSystemState(context.Background(), key)
	if err != nil {
		t.Fatalf("GetSystemState failed: %v", err)
	}
	if got != value {
		t.Errorf("expected %s, got %s", value, got)
	}

	// 3. Update (Upsert)
	newValue := "evt_200"
	if err := store.SetSystemState(context.Background(), key, newValue); err != nil {
		t.Fatalf("SetSystemState (update) failed: %v", err)
	}

	gotUpdated, err := store.GetSystemState(context.Background(), key)
	if err != nil {
		t.Fatalf("GetSystemState failed: %v", err)
	}
	if gotUpdated != newValue {
		t.Errorf("expected %s, got %s", newValue, gotUpdated)
	}
}

func TestUsageStats(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	baseHour := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	baseDay := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	stats := []UsageStat{
		{
			BucketTs:   baseHour,
			ProviderID: "p1",
			PoolID:     "pool1",
			IdentityID: "u1",
			ScopeID:    "s1",
			TotalUsage: 100,
			MinUsage:   10,
			MaxUsage:   50,
			EventCount: 5,
		},
		{
			BucketTs:   baseDay,
			ProviderID: "p1",
			PoolID:     "pool1",
			IdentityID: "u1",
			ScopeID:    "s1",
			TotalUsage: 1000,
			MinUsage:   10,
			MaxUsage:   100,
			EventCount: 50,
		},
	}

	// 1. Upsert
	if err := store.UpsertUsageStats(context.Background(), stats); err != nil {
		t.Fatalf("UpsertUsageStats failed: %v", err)
	}

	// 2. Query Hourly
	hourly, err := store.GetUsageStats(context.Background(), UsageFilter{
		Bucket:     "hour",
		From:       baseHour,
		To:         baseHour.Add(1 * time.Hour),
		ProviderID: "p1",
	})
	if err != nil {
		t.Fatalf("GetUsageStats (hourly) failed: %v", err)
	}
	if len(hourly) != 1 {
		t.Errorf("expected 1 hourly stat, got %d", len(hourly))
	} else {
		if hourly[0].TotalUsage != 100 {
			t.Errorf("expected total usage 100, got %d", hourly[0].TotalUsage)
		}
	}

	// 3. Query Daily
	daily, err := store.GetUsageStats(context.Background(), UsageFilter{
		Bucket: "day",
		From:   baseDay,
		To:     baseDay.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsageStats (daily) failed: %v", err)
	}
	if len(daily) != 1 {
		t.Errorf("expected 1 daily stat, got %d", len(daily))
	}

	// 4. Update (Merge)
	newStats := []UsageStat{
		{
			BucketTs:   baseHour,
			ProviderID: "p1",
			PoolID:     "pool1",
			IdentityID: "u1",
			ScopeID:    "s1",
			TotalUsage: 50, // Add 50
			MinUsage:   5,  // Lower min
			MaxUsage:   60, // Higher max
			EventCount: 2,  // Add 2
		},
	}
	if err := store.UpsertUsageStats(context.Background(), newStats); err != nil {
		t.Fatalf("UpsertUsageStats (merge) failed: %v", err)
	}

	updated, err := store.GetUsageStats(context.Background(), UsageFilter{
		Bucket: "hour",
		From:   baseHour,
		To:     baseHour.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("GetUsageStats (updated) failed: %v", err)
	}
	u := updated[0]
	if u.TotalUsage != 150 { // 100 + 50
		t.Errorf("expected merged total 150, got %d", u.TotalUsage)
	}
	if u.MinUsage != 5 { // min(10, 5)
		t.Errorf("expected merged min 5, got %d", u.MinUsage)
	}
	if u.MaxUsage != 60 { // max(50, 60)
		t.Errorf("expected merged max 60, got %d", u.MaxUsage)
	}
	if u.EventCount != 7 { // 5 + 2
		t.Errorf("expected merged count 7, got %d", u.EventCount)
	}

	// 5. Invalid Bucket TS (Upsert should fail)
	badStat := []UsageStat{{
		BucketTs: baseHour.Add(1 * time.Minute), // Not top of hour
	}}
	if err := store.UpsertUsageStats(context.Background(), badStat); err == nil {
		t.Errorf("expected error for invalid bucket timestamp, got nil")
	}
}

func TestWebhooks(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	wh := &WebhookConfig{
		WebhookID: "wh_1",
		URL:       "http://example.com",
		Secret:    "secret",
		Events:    []string{"usage_observed"},
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}

	// 1. Register
	if err := store.RegisterWebhook(context.Background(), wh); err != nil {
		t.Fatalf("RegisterWebhook failed: %v", err)
	}

	// 2. List
	list, err := store.ListWebhooks(context.Background())
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(list))
	} else if list[0].WebhookID != "wh_1" {
		t.Errorf("expected wh_1, got %s", list[0].WebhookID)
	}

	// 3. Delete
	if err := store.DeleteWebhook(context.Background(), "wh_1"); err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}

	// 4. List Empty
	listEmpty, err := store.ListWebhooks(context.Background())
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}
	if len(listEmpty) != 0 {
		t.Errorf("expected 0 webhooks, got %d", len(listEmpty))
	}
}
