package engine

import (
	"fmt"
	"strings"
	"time"
)

// Matches checks if the given time falls within the TimeWindow.
// Returns true if matched, or if TimeWindow is nil/empty.
func (tw *TimeWindow) Matches(t time.Time) (bool, error) {
	if tw == nil {
		return true, nil
	}

	// 1. Check Location
	targetTime := t
	if tw.Location != "" {
		loc, err := time.LoadLocation(tw.Location)
		if err != nil {
			return false, fmt.Errorf("invalid location '%s': %w", tw.Location, err)
		}
		targetTime = t.In(loc)
	}

	// 2. Check Day of Week
	if len(tw.Days) > 0 {
		currentDay := targetTime.Weekday().String() // "Sunday", "Monday"...
		matchedDay := false
		for _, d := range tw.Days {
			// Normalize comparison to handle "Mon", "mon", "Monday"
			d = strings.ToLower(strings.TrimSpace(d))
			cur := strings.ToLower(currentDay)

			// Check for 3-letter prefix match (e.g. "mon" matches "monday")
			if len(d) >= 3 && strings.HasPrefix(cur, d) {
				matchedDay = true
				break
			}
		}
		if !matchedDay {
			return false, nil
		}
	}

	// 3. Check Time Range
	if tw.StartTime != "" && tw.EndTime != "" {
		startMin, err := parseTimeOfDay(tw.StartTime)
		if err != nil {
			return false, err
		}
		endMin, err := parseTimeOfDay(tw.EndTime)
		if err != nil {
			return false, err
		}

		currentMin := targetTime.Hour()*60 + targetTime.Minute()

		if startMin <= endMin {
			// Normal range: 09:00 - 17:00
			if currentMin < startMin || currentMin > endMin {
				return false, nil
			}
		} else {
			// Cross-midnight range: 22:00 - 06:00
			// Matched if >= 22:00 OR <= 06:00
			if currentMin < startMin && currentMin > endMin {
				return false, nil
			}
		}
	}

	return true, nil
}

func parseTimeOfDay(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("invalid time format '%s' (expected HH:MM): %w", s, err)
	}
	return t.Hour()*60 + t.Minute(), nil
}
