package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// UsageReport generates CSV reports for usage statistics.
type UsageReport struct {
	store ReportStore
}

// NewUsageReport creates a new UsageReport generator.
func NewUsageReport(s ReportStore) *UsageReport {
	return &UsageReport{store: s}
}

// Generate creates a CSV report for usage statistics based on the provided parameters.
func (r *UsageReport) Generate(ctx context.Context, params ReportParams) (io.Reader, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write CSV headers
	headers := []string{"bucket_ts", "provider", "pool", "identity", "scope", "total_usage", "event_count"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write headers: %w", err)
	}

	// Construct UsageFilter from params
	filter := store.UsageFilter{
		From: params.Start,
		To:   params.End,
	}

	// Default to "hour" bucket if not specified
	bucket := "hour"
	if b, ok := params.Filters["bucket"].(string); ok && b != "" {
		bucket = b
	}
	filter.Bucket = bucket

	// Apply other filters
	if providerID, ok := params.Filters["provider_id"].(string); ok && providerID != "" {
		filter.ProviderID = providerID
	}
	if poolID, ok := params.Filters["pool_id"].(string); ok && poolID != "" {
		filter.PoolID = poolID
	}
	if identityID, ok := params.Filters["identity_id"].(string); ok && identityID != "" {
		filter.IdentityID = identityID
	}
	if scopeID, ok := params.Filters["scope_id"].(string); ok && scopeID != "" {
		filter.ScopeID = scopeID
	}

	// Query usage stats
	stats, err := r.store.GetUsageStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage stats: %w", err)
	}

	// Process each stat
	for _, stat := range stats {
		row := []string{
			stat.BucketTs.Format("2006-01-02T15:04:05Z07:00"),
			stat.ProviderID,
			stat.PoolID,
			stat.IdentityID,
			stat.ScopeID,
			fmt.Sprintf("%d", stat.TotalUsage),
			fmt.Sprintf("%d", stat.EventCount),
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
