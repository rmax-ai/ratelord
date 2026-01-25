package engine

import (
	"fmt"

	"github.com/rmax/ratelord/pkg/store"
)

// Decision defines the outcome of an intent evaluation
type Decision string

const (
	DecisionApprove                  Decision = "approve"
	DecisionApproveWithModifications Decision = "approve_with_modifications"
	DecisionDenyWithReason           Decision = "deny_with_reason"
)

// Intent represents the request to perform an action
type Intent struct {
	IntentID     string
	IdentityID   string
	WorkloadID   string
	ScopeID      string
	ProviderID   string // Target provider (optional/inferred)
	PoolID       string // Target pool (optional/inferred)
	ExpectedCost int64  // Estimated consumption
}

// PolicyEvaluationResult captures the output of the policy engine
type PolicyEvaluationResult struct {
	Decision      Decision          `json:"decision"`
	Reason        string            `json:"reason"`
	Modifications map[string]string `json:"modifications,omitempty"`
}

// PolicyEngine is responsible for arbitrating intents
type PolicyEngine struct {
	usage *UsageProjection
}

// NewPolicyEngine creates a new policy engine instance
func NewPolicyEngine(usage *UsageProjection) *PolicyEngine {
	return &PolicyEngine{
		usage: usage,
	}
}

// Evaluate checks an intent against current policies and usage
func (pe *PolicyEngine) Evaluate(intent Intent) PolicyEvaluationResult {
	// 1. Check Hard Limits (Basic Arithmetic first)
	// If we know the pool, check if we have budget.
	if intent.ProviderID != "" && intent.PoolID != "" {
		poolState, exists := pe.usage.GetPoolState(intent.ProviderID, intent.PoolID)
		if exists {
			// Basic check: do we have enough remaining?
			// Note: This is a simplistic check. Real policy would use forecasts.
			// But M5.2 focuses on wiring.
			if poolState.Remaining < intent.ExpectedCost {
				return PolicyEvaluationResult{
					Decision: DecisionDenyWithReason,
					Reason:   fmt.Sprintf("insufficient_budget: remaining %d < cost %d", poolState.Remaining, intent.ExpectedCost),
				}
			}
		}
	}

	// 2. Default Approval (Soft Rule)
	// If no hard limit is hit, we approve.
	// In the future, this is where "Yellow Zone" logic would apply shaping.
	return PolicyEvaluationResult{
		Decision: DecisionApprove,
		Reason:   "policy:default_allow",
	}
}

// ConvertToEvent converts an evaluation result into an event payload
func (pe *PolicyEngine) ConvertToEvent(intent Intent, result PolicyEvaluationResult) store.Event {
	// This helper constructs the decision event.
	// Implementation deferred to the API handler usually, but good to have helper here.
	// For now, we return a partial event or just let the caller handle it.
	// Returning empty event as placeholder.
	return store.Event{}
}
