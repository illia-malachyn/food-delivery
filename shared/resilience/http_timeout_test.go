package resilience

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTimeoutHandlerReturnsServiceUnavailableWhenHandlerExceedsDeadline(t *testing.T) {
	t.Parallel()

	handler := NewTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		_, _ = w.Write([]byte("late"))
	}), time.Nanosecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.JSONEq(t, timeoutResponseBody, rec.Body.String())
}

func TestNewTimeoutHandlerAllowsNonPositiveTimeoutToDisableDeadline(t *testing.T) {
	t.Parallel()

	handler := NewTimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), 0)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}
