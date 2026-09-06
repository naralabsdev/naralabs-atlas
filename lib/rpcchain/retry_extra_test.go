package rpcchain_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/lib/rpcchain"
)

func TestRetryStopsOnNonRetryable(t *testing.T) {
	attempts := 0
	err := rpcchain.Do(context.Background(), rpcchain.RetryPolicy{MaxAttempts: 5}, func(context.Context) error {
		attempts++
		return errors.New("invalid argument")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryTransientNetworkError(t *testing.T) {
	attempts := 0
	err := rpcchain.Do(context.Background(), rpcchain.RetryPolicy{
		MaxAttempts: 2,
		BaseBackoff: time.Millisecond,
		Jitter:      false,
	}, func(context.Context) error {
		attempts++
		if attempts == 1 {
			return rpcchain.Retryable(&net.OpError{Op: "dial", Err: errors.New("timeout")})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts=%d", attempts)
	}
}

func TestDefaultRetryPolicySafeDefaults(t *testing.T) {
	p := rpcchain.DefaultRetryPolicy()
	if p.MaxAttempts < 1 || p.BaseBackoff <= 0 {
		t.Fatalf("policy=%+v", p)
	}
}
