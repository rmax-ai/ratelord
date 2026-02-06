package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

type RollupWorker struct {
	store *store.Store
}

func NewRollupWorker(st *store.Store) *RollupWorker {
	return &RollupWorker{store: st}
}

func (r *RollupWorker) Run(ctx context.Context) {
	log.Println("Starting rollup worker")

	ticker := time.NewTicker(30 * time.Second) // Run every 30 seconds for demo
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Rollup worker stopping")
			return
		case <-ticker.C:
			if err := r.ProcessBatch(ctx); err != nil {
				log.Printf("Rollup batch failed: %v", err)
			}
		}
	}
}

func (r *RollupWorker) ProcessBatch(ctx context.Context) error {
	// Get high water mark
	hwmStr, err := r.store.GetSystemState(ctx, "rollup_hwm_ts")
	if err != nil && err.Error() != "key not found: rollup_hwm_ts" {
		return fmt.Errorf("failed to get rollup_hwm_ts: %w", err)
	}
	var since time.Time
	if hwmStr != "" {
		if ts, err := time.Parse(time.RFC3339, hwmStr); err == nil {
			since = ts
		}
	}

	// Read batch of events
	events, err := r.store.ReadEvents(ctx, since, 1000) // Limit to 1000 events
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	if len(events) == 0 {
		return nil // No new events
	}

	// Filter for usage_observed
	var usageEvents []*store.Event
	for _, evt := range events {
		if evt.EventType == store.EventTypeUsageObserved {
			usageEvents = append(usageEvents, evt)
		}
	}

	if len(usageEvents) == 0 {
		// Update hwm anyway
		lastTs := events[len(events)-1].TsIngest
		return r.store.SetSystemState(ctx, "rollup_hwm_ts", lastTs.Format(time.RFC3339))
	}

	// Group and aggregate
	groups := make(map[string]*store.UsageStat)

	for _, evt := range usageEvents {
		var payload struct {
			ProviderID string `json:"provider_id"`
			PoolID     string `json:"pool_id"`
			Used       *int   `json:"used,omitempty"`
			Delta      *int   `json:"delta,omitempty"`
		}
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			log.Printf("Failed to unmarshal usage payload: %v", err)
			continue
		}

		if payload.ProviderID == "" || payload.PoolID == "" {
			continue
		}

		// Bucket hour: truncate TsEvent to hour
		bucket := evt.TsEvent.Truncate(time.Hour)

		key := fmt.Sprintf("%s|%s|%s|%s|%s", bucket.Format(time.RFC3339), payload.ProviderID, payload.PoolID, evt.Dimensions.IdentityID, evt.Dimensions.ScopeID)

		stat, exists := groups[key]
		if !exists {
			stat = &store.UsageStat{
				BucketTs:   bucket,
				ProviderID: payload.ProviderID,
				PoolID:     payload.PoolID,
				IdentityID: evt.Dimensions.IdentityID,
				ScopeID:    evt.Dimensions.ScopeID,
				TotalUsage: 0,
				MinUsage:   0,
				MaxUsage:   0,
				EventCount: 0,
			}
			groups[key] = stat
		}

		stat.EventCount++

		var value int
		if payload.Delta != nil {
			value = *payload.Delta
			stat.TotalUsage += value
		} else if payload.Used != nil {
			value = *payload.Used
		} else {
			continue
		}

		if stat.EventCount == 1 {
			stat.MinUsage = value
			stat.MaxUsage = value
		} else {
			if value < stat.MinUsage {
				stat.MinUsage = value
			}
			if value > stat.MaxUsage {
				stat.MaxUsage = value
			}
		}
	}

	// Prepare stats slice
	var stats []store.UsageStat
	for _, stat := range groups {
		stats = append(stats, *stat)
	}

	// Upsert
	if err := r.store.UpsertUsageStats(ctx, stats); err != nil {
		return fmt.Errorf("failed to upsert usage stats: %w", err)
	}

	// Update hwm
	lastTs := events[len(events)-1].TsIngest
	return r.store.SetSystemState(ctx, "rollup_hwm_ts", lastTs.Format(time.RFC3339))
}
