package forecast

import (
	"math"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
)

func TestLinearModel_Predict_CostBurnRate(t *testing.T) {
	model := &LinearModel{}
	now := time.Now()
	// Create history where cost increases by 200 MicroUSD per 5 seconds (40 MicroUSD/sec)
	history := []UsagePoint{
		{Timestamp: now.Add(-10 * time.Second), Used: 10, Cost: currency.MicroUSD(100)},
		{Timestamp: now.Add(-5 * time.Second), Used: 20, Cost: currency.MicroUSD(300)},
		{Timestamp: now, Used: 30, Cost: currency.MicroUSD(500)},
	}
	resetAt := now.Add(100 * time.Second)
	remaining := int64(1000)

	forecast, err := model.Predict(history, remaining, resetAt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Expected cost burn rate = 40.0 MicroUSD/sec
	expectedCostBurnRate := 40.0
	if math.Abs(forecast.CostBurnRate.Mean-expectedCostBurnRate) > 0.001 {
		t.Errorf("Expected cost burn rate %f, got %f", expectedCostBurnRate, forecast.CostBurnRate.Mean)
	}

	if forecast.CostBurnRate.Unit != "MicroUSD/sec" {
		t.Errorf("Expected unit 'MicroUSD/sec', got %s", forecast.CostBurnRate.Unit)
	}

	// Variance should be close to 0 as points are perfectly linear
	if forecast.CostBurnRate.Variance > 0.001 {
		t.Errorf("Expected low variance, got %f", forecast.CostBurnRate.Variance)
	}
}
