package engine

import (
	"os"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/graph"
)

func TestPolicyEngine_Legacy(t *testing.T) {
	// Setup with NO policies loaded
	usage := NewUsageProjection()
	usage.LoadState("evt-1", time.Now(), []PoolState{
		{
			ProviderID: "prov-1",
			PoolID:     "pool-1",
			Remaining:  50,
			Used:       50,
		},
	})

	pe := NewPolicyEngine(usage, graph.NewProjection())
	// Note: We don't call UpdatePolicies, so policies is nil -> evaluateLegacy used

	// 1. Test Default Allow (Provider/Pool not set or not found)
	res := pe.Evaluate(Intent{
		IdentityID:   "user-1",
		ScopeID:      "global",
		ExpectedCost: 10,
	})
	if res.Decision != DecisionApprove {
		t.Errorf("Expected DefaultApprove, got %s", res.Decision)
	}

	// 2. Test Hard Limit Pass
	res = pe.Evaluate(Intent{
		ProviderID:   "prov-1",
		PoolID:       "pool-1",
		ExpectedCost: 10,
	})
	if res.Decision != DecisionApprove {
		t.Errorf("Expected Approve, got %s", res.Decision)
	}

	// 3. Test Hard Limit Fail
	res = pe.Evaluate(Intent{
		ProviderID:   "prov-1",
		PoolID:       "pool-1",
		ExpectedCost: 60, // Remaining is 50
	})
	if res.Decision != DecisionDenyWithReason {
		t.Errorf("Expected DenyWithReason, got %s", res.Decision)
	}
}

func TestLoadPolicyConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "policy*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	configJSON := `{
		"policies": [
			{
				"id": "pol-1",
				"scope": "global",
				"rules": []
			}
		]
	}`
	tmpFile.Write([]byte(configJSON))
	tmpFile.Close()

	config, err := LoadPolicyConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadPolicyConfig failed: %v", err)
	}
	if len(config.Policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(config.Policies))
	}
	if config.Policies[0].ID != "pol-1" {
		t.Errorf("Expected ID pol-1, got %s", config.Policies[0].ID)
	}
}
