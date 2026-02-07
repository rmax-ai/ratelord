package redis

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/engine/currency"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
)

// RunUsageStoreTests runs a comprehensive test suite against a UsageStore implementation
func RunUsageStoreTests(t *testing.T, store engine.UsageStore) {
	t.Run("Set and Get", func(t *testing.T) {
		// Clear store
		store.Clear()

		state := engine.PoolState{
			ProviderID:  "test-provider",
			PoolID:      "test-pool",
			Used:        100,
			Remaining:   900,
			Cost:        50000, // 50 cents in microUSD
			ResetAt:     time.Now().Add(time.Hour),
			LastUpdated: time.Now(),
			LatestForecast: &forecast.Forecast{
				TTE: forecast.TimeToExhaustion{
					P50Seconds: 3600,
					P90Seconds: 7200,
					P99Seconds: 10800,
				},
			},
		}

		store.Set(state)

		retrieved, ok := store.Get("test-provider", "test-pool")
		if !ok {
			t.Fatal("Expected to find pool state")
		}

		if retrieved.ProviderID != state.ProviderID ||
			retrieved.PoolID != state.PoolID ||
			retrieved.Used != state.Used ||
			retrieved.Remaining != state.Remaining ||
			retrieved.Cost != state.Cost {
			t.Errorf("Retrieved state doesn't match set state: got %+v, want %+v", retrieved, state)
		}

		// Check forecast
		if retrieved.LatestForecast == nil {
			t.Error("Expected LatestForecast to be set")
		} else if retrieved.LatestForecast.TTE.P50Seconds != state.LatestForecast.TTE.P50Seconds {
			t.Errorf("Forecast P50 doesn't match: got %d, want %d", retrieved.LatestForecast.TTE.P50Seconds, state.LatestForecast.TTE.P50Seconds)
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		store.Clear()
		_, ok := store.Get("non-existent", "pool")
		if ok {
			t.Error("Expected not to find non-existent pool")
		}
	})

	t.Run("Increment existing pool", func(t *testing.T) {
		store.Clear()

		// Set initial state
		initial := engine.PoolState{
			ProviderID:  "test-provider",
			PoolID:      "test-pool",
			Used:        50,
			Remaining:   950,
			Cost:        25000,
			LastUpdated: time.Now(),
		}
		store.Set(initial)

		// Increment
		store.Increment("test-provider", "test-pool", 25, -25, 12500)

		retrieved, ok := store.Get("test-provider", "test-pool")
		if !ok {
			t.Fatal("Expected to find pool after increment")
		}

		expectedUsed := int64(75)
		expectedRemaining := int64(925)
		expectedCost := currency.MicroUSD(37500)

		if retrieved.Used != expectedUsed {
			t.Errorf("Used: got %d, want %d", retrieved.Used, expectedUsed)
		}
		if retrieved.Remaining != expectedRemaining {
			t.Errorf("Remaining: got %d, want %d", retrieved.Remaining, expectedRemaining)
		}
		if retrieved.Cost != expectedCost {
			t.Errorf("Cost: got %d, want %d", retrieved.Cost, expectedCost)
		}
	})

	t.Run("Increment new pool", func(t *testing.T) {
		store.Clear()

		store.Increment("new-provider", "new-pool", 10, -10, 5000)

		retrieved, ok := store.Get("new-provider", "new-pool")
		if !ok {
			t.Fatal("Expected to find newly created pool")
		}

		if retrieved.Used != 10 {
			t.Errorf("Used: got %d, want 10", retrieved.Used)
		}
		if retrieved.Remaining != -10 {
			t.Errorf("Remaining: got %d, want -10", retrieved.Remaining)
		}
		if retrieved.Cost != 5000 {
			t.Errorf("Cost: got %d, want 5000", retrieved.Cost)
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		store.Clear()

		states := []engine.PoolState{
			{ProviderID: "p1", PoolID: "pool1", Used: 1, Remaining: 99},
			{ProviderID: "p1", PoolID: "pool2", Used: 2, Remaining: 98},
			{ProviderID: "p2", PoolID: "pool1", Used: 3, Remaining: 97},
		}

		for _, state := range states {
			store.Set(state)
		}

		all := store.GetAll()
		if len(all) != 3 {
			t.Errorf("Expected 3 pools, got %d", len(all))
		}

		// Check that all states are present (order may vary)
		found := make(map[string]bool)
		for _, s := range all {
			key := s.ProviderID + ":" + s.PoolID
			found[key] = true
			for _, expected := range states {
				if expected.ProviderID == s.ProviderID && expected.PoolID == s.PoolID {
					if s.Used != expected.Used || s.Remaining != expected.Remaining {
						t.Errorf("State mismatch for %s: got %+v, want %+v", key, s, expected)
					}
				}
			}
		}

		if len(found) != 3 {
			t.Errorf("Not all pools found in GetAll")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		store.Clear()

		store.Set(engine.PoolState{ProviderID: "p", PoolID: "pool", Used: 1})
		store.Clear()

		all := store.GetAll()
		if len(all) != 0 {
			t.Errorf("Expected no pools after clear, got %d", len(all))
		}

		_, ok := store.Get("p", "pool")
		if ok {
			t.Error("Expected pool to be gone after clear")
		}
	})

	t.Run("Concurrent Increment", func(t *testing.T) {
		store.Clear()

		const numGoroutines = 10
		const incrementsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					store.Increment("concurrency-test", "pool", 1, -1, 100)
				}
			}()
		}

		wg.Wait()

		retrieved, ok := store.Get("concurrency-test", "pool")
		if !ok {
			t.Fatal("Expected to find pool after concurrent increments")
		}

		expectedUsed := int64(numGoroutines * incrementsPerGoroutine)
		expectedRemaining := int64(-expectedUsed)
		expectedCost := currency.MicroUSD(expectedUsed * 100)

		if retrieved.Used != expectedUsed {
			t.Errorf("Concurrent Used: got %d, want %d", retrieved.Used, expectedUsed)
		}
		if retrieved.Remaining != expectedRemaining {
			t.Errorf("Concurrent Remaining: got %d, want %d", retrieved.Remaining, expectedRemaining)
		}
		if retrieved.Cost != expectedCost {
			t.Errorf("Concurrent Cost: got %d, want %d", retrieved.Cost, expectedCost)
		}
	})
}

func TestRedisUsageStore(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create store
	store := NewRedisUsageStore(client)

	// Run tests
	RunUsageStoreTests(t, store)
}
