package provider

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// MockProvider generates synthetic usage data for testing
type MockProvider struct {
	id     ProviderID
	mu     sync.Mutex
	pools  map[string]*MockPool
	config MockConfig
}

type MockConfig struct {
	Jitter    time.Duration
	ErrorRate float64
}

type MockPool struct {
	ID        string
	Limit     int64
	Used      int64
	ResetAt   time.Time
	ResetFreq time.Duration
}

// NewMockProvider creates a new mock provider with default pools
func NewMockProvider(id string) *MockProvider {
	mp := &MockProvider{
		id:    ProviderID(id),
		pools: make(map[string]*MockPool),
		config: MockConfig{
			Jitter:    100 * time.Millisecond,
			ErrorRate: 0.0, // Clean by default
		},
	}

	// Initialize with a default pool
	mp.pools["default"] = &MockPool{
		ID:        "default",
		Limit:     5000,
		Used:      0,
		ResetFreq: 1 * time.Hour,
		ResetAt:   time.Now().Add(1 * time.Hour),
	}

	return mp
}

// InjectUsage allows manually increasing usage on a pool to simulate drift
func (p *MockProvider) InjectUsage(poolID string, amount int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pool, ok := p.pools[poolID]
	if !ok {
		return fmt.Errorf("pool %s not found", poolID)
	}

	if pool.Used+amount > pool.Limit {
		pool.Used = pool.Limit
	} else {
		pool.Used += amount
	}
	return nil
}

func (p *MockProvider) ID() ProviderID {
	return p.id
}

func (p *MockProvider) Poll(ctx context.Context) (PollResult, error) {
	// Simulate network latency
	select {
	case <-ctx.Done():
		return PollResult{ProviderID: p.id, Status: "error", Error: ctx.Err()}, ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Update state (simulate burn)
	now := time.Now()
	observations := make([]UsageObservation, 0, len(p.pools))

	for _, pool := range p.pools {
		// Handle reset
		if now.After(pool.ResetAt) {
			pool.Used = 0
			pool.ResetAt = now.Add(pool.ResetFreq)
		}

		// Simulate usage (random burn)
		burn := rand.Int63n(10)
		if pool.Used+burn <= pool.Limit {
			pool.Used += burn
		}

		observations = append(observations, UsageObservation{
			PoolID:    pool.ID,
			Used:      pool.Used,
			Remaining: pool.Limit - pool.Used,
			Limit:     pool.Limit,
			ResetAt:   pool.ResetAt,
		})
	}

	return PollResult{
		ProviderID: p.id,
		Status:     "success",
		Timestamp:  now,
		Usage:      observations,
	}, nil
}
