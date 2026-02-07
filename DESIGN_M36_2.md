# Design: Cold Storage Offload (M36.2)

## Context
Ratelord uses an append-only SQLite event log. To manage disk space while preserving history, we need to offload older events to a "Cold Store" (Blob Storage).

## Goals
1.  **Offload**: Move events older than a configured threshold from SQLite to Blob Storage.
2.  **Safety**: Ensure events are persisted in Cold Storage before deletion from Hot Storage.
3.  **Replayability**: Archive format must support future replay (M36.3).
4.  **Local-First**: Initial implementation uses the local filesystem, extensible to S3/GCS.

## Architecture

### 1. Store Extensions
We need to extend `pkg/store` to support fetching and deleting specific batches of old events.

**New Methods:**
```go
// Fetch events older than a specific time, limited by batch size
func (s *Store) ReadCandidateEvents(ctx context.Context, before time.Time, limit int) ([]*Event, error)

// Delete a specific set of events by ID (safe deletion after archive)
func (s *Store) DeleteEvents(ctx context.Context, eventIDs []string) error
```

### 2. BlobStore Interface
Located in `pkg/blob/types.go`.

```go
package blob

import (
	"context"
	"io"
)

type BlobStore interface {
	// Put uploads content to the blob store.
	Put(ctx context.Context, key string, reader io.Reader) error

	// Get retrieves content from the blob store.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// List returns a list of keys matching the prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes a blob.
	Delete(ctx context.Context, key string) error
}
```

### 3. LocalBlobStore Implementation
Located in `pkg/blob/local_store.go`.
-   **Root**: Configurable directory (e.g., `./data/blobs`).
-   **Structure**: Maps keys directly to file paths.
-   **Atomicity**: Uses write-to-temp + rename to ensure partial writes don't corrupt data.

### 4. ArchiveWorker
Located in `pkg/engine/archive.go`.

**Configuration (`ArchiveConfig`):**
-   `Enabled`: boolean.
-   `Retention`: duration (e.g., `720h` for 30 days). Events older than this are archived.
-   `BatchSize`: integer (e.g., 1000).
-   `CheckInterval`: duration (e.g., `1h`).

**Workflow:**
1.  **Safety Check**:
    -   Get the latest **Snapshot**.
    -   Ensure we only archive events that are *older* than the snapshot. This guarantees that if we crash and recover from the snapshot, we don't need the archived events immediately to rebuild state.
    -   `SafeCutoff = min(Now - Retention, Snapshot.TsIngest)`

2.  **Batch Process**:
    -   `events = store.ReadCandidateEvents(ctx, SafeCutoff, BatchSize)`
    -   If `len(events) == 0`, sleep.

3.  **Serialize & Compress**:
    -   Format: **JSON Lines** (`.jsonl`). Each line is a full JSON `Event`.
    -   Compression: **Gzip**.
    -   Filename: `events/YYYY/MM/DD/{first_ts_ingest}_{last_ts_ingest}_{uuid}.jsonl.gz`
        -   Partitioning by date allows efficient listing/filtering later.
        -   Including timestamp range in filename aids searching.

4.  **Upload**:
    -   `blobStore.Put(key, compressed_data)`

5.  **Delete**:
    -   `store.DeleteEvents(ctx, [e.EventID for e in events])`
    -   This is safer than deleting by range, ensuring we exactly delete what we archived.

## Data Format (Archive)
Each file is a GZIP compressed stream of newline-delimited JSON objects matching the `Event` struct structure.

```json
{"event_id":"evt_1","type":"usage_observed","ts_ingest":"2023-01-01T00:00:00Z",...}
{"event_id":"evt_2","type":"usage_observed","ts_ingest":"2023-01-01T00:00:01Z",...}
```

## Integration Plan
1.  Implement `pkg/blob` package.
2.  Update `pkg/store` with new methods.
3.  Implement `ArchiveWorker` in `pkg/engine`.
4.  Update `ratelord-d` configuration to include `Archive` settings.
5.  Wire up `ArchiveWorker` in `main.go`.
