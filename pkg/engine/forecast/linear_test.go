package forecast

import (
	"testing"
	"time"
)

func TestLinearModel_Predict_PositiveBurnRate(t *testing.T) {
	model := &LinearModel{}
	now := time.Now()
	history := []UsagePoint{
		{Timestamp: now.Add(-10 * time.Second), Used: 10},
		{Timestamp: now.Add(-5 * time.Second), Used: 20},
		{Timestamp: now, Used: 30},
	}
	resetAt := now.Add(100 * time.Second)
	remaining := int64(1000) // Larger remaining to ensure TTE > TTR

	forecast, err := model.Predict(history, remaining, resetAt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if forecast.BurnRate.Mean <= 0 {
		t.Errorf("Expected positive burn rate, got %f", forecast.BurnRate.Mean)
	}

	if forecast.TTE.P50Seconds <= 0 {
		t.Errorf("Expected positive P50 TTE, got %d", forecast.TTE.P50Seconds)
	}

	if forecast.Risk.ProbabilityExhaustionBeforeReset != 0.0 {
		t.Errorf("Expected risk 0.0, got %f", forecast.Risk.ProbabilityExhaustionBeforeReset)
	}
}

func TestLinearModel_Predict_ZeroBurnRate(t *testing.T) {
	model := &LinearModel{}
	now := time.Now()
	history := []UsagePoint{
		{Timestamp: now.Add(-10 * time.Second), Used: 20},
		{Timestamp: now.Add(-5 * time.Second), Used: 20},
		{Timestamp: now, Used: 20},
	}
	resetAt := now.Add(100 * time.Second)
	remaining := int64(100)

	forecast, err := model.Predict(history, remaining, resetAt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if forecast.BurnRate.Mean != 0 {
		t.Errorf("Expected zero burn rate, got %f", forecast.BurnRate.Mean)
	}

	if forecast.TTE.P50Seconds != 9223372036854775807 { // math.MaxInt64
		t.Errorf("Expected infinite P50 TTE, got %d", forecast.TTE.P50Seconds)
	}

	if forecast.Risk.ProbabilityExhaustionBeforeReset != 0.0 {
		t.Errorf("Expected risk 0.0, got %f", forecast.Risk.ProbabilityExhaustionBeforeReset)
	}
}

func TestLinearModel_Predict_RiskExhaustion(t *testing.T) {
	model := &LinearModel{}
	now := time.Now()
	history := []UsagePoint{
		{Timestamp: now.Add(-10 * time.Second), Used: 10},
		{Timestamp: now.Add(-5 * time.Second), Used: 20},
		{Timestamp: now, Used: 30},
	}
	resetAt := now.Add(10 * time.Second) // Short reset time
	remaining := int64(10)               // Small remaining to ensure exhaustion before reset

	forecast, err := model.Predict(history, remaining, resetAt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if forecast.Risk.ProbabilityExhaustionBeforeReset != 1.0 {
		t.Errorf("Expected risk 1.0, got %f", forecast.Risk.ProbabilityExhaustionBeforeReset)
	}

	if forecast.Risk.SafetyMarginSeconds >= 0 {
		t.Errorf("Expected negative safety margin, got %d", forecast.Risk.SafetyMarginSeconds)
	}
}

func TestLinearModel_Predict_InsufficientHistory(t *testing.T) {
	model := &LinearModel{}
	history := []UsagePoint{
		{Timestamp: time.Now(), Used: 10},
	}
	resetAt := time.Now().Add(100 * time.Second)
	remaining := int64(100)

	_, err := model.Predict(history, remaining, resetAt)
	if err == nil {
		t.Fatal("Expected error for insufficient history")
	}
}
