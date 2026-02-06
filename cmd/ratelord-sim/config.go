package main

import (
	"time"
)

type Scenario struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Duration    time.Duration   `json:"duration" yaml:"duration"`
	Seed        int64           `json:"seed" yaml:"seed"` // Deterministic seed
	Topology    Topology        `json:"topology" yaml:"topology"`
	Agents      []AgentConfig   `json:"agents" yaml:"agents"`
	Sabotage    *SabotageConfig `json:"sabotage,omitempty" yaml:"sabotage,omitempty"`
}

type Topology struct {
	Identities []string `json:"identities" yaml:"identities"`
	Pools      []string `json:"pools" yaml:"pools"`
}

type AgentConfig struct {
	Name       string        `json:"name" yaml:"name"`
	Count      int           `json:"count" yaml:"count"`
	IdentityID string        `json:"identity_id" yaml:"identity_id"`
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
