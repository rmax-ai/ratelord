package engine

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/rmax-ai/ratelord/pkg/store"
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
	Decision      Decision               `json:"decision"`
	Reason        string                 `json:"reason"`
	Modifications map[string]interface{} `json:"modifications,omitempty"`
}

// PolicyEngine is responsible for arbitrating intents
type PolicyEngine struct {
	usage    *UsageProjection
	mu       sync.RWMutex
	policies *PolicyConfig
}

// NewPolicyEngine creates a new policy engine instance
func NewPolicyEngine(usage *UsageProjection) *PolicyEngine {
	return &PolicyEngine{
		usage: usage,
	}
}

// UpdatePolicies safely hot-swaps the current policies
func (pe *PolicyEngine) UpdatePolicies(newConfig *PolicyConfig) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.policies = newConfig
}

// Evaluate checks an intent against current policies and usage
func (pe *PolicyEngine) Evaluate(intent Intent) PolicyEvaluationResult {
	pe.mu.RLock()
	activePolicies := pe.policies
	pe.mu.RUnlock()

	// Fallback if no policy loaded (or for bootstrapping)
	if activePolicies == nil {
		return pe.evaluateLegacy(intent)
	}

	return pe.evaluateDynamic(intent, activePolicies)
}

func (pe *PolicyEngine) evaluateDynamic(intent Intent, config *PolicyConfig) PolicyEvaluationResult {
	// Simple linear scan of policies -> rules
	// In a real implementation, we'd have a compiled rule tree.

	// Fetch pool state once for the intent context
	var poolState PoolState
	var exists bool
	if intent.ProviderID != "" && intent.PoolID != "" {
		poolState, exists = pe.usage.GetPoolState(intent.ProviderID, intent.PoolID)
	}

	for _, policy := range config.Policies {
		// TODO: Match Scope (e.g. wildcard or exact match)
		// For now, assume "global" or match

		for _, rule := range policy.Rules {
			if pe.checkCondition(rule.Condition, intent, policy.Limit, poolState, exists) {
				return pe.applyAction(rule.Action, rule.Params, poolState)
			}
		}
	}

	// Default Allow
	return PolicyEvaluationResult{
		Decision: DecisionApprove,
		Reason:   "policy:default_allow",
	}
}

func (pe *PolicyEngine) checkCondition(cond string, intent Intent, limit int64, poolState PoolState, exists bool) bool {
	// Very basic DSL parser for M9.3
	// Supported: "remaining < X"

	if intent.ProviderID == "" || intent.PoolID == "" {
		return false
	}

	if !exists {
		return false
	}

	var remaining int64
	if limit > 0 {
		remaining = limit - poolState.Used
	} else {
		remaining = poolState.Remaining
	}

	var threshold int64
	// Try to parse "remaining < 100"
	if n, err := fmt.Sscanf(cond, "remaining < %d", &threshold); err == nil && n == 1 {
		return remaining < threshold
	}

	return false
}

func (pe *PolicyEngine) calculateWaitTime(providerID, poolID string) float64 {
	return pe.usage.CalculateWaitTime(providerID, poolID)
}

func (pe *PolicyEngine) applyAction(action string, params map[string]interface{}, poolState PoolState) PolicyEvaluationResult {
	switch action {
	case "deny":
		reason := "policy:rule_matched"
		if r, ok := params["reason"].(string); ok {
			reason = r
		}
		return PolicyEvaluationResult{
			Decision: DecisionDenyWithReason,
			Reason:   reason,
		}

	case "shape":
		var wait float64
		// If "wait_seconds" is explicitly provided
		if w, ok := params["wait_seconds"].(float64); ok {
			wait = w
		} else if wInt, ok := params["wait_seconds"].(int); ok {
			wait = float64(wInt)
		}

		// TODO: Implement algorithms like linear backoff if params["algorithm"] is set

		return PolicyEvaluationResult{
			Decision: DecisionApproveWithModifications,
			Modifications: map[string]interface{}{
				"wait_seconds": wait,
			},
			Reason: "policy:shaping_applied",
		}

	case "defer":
		// Wait until reset + jitter
		wait := pe.calculateWaitTime(poolState.ProviderID, poolState.PoolID)

		// Add jitter to avoid thundering herd (default 100ms - 1s)
		jitterMax := 1.0
		if j, ok := params["jitter_max_seconds"].(float64); ok {
			jitterMax = j
		}
		wait += rand.Float64() * jitterMax

		return PolicyEvaluationResult{
			Decision: DecisionApproveWithModifications,
			Modifications: map[string]interface{}{
				"wait_seconds": wait,
			},
			Reason: "policy:deferred_until_reset",
		}
	}

	return PolicyEvaluationResult{
		Decision: DecisionApprove,
		Reason:   "policy:rule_passed",
	}
}

// evaluateLegacy preserves the M5.2 hardcoded logic
func (pe *PolicyEngine) evaluateLegacy(intent Intent) PolicyEvaluationResult {
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
