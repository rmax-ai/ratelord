package client

import (
	"encoding/json"
	"time"
)

// Intent represents the agent's request to perform an action.
type Intent struct {
	// AgentID is the required identifier of the agent.
	AgentID string `json:"agent_id"`
	// IdentityID is the required identity/credential to use.
	IdentityID string `json:"identity_id"`
	// WorkloadID is the required type of task (e.g., "repo_scan").
	WorkloadID string `json:"workload_id"`
	// ScopeID is the required target (e.g., "repo:owner/name").
	ScopeID string `json:"scope_id"`
	// Urgency indicates priority: "high", "normal", "background" (default: "normal").
	Urgency string `json:"urgency,omitempty"`
	// ExpectedCost is the optional cost estimate. Default: 1.0.
	ExpectedCost float64 `json:"expected_cost,omitempty"`
	// DurationHint is the estimated runtime in seconds.
	DurationHint float64 `json:"duration_hint,omitempty"`
	// ClientContext is an optional map of metadata.
	ClientContext map[string]any `json:"client_context,omitempty"`
}

// Modifications contains changes required by the daemon.
type Modifications struct {
	// WaitSeconds is the time the SDK slept (informational).
	WaitSeconds float64 `json:"wait_seconds"`
	// IdentitySwitch is set if the daemon forced an identity swap.
	IdentitySwitch string `json:"identity_switch,omitempty"`
}

// Decision represents the result of the negotiation.
type Decision struct {
	// Allowed is a derived helper (true if approved/modified).
	Allowed bool `json:"allowed"`
	// IntentID is the UUID assigned by the daemon.
	IntentID string `json:"intent_id"`
	// Status can be "approve", "approve_with_modifications", "deny_with_reason".
	Status string `json:"status"`
	// Modifications contains any changes required by the daemon.
	Modifications Modifications `json:"modifications,omitempty"`
	// Reason is populated if the request was denied.
	Reason string `json:"reason,omitempty"`
}

// Status represents the health check response.
type Status struct {
	// Status is the health status string (e.g. "ok").
	Status string `json:"status"`
	// Version is the daemon version.
	Version string `json:"version"`
}

// UnmarshalJSON implements custom unmarshaling for Decision to derive Allowed.
func (d *Decision) UnmarshalJSON(data []byte) error {
	type Alias Decision
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*d = Decision(*aux)

	// Derive Allowed based on Status
	d.Allowed = d.Status == "approve" || d.Status == "approve_with_modifications"
	return nil
}

// --- M39: MCP Support Types ---

// Event represents a system event.
type Event struct {
	EventID       string           `json:"event_id"`
	EventType     string           `json:"event_type"`
	SchemaVersion int              `json:"schema_version"`
	TsEvent       time.Time        `json:"ts_event"`
	TsIngest      time.Time        `json:"ts_ingest"`
	Source        EventSource      `json:"source"`
	Dimensions    EventDimensions  `json:"dimensions"`
	Correlation   EventCorrelation `json:"correlation"`
	Payload       json.RawMessage  `json:"payload"`
}

type EventSource struct {
	OriginKind string `json:"origin_kind"`
	OriginID   string `json:"origin_id"`
	WriterID   string `json:"writer_id"`
}

type EventDimensions struct {
	AgentID    string `json:"agent_id"`
	IdentityID string `json:"identity_id"`
	WorkloadID string `json:"workload_id"`
	ScopeID    string `json:"scope_id"`
}

type EventCorrelation struct {
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
}

// UsageStat represents aggregated usage statistics.
type UsageStat struct {
	BucketTs   time.Time `json:"bucket_ts"`
	ProviderID string    `json:"provider_id"`
	PoolID     string    `json:"pool_id"`
	IdentityID string    `json:"identity_id"`
	ScopeID    string    `json:"scope_id"`
	TotalUsage int       `json:"total_usage"`
	MinUsage   int       `json:"min_usage"`
	MaxUsage   int       `json:"max_usage"`
	EventCount int       `json:"event_count"`
}

// TrendsOptions defines filters for GetTrends.
type TrendsOptions struct {
	From       time.Time
	To         time.Time
	Bucket     string // "hour" or "day"
	ProviderID string
	PoolID     string
	IdentityID string
	ScopeID    string
}
