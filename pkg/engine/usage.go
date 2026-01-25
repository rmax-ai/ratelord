package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rmax/ratelord/pkg/store"
)

// PoolState represents the current usage state of a constraint pool
type PoolState struct {
	ProviderID  string    `json:"provider_id"`
	PoolID      string    `json:"pool_id"`
	Used        int64     `json:"used"`
	Remaining   int64     `json:"remaining"`
	ResetAt     time.Time `json:"reset_at"`
	LastUpdated time.Time `json:"last_updated"`
}

// UsageProjection maintains in-memory usage state per pool
type UsageProjection struct {
	mu    sync.RWMutex
	pools map[string]PoolState // Key: provider_id:pool_id
}

// NewUsageProjection creates a new empty projection
func NewUsageProjection() *UsageProjection {
	return &UsageProjection{
		pools: make(map[string]PoolState),
	}
}

func makePoolKey(providerID, poolID string) string {
	return fmt.Sprintf("%s:%s", providerID, poolID)
}

// Apply updates the projection with a single event
func (p *UsageProjection) Apply(event store.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event.EventType {
	case store.EventTypeUsageObserved:
		return p.applyUsage(event)
	case store.EventTypeResetObserved:
		return p.applyReset(event)
	}
	return nil
}

func (p *UsageProjection) applyUsage(event store.Event) error {
	var payload struct {
		ProviderID string `json:"provider_id"`
		PoolID     string `json:"pool_id"`
		Used       int64  `json:"used"`
		Remaining  int64  `json:"remaining"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal usage payload: %w", err)
	}

	key := makePoolKey(payload.ProviderID, payload.PoolID)
	state, exists := p.pools[key]
	if !exists {
		state = PoolState{
			ProviderID: payload.ProviderID,
			PoolID:     payload.PoolID,
		}
	}

	// Update state
	// Note: We might want to handle partial updates or deltas later.
	// For now, assume absolute values if provided.
	state.Used = payload.Used
	state.Remaining = payload.Remaining
	state.LastUpdated = event.TsIngest

	p.pools[key] = state
	return nil
}

func (p *UsageProjection) applyReset(event store.Event) error {
	var payload struct {
		ProviderID string    `json:"provider_id"`
		PoolID     string    `json:"pool_id"`
		ResetAt    time.Time `json:"reset_at"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal reset payload: %w", err)
	}

	key := makePoolKey(payload.ProviderID, payload.PoolID)
	state, exists := p.pools[key]
	if !exists {
		state = PoolState{
			ProviderID: payload.ProviderID,
			PoolID:     payload.PoolID,
		}
	}

	state.ResetAt = payload.ResetAt
	state.LastUpdated = event.TsIngest

	p.pools[key] = state
	return nil
}

// Replay rebuilds the projection from a slice of events
func (p *UsageProjection) Replay(events []*store.Event) error {
	for _, event := range events {
		if event == nil {
			continue
		}
		if err := p.Apply(*event); err != nil {
			return err
		}
	}
	return nil
}

// GetPoolState returns the state for a specific pool
func (p *UsageProjection) GetPoolState(providerID, poolID string) (PoolState, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, ok := p.pools[makePoolKey(providerID, poolID)]
	return state, ok
}
