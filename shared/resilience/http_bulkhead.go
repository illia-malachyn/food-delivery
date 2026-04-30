package resilience

import (
	"context"
	"net/http"
	"time"
)

func NewBulkheadRoundTripper(base http.RoundTripper, maxConcurrent int) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if maxConcurrent <= 0 {
		return base
	}

	slots := make(chan struct{}, maxConcurrent)
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
			return base.RoundTrip(req)
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})
}

func ContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}
