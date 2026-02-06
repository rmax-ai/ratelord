package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// SnapshotPayload defines the structure of the JSON blob stored in snapshots
type SnapshotPayload struct {
	Identities        []Identity                       `json:"identities"`
	Pools             []PoolState                      `json:"pools"`
	ProviderStates    map[string][]byte                `json:"provider_states"`
	ForecastHistories map[string][]forecast.UsagePoint `json:"forecast_histories"`
}

// SnapshotWorker periodically persists the state of projections to the store
type SnapshotWorker struct {
	store      *store.Store
	identities *IdentityProjection
	usage      *UsageProjection
	providers  *ProviderProjection
	forecasts  *forecast.ForecastProjection
	interval   time.Duration
}

// NewSnapshotWorker creates a new worker
func NewSnapshotWorker(st *store.Store, id *IdentityProjection, usage *UsageProjection, prov *ProviderProjection, fore *forecast.ForecastProjection, interval time.Duration) *SnapshotWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &SnapshotWorker{
		store:      st,
		identities: id,
		usage:      usage,
		providers:  prov,
		forecasts:  fore,
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
	// Providers and Forecasts don't track "LastEventID" explicitly in the same way,
	// relying on event stream integrity. We assume they are up to date with the stream processed by ID/Usage projections.
	// Since all projections are updated in the same replay loop or stream processing,
	// taking the Minimum of (idTime, usageTime) is safe enough as the checkpoint.

	providerStates := w.providers.GetAllStates()
	forecastHistories := w.forecasts.GetAllHistories()

	// Determine the "safe" checkpoint.
	var safeEventID string
	if idEventID == "" || usageEventID == "" {
		// If mostly empty, maybe check if we have other data?
		// For now, strict check on core projections.
		// If system is fresh, we might skip snapshotting until some activity.
		return fmt.Errorf("cannot snapshot: not all projections have processed events (id=%s, usage=%s)", idEventID, usageEventID)
	}

	if idTime.Before(usageTime) {
		safeEventID = idEventID
	} else {
		safeEventID = usageEventID
	}

	payload := SnapshotPayload{
		Identities:        identities,
		Pools:             pools,
		ProviderStates:    providerStates,
		ForecastHistories: forecastHistories,
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

// LoadLatestSnapshot attempts to load the latest snapshot from the store.
// If successful, it restores the projections and returns the ingestion time of the last processed event.
// If no snapshot exists, it returns a zero time (indicating full replay is needed).
func LoadLatestSnapshot(ctx context.Context, st *store.Store, idProj *IdentityProjection, usageProj *UsageProjection, provProj *ProviderProjection, foreProj *forecast.ForecastProjection) (time.Time, error) {
	snap, err := st.GetLatestSnapshot(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest snapshot: %w", err)
	}
	if snap == nil {
		return time.Time{}, nil // No snapshot, full replay
	}

	// Fetch the checkpoint event to get the accurate ingest timestamp
	checkpointEvent, err := st.GetEvent(ctx, snap.LastEventID)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get checkpoint event %s: %w", snap.LastEventID, err)
	}
	if checkpointEvent == nil {
		return time.Time{}, fmt.Errorf("checkpoint event %s not found (inconsistent state)", snap.LastEventID)
	}

	var payload SnapshotPayload
	if err := json.Unmarshal(snap.Payload, &payload); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal snapshot payload: %w", err)
	}

	// Restore Projections
	idProj.LoadState(string(snap.LastEventID), checkpointEvent.TsIngest, payload.Identities)
	usageProj.LoadState(string(snap.LastEventID), checkpointEvent.TsIngest, payload.Pools)

	if payload.ProviderStates != nil {
		provProj.LoadState(payload.ProviderStates)
	}
	if payload.ForecastHistories != nil {
		foreProj.LoadHistories(payload.ForecastHistories)
	}

	return checkpointEvent.TsIngest, nil
}
