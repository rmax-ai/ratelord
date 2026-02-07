package forecast

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestForecastProjection(t *testing.T) {
	windowSize := 5
	fp := NewForecastProjection(windowSize)

	poolID := "test-pool"
	now := time.Now()

	// Test case 1: Add a single usage observation
	t.Run("AddObservation", func(t *testing.T) {
		payload := map[string]interface{}{
			"pool_id":   poolID,
			"remaining": 100,
			"used":      10,
			"cost":      100,
		}
		payloadBytes, _ := json.Marshal(payload)

		event := &store.Event{
			TsEvent: now,
			Payload: payloadBytes,
		}

		fp.OnUsageObserved(event)

		history := fp.GetHistory(poolID)
		assert.Len(t, history, 1)
		assert.Equal(t, int64(10), history[0].Used)
		assert.Equal(t, int64(100), history[0].Remaining)
		assert.Equal(t, currency.MicroUSD(100), history[0].Cost)
	})

	// Test case 2: Add multiple observations up to window size
	t.Run("FillWindow", func(t *testing.T) {
		for i := 0; i < windowSize; i++ {
			payload := map[string]interface{}{
				"pool_id":   poolID,
				"remaining": 100 - i,
				"used":      10 + i,
				"cost":      100 + i,
			}
			payloadBytes, _ := json.Marshal(payload)

			event := &store.Event{
				TsEvent: now.Add(time.Duration(i+1) * time.Minute),
				Payload: payloadBytes,
			}

			fp.OnUsageObserved(event)
		}

		history := fp.GetHistory(poolID)
		// It should still just keep the last `windowSize` items (or less if ring is not full, but here we added 1 (prev) + 5 = 6 total)
		// Wait, ring buffer overwrites.
		// Previous test added 1. This loop adds 5. Total 6. Window 5.
		// Should have 5 items.
		assert.Len(t, history, windowSize)
	})

	// Test case 3: Verify order (oldest to newest)
	t.Run("VerifyOrder", func(t *testing.T) {
		history := fp.GetHistory(poolID)
		assert.Len(t, history, windowSize)

		// The first item should be the 2nd item added overall (since 1st was overwritten)
		// timestamps should be increasing
		for i := 0; i < len(history)-1; i++ {
			assert.True(t, history[i].Timestamp.Before(history[i+1].Timestamp) || history[i].Timestamp.Equal(history[i+1].Timestamp))
		}
	})

	// Test case 4: GetAllHistories
	t.Run("GetAllHistories", func(t *testing.T) {
		all := fp.GetAllHistories()
		assert.Contains(t, all, poolID)
		assert.Len(t, all[poolID], windowSize)
	})

	// Test case 5: LoadHistories
	t.Run("LoadHistories", func(t *testing.T) {
		newFP := NewForecastProjection(windowSize)
		histories := fp.GetAllHistories()
		newFP.LoadHistories(histories)

		loaded := newFP.GetHistory(poolID)
		assert.Len(t, loaded, windowSize)
		assert.Equal(t, fp.GetHistory(poolID), loaded)
	})

	// Test case 6: Bad Payload
	t.Run("BadPayload", func(t *testing.T) {
		event := &store.Event{
			TsEvent: now,
			Payload: []byte("invalid json"),
		}
		// Should not panic
		fp.OnUsageObserved(event)
	})

	// Test case 7: Non-existent pool
	t.Run("NonExistentPool", func(t *testing.T) {
		history := fp.GetHistory("non-existent")
		assert.Nil(t, history)
	})
}
