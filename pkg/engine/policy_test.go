package engine

import (
	"testing"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyWithLimit(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	// Simulate pool state: Used=95, Remaining=5 (but since Limit=100, remaining=100-95=5)
	// Note: In checkCondition, if limit > 0, remaining = limit - poolState.Used
	// So we need to set poolState.Used = 95

	// Create engine
	engine := NewPolicyEngine(usage, nil)

	// Create policy config
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

	// Update policies
	engine.UpdatePolicies(config)

	// Create intent
	intent := Intent{
		IntentID:     "test_intent",
		IdentityID:   "user1",
		WorkloadID:   "workload1",
		ScopeID:      "global",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	// Set up pool state manually (since we can't easily inject events here)
	// We need to access the private pools map, but since it's private, we'll use a different approach
	// Actually, for testing, we can create a mock or use the Apply method, but to keep it simple,
	// let's modify the test to set the pool state directly by accessing the map via reflection or by making it accessible.
	// Wait, better: since checkCondition fetches from usage.GetPoolState, and we have control over usage,
	// but GetPoolState returns zero values if not set. So we need to set it.

	// To set the pool state, we can use the Apply method with a fake event.
	// But to make it simple, let's add a method to set pool state for testing, but since we can't modify the code,
	// let's use reflection or just test the checkCondition directly.

	// Actually, let's test the full Evaluate by setting up the usage projection properly.
	// We can call Apply with a usage event.

	// Create a fake usage event
	event := store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":95,"remaining":5}`),
	}
	usage.Apply(event)

	// Now evaluate
	result := engine.Evaluate(intent)

	// Check result
	if result.Decision != DecisionDenyWithReason {
		t.Errorf("Expected DecisionDenyWithReason, got %s", result.Decision)
	}
	if result.Reason != "insufficient remaining budget" {
		t.Errorf("Expected reason 'insufficient remaining budget', got '%s'", result.Reason)
	}
}

func TestMalformedCondition(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage, nil)

	// Create policy config with malformed condition
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "test_policy",
				Scope: "global",
				Type:  "hard",
				Limit: 100,
				Rules: []RuleDefinition{
					{
						Name:      "malformed",
						Condition: "invalid syntax",
						Action:    "deny",
					},
				},
			},
		},
	}

	// Update policies
	engine.UpdatePolicies(config)

	// Create intent
	intent := Intent{
		IntentID:     "test_intent",
		IdentityID:   "user1",
		WorkloadID:   "workload1",
		ScopeID:      "global",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	// Set up pool state
	event := store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":50,"remaining":50}`),
	}
	usage.Apply(event)

	// Evaluate
	result := engine.Evaluate(intent)

	// Since condition is malformed, checkCondition returns false, so rule doesn't match, default allow
	if result.Decision != DecisionApprove {
		t.Errorf("Expected DecisionApprove for malformed condition, got %s", result.Decision)
	}
	if result.Reason != "policy:default_allow" {
		t.Errorf("Expected reason 'policy:default_allow', got '%s'", result.Reason)
	}
}
