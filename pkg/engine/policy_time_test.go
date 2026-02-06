package engine

import (
	"testing"
	"time"
)

func TestTimeWindow_Matches(t *testing.T) {
	tests := []struct {
		name      string
		window    *TimeWindow
		checkTime time.Time
		want      bool
		wantErr   bool
	}{
		{
			name:      "Nil window matches everything",
			window:    nil,
			checkTime: time.Now(),
			want:      true,
		},
		{
			name: "Day match - Monday",
			window: &TimeWindow{
				Days: []string{"Mon"},
			},
			// a Monday
			checkTime: time.Date(2023, 10, 23, 10, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name: "Day mismatch - Tuesday",
			window: &TimeWindow{
				Days: []string{"Mon"},
			},
			// a Tuesday
			checkTime: time.Date(2023, 10, 24, 10, 0, 0, 0, time.UTC),
			want:      false,
		},
		{
			name: "Time range match (within)",
			window: &TimeWindow{
				StartTime: "09:00",
				EndTime:   "17:00",
			},
			checkTime: time.Date(2023, 10, 23, 12, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name: "Time range mismatch (before)",
			window: &TimeWindow{
				StartTime: "09:00",
				EndTime:   "17:00",
			},
			checkTime: time.Date(2023, 10, 23, 8, 0, 0, 0, time.UTC),
			want:      false,
		},
		{
			name: "Time range mismatch (after)",
			window: &TimeWindow{
				StartTime: "09:00",
				EndTime:   "17:00",
			},
			checkTime: time.Date(2023, 10, 23, 18, 0, 0, 0, time.UTC),
			want:      false,
		},
		{
			name: "Cross-midnight match (late)",
			window: &TimeWindow{
				StartTime: "22:00",
				EndTime:   "06:00",
			},
			checkTime: time.Date(2023, 10, 23, 23, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name: "Cross-midnight match (early)",
			window: &TimeWindow{
				StartTime: "22:00",
				EndTime:   "06:00",
			},
			checkTime: time.Date(2023, 10, 24, 4, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name: "Cross-midnight mismatch (middle)",
			window: &TimeWindow{
				StartTime: "22:00",
				EndTime:   "06:00",
			},
			checkTime: time.Date(2023, 10, 23, 12, 0, 0, 0, time.UTC),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.window.Matches(tt.checkTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("Matches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Matches() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeWindow_Location(t *testing.T) {
	// Separate test for location to handle envs without tzdata
	_, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("Skipping location test: %v", err)
	}

	window := &TimeWindow{
		StartTime: "09:00", // 9 AM EST
		EndTime:   "17:00",
		Location:  "America/New_York",
	}

	// 15:00 UTC = 10:00 EST (Jan 1) -> Match
	checkTime := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)
	got, err := window.Matches(checkTime)
	if err != nil {
		t.Fatalf("Matches error: %v", err)
	}
	if !got {
		t.Errorf("Expected match for 15:00 UTC (10:00 EST) in 09:00-17:00 EST window")
	}

	// 12:00 UTC = 07:00 EST (Jan 1) -> No Match
	checkTime2 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	got2, err := window.Matches(checkTime2)
	if err != nil {
		t.Fatalf("Matches error: %v", err)
	}
	if got2 {
		t.Errorf("Expected NO match for 12:00 UTC (07:00 EST) in 09:00-17:00 EST window")
	}
}
