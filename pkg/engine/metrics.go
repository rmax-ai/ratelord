package engine

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RatelordUsage tracks the current used amount for a pool
	RatelordUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ratelord_usage",
			Help: "Current usage amount for a provider pool",
		},
		[]string{"provider_id", "pool_id"},
	)

	// RatelordLimit tracks the remaining capacity (or limit)
	RatelordLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ratelord_limit",
			Help: "Current remaining capacity for a provider pool",
		},
		[]string{"provider_id", "pool_id"},
	)

	// RatelordIntentTotal tracks the number of intents processed
	RatelordIntentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelord_intent_total",
			Help: "Total number of intents processed",
		},
		[]string{"identity_id", "decision"},
	)

	// RatelordForecastSeconds tracks the predicted time to exhaustion
	RatelordForecastSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ratelord_forecast_seconds",
			Help: "Predicted seconds until exhaustion",
		},
		[]string{"provider_id", "pool_id"},
	)
)

func init() {
	// Register metrics with the default registry
	prometheus.MustRegister(RatelordUsage)
	prometheus.MustRegister(RatelordLimit)
	prometheus.MustRegister(RatelordIntentTotal)
	prometheus.MustRegister(RatelordForecastSeconds)
}
