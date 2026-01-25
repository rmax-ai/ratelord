package api

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
	IntentID   string `json:"intent_id"`
	Decision   string `json:"decision"` // approve, deny, modify
	Reason     string `json:"reason,omitempty"`
	ModifiedBy string `json:"modified_by,omitempty"` // if decision=modify
	ValidUntil string `json:"valid_until,omitempty"` // ISO8601
}

// IdentityRegistration matches the payload for POST /v1/identities
type IdentityRegistration struct {
	IdentityID string                 `json:"identity_id"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// IdentityResponse matches the response for POST /v1/identities
type IdentityResponse struct {
	IdentityID string `json:"identity_id"`
	Status     string `json:"status"`
	EventID    string `json:"event_id"`
}
