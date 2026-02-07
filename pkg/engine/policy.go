package engine

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/graph"
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
	Warnings      []string               `json:"warnings,omitempty"`
	Trace         []RuleTrace            `json:"trace,omitempty"`
}

// RuleTrace provides explainability for each rule evaluation
type RuleTrace struct {
	PolicyID  string `json:"policy_id"`
	RuleIndex int    `json:"rule_index"`
	Condition string `json:"condition"`
	Result    bool   `json:"result"`
	Reason    string `json:"reason,omitempty"` // e.g. "passed", "failed: remaining < 100"
}

// PolicyEngine is responsible for arbitrating intents
type PolicyEngine struct {
	usage      *UsageProjection
	mu         sync.RWMutex
	policies   *PolicyConfig
	policyMap  map[string]PolicyDefinition
	controller *DelayController
	graph      *graph.Projection
}

// NewPolicyEngine creates a new policy engine instance
func NewPolicyEngine(usage *UsageProjection, graphProj *graph.Projection) *PolicyEngine {
	return &PolicyEngine{
		usage:      usage,
		controller: NewDelayController(1.0),
		graph:      graphProj,
		policyMap:  make(map[string]PolicyDefinition),
	}
}

// UpdatePolicies safely hot-swaps the current policies
func (pe *PolicyEngine) UpdatePolicies(newConfig *PolicyConfig) {
	pe.mu.Lock()
	pe.policies = newConfig
	// Rebuild map as a new object (COW)
	newMap := make(map[string]PolicyDefinition)
	if newConfig != nil {
		for _, p := range newConfig.Policies {
			newMap[p.ID] = p
		}
	}
	pe.policyMap = newMap
	pe.mu.Unlock()

	// Sync with graph outside the lock to avoid potential deadlocks if graph methods lock
	// Although here pe.graph is concurrent safe.
	pe.syncGraph(newConfig)
}

func (pe *PolicyEngine) syncGraph(config *PolicyConfig) {
	if pe.graph == nil {
		return
	}
	for _, policy := range config.Policies {
		props := make(map[string]string)
		props["type"] = policy.Type
		props["limit"] = fmt.Sprintf("%d", policy.Limit)
		// Use policy ID as Constraint ID
		pe.graph.AddConstraint(policy.ID, policy.Scope, props)
	}
}

// Evaluate checks an intent against current policies and usage
func (pe *PolicyEngine) Evaluate(intent Intent) PolicyEvaluationResult {
	pe.mu.RLock()
	activePolicies := pe.policies
	activeMap := pe.policyMap
	pe.mu.RUnlock()

	// Fallback if no policy loaded (or for bootstrapping)
	if activePolicies == nil {
		return pe.evaluateLegacy(intent)
	}

	return pe.evaluateDynamic(intent, activeMap)
}

func (pe *PolicyEngine) evaluateDynamic(intent Intent, policyMap map[string]PolicyDefinition) PolicyEvaluationResult {
	// Identify relevant policies via Graph
	var policiesToEvaluate []PolicyDefinition

	// Helper to append policies for a scope
	addPoliciesForScope := func(scopeID string) {
		nodes, err := pe.graph.FindConstraintsForScope(scopeID)
		if err != nil {
			return
		}
		for _, n := range nodes {
			if p, ok := policyMap[n.ID]; ok {
				policiesToEvaluate = append(policiesToEvaluate, p)
			}
		}
	}

	// 1. Specific Scope
	if intent.ScopeID != "" {
		addPoliciesForScope(intent.ScopeID)
	}

	// 2. Global Scope (if different)
	if intent.ScopeID != "global" {
		addPoliciesForScope("global")
	}

	// Fetch pool state once for the intent context
	var poolState PoolState
	var exists bool
	if intent.ProviderID != "" && intent.PoolID != "" {
		poolState, exists = pe.usage.GetPoolState(intent.ProviderID, intent.PoolID)
	}

	var trace []RuleTrace
	ruleIndex := 0

	for _, policy := range policiesToEvaluate {
		for _, rule := range policy.Rules {
			// Check TimeWindow if present
			if rule.TimeWindow != nil {
				match, err := rule.TimeWindow.Matches(time.Now())
				if err != nil {
					// TODO: Log warning about invalid time window
					continue
				}
				if !match {
					continue
				}
			}

			result, reason := pe.checkCondition(rule.Condition, intent, policy.Limit, poolState, exists)
			trace = append(trace, RuleTrace{
				PolicyID:  policy.ID,
				RuleIndex: ruleIndex,
				Condition: rule.Condition,
				Result:    result,
				Reason:    reason,
			})
			ruleIndex++

			if result {
				return pe.applyAction(rule.Action, rule.Params, poolState, trace)
			}
		}
	}

	// Default Allow
	return PolicyEvaluationResult{
		Decision: DecisionApprove,
		Reason:   "policy:default_allow",
		Trace:    trace,
	}
}

func (pe *PolicyEngine) checkCondition(cond string, intent Intent, limit int64, poolState PoolState, exists bool) (bool, string) {
	// Very basic DSL parser for M9.3 & M29.3
	// Supported:
	// - "remaining < X"
	// - "cost > X"
	// - "forecast_tte < X"
	// - "provider_id == X"

	// 1. Check provider_id (independent of pool state)
	var pid string
	if n, err := fmt.Sscanf(cond, "provider_id == %q", &pid); err == nil && n == 1 {
		if intent.ProviderID == pid {
			return true, "passed: provider_id matches"
		}
		return false, "failed: provider_id does not match"
	}
	// Also try without quotes just in case
	if n, err := fmt.Sscanf(cond, "provider_id == %s", &pid); err == nil && n == 1 {
		if intent.ProviderID == pid {
			return true, "passed: provider_id matches"
		}
		return false, "failed: provider_id does not match"
	}

	if intent.ProviderID == "" || intent.PoolID == "" {
		return false, "failed: provider_id or pool_id not set"
	}

	if !exists {
		return false, "failed: pool state not found"
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
		if remaining < threshold {
			return true, fmt.Sprintf("passed: remaining %d < %d", remaining, threshold)
		}
		return false, fmt.Sprintf("failed: remaining %d >= %d", remaining, threshold)
	}

	// Try to parse "cost > 5000000" (MicroUSD)
	var costThreshold int64
	if n, err := fmt.Sscanf(cond, "cost > %d", &costThreshold); err == nil && n == 1 {
		if poolState.Cost > currency.MicroUSD(costThreshold) {
			return true, fmt.Sprintf("passed: cost %d > %d", poolState.Cost, costThreshold)
		}
		return false, fmt.Sprintf("failed: cost %d <= %d", poolState.Cost, costThreshold)
	}

	// Try to parse "forecast_tte < 3600" (seconds)
	var tteThreshold float64
	if n, err := fmt.Sscanf(cond, "forecast_tte < %f", &tteThreshold); err == nil && n == 1 {
		if poolState.LatestForecast != nil {
			if float64(poolState.LatestForecast.TTE.P99Seconds) < tteThreshold {
				return true, fmt.Sprintf("passed: forecast_tte %.2f < %.2f", float64(poolState.LatestForecast.TTE.P99Seconds), tteThreshold)
			}
			return false, fmt.Sprintf("failed: forecast_tte %.2f >= %.2f", float64(poolState.LatestForecast.TTE.P99Seconds), tteThreshold)
		}
		return false, "failed: no forecast available"
	}

	return false, "failed: unrecognized condition"
}

func (pe *PolicyEngine) calculateWaitTime(providerID, poolID string) float64 {
	return pe.usage.CalculateWaitTime(providerID, poolID)
}

func (pe *PolicyEngine) applyAction(action string, params map[string]interface{}, poolState PoolState, trace []RuleTrace) PolicyEvaluationResult {
	switch action {
	case "deny":
		reason := "policy:rule_matched"
		if r, ok := params["reason"].(string); ok {
			reason = r
		}
		return PolicyEvaluationResult{
			Decision: DecisionDenyWithReason,
			Reason:   reason,
			Trace:    trace,
		}

	case "warn":
		msg := "policy:warning"
		if m, ok := params["message"].(string); ok {
			msg = m
		}
		return PolicyEvaluationResult{
			Decision: DecisionApprove,
			Reason:   "policy:rule_passed_with_warning",
			Warnings: []string{msg},
			Trace:    trace,
		}

	case "shape", "delay":
		var wait float64
		var kp float64
		// Check for kp parameter
		if k, ok := params["kp"].(float64); ok {
			kp = k
		} else if kInt, ok := params["kp"].(int); ok {
			kp = float64(kInt)
		}
		// Check if algorithm is "dynamic"
		if alg, ok := params["algorithm"].(string); ok && alg == "dynamic" {
			wait = pe.controller.CalculateWait(poolState, time.Now(), kp).Seconds()
		} else {
			// If "wait_seconds" is explicitly provided
			if w, ok := params["wait_seconds"].(float64); ok {
				wait = w
			} else if wInt, ok := params["wait_seconds"].(int); ok {
				wait = float64(wInt)
			}
		}

		return PolicyEvaluationResult{
			Decision: DecisionApproveWithModifications,
			Modifications: map[string]interface{}{
				"wait_seconds": wait,
			},
			Reason: "policy:shaping_applied",
			Trace:  trace,
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
			Trace:  trace,
		}
	}

	return PolicyEvaluationResult{
		Decision: DecisionApprove,
		Reason:   "policy:rule_passed",
		Trace:    trace,
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
