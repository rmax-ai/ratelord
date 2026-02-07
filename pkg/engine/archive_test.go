package engine

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rmax-ai/ratelord/pkg/blob"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestArchiveWorker_ProcessBatch(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "ratelord-archive-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "ratelord.db")
	blobDir := filepath.Join(tmpDir, "blobs")

	// Initialize store
	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer dbStore.Close()

	// Initialize blob store
	blobStore := blob.NewLocalBlobStore(blobDir)

	// Create events: 5 old, 5 new
	now := time.Now().UTC()
	retention := time.Hour
	oldTime := now.Add(-2 * retention)    // Older than retention
	newTime := now.Add(-30 * time.Minute) // Newer than retention

	ctx := context.Background()

	// Insert 5 old events
	for i := 0; i < 5; i++ {
		event := &store.Event{
			EventID:       store.EventID(uuid.New().String()),
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       oldTime,
			TsIngest:      oldTime,
			Source: store.EventSource{
				OriginKind: "test",
				OriginID:   "test-origin",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    "test-agent",
				IdentityID: "test-identity",
				WorkloadID: "test-workload",
				ScopeID:    "test-scope",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: uuid.New().String(),
			},
			Payload: json.RawMessage(`{"usage": 100}`),
		}
		if err := dbStore.AppendEvent(ctx, event); err != nil {
			t.Fatalf("failed to append old event %d: %v", i, err)
		}
	}

	// Insert 5 new events
	for i := 0; i < 5; i++ {
		event := &store.Event{
			EventID:       store.EventID(uuid.New().String()),
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       newTime,
			TsIngest:      newTime,
			Source: store.EventSource{
				OriginKind: "test",
				OriginID:   "test-origin",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    "test-agent",
				IdentityID: "test-identity",
				WorkloadID: "test-workload",
				ScopeID:    "test-scope",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: uuid.New().String(),
			},
			Payload: json.RawMessage(`{"usage": 200}`),
		}
		if err := dbStore.AppendEvent(ctx, event); err != nil {
			t.Fatalf("failed to append new event %d: %v", i, err)
		}
	}

	// Create ArchiveWorker
	config := ArchiveConfig{
		Enabled:       true,
		Retention:     retention,
		BatchSize:     10,
		CheckInterval: time.Minute,
	}
	worker := NewArchiveWorker(dbStore, blobStore, config)

	// Process batch
	if err := worker.processBatch(ctx); err != nil {
		t.Fatalf("processBatch failed: %v", err)
	}

	// Verify: ReadEvents should return only the 5 new events
	events, err := dbStore.ReadEvents(ctx, time.Time{}, 100)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 events remaining, got %d", len(events))
	}
	for _, event := range events {
		if event.TsIngest.Before(newTime.Add(-time.Minute)) {
			t.Errorf("found old event that should have been archived: %v", event.EventID)
		}
	}

	// Verify: BlobStore should have at least one file
	files, err := blobStore.List(ctx, "events/")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected at least one archived file, got none")
	}

	// Read and verify the archived file
	for _, file := range files {
		reader, err := blobStore.Get(ctx, file)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", file, err)
		}

		// Gunzip
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			reader.Close()
			t.Fatalf("gzip.NewReader failed: %v", err)
		}

		// Read all data
		data, err := io.ReadAll(gzReader)
		gzReader.Close()
		reader.Close()
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		// Decode JSON Lines
		lines := bytes.Split(data, []byte("\n"))
		var archivedEvents []*store.Event
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var event store.Event
			if err := json.Unmarshal(line, &event); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			archivedEvents = append(archivedEvents, &event)
		}

		// Should have 5 old events
		if len(archivedEvents) != 5 {
			t.Errorf("expected 5 archived events, got %d", len(archivedEvents))
		}
		for _, event := range archivedEvents {
			if !event.TsIngest.Equal(oldTime) {
				t.Errorf("archived event has wrong TsIngest: %v", event.TsIngest)
			}
		}
	}
}
