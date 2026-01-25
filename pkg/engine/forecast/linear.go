package forecast

import (
	"errors"
	"math"
	"time"
)

// LinearModel implements the Model interface using linear regression for burn rate prediction
type LinearModel struct{}

// Predict calculates the forecast using linear regression on the usage history
func (m *LinearModel) Predict(history []UsagePoint, currentRemaining int64, resetAt time.Time) (Forecast, error) {
	if len(history) < 2 {
		return Forecast{}, errors.New("insufficient history for prediction")
	}

	// Perform linear regression: Used = a + b * time
	// Time in seconds since first point
	startTime := history[0].Timestamp.Unix()
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(history))

	for _, point := range history {
		x := float64(point.Timestamp.Unix() - startTime)
		y := float64(point.Used)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Slope b = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		// No variation in time, can't fit line
		return Forecast{}, errors.New("no time variation in history")
	}
	b := (n*sumXY - sumX*sumY) / denom
	a := (sumY - b*sumX) / n

	// Burn rate is slope b (per second)
	meanBurnRate := b
	if meanBurnRate <= 0 {
		// If not burning, infinite TTE
		return Forecast{
			TTE: TimeToExhaustion{
				P50Seconds: math.MaxInt64,
				P90Seconds: math.MaxInt64,
				P99Seconds: math.MaxInt64,
			},
			Risk: Risk{
				ProbabilityExhaustionBeforeReset: 0.0,
				SafetyMarginSeconds:              math.MaxInt64,
				TTRSeconds:                       int64(resetAt.Sub(time.Now()).Seconds()),
			},
			BurnRate: BurnRate{
				Mean:     meanBurnRate,
				Variance: 0,
				Unit:     "per second",
			},
		}, nil
	}

	// Calculate residuals for variance
	var sumResidualsSq float64
	for _, point := range history {
		x := float64(point.Timestamp.Unix() - startTime)
		predicted := a + b*x
		residual := float64(point.Used) - predicted
		sumResidualsSq += residual * residual
	}
	variance := sumResidualsSq / (n - 2) // degrees of freedom

	stdDev := math.Sqrt(variance)

	// TTE calculations
	p50Rate := meanBurnRate
	p90Rate := meanBurnRate + 1.645*stdDev // approx 90th percentile
	p99Rate := meanBurnRate + 2*stdDev     // approx 99th percentile

	p50TTE := int64(float64(currentRemaining) / p50Rate)
	p90TTE := int64(float64(currentRemaining) / p90Rate)
	p99TTE := int64(float64(currentRemaining) / p99Rate)

	// Risk calculation
	now := time.Now()
	ttrSeconds := int64(resetAt.Sub(now).Seconds())
	safetyMargin := p99TTE - ttrSeconds
	probExhaustion := 0.0
	if safetyMargin < 0 {
		probExhaustion = 1.0
	}

	return Forecast{
		TTE: TimeToExhaustion{
			P50Seconds: p50TTE,
			P90Seconds: p90TTE,
			P99Seconds: p99TTE,
		},
		Risk: Risk{
			ProbabilityExhaustionBeforeReset: probExhaustion,
			SafetyMarginSeconds:              safetyMargin,
			TTRSeconds:                       ttrSeconds,
		},
		BurnRate: BurnRate{
			Mean:     meanBurnRate,
			Variance: variance,
			Unit:     "per second",
		},
	}, nil
}
