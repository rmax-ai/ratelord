package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestProviderProjection(t *testing.T) {
	proj := NewProviderProjection()

	// 1. Test Apply
	stateData := []byte(`{"status": "ok"}`)
	payloadMap := map[string]interface{}{
		"provider_id": "prov-1",
		"state":       string(stateData),
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	event := store.Event{
		EventID:   "evt-1",
		EventType: store.EventTypeProviderPollObserved,
		Payload:   payloadBytes,
		TsEvent:   time.Now(),
	}

	proj.Apply(event)

	// 2. Test GetState
	gotState := proj.GetState(provider.ProviderID("prov-1"))
	if string(gotState) != string(stateData) {
		t.Errorf("Expected state %s, got %s", stateData, gotState)
	}

	// 3. Test GetAllStates
	allStates := proj.GetAllStates()
	if len(allStates) != 1 {
		t.Errorf("Expected 1 state, got %d", len(allStates))
	}
	if string(allStates["prov-1"]) != string(stateData) {
		t.Errorf("Mismatch in GetAllStates")
	}

	// 4. Test Replay
	events := []*store.Event{}
	for i := 0; i < 2; i++ {
		pMap := map[string]interface{}{
			"provider_id": "prov-replay",
			"state":       "replay-state",
		}
		pBytes, _ := json.Marshal(pMap)
		events = append(events, &store.Event{
			EventType: store.EventTypeProviderPollObserved,
			Payload:   pBytes,
		})
	}
	proj.Replay(events)

	if string(proj.GetState("prov-replay")) != "replay-state" {
		t.Error("Replay failed to update state")
	}

	// 5. Test LoadState
	newState := map[string][]byte{
		"prov-load": []byte("loaded"),
	}
	proj.LoadState(newState)
	if string(proj.GetState("prov-load")) != "loaded" {
		t.Error("LoadState failed")
	}
	// Verify previous state is gone (LoadState replaces map)
	if proj.GetState("prov-1") != nil {
		t.Error("LoadState did not replace existing state")
	}
}
