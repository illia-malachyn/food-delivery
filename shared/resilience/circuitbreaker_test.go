package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerOpensAfterFailureThreshold(t *testing.T) {
	t.Parallel()

	breaker := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenTimeout:      time.Minute,
	})
	operationErr := errors.New("provider unavailable")

	require.ErrorIs(t, breaker.Do(context.Background(), func(context.Context) error {
		return operationErr
	}), operationErr)
	require.ErrorIs(t, breaker.Do(context.Background(), func(context.Context) error {
		return operationErr
	}), operationErr)

	called := false
	err := breaker.Do(context.Background(), func(context.Context) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, ErrCircuitOpen)
	assert.False(t, called)
	assert.Equal(t, CircuitOpen, breaker.State())
}

func TestCircuitBreakerHalfOpenClosesAfterProbeSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now()
	breaker := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		OpenTimeout:      10 * time.Second,
		Now:              func() time.Time { return now },
	})
	operationErr := errors.New("provider unavailable")

	require.ErrorIs(t, breaker.Do(context.Background(), func(context.Context) error {
		return operationErr
	}), operationErr)
	assert.Equal(t, CircuitOpen, breaker.State())

	now = now.Add(10 * time.Second)
	require.NoError(t, breaker.Do(context.Background(), func(context.Context) error {
		return nil
	}))
	assert.Equal(t, CircuitClosed, breaker.State())
}
