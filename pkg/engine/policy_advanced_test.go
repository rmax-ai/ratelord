package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyShape(t *testing.T) {
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage, nil)

	// Define policy with shape action
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "shape_policy",
				Scope: "global",
				Type:  "soft",
				Rules: []RuleDefinition{
					{
						Name:      "shape_traffic",
						Condition: "remaining < 1000",
						Action:    "shape",
						Params: map[string]interface{}{
							"wait_seconds": 2.5,
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Set usage: 4500 used, 500 remaining (triggers shape since 500 < 1000)
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":4500,"remaining":500}`),
	})

	intent := Intent{
		IntentID:   "intent_shape",
		ProviderID: "p1",
		PoolID:     "pool1",
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected DecisionApproveWithModifications, got %s", result.Decision)
	}

	waitVal, ok := result.Modifications["wait_seconds"]
	if !ok {
		t.Errorf("Expected wait_seconds modification")
	} else {
		wait, ok := waitVal.(float64)
		if !ok || wait != 2.5 {
			t.Errorf("Expected wait_seconds 2.5, got %v", waitVal)
		}
	}
}

func TestPolicyDefer(t *testing.T) {
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage, nil)

	// Define policy with defer action
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "defer_policy",
				Scope: "global",
				Type:  "soft",
				Rules: []RuleDefinition{
					{
						Name:      "defer_traffic",
						Condition: "remaining < 100",
						Action:    "defer",
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Set usage: remaining 50 (triggers defer since 50 < 100)
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":4950,"remaining":50}`),
	})

	// Mock ResetAt to be Now() + 5s
	resetTime := time.Now().Add(5 * time.Second)
	resetPayload, _ := json.Marshal(map[string]interface{}{
		"provider_id": "p1",
		"pool_id":     "pool1",
		"reset_at":    resetTime,
	})

	usage.Apply(store.Event{
		EventType: store.EventTypeResetObserved,
		Payload:   resetPayload,
	})

	intent := Intent{
		IntentID:   "intent_defer",
		ProviderID: "p1",
		PoolID:     "pool1",
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected DecisionApproveWithModifications, got %s", result.Decision)
	}

	waitVal, ok := result.Modifications["wait_seconds"]
	if !ok {
		t.Errorf("Expected wait_seconds modification")
	} else {
		wait, ok := waitVal.(float64)
		if !ok {
			t.Errorf("Expected float64 for wait_seconds, got %T", waitVal)
		} else {
			// Allow small delta for jitter (jitter max 1.0s by default)
			if wait < 4.9 || wait > 6.1 {
				t.Errorf("Expected wait_seconds approx 5.0, got %v", wait)
			}
		}
	}
}
