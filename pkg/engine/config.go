package engine

// PolicyConfig represents the top-level structure of policy.json
type PolicyConfig struct {
	Policies  []PolicyDefinition          `json:"policies" yaml:"policies"`
	Providers ProvidersConfig             `json:"providers,omitempty" yaml:"providers,omitempty"`
	Pricing   map[string]map[string]int64 `json:"pricing,omitempty" yaml:"pricing,omitempty"`
	Units     map[string]string           `json:"units,omitempty" yaml:"units,omitempty"` // provider_id -> unit_name
	Retention *RetentionConfig            `json:"retention,omitempty" yaml:"retention,omitempty"`
}

// RetentionConfig defines data lifecycle rules
type RetentionConfig struct {
	Enabled       bool              `json:"enabled" yaml:"enabled"`
	DefaultTTL    string            `json:"default_ttl" yaml:"default_ttl"`                           // e.g., "720h"
	ByType        map[string]string `json:"by_type,omitempty" yaml:"by_type,omitempty"`               // event_type -> "24h"
	CheckInterval string            `json:"check_interval,omitempty" yaml:"check_interval,omitempty"` // e.g., "1h"
}

// GetCost looks up the cost per unit for a given provider and pool.
// Returns 0 if not found.
func (c *PolicyConfig) GetCost(providerID, poolID string) int64 {
	if c.Pricing == nil {
		return 0
	}
	if pools, ok := c.Pricing[providerID]; ok {
		if cost, ok := pools[poolID]; ok {
			return cost
		}
	}
	return 0
}

// GetUnit returns the unit for a given provider, defaulting to "requests".
func (c *PolicyConfig) GetUnit(providerID string) string {
	if c.Units != nil {
		if unit, ok := c.Units[providerID]; ok {
			return unit
		}
	}
	return "requests"
}

// ProvidersConfig holds configuration for various providers
type ProvidersConfig struct {
	GitHub []GitHubConfig `json:"github,omitempty" yaml:"github,omitempty"`
	OpenAI []OpenAIConfig `json:"openai,omitempty" yaml:"openai,omitempty"`
}

// GitHubConfig defines configuration for the GitHub provider
type GitHubConfig struct {
	ID            string `json:"id" yaml:"id"`
	TokenEnvVar   string `json:"token_env_var" yaml:"token_env_var"` // Prefer env var name for security
	EnterpriseURL string `json:"enterprise_url,omitempty" yaml:"enterprise_url,omitempty"`
}

// OpenAIConfig defines configuration for the OpenAI provider
type OpenAIConfig struct {
	ID          string `json:"id" yaml:"id"`
	TokenEnvVar string `json:"token_env_var" yaml:"token_env_var"`
	OrgID       string `json:"org_id,omitempty" yaml:"org_id,omitempty"` // Optional: for 'OpenAI-Organization' header
	BaseURL     string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
}

// PolicyDefinition maps a high-level policy block
type PolicyDefinition struct {
	ID    string           `json:"id" yaml:"id"`
	Scope string           `json:"scope" yaml:"scope"` // e.g., "global", "env:dev"
	Type  string           `json:"type" yaml:"type"`   // "hard", "soft"
	Limit int64            `json:"limit,omitempty" yaml:"limit,omitempty"`
	Rules []RuleDefinition `json:"rules" yaml:"rules"`
}

// RuleDefinition maps individual logic rules
type RuleDefinition struct {
	Name       string                 `json:"name" yaml:"name"`
	Condition  string                 `json:"condition" yaml:"condition"` // Simple DSL: "remaining < 100"
	Action     string                 `json:"action" yaml:"action"`       // "approve", "deny", "shape"
	Params     map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
	TimeWindow *TimeWindow            `json:"time_window,omitempty" yaml:"time_window,omitempty"`
}

// TimeWindow defines a temporal constraint for a rule
type TimeWindow struct {
	StartTime string   `json:"start_time,omitempty" yaml:"start_time,omitempty"` // HH:MM (24-hour)
	EndTime   string   `json:"end_time,omitempty" yaml:"end_time,omitempty"`     // HH:MM (24-hour)
	Days      []string `json:"days,omitempty" yaml:"days,omitempty"`             // ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
	Location  string   `json:"location,omitempty" yaml:"location,omitempty"`     // e.g., "America/New_York" (defaults to UTC)
}
