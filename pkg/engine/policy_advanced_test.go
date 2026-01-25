package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyShapeAndDefer(t *testing.T) {
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	// Define policies with shape and defer actions
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "shaping_policy",
				Scope: "global",
				Type:  "soft",
				Rules: []RuleDefinition{
					{
						Name:      "defer_traffic",
						Condition: "remaining < 10",
						Action:    "defer",
					},
					{
						Name:      "shape_traffic",
						Condition: "remaining < 50",
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

	// 1. Test Shape
	// Set remaining = 40 (triggers shape, but not defer)
	// used = 60, remaining = 40
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":60,"remaining":40}`),
	})

	intentShape := Intent{
		IntentID:   "intent_shape",
		ProviderID: "p1",
		PoolID:     "pool1",
	}

	resultShape := engine.Evaluate(intentShape)

	if resultShape.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected DecisionApproveWithModifications for shape, got %s", resultShape.Decision)
	}
	if resultShape.Reason != "policy:shaping_applied" {
		t.Errorf("Expected reason 'policy:shaping_applied', got '%s'", resultShape.Reason)
	}

	waitVal, ok := resultShape.Modifications["wait_seconds"]
	if !ok {
		t.Errorf("Expected wait_seconds modification")
	} else {
		wait, ok := waitVal.(float64)
		if !ok || wait != 2.5 {
			t.Errorf("Expected wait_seconds 2.5, got %v", waitVal)
		}
	}

	// 2. Test Defer
	// Set remaining = 5 (triggers defer)
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":95,"remaining":5}`),
	})

	// Set ResetAt to 10 seconds from now
	resetTime := time.Now().Add(10 * time.Second)
	resetPayload, _ := json.Marshal(map[string]interface{}{
		"provider_id": "p1",
		"pool_id":     "pool1",
		"reset_at":    resetTime,
	})

	usage.Apply(store.Event{
		EventType: store.EventTypeResetObserved,
		Payload:   resetPayload,
	})

	intentDefer := Intent{
		IntentID:   "intent_defer",
		ProviderID: "p1",
		PoolID:     "pool1",
	}

	resultDefer := engine.Evaluate(intentDefer)

	if resultDefer.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected DecisionApproveWithModifications for defer, got %s", resultDefer.Decision)
	}
	if resultDefer.Reason != "policy:deferred_until_reset" {
		t.Errorf("Expected reason 'policy:deferred_until_reset', got '%s'", resultDefer.Reason)
	}

	waitValDefer, ok := resultDefer.Modifications["wait_seconds"]
	if !ok {
		t.Errorf("Expected wait_seconds modification for defer")
	} else {
		wait, ok := waitValDefer.(float64)
		// Check range since time.Now() moves
		if !ok || wait < 9.0 || wait > 10.0 {
			t.Errorf("Expected wait_seconds ~10.0, got %v", waitValDefer)
		}
	}
}
