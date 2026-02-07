package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// AccessLogReport generates CSV reports for access log events (intent decisions).
type AccessLogReport struct {
	store ReportStore
}

// NewAccessLogReport creates a new AccessLogReport generator.
func NewAccessLogReport(s ReportStore) *AccessLogReport {
	return &AccessLogReport{store: s}
}

// Generate creates a CSV report for access log events based on the provided parameters.
func (r *AccessLogReport) Generate(ctx context.Context, params ReportParams) (io.Reader, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write CSV headers
	headers := []string{"timestamp", "identity_id", "workload_id", "scope_id", "decision", "reason", "latency_ms"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write headers: %w", err)
	}

	// Construct EventFilter from params
	filter := store.EventFilter{
		From:       params.Start,
		To:         params.End,
		EventTypes: []store.EventType{store.EventTypeIntentDecided},
	}

	// Apply filters from params.Filters if present
	if identityID, ok := params.Filters["identity_id"].(string); ok && identityID != "" {
		filter.IdentityID = identityID
	}
	if scopeID, ok := params.Filters["scope_id"].(string); ok && scopeID != "" {
		filter.ScopeID = scopeID
	}

	// Query events
	events, err := r.store.QueryEvents(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	// Process each event
	for _, event := range events {
		// Parse payload
		var payload engine.PolicyEvaluationResult
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload for event %s: %w", event.EventID, err)
		}

		// Prepare row
		reason := payload.Reason
		if reason == "" && payload.Decision == engine.DecisionDenyWithReason {
			reason = "unknown" // fallback if reason is missing
		}

		// Latency: for now, set to 0 as it's not directly available in the payload
		// In a full implementation, this might require correlating with intent_submitted events
		latencyMs := "0"

		row := []string{
			event.TsEvent.Format(time.RFC3339),
			event.Dimensions.IdentityID,
			event.Dimensions.WorkloadID,
			event.Dimensions.ScopeID,
			string(payload.Decision),
			reason,
			latencyMs,
		}

		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush writer: %w", err)
	}

	return buf, nil
}
