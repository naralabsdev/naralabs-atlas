package rpcchain

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"strings"
	"time"
)

type RetryPolicy struct {
	MaxAttempts   int
	BaseBackoff   time.Duration
	MaxBackoff    time.Duration
	MaxRetryAfter time.Duration
	Jitter        bool
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   5,
		BaseBackoff:   500 * time.Millisecond,
		MaxBackoff:    30 * time.Second,
		MaxRetryAfter: 60 * time.Second,
		Jitter:        true,
	}
}

type RetryableError struct {
	Err        error
	Retryable  bool
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err, Retryable: isTransient(err)}
}

func WithRetryAfter(err error, after time.Duration) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err, Retryable: true, RetryAfter: after}
}

func Do(ctx context.Context, policy RetryPolicy, fn func(context.Context) error) error {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.BaseBackoff <= 0 {
		policy.BaseBackoff = 500 * time.Millisecond
	}
	if policy.MaxBackoff <= 0 {
		policy.MaxBackoff = 30 * time.Second
	}
	if policy.MaxRetryAfter <= 0 {
		policy.MaxRetryAfter = 60 * time.Second
	}

	var lastErr error
	backoff := policy.BaseBackoff
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		var re *RetryableError
		if !errors.As(err, &re) || !re.Retryable {
			return err
		}
		if attempt >= policy.MaxAttempts {
			break
		}

		wait := backoff
		if re.RetryAfter > 0 {
			wait = re.RetryAfter
			if wait > policy.MaxRetryAfter {
				wait = policy.MaxRetryAfter
			}
		} else if policy.Jitter {
			wait = jitterDuration(wait)
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		if re.RetryAfter <= 0 {
			backoff *= 2
			if backoff > policy.MaxBackoff {
				backoff = policy.MaxBackoff
			}
		}
	}
	return lastErr
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporarily unavailable") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary())
}

func jitterDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	// Uniform in [0.5x, 1.5x)
	factor := 0.5 + rand.Float64()
	return time.Duration(float64(base) * factor)
}
