package engine

import "time"

type DelayController struct {
	Kp float64
}

func NewDelayController(kp float64) *DelayController {
	return &DelayController{Kp: kp}
}

func (dc *DelayController) CalculateWait(state PoolState, now time.Time, kp float64) time.Duration {
	if state.LatestForecast == nil {
		return 0
	}
	timeToReset := state.ResetAt.Sub(now)
	if timeToReset <= 0 {
		return 0
	}
	targetBurn := float64(state.Remaining) / timeToReset.Seconds()
	currentBurn := state.LatestForecast.BurnRate.Mean
	error := currentBurn - targetBurn
	if error <= 0 {
		return 0
	}
	kpToUse := dc.Kp
	if kp > 0 {
		kpToUse = kp
	}
	delaySeconds := kpToUse * (currentBurn/targetBurn - 1.0)
	if delaySeconds > 30 {
		delaySeconds = 30
	}
	return time.Duration(delaySeconds * float64(time.Second))
}
