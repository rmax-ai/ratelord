package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// SnapshotPayload defines the structure of the JSON blob stored in snapshots
type SnapshotPayload struct {
	Identities []Identity  `json:"identities"`
	Pools      []PoolState `json:"pools"`
}

// SnapshotWorker periodically persists the state of projections to the store
type SnapshotWorker struct {
	store      *store.Store
	identities *IdentityProjection
	usage      *UsageProjection
	interval   time.Duration
}

// NewSnapshotWorker creates a new worker
func NewSnapshotWorker(st *store.Store, id *IdentityProjection, usage *UsageProjection, interval time.Duration) *SnapshotWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &SnapshotWorker{
		store:      st,
		identities: id,
		usage:      usage,
		interval:   interval,
	}
}

// Run starts the snapshot loop
func (w *SnapshotWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	fmt.Println(`{"level":"info","msg":"snapshot_worker_started"}`)

	for {
		select {
		case <-ctx.Done():
			fmt.Println(`{"level":"info","msg":"snapshot_worker_stopped"}`)
			return
		case <-ticker.C:
			if err := w.TakeSnapshot(ctx); err != nil {
				fmt.Printf(`{"level":"error","msg":"snapshot_failed","error":"%v"}`+"\n", err)
			} else {
				fmt.Println(`{"level":"info","msg":"snapshot_created"}`)
			}
		}
	}
}

// TakeSnapshot captures the current state and saves it to the store
func (w *SnapshotWorker) TakeSnapshot(ctx context.Context) error {
	idEventID, idTime, identities := w.identities.GetState()
	usageEventID, usageTime, pools := w.usage.GetState()

	// Determine the "safe" checkpoint.
	// We must choose the EventID that corresponds to the OLDER timestamp,
	// because the snapshot guarantees that state is valid AT LEAST up to that point.
	// If we chose the newer one, we might claim to have processed events that one projection hasn't seen yet.
	// Replay logic will start AFTER this EventID.
	// If we restart, we replay events > safeEventID.
	// Since both projections have applied safeEventID (because it's the older one, or equal),
	// replaying events > safeEventID is correct (idempotency handles re-applying if needed, but append-only log replay usually assumes exactly-once delivery from a point).
	// Note: Our projections are idempotent-ish (Apply overwrites state), so replaying a few extra events is fine.
	// Replaying FEWER events is bad (missing data).
	// So we need to pick the EventID corresponding to the OLDER timestamp.

	var safeEventID string
	// If either is empty (start of time), we can't really snapshot safely unless we assume empty state.
	// But if one is empty and other isn't, we should probably skip snapshot or use the empty one?
	// If idEventID is empty, it means no identities registered. safeEventID = usageEventID?
	// No, if idEventID is empty, we haven't processed any identity events.
	// If we use usageEventID (which is > 0), and replay from there, we skip identity events < usageEventID?
	// Yes, potentially. So if one is empty, we probably shouldn't snapshot, or we use the empty one (if the store allows it).
	// The store requires `last_event_id` NOT NULL.
	// So we need at least one event to have been processed by BOTH?
	// Or we just take the one with older timestamp. empty time is time.Time{}.

	if idEventID == "" || usageEventID == "" {
		// Not enough state to snapshot
		return fmt.Errorf("cannot snapshot: not all projections have processed events (id=%s, usage=%s)", idEventID, usageEventID)
	}

	if idTime.Before(usageTime) {
		safeEventID = idEventID
	} else {
		safeEventID = usageEventID
	}

	payload := SnapshotPayload{
		Identities: identities,
		Pools:      pools,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot payload: %w", err)
	}

	snap := &store.Snapshot{
		SnapshotID:    fmt.Sprintf("snap_%d", time.Now().UnixNano()),
		SchemaVersion: 1,
		TsSnapshot:    time.Now().UTC(),
		LastEventID:   store.EventID(safeEventID),
		Payload:       payloadJSON,
	}

	if err := w.store.SaveSnapshot(ctx, snap); err != nil {
		return fmt.Errorf("store save failed: %w", err)
	}

	return nil
}
