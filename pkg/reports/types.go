package reports

import (
	"context"
	"io"
	"time"
)

type ReportType string

const (
	ReportTypeAccessLog ReportType = "access_log"
	ReportTypeUsage     ReportType = "usage"
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

type Generator interface {
	Generate(ctx context.Context, params ReportParams) (io.Reader, error)
}
