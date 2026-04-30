package resilience

import (
	"net/http"
	"time"
)

const timeoutResponseBody = `{"error":"request timeout"}`

func NewTimeoutHandler(next http.Handler, timeout time.Duration) http.Handler {
	if timeout <= 0 {
		return next
	}

	return http.TimeoutHandler(next, timeout, timeoutResponseBody)
}
