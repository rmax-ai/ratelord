package api

import "time"

// IntentRequest matches the POST /v1/intent body schema
type IntentRequest struct {
	AgentID       string                 `json:"agent_id"`
	IdentityID    string                 `json:"identity_id"`
	ScopeID       string                 `json:"scope_id"`
	WorkloadID    string                 `json:"workload_id"`
	Priority      string                 `json:"priority,omitempty"` // low, normal, critical
	Description   string                 `json:"description,omitempty"`
	ClientContext map[string]interface{} `json:"client_context,omitempty"`
}

// DecisionResponse matches the response for POST /v1/intent
type DecisionResponse struct {
	IntentID      string                 `json:"intent_id"`
	Decision      string                 `json:"decision"` // approve, deny, modify
	Reason        string                 `json:"reason,omitempty"`
	Modifications map[string]interface{} `json:"modifications,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	ModifiedBy    string                 `json:"modified_by,omitempty"` // if decision=modify
	ValidUntil    string                 `json:"valid_until,omitempty"` // ISO8601
}

// IdentityRegistration matches the payload for POST /v1/identities
type IdentityRegistration struct {
	IdentityID string                 `json:"identity_id"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Token      string                 `json:"token,omitempty"` // Optional: If provided, sets the authentication token
}

// IdentityResponse matches the response for POST /v1/identities
type IdentityResponse struct {
	IdentityID string `json:"identity_id"`
	Status     string `json:"status"`
	EventID    string `json:"event_id"`
	Token      string `json:"token,omitempty"` // Returned only if generated
}

// GrantRequest matches the POST /v1/federation/grant body schema
type GrantRequest struct {
	FollowerID string                 `json:"follower_id"`
	ProviderID string                 `json:"provider_id"` // Added for M38.1
	PoolID     string                 `json:"pool_id"`
	Amount     int64                  `json:"amount"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Added for M34.2
}

// GrantResponse matches the response for POST /v1/federation/grant
type GrantResponse struct {
	Granted         int64     `json:"granted"`
	ValidUntil      time.Time `json:"valid_until"`
	RemainingGlobal int64     `json:"remaining_global,omitempty"`
}
