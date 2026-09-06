package rpcchain

import (
	"context"
	"testing"
	"time"
)

func TestThrottleWaitImmediateWhenIdle(t *testing.T) {
	th := NewThrottle(100)
	start := time.Now()
	if err := th.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 20*time.Millisecond {
		t.Fatal("first wait should be immediate")
	}
}

func TestThrottleWaitRespectsContextCancel(t *testing.T) {
	th := NewThrottle(1)
	_ = th.Wait(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := th.Wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestNilThrottleIsNoop(t *testing.T) {
	var th *Throttle
	if err := th.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}
