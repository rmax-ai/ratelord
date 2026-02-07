package client

import (
	"testing"
	"time"
)

func TestExponentialBackoff_Next(t *testing.T) {
	b := &ExponentialBackoff{
		Base:   100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2.0,
		Jitter: 0.0, // Disable jitter for deterministic checks
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at Max
		{5, 1 * time.Second}, // Capped at Max
	}

	for _, tt := range tests {
		got := b.Next(tt.attempt)
		if got != tt.expected {
			t.Errorf("Next(%d) = %v; want %v", tt.attempt, got, tt.expected)
		}
	}
}

func TestExponentialBackoff_Jitter(t *testing.T) {
	b := &ExponentialBackoff{
		Base:   100 * time.Millisecond,
		Max:    5 * time.Second,
		Factor: 2.0,
		Jitter: 0.1, // 10% jitter
	}

	// Run multiple times to ensure we stay within bounds
	for i := 0; i < 100; i++ {
		got := b.Next(0)
		min := 90 * time.Millisecond  // 100 * 0.9
		max := 110 * time.Millisecond // 100 * 1.1

		if got < min || got > max {
			t.Errorf("Next(0) with jitter = %v; want between %v and %v", got, min, max)
		}
	}
}
