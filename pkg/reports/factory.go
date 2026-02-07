package reports

import (
	"fmt"
)

// NewReportGenerator creates a report generator based on the report type.
func NewReportGenerator(reportType ReportType, s ReportStore) (Generator, error) {
	switch reportType {
	case ReportTypeAccessLog:
		return NewAccessLogReport(s), nil
	case ReportTypeUsage:
		return NewUsageReport(s), nil
	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
}
