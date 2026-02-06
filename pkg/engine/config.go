package engine

// PolicyConfig represents the top-level structure of policy.json
type PolicyConfig struct {
	Policies  []PolicyDefinition `json:"policies"`
	Providers ProvidersConfig    `json:"providers,omitempty"`
}

// ProvidersConfig holds configuration for various providers
type ProvidersConfig struct {
	GitHub []GitHubConfig `json:"github,omitempty"`
	OpenAI []OpenAIConfig `json:"openai,omitempty"`
}

// GitHubConfig defines configuration for the GitHub provider
type GitHubConfig struct {
	ID            string `json:"id"`
	TokenEnvVar   string `json:"token_env_var"` // Prefer env var name for security
	EnterpriseURL string `json:"enterprise_url,omitempty"`
}

// OpenAIConfig defines configuration for the OpenAI provider
type OpenAIConfig struct {
	ID          string `json:"id"`
	TokenEnvVar string `json:"token_env_var"`
	OrgID       string `json:"org_id,omitempty"` // Optional: for 'OpenAI-Organization' header
	BaseURL     string `json:"base_url,omitempty"`
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
	Name       string                 `json:"name"`
	Condition  string                 `json:"condition"` // Simple DSL: "remaining < 100"
	Action     string                 `json:"action"`    // "approve", "deny", "shape"
	Params     map[string]interface{} `json:"params,omitempty"`
	TimeWindow *TimeWindow            `json:"time_window,omitempty"`
}

// TimeWindow defines a temporal constraint for a rule
type TimeWindow struct {
	StartTime string   `json:"start_time,omitempty"` // HH:MM (24-hour)
	EndTime   string   `json:"end_time,omitempty"`   // HH:MM (24-hour)
	Days      []string `json:"days,omitempty"`       // ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
	Location  string   `json:"location,omitempty"`   // e.g., "America/New_York" (defaults to UTC)
}
