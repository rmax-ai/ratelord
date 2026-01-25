package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// Poller manages the polling loop for registered providers
type Poller struct {
	store      *store.Store
	providers  []provider.Provider
	interval   time.Duration
	forecaster *forecast.Forecaster
	mu         sync.RWMutex
}

// NewPoller creates a new poller instance
func NewPoller(store *store.Store, interval time.Duration, forecaster *forecast.Forecaster) *Poller {
	return &Poller{
		store:      store,
		providers:  make([]provider.Provider, 0),
		interval:   interval,
		forecaster: forecaster,
	}
}

// Register adds a provider to the polling list
func (p *Poller) Register(provider provider.Provider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers = append(p.providers, provider)
}

// GetProvider returns a registered provider by ID (helper for testing/debugging)
func (p *Poller) GetProvider(id provider.ProviderID) provider.Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, prov := range p.providers {
		if prov.ID() == id {
			return prov
		}
	}
	return nil
}

// Start begins the polling loop in a background goroutine
func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Println("Poller started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Poller stopping due to context cancellation")
			return
		case <-ticker.C:
			p.pollAll(ctx)
		}
	}
}

// pollAll polls all registered providers
func (p *Poller) pollAll(ctx context.Context) {
	p.mu.RLock()
	providers := make([]provider.Provider, len(p.providers))
	copy(providers, p.providers)
	p.mu.RUnlock()

	for _, prov := range providers {
		p.poll(ctx, prov)
	}
}

// poll performs a single poll on a provider and emits events
func (p *Poller) poll(ctx context.Context, prov provider.Provider) {
	result, err := prov.Poll(ctx)
	if err != nil {
		log.Printf("Poll failed for provider %s: %v", prov.ID(), err)
		// TODO: emit provider_error event if needed
		return
	}

	now := time.Now().UTC()
	correlationID := fmt.Sprintf("poll_%s_%d", prov.ID(), now.Unix())

	// Emit provider_poll_observed event
	pollEvent := &store.Event{
		EventID:       store.EventID(fmt.Sprintf("poll_%s_%d", prov.ID(), now.UnixNano())),
		EventType:     store.EventTypeProviderPollObserved,
		SchemaVersion: 1,
		TsEvent:       result.Timestamp,
		TsIngest:      now,
		Source: store.EventSource{
			OriginKind: "daemon",
			OriginID:   "poller",
			WriterID:   "ratelord-d",
		},
		Dimensions: store.EventDimensions{
			AgentID:    store.SentinelSystem,
			IdentityID: store.SentinelGlobal,
			WorkloadID: store.SentinelSystem,
			ScopeID:    store.SentinelGlobal,
		},
		Correlation: store.EventCorrelation{
			CorrelationID: correlationID,
			CausationID:   store.SentinelUnknown, // No parent for poll
		},
	}

	pollPayload := map[string]interface{}{
		"provider_id": string(result.ProviderID),
		"status":      result.Status,
		"observation_summary": map[string]interface{}{
			"usage_count": len(result.Usage),
		},
	}
	payloadBytes, _ := json.Marshal(pollPayload)
	pollEvent.Payload = payloadBytes

	if err := p.store.AppendEvent(ctx, pollEvent); err != nil {
		log.Printf("Failed to append poll event: %v", err)
		return
	}

	// Emit usage_observed events for each observation
	for i, obs := range result.Usage {
		usageEvent := &store.Event{
			EventID:       store.EventID(fmt.Sprintf("usage_%s_%s_%d_%d", prov.ID(), obs.PoolID, now.UnixNano(), i)),
			EventType:     store.EventTypeUsageObserved,
			SchemaVersion: 1,
			TsEvent:       result.Timestamp,
			TsIngest:      now,
			Source: store.EventSource{
				OriginKind: "daemon",
				OriginID:   "poller",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: store.SentinelGlobal,
				WorkloadID: store.SentinelSystem,
				ScopeID:    store.SentinelGlobal,
			},
			Correlation: store.EventCorrelation{
				CorrelationID: correlationID,
				CausationID:   string(pollEvent.EventID),
			},
		}

		usagePayload := map[string]interface{}{
			"provider_id": string(result.ProviderID),
			"pool_id":     obs.PoolID,
			"units":       "requests", // Assuming requests for mock; TODO: make configurable
			"remaining":   obs.Remaining,
			"used":        obs.Used,
		}
		payloadBytes, _ := json.Marshal(usagePayload)
		usageEvent.Payload = payloadBytes

		if err := p.store.AppendEvent(ctx, usageEvent); err != nil {
			log.Printf("Failed to append usage event: %v", err)
		} else if p.forecaster != nil {
			// Trigger forecast computation
			p.forecaster.OnUsageObserved(ctx, usageEvent)
		}
	}
}
