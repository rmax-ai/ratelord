package store

import (
	"context"
	"encoding/json"
	"time"
)

// EventType represents the kind of event.
type EventType string

const (
	EventTypeProviderPollObserved EventType = "provider_poll_observed"
	EventTypeProviderError        EventType = "provider_error"
	EventTypeConstraintObserved   EventType = "constraint_observed"
	EventTypeResetObserved        EventType = "reset_observed"
	EventTypeUsageObserved        EventType = "usage_observed"
	EventTypeForecastComputed     EventType = "forecast_computed"
	EventTypeIntentSubmitted      EventType = "intent_submitted"
	EventTypeIntentDecided        EventType = "intent_decided"
	EventTypePolicyTriggered      EventType = "policy_triggered"
	EventTypeThrottleAdvised      EventType = "throttle_advised"
	EventTypeIdentityRegistered   EventType = "identity_registered"
	EventTypeIdentityDeleted      EventType = "identity_deleted"
	EventTypePolicyUpdated        EventType = "policy_updated"
	EventTypeGrantIssued          EventType = "grant_issued"
)

// Lease represents a distributed lock or leadership claim.
type Lease struct {
	Name      string    `json:"name"`
	HolderID  string    `json:"holder_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Version   int64     `json:"version"` // For CAS (Compare-And-Swap) logic
	Epoch     int64     `json:"epoch"`   // Monotonically increasing election term
}

// LeaseStore defines the interface for acquiring and renewing leases.
type LeaseStore interface {
	// Acquire tries to acquire the lease. Returns true if successful.
	// If the lease is already held by holderID, it renews it.
	Acquire(ctx context.Context, name, holderID string, ttl time.Duration) (bool, error)

	// Renew updates the expiry of an existing lease held by holderID.
	// Returns error if the lease is lost or stolen.
	Renew(ctx context.Context, name, holderID string, ttl time.Duration) error

	// Release releases the lease if held by holderID.
	Release(ctx context.Context, name, holderID string) error

	// Get returns the current lease state.
	Get(ctx context.Context, name string) (*Lease, error)
}

// EventID is a unique identifier for an event.
type EventID string

// Event represents the canonical envelope for all system events.
// See DATA_MODEL.md for field definitions.
type Event struct {
	EventID       EventID          `json:"event_id"`
	EventType     EventType        `json:"event_type"`
	SchemaVersion int              `json:"schema_version"`
	TsEvent       time.Time        `json:"ts_event"`
	TsIngest      time.Time        `json:"ts_ingest"`
	Epoch         int64            `json:"epoch,omitempty"` // Leadership epoch at generation time
	Source        EventSource      `json:"source"`
	Dimensions    EventDimensions  `json:"dimensions"`
	Correlation   EventCorrelation `json:"correlation"`
	Payload       json.RawMessage  `json:"payload"`
}

// EventSource describes the origin of the event.
type EventSource struct {
	OriginKind string `json:"origin_kind"` // daemon, provider, client, operator
	OriginID   string `json:"origin_id"`
	WriterID   string `json:"writer_id"` // Always "ratelord-d"
}

// EventDimensions are the mandatory scopes for every event.
type EventDimensions struct {
	AgentID    string `json:"agent_id"`
	IdentityID string `json:"identity_id"`
	WorkloadID string `json:"workload_id"`
	ScopeID    string `json:"scope_id"`
}

// EventCorrelation groups events logically.
type EventCorrelation struct {
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
}

// UsageStat represents aggregated usage statistics for a time bucket.
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

// UsageFilter defines filters for querying usage statistics.
type UsageFilter struct {
	From       time.Time
	To         time.Time
	Bucket     string // "hour" or "day"
	ProviderID string
	PoolID     string
	IdentityID string
	ScopeID    string
}

// EventFilter defines filters for querying events.
type EventFilter struct {
	From       time.Time
	To         time.Time
	EventTypes []EventType
	IdentityID string
	ScopeID    string
	Limit      int
}

// WebhookConfig represents a registered webhook endpoint for event notifications.
type WebhookConfig struct {
	WebhookID string    `json:"webhook_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret"` // Shared secret for HMAC signature verification
	Events    []string  `json:"events"` // List of event types to subscribe to
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// Sentinel constants for unknown/global dimensions.
const (
	SentinelSystem  = "sentinel:system"
	SentinelGlobal  = "sentinel:global"
	SentinelUnknown = "sentinel:unknown"
)

// Snapshot represents a point-in-time capture of the system state.
type Snapshot struct {
	SnapshotID    string          `json:"snapshot_id"`
	SchemaVersion int             `json:"schema_version"`
	TsSnapshot    time.Time       `json:"ts_snapshot"`
	LastEventID   EventID         `json:"last_event_id"`
	Payload       json.RawMessage `json:"payload"`
}
