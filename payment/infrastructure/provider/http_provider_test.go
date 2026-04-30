package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/payment/infrastructure/provider"
	"github.com/illia-malachyn/food-delivery/shared/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPPaymentProviderUsesCircuitBreaker(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: resilience.NewCircuitBreakerRoundTripper(
			http.DefaultTransport,
			resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
				FailureThreshold: 1,
				OpenTimeout:      time.Minute,
			}),
		),
	}
	paymentProvider := provider.NewHTTPPaymentProvider(server.URL, client)

	require.Error(t, paymentProvider.Capture(context.Background(), "payment-1", 1000, "USD"))
	err := paymentProvider.Capture(context.Background(), "payment-1", 1000, "USD")
	require.ErrorIs(t, err, resilience.ErrCircuitOpen)
	assert.Equal(t, 1, calls)
}

func TestHTTPPaymentProviderRetriesRetryableStatuses(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: resilience.NewCircuitBreakerRoundTripper(
			resilience.NewRetryRoundTripper(http.DefaultTransport, resilience.RetryPolicy{
				MaxAttempts: 2,
				Sleep: func(context.Context, time.Duration) error {
					return nil
				},
			}),
			resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
				FailureThreshold: 1,
				OpenTimeout:      time.Minute,
			}),
		),
	}
	paymentProvider := provider.NewHTTPPaymentProvider(server.URL, client)

	require.NoError(t, paymentProvider.Capture(context.Background(), "payment-1", 1000, "USD"))
	assert.Equal(t, 2, calls)
}
