package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// Identity represents the read-model for an identity
type Identity struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Metadata  map[string]interface{} `json:"metadata"`
	TokenHash string                 `json:"-"` // Never export token hash in JSON
}

// IdentityProjection maintains an in-memory list of registered identities
type IdentityProjection struct {
	mu             sync.RWMutex
	identities     map[string]Identity
	tokenMap       map[string]string // hash -> identityID
	lastEventID    string
	lastIngestTime time.Time
}

// NewIdentityProjection creates a new empty projection
func NewIdentityProjection() *IdentityProjection {
	return &IdentityProjection{
		identities: make(map[string]Identity),
		tokenMap:   make(map[string]string),
	}
}

// Apply updates the projection with a single event
func (p *IdentityProjection) Apply(event store.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastEventID = string(event.EventID)
	p.lastIngestTime = event.TsIngest

	switch event.EventType {
	case store.EventTypeIdentityRegistered:
		var payload struct {
			Kind      string                 `json:"kind"`
			Metadata  map[string]interface{} `json:"metadata"`
			TokenHash string                 `json:"token_hash"`
		}

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for event %s: %w", event.EventID, err)
		}

		// Dimensions.IdentityID is the source of truth for the ID
		id := event.Dimensions.IdentityID
		p.identities[id] = Identity{
			ID:        id,
			Kind:      payload.Kind,
			Metadata:  payload.Metadata,
			TokenHash: payload.TokenHash,
		}

		if payload.TokenHash != "" {
			p.tokenMap[payload.TokenHash] = id
		}

	case store.EventTypeIdentityDeleted:
		id := event.Dimensions.IdentityID
		if identity, exists := p.identities[id]; exists {
			if identity.TokenHash != "" {
				delete(p.tokenMap, identity.TokenHash)
			}
			delete(p.identities, id)
		}

	default:
		// Ignore irrelevant events
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

// GetByTokenHash looks up an identity by its token hash
func (p *IdentityProjection) GetByTokenHash(hash string) (Identity, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	id, ok := p.tokenMap[hash]
	if !ok {
		return Identity{}, false
	}
	return p.identities[id], true
}

// LoadState restores the projection state from a snapshot
func (p *IdentityProjection) LoadState(lastEventID string, lastIngestTime time.Time, identities []Identity) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastEventID = lastEventID
	p.lastIngestTime = lastIngestTime

	// Clear existing state? Or merge? Snapshots are usually full state.
	p.identities = make(map[string]Identity)
	p.tokenMap = make(map[string]string)

	for _, id := range identities {
		p.identities[id.ID] = id
		if id.TokenHash != "" {
			p.tokenMap[id.TokenHash] = id.ID
		}
	}
}

// GetState returns the current state and the last applied event ID/Timestamp
func (p *IdentityProjection) GetState() (string, time.Time, []Identity) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := make([]Identity, 0, len(p.identities))
	for _, id := range p.identities {
		list = append(list, id)
	}
	return p.lastEventID, p.lastIngestTime, list
}
