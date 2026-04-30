package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/illia-malachyn/food-delivery/payment/application"
	"github.com/illia-malachyn/food-delivery/payment/domain"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentHandler_Success(t *testing.T) {
	t.Parallel()

	service := application.NewPaymentService(repositoryStub{
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{"order_id":"order-1","amount":1200,"currency":"USD"}`))
	rec := httptest.NewRecorder()

	CreatePaymentHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response createPaymentResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	_, err = uuid.Parse(response.ID)
	require.NoError(t, err)
}

func TestCreatePaymentHandler_InvalidJSON(t *testing.T) {
	t.Parallel()

	service := application.NewPaymentService(repositoryStub{
		saveFn: func(_ context.Context, _ *domain.Payment, _ []domain.DomainEvent) error {
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{"order_id":`))
	rec := httptest.NewRecorder()

	CreatePaymentHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
