package engine

// PolicyConfig represents the top-level structure of policy.json
type PolicyConfig struct {
	Policies  []PolicyDefinition `json:"policies"`
	Providers ProvidersConfig    `json:"providers,omitempty"`
}

// ProvidersConfig holds configuration for various providers
type ProvidersConfig struct {
	GitHub []GitHubConfig `json:"github,omitempty"`
}

// GitHubConfig defines configuration for the GitHub provider
type GitHubConfig struct {
	ID            string `json:"id"`
	TokenEnvVar   string `json:"token_env_var"` // Prefer env var name for security
	EnterpriseURL string `json:"enterprise_url,omitempty"`
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
