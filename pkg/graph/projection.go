package graph

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// Projection maintains the in-memory constraint graph.
type Projection struct {
	mu               sync.RWMutex
	graph            *Graph
	lastEventID      string
	lastIngestTime   time.Time
	scopeConstraints map[string][]string // scopeID -> []constraintID
}

// NewProjection creates a new empty graph projection.
func NewProjection() *Projection {
	return &Projection{
		graph:            NewGraph(),
		scopeConstraints: make(map[string][]string),
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
	case store.EventTypePolicyUpdated:
		return p.handlePolicyUpdated(event)
	case store.EventTypeProviderPollObserved:
		return p.handleProviderPollObserved(event)
	}

	return nil
}

func (p *Projection) handleProviderPollObserved(event store.Event) error {
	var payload struct {
		ProviderID string `json:"provider_id"`
		Status     string `json:"status"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	if payload.ProviderID == "" {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure Provider Node exists
	// We map ProviderID to a Node
	if _, exists := p.graph.Nodes[payload.ProviderID]; !exists {
		p.graph.Nodes[payload.ProviderID] = &Node{
			ID:         payload.ProviderID,
			Type:       NodeResource, // Providers are resources in our graph taxonomy
			Label:      payload.ProviderID,
			Properties: map[string]string{"type": "provider"},
		}
	}

	return nil
}

func (p *Projection) handlePolicyUpdated(event store.Event) error {
	// Define a partial struct matching PolicyConfig to avoid cyclic dependency on pkg/engine
	var payload struct {
		Policies []struct {
			ID    string `json:"id"`
			Scope string `json:"scope"`
			Type  string `json:"type"`
			Limit int64  `json:"limit"`
		} `json:"policies"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	for _, policy := range payload.Policies {
		props := map[string]string{
			"type":  policy.Type,
			"limit": fmt.Sprintf("%d", policy.Limit),
		}
		// AddConstraint is internal, we can call it here but we are already holding the lock
		// So we should refactor AddConstraint or call internal version.
		// Since AddConstraint takes a lock, we can't call it from here (Apply holds lock).
		// We need an internal helper.
		p.addConstraintLocked(policy.ID, policy.Scope, props)
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

// EnsureNode adds a node if it doesn't exist.
func (p *Projection) EnsureNode(id string, nodeType NodeType) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.graph.Nodes[id]; !exists {
		p.graph.Nodes[id] = &Node{
			ID:         id,
			Type:       nodeType,
			Label:      id,
			Properties: make(map[string]string),
		}
	}
}

// AddConstraint adds a constraint node and links it to a scope.
func (p *Projection) AddConstraint(id, scope string, props map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addConstraintLocked(id, scope, props)
}

// addConstraintLocked performs the logic of AddConstraint without locking.
// Must be called with p.mu held.
func (p *Projection) addConstraintLocked(id, scope string, props map[string]string) {
	// Add Constraint Node
	cNode := &Node{
		ID:         id,
		Type:       NodeConstraint,
		Label:      id,
		Properties: props,
	}
	p.graph.Nodes[id] = cNode

	// Ensure Scope Node exists
	if _, exists := p.graph.Nodes[scope]; !exists {
		p.graph.Nodes[scope] = &Node{
			ID:    scope,
			Type:  NodeScope,
			Label: scope,
		}
	}

	// Update Adjacency Index
	// Check if already exists to avoid duplicates?
	// For O(1) we trust the map, but let's check duplicates if we replay.
	// Simple slice append for now.
	p.scopeConstraints[scope] = append(p.scopeConstraints[scope], id)

	// Link Constraint -> AppliesTo -> Scope
	// Check if edge exists? For now just append, maybe dedupe later or allow multiples
	// Ideally we check uniqueness.
	edge := &Edge{
		FromID: id,
		ToID:   scope,
		Type:   EdgeAppliesTo,
	}
	p.graph.Edges = append(p.graph.Edges, edge)
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

// FindConstraintsForScope returns all constraint nodes that apply to the given scope.
func (p *Projection) FindConstraintsForScope(scopeID string) ([]*Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var constraints []*Node

	// Use Adjacency Index for O(1) lookup
	if ids, ok := p.scopeConstraints[scopeID]; ok {
		for _, id := range ids {
			if node, exists := p.graph.Nodes[id]; exists {
				constraints = append(constraints, node)
			}
		}
		return constraints, nil
	}

	return constraints, nil
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
