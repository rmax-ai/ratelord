package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// UsageStore abstracts the storage of pool state
type UsageStore interface {
	Get(providerID, poolID string) (PoolState, bool)
	Set(state PoolState)
	GetAll() []PoolState
	Clear()
}

// MemoryUsageStore implements UsageStore using an in-memory map
type MemoryUsageStore struct {
	pools map[string]PoolState
}

func NewMemoryUsageStore() *MemoryUsageStore {
	return &MemoryUsageStore{
		pools: make(map[string]PoolState),
	}
}

func (s *MemoryUsageStore) Get(providerID, poolID string) (PoolState, bool) {
	state, ok := s.pools[makePoolKey(providerID, poolID)]
	return state, ok
}

func (s *MemoryUsageStore) Set(state PoolState) {
	s.pools[makePoolKey(state.ProviderID, state.PoolID)] = state
}

func (s *MemoryUsageStore) GetAll() []PoolState {
	list := make([]PoolState, 0, len(s.pools))
	for _, pool := range s.pools {
		list = append(list, pool)
	}
	return list
}

func (s *MemoryUsageStore) Clear() {
	s.pools = make(map[string]PoolState)
}

// PoolState represents the current usage state of a constraint pool
type PoolState struct {
	ProviderID     string             `json:"provider_id"`
	PoolID         string             `json:"pool_id"`
	Used           int64              `json:"used"`
	Remaining      int64              `json:"remaining"`
	Cost           currency.MicroUSD  `json:"cost,omitempty"`
	ResetAt        time.Time          `json:"reset_at"`
	LastUpdated    time.Time          `json:"last_updated"`
	LatestForecast *forecast.Forecast `json:"latest_forecast,omitempty"`
}

// UsageProjection maintains in-memory usage state per pool
type UsageProjection struct {
	mu             sync.RWMutex
	store          UsageStore
	lastEventID    string
	lastIngestTime time.Time
}

// NewUsageProjection creates a new empty projection with in-memory storage
func NewUsageProjection() *UsageProjection {
	return NewUsageProjectionWithStore(NewMemoryUsageStore())
}

// NewUsageProjectionWithStore creates a new projection with a specific backing store
func NewUsageProjectionWithStore(store UsageStore) *UsageProjection {
	return &UsageProjection{
		store: store,
	}
}

// makePoolKey generates a unique key for a pool
func makePoolKey(providerID, poolID string) string {
	return fmt.Sprintf("%s:%s", providerID, poolID)
}

// Apply updates the projection with a single event
func (p *UsageProjection) Apply(event store.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastEventID = string(event.EventID)
	p.lastIngestTime = event.TsIngest

	switch event.EventType {
	case store.EventTypeUsageObserved:
		return p.applyUsage(event)
	case store.EventTypeResetObserved:
		return p.applyReset(event)
	case store.EventTypeForecastComputed:
		return p.applyForecast(event)
	}
	return nil
}

func (p *UsageProjection) applyForecast(event store.Event) error {
	var payload struct {
		ProviderID string            `json:"provider_id"`
		PoolID     string            `json:"pool_id"`
		Forecast   forecast.Forecast `json:"forecast"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal forecast payload: %w", err)
	}

	RatelordForecastSeconds.WithLabelValues(payload.ProviderID, payload.PoolID).Set(float64(payload.Forecast.TTE.P99Seconds))

	state, exists := p.store.Get(payload.ProviderID, payload.PoolID)
	if !exists {
		state = PoolState{
			ProviderID: payload.ProviderID,
			PoolID:     payload.PoolID,
		}
	}

	state.LatestForecast = &payload.Forecast
	state.LastUpdated = event.TsIngest

	p.store.Set(state)

	return nil
}

func (p *UsageProjection) applyUsage(event store.Event) error {
	var payload struct {
		ProviderID string            `json:"provider_id"`
		PoolID     string            `json:"pool_id"`
		Used       int64             `json:"used"`
		Remaining  int64             `json:"remaining"`
		Cost       currency.MicroUSD `json:"cost"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal usage payload: %w", err)
	}

	state, exists := p.store.Get(payload.ProviderID, payload.PoolID)
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
	state.Cost = payload.Cost
	state.LastUpdated = event.TsIngest

	p.store.Set(state)

	// Update metrics
	RatelordUsage.WithLabelValues(payload.ProviderID, payload.PoolID).Set(float64(payload.Used))
	RatelordLimit.WithLabelValues(payload.ProviderID, payload.PoolID).Set(float64(payload.Remaining))

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

	state, exists := p.store.Get(payload.ProviderID, payload.PoolID)
	if !exists {
		state = PoolState{
			ProviderID: payload.ProviderID,
			PoolID:     payload.PoolID,
		}
	}

	state.ResetAt = payload.ResetAt
	state.LastUpdated = event.TsIngest

	p.store.Set(state)
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

// LoadState restores the projection state from a snapshot
func (p *UsageProjection) LoadState(lastEventID string, lastIngestTime time.Time, pools []PoolState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastEventID = lastEventID
	p.lastIngestTime = lastIngestTime

	p.store.Clear()

	for _, pool := range pools {
		p.store.Set(pool)
	}
}

// GetPoolState returns the state for a specific pool
func (p *UsageProjection) GetPoolState(providerID, poolID string) (PoolState, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.store.Get(providerID, poolID)
}

// GetState returns the current state and the last applied event ID/Timestamp
func (p *UsageProjection) GetState() (string, time.Time, []PoolState) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lastEventID, p.lastIngestTime, p.store.GetAll()
}

// CalculateWaitTime returns the seconds until reset for a pool
func (p *UsageProjection) CalculateWaitTime(providerID, poolID string) float64 {
	state, exists := p.GetPoolState(providerID, poolID)
	if !exists || state.ResetAt.IsZero() {
		return 0
	}
	timeLeft := time.Until(state.ResetAt)
	if timeLeft > 0 {
		return timeLeft.Seconds()
	}
	return 0
}
