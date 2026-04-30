package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkFailedHandler_Success(t *testing.T) {
	t.Parallel()

	payment, err := domain.NewPayment("order-1", 1200, "USD")
	require.NoError(t, err)
	_ = payment.FlushEvents()

	service := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, id string) (*domain.Payment, error) {
			assert.Equal(t, payment.ID(), id)
			return payment, nil
		},
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/"+payment.ID()+"/fail", strings.NewReader(`{"reason":"declined"}`))
	req.SetPathValue("id", payment.ID())
	rec := httptest.NewRecorder()

	MarkFailedHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, domain.PaymentStatusFailed, payment.Status())
	assert.Equal(t, "declined", payment.FailureReason())
}

func TestMarkFailedHandler_InvalidJSON(t *testing.T) {
	t.Parallel()

	service := application.NewPaymentService(repositoryStub{})

	req := httptest.NewRequest(http.MethodPost, "/payments/any/fail", strings.NewReader(`{"reason":`))
	req.SetPathValue("id", "any")
	rec := httptest.NewRecorder()

	MarkFailedHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
