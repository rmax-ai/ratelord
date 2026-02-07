package reports

import (
	"context"
	"io"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

type ReportType string

const (
	ReportTypeAccessLog ReportType = "access_log"
	ReportTypeUsage     ReportType = "usage"
	ReportTypeEvents    ReportType = "events"
)

type ReportFormat string

const (
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatJSON ReportFormat = "json"
)

type ReportParams struct {
	Start   time.Time
	End     time.Time
	Filters map[string]interface{}
}

// ReportStore defines the interface for data access required by reports.
type ReportStore interface {
	QueryEvents(ctx context.Context, filter store.EventFilter) ([]*store.Event, error)
	GetUsageStats(ctx context.Context, filter store.UsageFilter) ([]store.UsageStat, error)
}

type Generator interface {
	Generate(ctx context.Context, params ReportParams) (io.Reader, error)
}
