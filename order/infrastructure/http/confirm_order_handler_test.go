package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illia-malachyn/food-delivery/order/application"
	mockapp "github.com/illia-malachyn/food-delivery/order/application/mocks"
	"github.com/illia-malachyn/food-delivery/order/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfirmOrderHandler_Success(t *testing.T) {
	t.Parallel()

	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	order, err := domain.ReconstructOrder("order-1", "user-1", "item-1", 2, domain.OrderStatusPlaced)
	require.NoError(t, err)

	repository.EXPECT().
		GetOrderById(mock.Anything, "order-1").
		Return(order, nil).
		Once()

	upcaster.EXPECT().
		Upcast(mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			return len(events) == 0
		})).
		Return([]application.IntegrationEvent{}).
		Once()

	repository.EXPECT().
		SaveOrder(mock.Anything, mock.MatchedBy(func(saved *domain.Order) bool {
			return saved != nil && saved.Status() == domain.OrderStatusConfirmed
		}), []application.IntegrationEvent{}).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/orders/order-1/confirm", nil)
	req.SetPathValue("id", "order-1")
	rec := httptest.NewRecorder()

	ConfirmOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestConfirmOrderHandler_ValidatesOrderID(t *testing.T) {
	t.Parallel()

	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	req := httptest.NewRequest(http.MethodPost, "/orders//confirm", nil)
	rec := httptest.NewRecorder()

	ConfirmOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConfirmOrderHandler_ReturnsBadRequestOnServiceError(t *testing.T) {
	t.Parallel()

	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	repository.EXPECT().
		GetOrderById(mock.Anything, "missing").
		Return(nil, errors.New("not found")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/orders/missing/confirm", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	ConfirmOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
