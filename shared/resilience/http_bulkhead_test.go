package resilience

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBulkheadRoundTripperHonorsContextWhenFull(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	base := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		entered <- struct{}{}
		<-release
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})
	transport := NewBulkheadRoundTripper(base, 1)

	firstReq, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)
	firstDone := make(chan error, 1)
	go func() {
		_, err := transport.RoundTrip(firstReq)
		firstDone <- err
	}()
	<-entered

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	secondReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(secondReq)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	close(release)
	require.NoError(t, <-firstDone)
}
