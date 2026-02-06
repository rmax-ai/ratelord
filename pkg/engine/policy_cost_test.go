package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestPolicyCostCondition(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	// Create policy config with cost rule
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "cost_policy",
				Scope: "global",
				Rules: []RuleDefinition{
					{
						Name:      "high_cost",
						Condition: "cost > 5000000", // > 5 USD
						Action:    "deny",
						Params: map[string]interface{}{
							"reason": "cost limit exceeded",
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Setup usage state with high cost
	// cost = 6000000 (6 USD)
	payload := struct {
		ProviderID string            `json:"provider_id"`
		PoolID     string            `json:"pool_id"`
		Used       int64             `json:"used"`
		Remaining  int64             `json:"remaining"`
		Cost       currency.MicroUSD `json:"cost"`
	}{
		ProviderID: "p1",
		PoolID:     "pool1",
		Used:       10,
		Remaining:  90,
		Cost:       6000000,
	}
	payloadBytes, _ := json.Marshal(payload)

	usage.Apply(store.Event{
		EventType: store.EventTypeUsageObserved,
		Payload:   payloadBytes,
		TsIngest:  time.Now(),
	})

	// Evaluate Intent
	intent := Intent{
		IntentID:     "intent1",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionDenyWithReason {
		t.Errorf("Expected Deny, got %s", result.Decision)
	}
	if result.Reason != "cost limit exceeded" {
		t.Errorf("Expected 'cost limit exceeded', got %s", result.Reason)
	}
}

func TestPolicyForecastTTECondition(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	// Create policy config with TTE rule
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "tte_policy",
				Scope: "global",
				Rules: []RuleDefinition{
					{
						Name:      "low_tte",
						Condition: "forecast_tte < 3600.0", // < 1 hour
						Action:    "warn",
						Params: map[string]interface{}{
							"message": "running out of budget soon",
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Setup forecast state
	// TTE = 1800s (30 mins)
	forecastPayload := struct {
		ProviderID string            `json:"provider_id"`
		PoolID     string            `json:"pool_id"`
		Forecast   forecast.Forecast `json:"forecast"`
	}{
		ProviderID: "p1",
		PoolID:     "pool1",
		Forecast: forecast.Forecast{
			TTE: forecast.TimeToExhaustion{
				P99Seconds: 1800,
			},
		},
	}
	payloadBytes, _ := json.Marshal(forecastPayload)

	usage.Apply(store.Event{
		EventType: store.EventTypeForecastComputed,
		Payload:   payloadBytes,
		TsIngest:  time.Now(),
	})

	// Also need to ensure the pool exists in usage projection (ForecastComputed adds it if missing)
	// Evaluate Intent
	intent := Intent{
		IntentID:     "intent1",
		ProviderID:   "p1",
		PoolID:       "pool1",
		ExpectedCost: 1,
	}

	result := engine.Evaluate(intent)

	if result.Decision != DecisionApprove {
		t.Errorf("Expected Approve (with warning), got %s", result.Decision)
	}
	if len(result.Warnings) == 0 || result.Warnings[0] != "running out of budget soon" {
		t.Errorf("Expected warning 'running out of budget soon', got %v", result.Warnings)
	}
}

func TestPolicyProviderIDCondition(t *testing.T) {
	// Setup usage projection
	usage := NewUsageProjection()
	engine := NewPolicyEngine(usage)

	// Create policy config with provider check
	config := &PolicyConfig{
		Policies: []PolicyDefinition{
			{
				ID:    "provider_policy",
				Scope: "global",
				Rules: []RuleDefinition{
					{
						Name:      "check_provider",
						Condition: "provider_id == \"p_expensive\"",
						Action:    "deny",
						Params: map[string]interface{}{
							"reason": "provider blocked",
						},
					},
				},
			},
		},
	}
	engine.UpdatePolicies(config)

	// Intent for blocked provider
	intent1 := Intent{
		IntentID:   "intent1",
		ProviderID: "p_expensive",
		PoolID:     "pool1",
	}

	result1 := engine.Evaluate(intent1)
	if result1.Decision != DecisionDenyWithReason {
		t.Errorf("Expected Deny for p_expensive, got %s", result1.Decision)
	}

	// Intent for allowed provider
	intent2 := Intent{
		IntentID:   "intent2",
		ProviderID: "p_cheap",
		PoolID:     "pool1",
	}
	result2 := engine.Evaluate(intent2)
	if result2.Decision != DecisionApprove {
		t.Errorf("Expected Approve for p_cheap, got %s", result2.Decision)
	}
}
