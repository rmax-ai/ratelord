package engine

// PolicyConfig represents the top-level structure of policy.json
type PolicyConfig struct {
	Policies []PolicyDefinition `json:"policies"`
}

// PolicyDefinition maps a high-level policy block
type PolicyDefinition struct {
	ID    string           `json:"id"`
	Scope string           `json:"scope"` // e.g., "global", "env:dev"
	Type  string           `json:"type"`  // "hard", "soft"
	Limit int64            `json:"limit,omitempty"`
	Rules []RuleDefinition `json:"rules"`
}

// RuleDefinition maps individual logic rules
type RuleDefinition struct {
	Name      string                 `json:"name"`
	Condition string                 `json:"condition"` // Simple DSL: "remaining < 100"
	Action    string                 `json:"action"`    // "approve", "deny", "shape"
	Params    map[string]interface{} `json:"params,omitempty"`
}
