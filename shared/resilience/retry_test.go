package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryPolicyRetriesWithExponentialBackoff(t *testing.T) {
	t.Parallel()

	transientErr := errors.New("transient")
	attempts := 0
	var delays []time.Duration
	policy := NewRetryPolicy(RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Jitter:         0,
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	})

	err := policy.Do(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return transientErr
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}, delays)
}

func TestRetryPolicyStopsOnNonRetryableError(t *testing.T) {
	t.Parallel()

	permanentErr := errors.New("permanent")
	attempts := 0
	policy := NewRetryPolicy(RetryPolicy{
		MaxAttempts: 5,
		IsRetryable: func(error) bool {
			return false
		},
		Sleep: func(context.Context, time.Duration) error {
			t.Fatal("sleep should not be called for non-retryable errors")
			return nil
		},
	})

	err := policy.Do(context.Background(), func(context.Context) error {
		attempts++
		return permanentErr
	})

	require.ErrorIs(t, err, permanentErr)
	assert.Equal(t, 1, attempts)
}
