package rpcchain_test

import (
	"context"
	"errors"
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

func TestRetryHonorsRetryAfter(t *testing.T) {
	start := time.Now()
	attempts := 0
	err := rpcchain.Do(context.Background(), rpcchain.RetryPolicy{
		MaxAttempts:   2,
		BaseBackoff:   time.Second,
		MaxRetryAfter: time.Second,
	}, func(context.Context) error {
		attempts++
		if attempts == 1 {
			return rpcchain.WithRetryAfter(errors.New("429"), 50*time.Millisecond)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatal("expected retry-after wait")
	}
}
