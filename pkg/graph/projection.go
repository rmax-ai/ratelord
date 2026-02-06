package graph

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// Projection maintains the in-memory constraint graph.
type Projection struct {
	mu             sync.RWMutex
	graph          *Graph
	lastEventID    string
	lastIngestTime time.Time
}

// NewProjection creates a new empty graph projection.
func NewProjection() *Projection {
	return &Projection{
		graph: NewGraph(),
	}
}

// Apply updates the projection with a single event.
func (p *Projection) Apply(event store.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastEventID = string(event.EventID)
	p.lastIngestTime = event.TsIngest

	switch event.EventType {
	case store.EventTypeIdentityRegistered:
		return p.handleIdentityRegistered(event)
		// TODO: Handle PolicyUpdated to build constraint/pool nodes
		// TODO: Handle ProviderObserved to build provider nodes?
	}

	return nil
}

func (p *Projection) handleIdentityRegistered(event store.Event) error {
	var payload struct {
		Kind      string                 `json:"kind"`
		Metadata  map[string]interface{} `json:"metadata"`
		TokenHash string                 `json:"token_hash"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	id := event.Dimensions.IdentityID
	node := &Node{
		ID:         id,
		Type:       NodeIdentity,
		Label:      id, // Use ID as label for now, or extract from metadata
		Properties: make(map[string]string),
	}
	if payload.Kind != "" {
		node.Properties["kind"] = payload.Kind
	}

	p.graph.AddNode(node)
	return nil
}

// Replay rebuilds the projection from a slice of events.
func (p *Projection) Replay(events []*store.Event) error {
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

// GetGraph returns a snapshot of the current graph.
func (p *Projection) GetGraph() *Graph {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a deep copy if we want to be safe, or just the pointer if we trust the caller not to mutate.
	// For now, return the pointer but the caller should treat it as read-only or we should implement Clone().
	// Given this is an in-memory projection for the daemon, returning the pointer is risky if concurrent reads happen.
	// But `Graph` struct has a map. Let's return a clone for safety if it's small, or rely on the lock being held during read if we expose a "Read(func(*Graph))" method.
	// For simplicity in this first pass, let's clone the top level structure but share nodes (assuming nodes are immutable once added? No, they might be updated).
	// Let's just return the struct as is for now, but note the concurrency risk.
	// Better approach: Clone.

	newGraph := NewGraph()
	for k, v := range p.graph.Nodes {
		// Shallow copy of node is fine if we don't mutate map properties concurrently
		n := *v
		newGraph.Nodes[k] = &n
	}
	for _, e := range p.graph.Edges {
		edge := *e
		newGraph.Edges = append(newGraph.Edges, &edge)
	}
	return newGraph
}
