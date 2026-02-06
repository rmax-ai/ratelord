package engine

import (
	"testing"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyWarn(t *testing.T) {
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "test_policy",
				Scope: "global",
				Type:  "soft",
				Limit: 100,
				Rules: []RuleDefinition{
					{
						Name:      "warn_rule",
						Condition: "remaining < 20",
						Action:    "warn",
						Params: map[string]interface{}{
							"message": "Approaching limit",
						},
					},
				},
			},
		},
	}

	engine.UpdatePolicies(config)

	intent := Intent{
		IntentID:     "test_intent",
		IdentityID:   "user1",
		WorkloadID:   "workload1",
		ScopeID:      "global",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	// Set usage so remaining is 10 ( < 20 )
	event := store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":90,"remaining":10}`),
	}
	usage.Apply(event)

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApprove {
		t.Errorf("Expected DecisionApprove, got %s", result.Decision)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0] != "Approaching limit" {
		t.Errorf("Expected warning 'Approaching limit', got '%s'", result.Warnings[0])
	}
}

func TestPolicyDelay(t *testing.T) {
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "test_policy",
				Scope: "global",
				Type:  "soft",
				Limit: 100,
				Rules: []RuleDefinition{
					{
						Name:      "delay_rule",
						Condition: "remaining < 10",
						Action:    "delay",
						Params: map[string]interface{}{
							"wait_seconds": 2.5,
						},
					},
				},
			},
		},
	}

	engine.UpdatePolicies(config)

	intent := Intent{
		IntentID:     "test_intent",
		IdentityID:   "user1",
		WorkloadID:   "workload1",
		ScopeID:      "global",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	// Set usage so remaining is 5 ( < 10 )
	event := store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":95,"remaining":5}`),
	}
	usage.Apply(event)

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected DecisionApproveWithModifications, got %s", result.Decision)
	}
	if wait, ok := result.Modifications["wait_seconds"].(float64); !ok || wait != 2.5 {
		t.Errorf("Expected wait_seconds 2.5, got %v", result.Modifications["wait_seconds"])
	}
}
