package engine

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// Identity represents the read-model for an identity
type Identity struct {
	ID       string                 `json:"id"`
	Kind     string                 `json:"kind"`
	Metadata map[string]interface{} `json:"metadata"`
}

// IdentityProjection maintains an in-memory list of registered identities
type IdentityProjection struct {
	mu         sync.RWMutex
	identities map[string]Identity
}

// NewIdentityProjection creates a new empty projection
func NewIdentityProjection() *IdentityProjection {
	return &IdentityProjection{
		identities: make(map[string]Identity),
	}
}

// Apply updates the projection with a single event
func (p *IdentityProjection) Apply(event store.Event) error {
	if event.EventType != store.EventTypeIdentityRegistered {
		return nil // Ignore irrelevant events
	}

	var payload struct {
		Kind     string                 `json:"kind"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload for event %s: %w", event.EventID, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Dimensions.IdentityID is the source of truth for the ID
	id := event.Dimensions.IdentityID
	p.identities[id] = Identity{
		ID:       id,
		Kind:     payload.Kind,
		Metadata: payload.Metadata,
	}

	return nil
}

// Replay rebuilds the projection from a slice of events
func (p *IdentityProjection) Replay(events []*store.Event) error {
	for _, event := range events {
		if event == nil {
			continue
		}
		if err := p.Apply(*event); err != nil {
			// Log error but continue replaying? For now return error.
			return err
		}
	}
	return nil
}

// GetAll returns a list of all identities
func (p *IdentityProjection) GetAll() []Identity {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := make([]Identity, 0, len(p.identities))
	for _, id := range p.identities {
		list = append(list, id)
	}
	return list
}

// Get looks up a specific identity
func (p *IdentityProjection) Get(id string) (Identity, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	identity, ok := p.identities[id]
	return identity, ok
}
