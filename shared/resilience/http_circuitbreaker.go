package resilience

import (
	"context"
	"net/http"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func NewCircuitBreakerRoundTripper(base http.RoundTripper, breaker *CircuitBreaker) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if breaker == nil {
		breaker = NewCircuitBreaker(CircuitBreakerConfig{})
	}

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var resp *http.Response
		err := breaker.Do(req.Context(), func(ctx context.Context) error {
			var roundTripErr error
			resp, roundTripErr = base.RoundTrip(req)
			if roundTripErr != nil {
				return roundTripErr
			}
			if isRetryableHTTPStatus(resp.StatusCode) {
				return HTTPStatusError{StatusCode: resp.StatusCode}
			}
			return nil
		})
		if err != nil {
			if resp != nil {
				_ = resp.Body.Close()
			}
			return nil, err
		}
		return resp, nil
	})
}

type HTTPStatusError struct {
	StatusCode int
}

func (e HTTPStatusError) Error() string {
	return http.StatusText(e.StatusCode)
}
