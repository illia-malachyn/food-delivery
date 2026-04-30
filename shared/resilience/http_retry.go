package resilience

import (
	"context"
	"net/http"
)

func NewRetryRoundTripper(base http.RoundTripper, policy RetryPolicy) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	policy = NewRetryPolicy(policy)
	if policy.IsRetryable == nil {
		policy.IsRetryable = isHTTPRetryableError
	}

	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var resp *http.Response
		err := policy.Do(req.Context(), func(ctx context.Context) error {
			attemptReq := req.Clone(ctx)
			if req.Body != nil && req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return err
				}
				attemptReq.Body = body
			}

			var roundTripErr error
			resp, roundTripErr = base.RoundTrip(attemptReq)
			if roundTripErr != nil {
				return roundTripErr
			}
			if isRetryableHTTPStatus(resp.StatusCode) {
				_ = resp.Body.Close()
				return HTTPStatusError{StatusCode: resp.StatusCode}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func isHTTPRetryableError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(HTTPStatusError)
	return ok
}

func isRetryableHTTPStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}
