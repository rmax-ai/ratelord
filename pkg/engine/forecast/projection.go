package forecast

import (
	"container/ring"
	"encoding/json"
	"sync"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// ForecastProjection maintains sliding window history of usage observations per pool
type ForecastProjection struct {
	mu         sync.RWMutex
	histories  map[string]*ring.Ring // poolID -> circular buffer of UsagePoint
	windowSize int
}

// NewForecastProjection creates a new forecast projection with the specified window size
func NewForecastProjection(windowSize int) *ForecastProjection {
	return &ForecastProjection{
		histories:  make(map[string]*ring.Ring),
		windowSize: windowSize,
	}
}

// OnUsageObserved updates the history for a pool with a new usage observation
func (fp *ForecastProjection) OnUsageObserved(event *store.Event) {
	var payload struct {
		PoolID    string            `json:"pool_id"`
		Remaining int64             `json:"remaining"`
		Used      int64             `json:"used"`
		Cost      currency.MicroUSD `json:"cost"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		// Log error but don't panic
		return
	}

	point := UsagePoint{
		Timestamp: event.TsEvent,
		Used:      payload.Used,
		Remaining: payload.Remaining,
		Cost:      payload.Cost,
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	r, exists := fp.histories[payload.PoolID]
	if !exists {
		r = ring.New(fp.windowSize)
		fp.histories[payload.PoolID] = r
	}

	r.Value = point
	r = r.Next()
	fp.histories[payload.PoolID] = r
}

// GetHistory returns the usage history for a pool as a slice in chronological order
func (fp *ForecastProjection) GetHistory(poolID string) []UsagePoint {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	r, exists := fp.histories[poolID]
	if !exists {
		return nil
	}

	var history []UsagePoint
	// Collect from the ring in order: start from the oldest point
	// Fix bug: include all elements
	// r is the next write position (oldest or empty)
	start := r
	if start.Value != nil {
		if point, ok := start.Value.(UsagePoint); ok {
			history = append(history, point)
		}
	}
	for p := start.Next(); p != start; p = p.Next() {
		if p.Value != nil {
			if point, ok := p.Value.(UsagePoint); ok {
				history = append(history, point)
			}
		}
	}

	return history
}

// GetAllHistories returns the usage history for all pools
func (fp *ForecastProjection) GetAllHistories() map[string][]UsagePoint {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	result := make(map[string][]UsagePoint)
	for poolID, r := range fp.histories {
		if r == nil {
			continue
		}
		var history []UsagePoint
		r.Do(func(v interface{}) {
			if v != nil {
				if point, ok := v.(UsagePoint); ok {
					history = append(history, point)
				}
			}
		})
		result[poolID] = history
	}
	return result
}

// LoadHistories restores the usage history for all pools
func (fp *ForecastProjection) LoadHistories(histories map[string][]UsagePoint) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.histories = make(map[string]*ring.Ring)
	for poolID, points := range histories {
		if len(points) == 0 {
			continue
		}
		// Create a new ring
		r := ring.New(fp.windowSize)

		startIdx := 0
		if len(points) > fp.windowSize {
			startIdx = len(points) - fp.windowSize
		}

		for i := startIdx; i < len(points); i++ {
			r.Value = points[i]
			r = r.Next()
		}
		fp.histories[poolID] = r
	}
}
