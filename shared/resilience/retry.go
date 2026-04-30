package resilience

import (
	"context"
	"math/rand"
	"time"
)

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Jitter         float64
	IsRetryable    func(error) bool
	Sleep          func(context.Context, time.Duration) error
	RandInt63n     func(int64) int64
}

func NewRetryPolicy(config RetryPolicy) RetryPolicy {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = 100 * time.Millisecond
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 2 * time.Second
	}
	if config.Jitter < 0 {
		config.Jitter = 0
	}
	if config.Jitter > 1 {
		config.Jitter = 1
	}
	if config.IsRetryable == nil {
		config.IsRetryable = func(err error) bool { return err != nil }
	}
	if config.Sleep == nil {
		config.Sleep = func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	if config.RandInt63n == nil {
		config.RandInt63n = rand.Int63n
	}
	return config
}

func (p RetryPolicy) Do(ctx context.Context, operation func(context.Context) error) error {
	p = NewRetryPolicy(p)

	var lastErr error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = operation(ctx)
		if lastErr == nil || !p.IsRetryable(lastErr) || attempt == p.MaxAttempts {
			return lastErr
		}

		if err := p.Sleep(ctx, p.backoffForAttempt(attempt)); err != nil {
			return err
		}
	}

	return lastErr
}

func (p RetryPolicy) backoffForAttempt(attempt int) time.Duration {
	delay := p.InitialBackoff
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= p.MaxBackoff {
			delay = p.MaxBackoff
			break
		}
	}

	if delay > p.MaxBackoff {
		delay = p.MaxBackoff
	}

	if p.Jitter == 0 || delay <= 0 {
		return delay
	}

	spread := int64(float64(delay) * p.Jitter)
	if spread <= 0 {
		return delay
	}

	minDelay := int64(delay) - spread
	return time.Duration(minDelay + p.RandInt63n(spread*2+1))
}
