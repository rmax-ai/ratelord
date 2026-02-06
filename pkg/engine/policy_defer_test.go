package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyEngine_DeferAction(t *testing.T) {
	// 1. Setup UsageProjection
	usage := NewUsageProjection()

	// 2. Set ResetAt to 10 seconds in the future
	futureReset := time.Now().Add(10 * time.Second)
	resetPayload, _ := json.Marshal(map[string]interface{}{
		"provider_id": "test-provider",
		"pool_id":     "test-pool",
		"reset_at":    futureReset,
	})

	err := usage.Apply(store.Event{
		EventType: store.EventTypeResetObserved,
		Payload:   resetPayload,
		TsIngest:  time.Now(),
	})
	if err != nil {
		t.Fatalf("Failed to apply reset event: %v", err)
	}

	// 3. Setup PolicyEngine with a "defer" policy
	engine := NewPolicyEngine(usage, nil)

	// Create a policy that defers if remaining < 100 (we'll simulate low remaining)
	// First set low remaining
	usagePayload, _ := json.Marshal(map[string]interface{}{
		"provider_id": "test-provider",
		"pool_id":     "test-pool",
		"used":        950,
		"remaining":   50,
	})
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   usagePayload,
		TsIngest:  time.Now(),
	})

	policyConfig := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID: "defer-policy",
				Rules: []RuleDefinition{
					{
						Condition: "remaining < 100",
						Action:    "defer",
						Params: map[string]interface{}{
							"jitter_max_seconds": 0.0, // Disable jitter for precise check
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(policyConfig)

	// 4. Evaluate Intent
	intent := Intent{
		IntentID:     "intent-1",
		ProviderID:   "test-provider",
		PoolID:       "test-pool",
		ExpectedCost: 1,
	}

	result := engine.Evaluate(intent)

	// 5. Verify Result
	if result.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected decision %s, got %s", DecisionApproveWithModifications, result.Decision)
	}

	if result.Reason != "policy:deferred_until_reset" {
		t.Errorf("Expected reason 'policy:deferred_until_reset', got '%s'", result.Reason)
	}

	waitSeconds, ok := result.Modifications["wait_seconds"].(float64)
	if !ok {
		t.Fatalf("Modifications['wait_seconds'] missing or not float64")
	}

	// Expect waitSeconds to be around 10s (allow some epsilon for execution time)
	if waitSeconds < 9.0 || waitSeconds > 11.0 {
		t.Errorf("Expected wait_seconds around 10.0, got %f", waitSeconds)
	}
}

func TestPolicyEngine_DeferAction_NoResetAt(t *testing.T) {
	// Test behavior when ResetAt is missing (should default to 0 wait?)
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage, nil)

	// Set usage to trigger condition, but NO reset time
	usagePayload, _ := json.Marshal(map[string]interface{}{
		"provider_id": "test-provider",
		"pool_id":     "test-pool",
		"used":        950,
		"remaining":   50,
	})
	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   usagePayload,
		TsIngest:  time.Now(),
	})

	policyConfig := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID: "defer-policy",
				Rules: []RuleDefinition{
					{
						Condition: "remaining < 100",
						Action:    "defer",
					},
				},
			},
		},
	}
	engine.UpdatePolicies(policyConfig)

	intent := Intent{
		ProviderID:   "test-provider",
		PoolID:       "test-pool",
		ExpectedCost: 1,
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApproveWithModifications {
		t.Errorf("Expected decision %s, got %s", DecisionApproveWithModifications, result.Decision)
	}

	waitSeconds, ok := result.Modifications["wait_seconds"].(float64)
	if !ok {
		t.Fatalf("Modifications['wait_seconds'] missing")
	}

	// Without ResetAt, wait should be roughly 0 (plus maybe jitter)
	// The current implementation returns 0 if ResetAt is zero.
	// Jitter is added on top. Default jitter is 0-1.0s.
	if waitSeconds > 2.0 {
		t.Errorf("Expected small wait_seconds (jitter only), got %f", waitSeconds)
	}
}
