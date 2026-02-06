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

	startTime := history[0].Timestamp.Unix()

	// Calculate Usage Burn Rate
	usageSlope, usageVariance, _, err := calculateSlope(history, startTime, func(p UsagePoint) float64 {
		return float64(p.Used)
	})
	if err != nil {
		return Forecast{}, err
	}

	// Calculate Cost Burn Rate
	costSlope, costVariance, _, err := calculateSlope(history, startTime, func(p UsagePoint) float64 {
		return float64(p.Cost)
	})
	if err != nil {
		// If cost calculation fails (e.g., constant cost), we can still proceed but maybe with 0 burn rate?
		// For now, let's treat it as an error if we can't calculate slope,
		// but actually constant cost implies 0 slope which is fine.
		// The helper returns error if no time variation, which is checked once.
		// But if we have valid time variation, we should be fine.
		return Forecast{}, err
	}

	meanBurnRate := usageSlope
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
			CostBurnRate: BurnRate{
				Mean:     costSlope,
				Variance: costVariance,
				Unit:     "MicroUSD/sec",
			},
		}, nil
	}

	stdDev := math.Sqrt(usageVariance)

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
			Variance: usageVariance,
			Unit:     "per second",
		},
		CostBurnRate: BurnRate{
			Mean:     costSlope,
			Variance: costVariance,
			Unit:     "MicroUSD/sec",
		},
	}, nil
}

// calculateSlope performs linear regression y = a + bx
func calculateSlope(history []UsagePoint, startTime int64, valueExtractor func(p UsagePoint) float64) (slope float64, variance float64, intercept float64, err error) {
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(history))

	for _, point := range history {
		x := float64(point.Timestamp.Unix() - startTime)
		y := valueExtractor(point)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Slope b = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		// No variation in time, can't fit line
		return 0, 0, 0, errors.New("no time variation in history")
	}
	b := (n*sumXY - sumX*sumY) / denom
	a := (sumY - b*sumX) / n

	// Calculate residuals for variance
	var sumResidualsSq float64
	for _, point := range history {
		x := float64(point.Timestamp.Unix() - startTime)
		y := valueExtractor(point)
		predicted := a + b*x
		residual := y - predicted
		sumResidualsSq += residual * residual
	}
	// degrees of freedom = n - 2
	if n > 2 {
		variance = sumResidualsSq / (n - 2)
	} else {
		variance = 0 // Not enough points for variance
	}

	return b, variance, a, nil
}
