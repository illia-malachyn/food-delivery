package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkPaidHandler_Success(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/payments/"+payment.ID()+"/pay", nil)
	req.SetPathValue("id", payment.ID())
	rec := httptest.NewRecorder()

	MarkPaidHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, domain.PaymentStatusPaid, payment.Status())
}

func TestMarkPaidHandler_NotFound(t *testing.T) {
	t.Parallel()

	service := application.NewPaymentService(repositoryStub{
		getByIDFn: func(_ context.Context, _ string) (*domain.Payment, error) {
			return nil, application.ErrPaymentNotFound
		},
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/payments/missing/pay", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	MarkPaidHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
