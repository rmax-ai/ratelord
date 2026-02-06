package graph

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestGraphProjection_Apply_IdentityRegistered(t *testing.T) {
	proj := NewProjection()

	payload := map[string]interface{}{
		"kind":       "service",
		"token_hash": "hash123",
	}
	payloadBytes, _ := json.Marshal(payload)

	event := store.Event{
		EventID:   "evt-1",
		EventType: store.EventTypeIdentityRegistered,
		TsIngest:  time.Now(),
		Dimensions: store.EventDimensions{
			IdentityID: "identity-1",
		},
		Payload: payloadBytes,
	}

	if err := proj.Apply(event); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	g := proj.GetGraph()
	if len(g.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(g.Nodes))
	}

	node := g.Nodes["identity-1"]
	if node.Type != NodeIdentity {
		t.Errorf("Expected node type %s, got %s", NodeIdentity, node.Type)
	}
	if node.Properties["kind"] != "service" {
		t.Errorf("Expected kind 'service', got '%s'", node.Properties["kind"])
	}
}

func TestGraphProjection_AddConstraint(t *testing.T) {
	proj := NewProjection()

	props := map[string]string{
		"type":  "hard",
		"limit": "100",
	}
	proj.AddConstraint("policy-1", "global", props)

	g := proj.GetGraph()

	// Check Constraint Node
	cNode, exists := g.Nodes["policy-1"]
	if !exists {
		t.Fatal("Constraint node not found")
	}
	if cNode.Type != NodeConstraint {
		t.Errorf("Expected type Constraint, got %s", cNode.Type)
	}
	if cNode.Properties["limit"] != "100" {
		t.Errorf("Expected limit 100, got %s", cNode.Properties["limit"])
	}

	// Check Scope Node
	sNode, exists := g.Nodes["global"]
	if !exists {
		t.Fatal("Scope node not found")
	}
	if sNode.Type != NodeScope {
		t.Errorf("Expected type Scope, got %s", sNode.Type)
	}

	// Check Edge
	if len(g.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(g.Edges))
	}
	edge := g.Edges[0]
	if edge.FromID != "policy-1" || edge.ToID != "global" || edge.Type != EdgeAppliesTo {
		t.Errorf("Edge mismatch: %+v", edge)
	}
}

func TestGraphProjection_FindConstraintsForScope(t *testing.T) {
	proj := NewProjection()
	proj.AddConstraint("p1", "global", nil)
	proj.AddConstraint("p2", "global", nil)
	proj.AddConstraint("p3", "other", nil)

	constraints, err := proj.FindConstraintsForScope("global")
	if err != nil {
		t.Fatalf("FindConstraintsForScope failed: %v", err)
	}

	if len(constraints) != 2 {
		t.Errorf("Expected 2 constraints, got %d", len(constraints))
	}
}
