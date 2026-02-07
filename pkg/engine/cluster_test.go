package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestClusterTopology_Apply(t *testing.T) {
	topology := NewClusterTopology()

	// Test case 1: Apply a valid grant issued event
	payload := struct {
		FollowerID string                 `json:"follower_id"`
		PoolID     string                 `json:"pool_id"`
		Amount     int                    `json:"amount"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}{
		FollowerID: "node-1",
		PoolID:     "pool-a",
		Amount:     10,
		Metadata:   map[string]interface{}{"region": "us-east"},
	}
	payloadBytes, _ := json.Marshal(payload)

	event := store.Event{
		EventID:    "evt-1",
		EventType:  store.EventTypeGrantIssued,
		Payload:    payloadBytes,
		TsEvent:    time.Now(),
		Dimensions: store.EventDimensions{},
	}

	if err := topology.Apply(event); err != nil {
		t.Fatalf("Failed to apply event: %v", err)
	}

	nodes := topology.GetNodes(time.Hour)
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
	if nodes[0].NodeID != "node-1" {
		t.Errorf("Expected node-1, got %s", nodes[0].NodeID)
	}
	if nodes[0].Status != "active" {
		t.Errorf("Expected active status, got %s", nodes[0].Status)
	}
	if nodes[0].Metadata["region"] != "us-east" {
		t.Errorf("Expected region us-east, got %v", nodes[0].Metadata["region"])
	}

	// Test case 2: Ignore non-grant events
	otherEvent := store.Event{
		EventID:   "evt-2",
		EventType: store.EventTypeIntentSubmitted, // Irrelevant type
		TsEvent:   time.Now(),
	}
	if err := topology.Apply(otherEvent); err != nil {
		t.Fatalf("Failed to apply irrelevant event: %v", err)
	}
	// Node count should remain same
	if len(topology.GetNodes(time.Hour)) != 1 {
		t.Errorf("Node count changed after irrelevant event")
	}

	// Test case 3: Invalid payload
	invalidEvent := store.Event{
		EventID:   "evt-3",
		EventType: store.EventTypeGrantIssued,
		Payload:   []byte("invalid-json"),
	}
	if err := topology.Apply(invalidEvent); err == nil {
		t.Error("Expected error for invalid payload, got nil")
	}
}

func TestClusterTopology_GetNodes_TTL(t *testing.T) {
	topology := NewClusterTopology()

	// Add an old node
	oldTime := time.Now().Add(-2 * time.Hour)
	payload := struct {
		FollowerID string `json:"follower_id"`
	}{
		FollowerID: "old-node",
	}
	payloadBytes, _ := json.Marshal(payload)

	event := store.Event{
		EventID:   "evt-old",
		EventType: store.EventTypeGrantIssued,
		Payload:   payloadBytes,
		TsEvent:   oldTime,
	}
	topology.Apply(event)

	// Check with 1h TTL -> should be offline
	nodes := topology.GetNodes(time.Hour)
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node")
	}
	if nodes[0].Status != "offline" {
		t.Errorf("Expected offline status for old node, got %s", nodes[0].Status)
	}

	// Check with 3h TTL -> should be active
	nodes = topology.GetNodes(3 * time.Hour)
	if nodes[0].Status != "active" {
		t.Errorf("Expected active status with long TTL, got %s", nodes[0].Status)
	}
}

func TestClusterTopology_Replay(t *testing.T) {
	topology := NewClusterTopology()

	events := []*store.Event{}
	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]string{"follower_id": "node-replay"})
		events = append(events, &store.Event{
			EventID:   store.EventID(string(rune(i))),
			EventType: store.EventTypeGrantIssued,
			Payload:   payload,
			TsEvent:   time.Now(),
		})
	}

	if err := topology.Replay(events); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if len(topology.GetNodes(time.Hour)) != 1 {
		t.Errorf("Expected 1 unique node")
	}
}
