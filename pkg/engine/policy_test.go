package engine

import (
	"testing"

	"github.com/rmax-ai/ratelord/pkg/graph"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyWithLimit(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	// Simulate pool state: Used=95, Remaining=5 (but since Limit=100, remaining=100-95=5)
	// Note: In checkCondition, if limit > 0, remaining = limit - poolState.Used
	// So we need to set poolState.Used = 95

	// Create graph projection
	graphProj := graph.NewProjection()

	// Create engine
	engine := NewPolicyEngine(usage, graphProj)

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
	graphProj := graph.NewProjection()
	engine := NewPolicyEngine(usage, graphProj)

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

func TestPolicyScopePrecedence(t *testing.T) {
	usage := NewUsageProjection()
	graphProj := graph.NewProjection()
	engine := NewPolicyEngine(usage, graphProj)

	// Config with Global Allow but Specific Deny
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "global_policy",
				Scope: "global",
				Type:  "soft",
				Rules: []RuleDefinition{
					{
						Name:      "always_allow",
						Condition: "remaining < 200", // True
						Action:    "approve",
					},
				},
			},
			{
				ID:    "specific_policy",
				Scope: "tenant_A",
				Type:  "hard",
				Rules: []RuleDefinition{
					{
						Name:      "always_deny",
						Condition: "remaining < 200", // True
						Action:    "deny",
						Params: map[string]interface{}{
							"reason": "tenant denied",
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Mock Usage
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   []byte(`{"provider_id":"p1","pool_id":"pool1","used":0,"remaining":100}`),
	})

	// Intent for Tenant A (Should Deny)
	intentA := Intent{
		ScopeID:      "tenant_A",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}
	resA := engine.Evaluate(intentA)
	if resA.Decision != DecisionDenyWithReason {
		t.Errorf("Expected Deny for tenant_A, got %s", resA.Decision)
	}

	// Intent for Tenant B (Should fall back to Global -> Approve)
	intentB := Intent{
		ScopeID:      "tenant_B",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}
	resB := engine.Evaluate(intentB)
	if resB.Decision != DecisionApprove {
		t.Errorf("Expected Approve for tenant_B, got %s", resB.Decision)
	}
}
