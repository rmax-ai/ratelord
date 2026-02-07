package reports

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/store"
)

type mockReportStore struct {
	events     []*store.Event
	usageStats []store.UsageStat
}

func (m *mockReportStore) QueryEvents(ctx context.Context, filter store.EventFilter) ([]*store.Event, error) {
	var results []*store.Event
	for _, e := range m.events {
		// Basic time filtering
		if !filter.From.IsZero() && e.TsEvent.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.TsEvent.After(filter.To) {
			continue
		}
		// Type filtering
		if len(filter.EventTypes) > 0 {
			found := false
			for _, t := range filter.EventTypes {
				if e.EventType == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		results = append(results, e)
	}
	return results, nil
}

func (m *mockReportStore) GetUsageStats(ctx context.Context, filter store.UsageFilter) ([]store.UsageStat, error) {
	return m.usageStats, nil
}

func TestEventReport(t *testing.T) {
	now := time.Now()
	events := []*store.Event{
		{
			EventID:    "evt1",
			EventType:  store.EventTypeIdentityRegistered,
			TsEvent:    now,
			Payload:    json.RawMessage(`{"foo":"bar"}`),
			Dimensions: store.EventDimensions{IdentityID: "id1", ScopeID: "scope1"},
		},
	}
	s := &mockReportStore{events: events}
	r := NewEventReport(s)

	params := ReportParams{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(1 * time.Hour),
	}

	reader, err := r.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 { // Header + 1 row
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	if records[1][0] != "evt1" {
		t.Errorf("Expected event ID evt1, got %s", records[1][0])
	}
}

func TestAccessLogReport(t *testing.T) {
	now := time.Now()
	payload, _ := json.Marshal(engine.PolicyEvaluationResult{
		Decision: engine.DecisionApprove,
		Reason:   "test",
	})
	events := []*store.Event{
		{
			EventID:    "evt1",
			EventType:  store.EventTypeIntentDecided,
			TsEvent:    now,
			Payload:    payload,
			Dimensions: store.EventDimensions{IdentityID: "id1", ScopeID: "scope1"},
		},
	}
	s := &mockReportStore{events: events}
	r := NewAccessLogReport(s)

	params := ReportParams{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(1 * time.Hour),
	}

	reader, err := r.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	if records[1][4] != "approve" {
		t.Errorf("Expected decision approve, got %s", records[1][4])
	}
}

func TestUsageReport(t *testing.T) {
	now := time.Now()
	stats := []store.UsageStat{
		{
			BucketTs:   now,
			ProviderID: "prov1",
			PoolID:     "pool1",
			TotalUsage: 100,
			EventCount: 10,
		},
	}
	s := &mockReportStore{usageStats: stats}
	r := NewUsageReport(s)

	params := ReportParams{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(1 * time.Hour),
	}

	reader, err := r.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
	if records[1][1] != "prov1" {
		t.Errorf("Expected provider prov1, got %s", records[1][1])
	}
}
