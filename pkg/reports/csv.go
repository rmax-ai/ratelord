package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// EventReport generates CSV reports for raw events.
type EventReport struct {
	store ReportStore
}

// NewEventReport creates a new EventReport generator.
func NewEventReport(s ReportStore) *EventReport {
	return &EventReport{store: s}
}

// Generate creates a CSV report for raw events based on the provided parameters.
func (r *EventReport) Generate(ctx context.Context, params ReportParams) (io.Reader, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write CSV headers
	headers := []string{"event_id", "timestamp", "type", "identity_id", "scope_id", "workload_id", "payload"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write headers: %w", err)
	}

	// Construct EventFilter from params
	filter := store.EventFilter{
		From: params.Start,
		To:   params.End,
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
		row := []string{
			string(event.EventID),
			event.TsEvent.Format(time.RFC3339),
			string(event.EventType),
			event.Dimensions.IdentityID,
			event.Dimensions.ScopeID,
			event.Dimensions.WorkloadID,
			string(event.Payload),
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
