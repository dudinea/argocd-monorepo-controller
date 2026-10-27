package controller

import (
	"math"
	"math/rand/v2"
	"time"
)

// Default values of the retry policy of the application queue.
const (
	DefaultRetryMaxAttempts   = 5
	DefaultRetryBaseDelay     = 10 * time.Second
	DefaultRetryMaxDelay      = 5 * time.Minute
	DefaultRetryBackoffFactor = 2.0
	DefaultRetryJitter        = 0.2
)

// RetryPolicy defines how an application whose handling failed with a
// transient error is retried.
type RetryPolicy struct {
	// MaxAttempts is the number of retries of a single application event.
	// Failed applications are not retried at all when it is not positive.
	MaxAttempts int
	// BaseDelay is the delay before the first retry.
	BaseDelay time.Duration
	// MaxDelay caps the delay between the retries. It is not capped when the
	// value is not positive.
	MaxDelay time.Duration
	// Factor is the multiplier applied to the delay of every next retry.
	Factor float64
	// Jitter is the fraction by which the delay is randomly varied, so that
	// the applications which failed together are not all retried at once.
	Jitter float64
}

// DefaultRetryPolicy returns the retry policy used unless it is configured.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: DefaultRetryMaxAttempts,
		BaseDelay:   DefaultRetryBaseDelay,
		MaxDelay:    DefaultRetryMaxDelay,
		Factor:      DefaultRetryBackoffFactor,
		Jitter:      DefaultRetryJitter,
	}
}

// Enabled reports whether the applications which failed are retried at all.
func (p RetryPolicy) Enabled() bool {
	return p.MaxAttempts > 0
}

// Backoff returns the delay before the given retry attempt, counted from 1.
// The delay grows exponentially with the number of the attempt, is never
// longer than MaxDelay, and is randomized by the Jitter fraction.
func (p RetryPolicy) Backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := p.clampDelay(float64(p.BaseDelay) * math.Pow(p.Factor, float64(attempt-1)))
	if p.Jitter <= 0 {
		return delay
	}
	// The delay is capped before the jitter is applied, so that the delays of
	// the applications which all reached the cap still differ from each other.
	return p.clampDelay(float64(delay) * (1 + p.Jitter*(2*rand.Float64()-1)))
}

// clampDelay keeps the delay within MaxDelay, and within the range a
// time.Duration can hold when the exponential growth overflows it.
func (p RetryPolicy) clampDelay(delay float64) time.Duration {
	maxDelay := time.Duration(math.MaxInt64)
	if p.MaxDelay > 0 {
		maxDelay = p.MaxDelay
	}
	switch {
	case delay < 0:
		return 0
	case delay >= float64(maxDelay):
		// converting a float which does not fit into a time.Duration, such as
		// the infinity the exponential growth reaches, overflows it
		return maxDelay
	default:
		return time.Duration(delay)
	}
}

// retryTimer is the handle of a retry which was scheduled but did not run yet.
type retryTimer interface {
	// Stop prevents the scheduled function from being run. It reports false
	// when the function already ran or was already stopped.
	Stop() bool
}

// timerFactory runs f after the delay d and returns the handle of the
// scheduled call. It is the only place where the application queue depends on
// the wall clock, which lets the tests run the retries when they want to.
type timerFactory func(d time.Duration, f func()) retryTimer

func realTimer(d time.Duration, f func()) retryTimer {
	return time.AfterFunc(d, f)
}
