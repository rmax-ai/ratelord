package engine

import (
	"testing"

	"github.com/rmax-ai/ratelord/pkg/graph"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyDebugMode(t *testing.T) {
	usage := NewUsageProjection()
	graphProj := graph.NewProjection()
	engine := NewPolicyEngine(usage, graphProj)

	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "test_policy",
				Scope: "global",
				Type:  "hard",
				Limit: 100,
				Rules: []RuleDefinition{
					{
						Name:      "low_remaining",
						Condition: "remaining < 10",
						Action:    "deny",
						Params: map[string]interface{}{
							"reason": "insufficient remaining budget",
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Set up pool state
	event := store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":95,"remaining":5}`),
	}
	usage.Apply(event)

	intent := Intent{
		IntentID:     "test_intent_debug",
		IdentityID:   "user1",
		WorkloadID:   "workload1",
		ScopeID:      "global",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
		Debug:        true, // Enable debug mode
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionDenyWithReason {
		t.Errorf("Expected DecisionDenyWithReason, got %s", result.Decision)
	}
}
