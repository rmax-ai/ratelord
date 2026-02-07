package forecast

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockModel satisfies the Model interface
type MockModel struct {
	mock.Mock
}

func (m *MockModel) Predict(history []UsagePoint, currentRemaining int64, resetAt time.Time) (Forecast, error) {
	args := m.Called(history, currentRemaining, resetAt)
	return args.Get(0).(Forecast), args.Error(1)
}

func TestForecaster(t *testing.T) {
	// Setup real store with temp file
	tmpDB, err := os.CreateTemp("", "ratelord-test-*.db")
	assert.NoError(t, err)
	defer os.Remove(tmpDB.Name())
	tmpDB.Close()

	s, err := store.NewStore(tmpDB.Name())
	assert.NoError(t, err)
	defer s.Close()

	// Setup components
	projection := NewForecastProjection(10)
	mockModel := new(MockModel)
	forecaster := NewForecaster(s, projection, mockModel, nil)

	// Set epoch func
	epoch := int64(100)
	forecaster.SetEpochFunc(func() int64 { return epoch })
	assert.Equal(t, epoch, forecaster.getEpoch())

	ctx := context.Background()
	poolID := "test-pool"
	providerID := "test-provider"

	// Add enough history to trigger prediction (requires >= 2 points)

	// Point 1
	payload1 := map[string]interface{}{
		"provider_id": providerID,
		"pool_id":     poolID,
		"remaining":   1000,
		"used":        10,
	}
	payloadBytes1, _ := json.Marshal(payload1)
	event1 := &store.Event{
		EventID:   "evt1",
		EventType: store.EventTypeUsageObserved,
		TsEvent:   time.Now().Add(-1 * time.Minute),
		Payload:   payloadBytes1,
	}

	// This shouldn't trigger prediction yet (need < 2 history)
	forecaster.OnUsageObserved(ctx, event1)
	mockModel.AssertNotCalled(t, "Predict")

	// Point 2
	payload2 := map[string]interface{}{
		"provider_id": providerID,
		"pool_id":     poolID,
		"remaining":   990,
		"used":        20,
	}
	payloadBytes2, _ := json.Marshal(payload2)
	event2 := &store.Event{
		EventID:   "evt2",
		EventType: store.EventTypeUsageObserved,
		TsEvent:   time.Now(),
		Payload:   payloadBytes2,
	}

	// Setup mock expectation
	expectedForecast := Forecast{
		TTE: TimeToExhaustion{P50Seconds: 3600},
	}
	mockModel.On("Predict", mock.Anything, int64(990), mock.Anything).Return(expectedForecast, nil)

	// This should trigger prediction
	forecaster.OnUsageObserved(ctx, event2)

	// Verify mock call
	mockModel.AssertExpectations(t)

	// Verify event emission
	// Wait a bit for async write if it were async, but AppendEvent is blocking in this implementation (or at least called synchronously in OnUsageObserved)

	// Read back events to verify forecast was emitted
	events, err := s.ReadRecentEvents(ctx, 10)
	assert.NoError(t, err)

	var forecastEvent *store.Event
	for _, e := range events {
		if e.EventType == store.EventTypeForecastComputed {
			forecastEvent = e
			break
		}
	}

	assert.NotNil(t, forecastEvent)
	assert.Equal(t, epoch, forecastEvent.Epoch)
	assert.Equal(t, "forecast_test-pool", forecastEvent.Correlation.CorrelationID[:18]) // check prefix
	assert.Equal(t, "evt2", forecastEvent.Correlation.CausationID)

	var forecastPayload struct {
		ProviderID string   `json:"provider_id"`
		PoolID     string   `json:"pool_id"`
		Forecast   Forecast `json:"forecast"`
	}
	err = json.Unmarshal(forecastEvent.Payload, &forecastPayload)
	assert.NoError(t, err)
	assert.Equal(t, providerID, forecastPayload.ProviderID)
	assert.Equal(t, poolID, forecastPayload.PoolID)
	assert.Equal(t, expectedForecast.TTE.P50Seconds, forecastPayload.Forecast.TTE.P50Seconds)
}
