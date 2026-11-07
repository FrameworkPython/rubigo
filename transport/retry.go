package transport

import (
	"context"
	"errors"
	"math"
	"time"
)

// RetryPolicy describes how a failed request should be retried.
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	RetryOn    func(status int, err error) bool
}

// DefaultRetryPolicy returns the retry policy used by ParsRubika by
// default: 5 attempts, 1-second base delay, capped at 30 seconds, and
// retry on transient network errors and 5xx responses.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 5,
		BaseDelay:  time.Second,
		MaxDelay:   30 * time.Second,
		RetryOn: func(status int, err error) bool {
			if err != nil {
				return true
			}
			return status == 0 || status >= 500 || status == 408 || status == 425 || status == 429
		},
	}
}

// Sleep waits for the exponential backoff of the supplied attempt
// number. The first attempt uses base delay; subsequent attempts double
// the delay each time, capped by `MaxDelay`.
func (p RetryPolicy) Sleep(ctx context.Context, attempt int) error {
	if p.BaseDelay <= 0 || attempt <= 0 {
		return nil
	}
	delay := time.Duration(math.Pow(2, float64(attempt-1))) * p.BaseDelay
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ShouldRetry reports whether the policy says the request should be
// retried for the given (status, err) pair.
func (p RetryPolicy) ShouldRetry(status int, err error) bool {
	if p.RetryOn == nil {
		return err != nil
	}
	return p.RetryOn(status, err)
}

// ErrMaxRetries is returned when a request exhausts all retries.
var ErrMaxRetries = errors.New("transport: max retries exceeded")
