package engine

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rmax-ai/ratelord/pkg/blob"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// ArchiveConfig holds configuration for the ArchiveWorker.
type ArchiveConfig struct {
	Enabled       bool          `json:"enabled"`
	Retention     time.Duration `json:"retention"`
	BatchSize     int           `json:"batch_size"`
	CheckInterval time.Duration `json:"check_interval"`
}

// ArchiveWorker handles archiving old events to blob storage.
type ArchiveWorker struct {
	store     *store.Store
	blobStore blob.BlobStore
	config    ArchiveConfig
}

// NewArchiveWorker creates a new ArchiveWorker.
func NewArchiveWorker(store *store.Store, blobStore blob.BlobStore, config ArchiveConfig) *ArchiveWorker {
	return &ArchiveWorker{
		store:     store,
		blobStore: blobStore,
		config:    config,
	}
}

// Run starts the archive worker loop.
func (w *ArchiveWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				// Log error (in a real system, use structured logging)
				fmt.Printf("archive worker error: %v\n", err)
			}
		}
	}
}

// processBatch processes a batch of events for archiving.
func (w *ArchiveWorker) processBatch(ctx context.Context) error {
	// Safety Check: Get the latest snapshot timestamp
	snapshotTime, err := w.store.GetLatestSnapshotTime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest snapshot time: %w", err)
	}

	now := time.Now().UTC()
	safeCutoff := now.Add(-w.config.Retention)
	if !snapshotTime.IsZero() && snapshotTime.Before(safeCutoff) {
		safeCutoff = snapshotTime
	}

	// Fetch candidate events
	events, err := w.store.ReadCandidateEvents(ctx, safeCutoff, w.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to read candidate events: %w", err)
	}
	if len(events) == 0 {
		return nil // Nothing to archive
	}

	// Serialize events to JSON Lines
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	encoder := json.NewEncoder(gzWriter)

	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			gzWriter.Close()
			return fmt.Errorf("failed to encode event %s: %w", event.EventID, err)
		}
	}

	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Generate key: events/YYYY/MM/DD/first_ts_ingest_last_ts_ingest_uuid.jsonl.gz
	firstEvent := events[0]
	lastEvent := events[len(events)-1]
	year, month, day := firstEvent.TsIngest.Date()
	key := fmt.Sprintf("events/%04d/%02d/%02d/%d_%d_%s.jsonl.gz",
		year, month, day,
		firstEvent.TsIngest.Unix(),
		lastEvent.TsIngest.Unix(),
		uuid.New().String(),
	)

	// Upload to blob store
	if err := w.blobStore.Put(ctx, key, &buf); err != nil {
		return fmt.Errorf("failed to upload archive to blob store: %w", err)
	}

	// Collect event IDs for deletion
	eventIDs := make([]string, len(events))
	for i, event := range events {
		eventIDs[i] = string(event.EventID)
	}

	// Delete events from store
	if err := w.store.DeleteEvents(ctx, eventIDs); err != nil {
		return fmt.Errorf("failed to delete archived events: %w", err)
	}

	return nil
}
