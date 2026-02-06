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
