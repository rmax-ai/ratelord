package forecast

import (
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine/currency"
)

// UsagePoint represents a point in time usage observation
type UsagePoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Used      int64             `json:"used"`
	Remaining int64             `json:"remaining"`
	Cost      currency.MicroUSD `json:"cost"`
}

// TimeToExhaustion represents the probabilistic time-to-exhaustion estimates
type TimeToExhaustion struct {
	P50Seconds int64 `json:"p50_seconds"`
	P90Seconds int64 `json:"p90_seconds"`
	P99Seconds int64 `json:"p99_seconds"`
}

// BurnRate represents the burn rate with mean and variance
type BurnRate struct {
	Mean     float64 `json:"mean"`
	Variance float64 `json:"variance"`
	Unit     string  `json:"unit"`
}

// Risk represents the risk assessment including probability of exhaustion before reset
type Risk struct {
	ProbabilityExhaustionBeforeReset float64 `json:"probability_exhaustion_before_reset"`
	SafetyMarginSeconds              int64   `json:"safety_margin_seconds"`
	TTRSeconds                       int64   `json:"ttr_seconds"`
}

// Forecast represents the complete forecast output
type Forecast struct {
	TTE          TimeToExhaustion `json:"tte"`
	Risk         Risk             `json:"risk"`
	BurnRate     BurnRate         `json:"burn_rate"`
	CostBurnRate BurnRate         `json:"cost_burn_rate"`
}

// Model defines the interface for prediction models
type Model interface {
	Predict(history []UsagePoint, currentRemaining int64, resetAt time.Time) (Forecast, error)
}
