//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Event struct {
	EventID   string
	EventType string
	TsEvent   time.Time
	Payload   []byte
}

type ForecastPayload struct {
	PoolID   string `json:"pool_id"`
	Forecast struct {
		TTE struct {
			P50Seconds int64 `json:"p50_seconds"`
			P99Seconds int64 `json:"p99_seconds"`
		} `json:"tte"`
		BurnRate struct {
			Mean float64 `json:"mean"`
		} `json:"burn_rate"`
	} `json:"forecast"`
}

type UsagePayload struct {
	PoolID    string `json:"pool_id"`
	Remaining int64  `json:"remaining"`
	Used      int64  `json:"used"`
}

func main() {
	db, err := sql.Open("sqlite3", "deploy/dogfood/ratelord.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT event_type, ts_event, payload 
		FROM events 
		WHERE event_type IN ('usage_observed', 'forecast_computed')
		ORDER BY ts_event ASC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var evt Event
		var payloadBytes []byte
		if err := rows.Scan(&evt.EventType, &evt.TsEvent, &payloadBytes); err != nil {
			log.Fatal(err)
		}
		evt.Payload = payloadBytes
		events = append(events, evt)
	}

	fmt.Printf("%-20s | %-8s | %-8s | %-10s | %-10s | %-10s | %-10s\n",
		"Timestamp", "Type", "Pool", "Used", "Pred.Rate", "Real.Rate", "Error")
	fmt.Println("---------------------------------------------------------------------------------------------")

	// Map to track usage history for calculating realized burn rate
	usageHistory := make(map[string][]struct {
		ts   time.Time
		used int64
	})

	// Pre-pass to build history
	for _, evt := range events {
		if evt.EventType == "usage_observed" {
			var p UsagePayload
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				continue
			}
			usageHistory[p.PoolID] = append(usageHistory[p.PoolID], struct {
				ts   time.Time
				used int64
			}{evt.TsEvent, p.Used})
		}
	}

	for _, evt := range events {
		tsStr := evt.TsEvent.Format("15:04:05")

		if evt.EventType == "usage_observed" {
			var p UsagePayload
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				continue
			}
			fmt.Printf("%-20s | %-8s | %-8s | %-10d | %-10s | %-10s | %-10s\n",
				tsStr, "USAGE", p.PoolID, p.Used, "-", "-", "-")
		} else if evt.EventType == "forecast_computed" {
			var p ForecastPayload
			if err := json.Unmarshal(evt.Payload, &p); err != nil {
				continue
			}

			// Calculate realized burn rate over next 1 minute (if data exists)
			realizedRate := 0.0
			hasRealized := false

			history := usageHistory[p.PoolID]
			// Find current point index
			idx := -1
			for i, h := range history {
				if h.ts.Equal(evt.TsEvent) || h.ts.After(evt.TsEvent) {
					idx = i
					break
				}
			}

			if idx != -1 && idx+1 < len(history) {
				// Look ahead up to 5 points or 1 minute
				endIdx := idx + 1
				for i := idx + 1; i < len(history); i++ {
					if history[i].ts.Sub(history[idx].ts) > 1*time.Minute {
						break
					}
					endIdx = i
				}

				if endIdx > idx {
					start := history[idx]
					end := history[endIdx]
					deltaUsed := float64(end.used - start.used)
					deltaTime := end.ts.Sub(start.ts).Seconds()
					if deltaTime > 0 {
						realizedRate = deltaUsed / deltaTime
						hasRealized = true
					}
				}
			}

			predRateStr := fmt.Sprintf("%.2f", p.Forecast.BurnRate.Mean)
			realRateStr := "-"
			errorStr := "-"

			if hasRealized {
				realRateStr = fmt.Sprintf("%.2f", realizedRate)
				diff := realizedRate - p.Forecast.BurnRate.Mean
				errorStr = fmt.Sprintf("%.2f", diff)
			}

			fmt.Printf("%-20s | %-8s | %-8s | %-10s | %-10s | %-10s | %-10s\n",
				tsStr, "FORECAST", p.PoolID, "-", predRateStr, realRateStr, errorStr)
		}
	}
}
