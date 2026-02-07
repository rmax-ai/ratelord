package simulation

import (
	"time"
)

// SimulationResult captures the final state of the simulation for reporting
type SimulationResult struct {
	ScenarioName  string                 `json:"scenario_name"`
	Duration      time.Duration          `json:"duration"`
	TotalRequests uint64                 `json:"total_requests"`
	TotalApproved uint64                 `json:"total_approved"`
	TotalDenied   uint64                 `json:"total_denied"`
	TotalModified uint64                 `json:"total_modified"`
	TotalErrors   uint64                 `json:"total_errors"`
	TotalInjected uint64                 `json:"total_injected"`
	AgentStats    map[string]*AgentStats `json:"agent_stats"`
	Invariants    []InvariantResult      `json:"invariants"`
	Success       bool                   `json:"success"`
}

type AgentStats struct {
	Requests uint64 `json:"requests"`
	Approved uint64 `json:"approved"`
	Denied   uint64 `json:"denied"`
	Modified uint64 `json:"modified"`
	Errors   uint64 `json:"errors"`
}

type InvariantResult struct {
	Metric   string `json:"metric"`
	Scope    string `json:"scope"`
	Expected string `json:"expected"` // e.g. "> 0.95"
	Actual   string `json:"actual"`   // e.g. "0.98"
	Passed   bool   `json:"passed"`
}

type Scenario struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Duration    time.Duration   `json:"duration" yaml:"duration"`
	Seed        int64           `json:"seed" yaml:"seed"` // Deterministic seed
	Topology    Topology        `json:"topology" yaml:"topology"`
	Agents      []AgentConfig   `json:"agents" yaml:"agents"`
	Sabotage    *SabotageConfig `json:"sabotage,omitempty" yaml:"sabotage,omitempty"`
	Invariants  []Invariant     `json:"invariants,omitempty" yaml:"invariants,omitempty"`
}

type Invariant struct {
	Metric    string  `json:"metric" yaml:"metric"`       // e.g., "approval_rate", "denial_rate", "latency_p99"
	Condition string  `json:"condition" yaml:"condition"` // e.g., ">", "<", ">=", "<="
	Value     float64 `json:"value" yaml:"value"`
	Scope     string  `json:"scope" yaml:"scope"` // "global" or specific agent name
}

type Topology struct {
	Identities []string `json:"identities" yaml:"identities"`
	Pools      []string `json:"pools" yaml:"pools"`
}

type AgentConfig struct {
	Name       string        `json:"name" yaml:"name"`
	Count      int           `json:"count" yaml:"count"`
	IdentityID string        `json:"identity_id" yaml:"identity_id"`
	ScopeID    string        `json:"scope_id" yaml:"scope_id"` // Target Scope (default: "default")
	Priority   string        `json:"priority" yaml:"priority"` // low, normal, high, critical
	Behavior   BehaviorType  `json:"behavior" yaml:"behavior"`
	Rate       int           `json:"rate" yaml:"rate"` // Requests per second
	Burst      int           `json:"burst" yaml:"burst"`
	Jitter     time.Duration `json:"jitter" yaml:"jitter"`
}

type BehaviorType string

const (
	BehaviorPeriodic BehaviorType = "periodic"
	BehaviorGreedy   BehaviorType = "greedy"
	BehaviorPoisson  BehaviorType = "poisson"
	BehaviorBursty   BehaviorType = "bursty"
)

type SabotageConfig struct {
	Enabled  bool          `json:"enabled" yaml:"enabled"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Amount   int64         `json:"amount" yaml:"amount"`
	Target   string        `json:"target" yaml:"target"` // "provider_id/pool_id"
}
