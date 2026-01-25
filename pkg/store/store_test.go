package store

import (
	"os"
	"path/filepath"
	"testing"
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
