package engine

import (
	"encoding/json"
	"sync"

	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

type ProviderProjection struct {
	mu     sync.RWMutex
	states map[string][]byte
}

func NewProviderProjection() *ProviderProjection {
	return &ProviderProjection{
		states: make(map[string][]byte),
	}
}

func (p *ProviderProjection) Apply(event store.Event) {
	if event.EventType == store.EventTypeProviderPollObserved {
		var payload map[string]interface{}
		json.Unmarshal(event.Payload, &payload)
		providerID, ok := payload["provider_id"].(string)
		if !ok {
			return
		}
		stateStr, ok := payload["state"].(string)
		if ok {
			state := []byte(stateStr)
			p.mu.Lock()
			p.states[providerID] = state
			p.mu.Unlock()
		}
	}
}

func (p *ProviderProjection) Replay(events []*store.Event) {
	for _, event := range events {
		p.Apply(*event)
	}
}

func (p *ProviderProjection) GetState(providerID provider.ProviderID) []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.states[string(providerID)]
}

func (p *ProviderProjection) GetAllStates() map[string][]byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	copyMap := make(map[string][]byte)
	for k, v := range p.states {
		clone := make([]byte, len(v))
		clone = append(clone[:0], v...)
		copyMap[k] = clone
	}
	return copyMap
}

func (p *ProviderProjection) LoadState(states map[string][]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.states = states
}
