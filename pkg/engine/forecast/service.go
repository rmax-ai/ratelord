package forecast

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rmax/ratelord/pkg/store"
)

// Forecaster ties together the projection, model, and event emission
type Forecaster struct {
	store      *store.Store
	projection *ForecastProjection
	model      Model
}

// NewForecaster creates a new forecaster instance
func NewForecaster(store *store.Store, projection *ForecastProjection, model Model) *Forecaster {
	return &Forecaster{
		store:      store,
		projection: projection,
		model:      model,
	}
}

// OnUsageObserved is called when a usage_observed event occurs
func (f *Forecaster) OnUsageObserved(ctx context.Context, event *store.Event) {
	var payload struct {
		PoolID    string `json:"pool_id"`
		Remaining int64  `json:"remaining"`
		Used      int64  `json:"used"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		log.Printf("Failed to unmarshal usage event payload: %v", err)
		return
	}

	// Update projection
	f.projection.OnUsageObserved(event)

	// Get history
	history := f.projection.GetHistory(payload.PoolID)
	if len(history) < 2 {
		// Not enough history for prediction
		return
	}

	// Assume reset at next day or something; for now, hardcode to 24 hours from now
	resetAt := time.Now().Add(24 * time.Hour) // TODO: get from pool config

	// Predict
	forecast, err := f.model.Predict(history, payload.Remaining, resetAt)
	if err != nil {
		log.Printf("Failed to compute forecast for pool %s: %v", payload.PoolID, err)
		return
	}

	// Emit forecast_computed event
	f.emitForecastComputed(ctx, payload.PoolID, forecast, event)
}

func (f *Forecaster) emitForecastComputed(ctx context.Context, poolID string, forecast Forecast, causationEvent *store.Event) {
	now := time.Now().UTC()
	correlationID := fmt.Sprintf("forecast_%s_%d", poolID, now.Unix())

	event := &store.Event{
		EventID:       store.EventID(fmt.Sprintf("forecast_%s_%d", poolID, now.UnixNano())),
		EventType:     store.EventTypeForecastComputed,
		SchemaVersion: 1,
		TsEvent:       now,
		TsIngest:      now,
		Source: store.EventSource{
			OriginKind: "daemon",
			OriginID:   "forecaster",
			WriterID:   "ratelord-d",
		},
		Dimensions: store.EventDimensions{
			AgentID:    store.SentinelSystem,
			IdentityID: store.SentinelGlobal,
			WorkloadID: store.SentinelSystem,
			ScopeID:    store.SentinelGlobal,
		},
		Correlation: store.EventCorrelation{
			CorrelationID: correlationID,
			CausationID:   string(causationEvent.EventID),
		},
	}

	payload := map[string]interface{}{
		"pool_id":  poolID,
		"forecast": forecast,
	}
	payloadBytes, _ := json.Marshal(payload)
	event.Payload = payloadBytes

	if err := f.store.AppendEvent(ctx, event); err != nil {
		log.Printf("Failed to append forecast event: %v", err)
	}
}
