package reports

import (
	"fmt"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// NewReportGenerator creates a report generator based on the report type.
func NewReportGenerator(reportType ReportType, s *store.Store) (Generator, error) {
	switch reportType {
	case ReportTypeAccessLog:
		return NewAccessLogReport(s), nil
	case ReportTypeUsage:
		return NewUsageReport(s), nil
	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
}
