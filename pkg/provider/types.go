package provider

import (
	"context"
	"time"
)

// ProviderID identifies a specific provider integration (e.g., "github", "openai")
type ProviderID string

// PollResult contains the observations from a single poll
type PollResult struct {
	ProviderID ProviderID
	Status     string // "success", "partial", "error"
	Error      error
	Timestamp  time.Time

	// Usage data
	Usage []UsageObservation
}

// UsageObservation represents a single point of usage data
type UsageObservation struct {
	PoolID    string
	Used      int64
	Remaining int64
	Limit     int64
	ResetAt   time.Time
}

// Provider defines the interface for external rate limit sources
type Provider interface {
	// ID returns the unique identifier for this provider
	ID() ProviderID

	// Poll retrieves the current usage state from the provider
	Poll(ctx context.Context) (PollResult, error)
}
