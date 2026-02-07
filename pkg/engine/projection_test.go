package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestIdentityProjection_Apply(t *testing.T) {
	proj := NewIdentityProjection()

	// 1. Test Registration
	meta := map[string]interface{}{"env": "prod"}
	payload := struct {
		Kind      string                 `json:"kind"`
		Metadata  map[string]interface{} `json:"metadata"`
		TokenHash string                 `json:"token_hash"`
	}{
		Kind:      "service",
		Metadata:  meta,
		TokenHash: "hash123",
	}
	payloadBytes, _ := json.Marshal(payload)

	regEvent := store.Event{
		EventID:    "evt-1",
		EventType:  store.EventTypeIdentityRegistered,
		Payload:    payloadBytes,
		Dimensions: store.EventDimensions{IdentityID: "id-1"},
	}

	if err := proj.Apply(regEvent); err != nil {
		t.Fatalf("Failed to apply registration: %v", err)
	}

	// Verify state
	id, ok := proj.Get("id-1")
	if !ok {
		t.Fatal("Identity not found after registration")
	}
	if id.Kind != "service" {
		t.Errorf("Expected kind service, got %s", id.Kind)
	}
	if id.TokenHash != "hash123" {
		t.Errorf("Expected token hash hash123, got %s", id.TokenHash)
	}

	// Verify lookup by hash
	idByHash, ok := proj.GetByTokenHash("hash123")
	if !ok {
		t.Fatal("Identity not found by hash")
	}
	if idByHash.ID != "id-1" {
		t.Errorf("Expected ID id-1, got %s", idByHash.ID)
	}

	// 2. Test Deletion
	delEvent := store.Event{
		EventID:    "evt-2",
		EventType:  store.EventTypeIdentityDeleted,
		Dimensions: store.EventDimensions{IdentityID: "id-1"},
	}

	if err := proj.Apply(delEvent); err != nil {
		t.Fatalf("Failed to apply deletion: %v", err)
	}

	if _, ok := proj.Get("id-1"); ok {
		t.Error("Identity still exists after deletion")
	}
	if _, ok := proj.GetByTokenHash("hash123"); ok {
		t.Error("Identity still found by hash after deletion")
	}
}

func TestIdentityProjection_Replay(t *testing.T) {
	proj := NewIdentityProjection()

	events := []*store.Event{}
	for i := 0; i < 3; i++ {
		id := string(rune('a' + i)) // a, b, c
		payload, _ := json.Marshal(map[string]interface{}{
			"kind": "user",
		})
		events = append(events, &store.Event{
			EventID:    store.EventID("evt-" + id),
			EventType:  store.EventTypeIdentityRegistered,
			Payload:    payload,
			Dimensions: store.EventDimensions{IdentityID: id},
		})
	}

	if err := proj.Replay(events); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	all := proj.GetAll()
	if len(all) != 3 {
		t.Errorf("Expected 3 identities, got %d", len(all))
	}
}

func TestIdentityProjection_State(t *testing.T) {
	proj := NewIdentityProjection()

	// Setup initial state via LoadState
	identities := []Identity{
		{ID: "id-load", Kind: "loaded", TokenHash: "loaded-hash"},
	}
	now := time.Now()
	proj.LoadState("last-evt", now, identities)

	// Verify loaded state
	lastEvt, lastTime, list := proj.GetState()
	if lastEvt != "last-evt" {
		t.Errorf("Expected last event last-evt, got %s", lastEvt)
	}
	if !lastTime.Equal(now) {
		t.Errorf("Time mismatch")
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 identity, got %d", len(list))
	}

	// Verify token map rebuild
	if _, ok := proj.GetByTokenHash("loaded-hash"); !ok {
		t.Error("Token map not rebuilt from LoadState")
	}
}
