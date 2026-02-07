package client

import (
	"math/rand"
	"time"
)

// BackoffStrategy defines how to calculate the next wait time.
type BackoffStrategy interface {
	Next(attempt int) time.Duration
}

// ExponentialBackoff implements exponential backoff with jitter.
type ExponentialBackoff struct {
	Base   time.Duration
	Max    time.Duration
	Factor float64
	Jitter float64 // 0.0 to 1.0
}

// DefaultBackoff returns a sensible default strategy.
// Base: 100ms, Max: 5s, Factor: 2.0, Jitter: 0.2
func DefaultBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		Base:   100 * time.Millisecond,
		Max:    5 * time.Second,
		Factor: 2.0,
		Jitter: 0.2,
	}
}

// Next calculates the wait duration for the given attempt (0-based).
func (b *ExponentialBackoff) Next(attempt int) time.Duration {
	if attempt < 0 {
		return b.Base
	}

	// Calculate exponential delay: Base * Factor^attempt
	delay := float64(b.Base)
	for i := 0; i < attempt; i++ {
		delay *= b.Factor
	}

	// Apply cap
	if delay > float64(b.Max) {
		delay = float64(b.Max)
	}

	// Apply jitter: delay = delay * (1 ± Jitter)
	// Actually, usually it's better to do: delay = delay * (1 + rand(-Jitter, +Jitter))
	// Or simply random between [0, delay) for "Full Jitter" which is best for thundering herds.
	// But let's stick to standard +/- jitter for now as it's more predictable for latency.
	// delay + delay * (rand.Float64()*2 - 1) * Jitter

	if b.Jitter > 0 {
		jitterFactor := (rand.Float64()*2 - 1) * b.Jitter // Range [-Jitter, +Jitter]
		delay += delay * jitterFactor
	}

	if delay < 0 {
		return 0
	}

	return time.Duration(delay)
}
