package rpcchain

import (
	"context"
	"time"
)

// Throttle limits outbound RPC request rate.
type Throttle struct {
	interval time.Duration
	last     time.Time
}

func NewThrottle(requestsPerSec float64) *Throttle {
	if requestsPerSec <= 0 {
		requestsPerSec = 10
	}
	return &Throttle{interval: time.Duration(float64(time.Second) / requestsPerSec)}
}

func (t *Throttle) Wait(ctx context.Context) error {
	if t == nil || t.interval <= 0 {
		return nil
	}
	elapsed := time.Since(t.last)
	if elapsed >= t.interval {
		t.last = time.Now()
		return nil
	}
	wait := t.interval - elapsed
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		t.last = time.Now()
		return nil
	}
}
