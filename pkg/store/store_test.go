package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	// Create a temp directory for the test db
	tmpDir, err := os.MkdirTemp("", "ratelord-store-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "ratelord.db")

	// Test initialization
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Verify file existence
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file was not created at %s", dbPath)
	}

	// Verify table existence
	// We can query the sqlite_master table
	var tableName string
	err = store.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&tableName)
	if err != nil {
		t.Fatalf("failed to query sqlite_master for events table: %v", err)
	}
	if tableName != "events" {
		t.Errorf("expected table 'events' to exist, but it was not found")
	}

	// Verify indices
	rows, err := store.db.Query("PRAGMA index_list('events')")
	if err != nil {
		t.Fatalf("failed to query index_list: %v", err)
	}
	defer rows.Close()

	foundIngestIndex := false
	foundCorrelationIndex := false

	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			// The number of columns returned by PRAGMA index_list varies by SQLite version.
			// Try scanning just the first few if this fails?
			// Actually, let's just check the names if we can.
			// standard columns: seq, name, unique, origin, partial
			t.Logf("scanning index row failed: %v", err)
			continue
		}
		if name == "idx_events_ts_ingest" {
			foundIngestIndex = true
		}
		if name == "idx_events_correlation" {
			foundCorrelationIndex = true
		}
	}

	if !foundIngestIndex {
		t.Errorf("idx_events_ts_ingest not found")
	}
	if !foundCorrelationIndex {
		t.Errorf("idx_events_correlation not found")
	}
}

func TestAppendEvent(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "ratelord-store-test-append")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "ratelord.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create a sample event
	evt := &Event{
		EventID:       "evt_123",
		EventType:     EventTypeUsageObserved,
		SchemaVersion: 1,
		TsEvent:       time.Now().UTC(),
		TsIngest:      time.Now().UTC(),
		Source: EventSource{
			OriginKind: "client",
			OriginID:   "cli_001",
			WriterID:   "ratelord-d",
		},
		Dimensions: EventDimensions{
			AgentID:    "agent_a",
			IdentityID: "user_1",
			WorkloadID: "wl_main",
			ScopeID:    "scope_global",
		},
		Correlation: EventCorrelation{
			CorrelationID: "corr_abc",
			CausationID:   "cause_xyz",
		},
		Payload: json.RawMessage(`{"tokens": 50}`),
	}

	// Execute AppendEvent
	if err := store.AppendEvent(context.Background(), evt); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	// Verify persistence by querying directly
	var count int
	err = store.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count events: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}

	// Verify specific fields
	var (
		id    string
		etype string
		pay   []byte
	)
	err = store.db.QueryRow("SELECT event_id, event_type, payload FROM events WHERE event_id = ?", "evt_123").Scan(&id, &etype, &pay)
	if err != nil {
		t.Fatalf("failed to query inserted event: %v", err)
	}

	if id != string(evt.EventID) {
		t.Errorf("expected event_id %s, got %s", evt.EventID, id)
	}
	if etype != string(evt.EventType) {
		t.Errorf("expected event_type %s, got %s", evt.EventType, etype)
	}
	if string(pay) != string(evt.Payload) {
		t.Errorf("expected payload %s, got %s", evt.Payload, pay)
	}
}

func TestReadEvents(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "ratelord-store-test-read")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "ratelord.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Seed multiple events with increasing timestamps
	baseTime := time.Now().UTC().Truncate(time.Second) // Truncate for stable comparison

	events := []*Event{
		{
			EventID:    "evt_1",
			TsIngest:   baseTime.Add(1 * time.Second),
			Payload:    json.RawMessage(`{"seq": 1}`),
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		},
		{
			EventID:    "evt_2",
			TsIngest:   baseTime.Add(2 * time.Second),
			Payload:    json.RawMessage(`{"seq": 2}`),
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		},
		{
			EventID:    "evt_3",
			TsIngest:   baseTime.Add(3 * time.Second),
			Payload:    json.RawMessage(`{"seq": 3}`),
			Dimensions: EventDimensions{AgentID: "a", IdentityID: "i", WorkloadID: "w", ScopeID: "s"},
		},
	}

	for _, evt := range events {
		// Fill mandatory fields
		evt.EventType = EventTypeUsageObserved
		evt.SchemaVersion = 1
		evt.TsEvent = evt.TsIngest
		evt.Source = EventSource{OriginKind: "test", OriginID: "test", WriterID: "test"}

		if err := store.AppendEvent(context.Background(), evt); err != nil {
			t.Fatalf("failed to seed event %s: %v", evt.EventID, err)
		}
	}

	// Test 1: Read all from beginning
	// Use a time definitely before the first event
	readAll, err := store.ReadEvents(context.Background(), baseTime, 10)
	if err != nil {
		t.Fatalf("ReadEvents (all) failed: %v", err)
	}
	if len(readAll) != 3 {
		t.Errorf("expected 3 events, got %d", len(readAll))
	} else {
		if readAll[0].EventID != "evt_1" {
			t.Errorf("expected first event to be evt_1, got %s", readAll[0].EventID)
		}
		if readAll[2].EventID != "evt_3" {
			t.Errorf("expected last event to be evt_3, got %s", readAll[2].EventID)
		}
	}

	// Test 2: Read from offset (after evt_1)
	readPartial, err := store.ReadEvents(context.Background(), baseTime.Add(1*time.Second), 10)
	if err != nil {
		t.Fatalf("ReadEvents (partial) failed: %v", err)
	}
	if len(readPartial) != 2 {
		t.Errorf("expected 2 events, got %d", len(readPartial))
	} else {
		if readPartial[0].EventID != "evt_2" {
			t.Errorf("expected first returned event to be evt_2, got %s", readPartial[0].EventID)
		}
	}

	// Test 3: Limit
	readLimit, err := store.ReadEvents(context.Background(), baseTime, 2)
	if err != nil {
		t.Fatalf("ReadEvents (limit) failed: %v", err)
	}
	if len(readLimit) != 2 {
		t.Errorf("expected 2 events (limit), got %d", len(readLimit))
	}
}
