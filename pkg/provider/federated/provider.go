package federated

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/api"
	"github.com/rmax-ai/ratelord/pkg/provider"
)

// FederatedProvider implements the Provider interface but sources limits from a Leader
type FederatedProvider struct {
	id          provider.ProviderID
	leaderURL   string
	followerID  string
	client      *http.Client
	mu          sync.RWMutex
	pools       map[string]*PoolState // poolID -> State
	defaultPoll time.Duration
}

type PoolState struct {
	Granted    int64
	UsedLocal  int64
	ValidUntil time.Time
}

// NewFederatedProvider creates a new follower provider
func NewFederatedProvider(id string, leaderURL string, followerID string) *FederatedProvider {
	return &FederatedProvider{
		id:          provider.ProviderID(id),
		leaderURL:   leaderURL,
		followerID:  followerID,
		client:      &http.Client{Timeout: 5 * time.Second},
		pools:       make(map[string]*PoolState),
		defaultPoll: 10 * time.Second,
	}
}

// ID returns the provider ID
func (p *FederatedProvider) ID() provider.ProviderID {
	return p.id
}

// TrackUsage increments the local usage counter for a pool
func (p *FederatedProvider) TrackUsage(poolID string, amount int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.pools[poolID]
	if !ok {
		state = &PoolState{}
		p.pools[poolID] = state
	}
	state.UsedLocal += amount
}

// Poll contacts the leader to refresh grants and reports current state
func (p *FederatedProvider) Poll(ctx context.Context) (provider.PollResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := provider.PollResult{
		ProviderID: p.id,
		Timestamp:  time.Now(),
		Status:     "success",
	}

	// For each known pool, check if we need a grant
	// Note: In a real implementation, we might need to know which pools *should* exist from config.
	// For now, we only manage pools we've seen usage for or that are pre-seeded.
	// TODO: Allow pre-seeding pools.

	for poolID, state := range p.pools {
		// Logic: If remaining is low (< 20%) or expired, ask for more.
		// Remaining = Granted - UsedLocal
		remaining := state.Granted - state.UsedLocal

		needsGrant := false
		if time.Now().After(state.ValidUntil) {
			needsGrant = true
		} else if state.Granted > 0 && float64(remaining)/float64(state.Granted) < 0.2 {
			needsGrant = true
		} else if state.Granted == 0 {
			needsGrant = true
		}

		if needsGrant {
			// Request Grant
			// Assume a default request amount or base it on burn rate.
			// Simple start: Ask for 1000 or 2x used.
			askAmount := int64(1000)

			req := api.GrantRequest{
				FollowerID: p.followerID,
				PoolID:     poolID,
				Amount:     askAmount,
			}

			// Call Leader
			granted, validUntil, err := p.requestGrant(ctx, req)
			if err != nil {
				// Log error but continue with other pools
				// TODO: Add error to result
				fmt.Printf("federated_provider: failed to get grant for %s: %v\n", poolID, err)
			} else {
				// Update State
				// Strategy: New grant *adds* to existing? Or replaces?
				// Leader implementation (M30.1) just returns an amount.
				// If Leader "deducts from global", we "add to local".
				state.Granted += granted
				state.ValidUntil = validUntil
				remaining = state.Granted - state.UsedLocal
			}
		}

		// Append Observation
		obs := provider.UsageObservation{
			PoolID: poolID,
			Used:   state.UsedLocal, // This is what we report as "Used" (relative to our grant window?)
			// Wait, if we report "UsedLocal", UsageProjection uses that.
			// Limit should be "Granted".
			// Remaining = Granted - UsedLocal.
			Limit:     state.Granted,
			Remaining: remaining,
			ResetAt:   state.ValidUntil,
		}
		result.Usage = append(result.Usage, obs)
	}

	return result, nil
}

func (p *FederatedProvider) requestGrant(ctx context.Context, req api.GrantRequest) (int64, time.Time, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return 0, time.Time{}, err
	}

	url := fmt.Sprintf("%s/v1/federation/grant", p.leaderURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, time.Time{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, time.Time{}, fmt.Errorf("leader returned status %d", resp.StatusCode)
	}

	var grantResp api.GrantResponse
	if err := json.NewDecoder(resp.Body).Decode(&grantResp); err != nil {
		return 0, time.Time{}, err
	}

	return grantResp.Granted, grantResp.ValidUntil, nil
}

// Restore is a no-op for now, or could restore grants
func (p *FederatedProvider) Restore(state []byte) error {
	return nil
}

// RegisterPool explicitly adds a pool to be managed
func (p *FederatedProvider) RegisterPool(poolID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pools[poolID]; !ok {
		p.pools[poolID] = &PoolState{}
	}
}

// UsageRouter implements api.UsageTracker and routes to registered providers
type UsageRouter struct {
	providers map[string]*FederatedProvider
	mu        sync.RWMutex
}

func NewUsageRouter() *UsageRouter {
	return &UsageRouter{
		providers: make(map[string]*FederatedProvider),
	}
}

func (r *UsageRouter) Register(p *FederatedProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[string(p.ID())] = p
}

func (r *UsageRouter) TrackUsage(providerID, poolID string, amount int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.providers[providerID]; ok {
		p.TrackUsage(poolID, amount)
	}
}
