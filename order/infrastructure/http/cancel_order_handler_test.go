package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/illia-malachyn/food-delivery/order/application"
	mockapp "github.com/illia-malachyn/food-delivery/order/application/mocks"
	"github.com/illia-malachyn/food-delivery/order/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCancelOrderHandler_Success(t *testing.T) {
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
			if len(events) != 1 {
				return false
			}
			cancelled, ok := events[0].(application.OrderCancelledEvent)
			return ok && cancelled.OrderID == "order-1" && cancelled.Reason == "payment failed"
		})).
		RunAndReturn(func(events []application.IntegrationEvent) []application.IntegrationEvent {
			return events
		}).
		Once()

	repository.EXPECT().
		SaveOrder(mock.Anything, mock.MatchedBy(func(saved *domain.Order) bool {
			return saved != nil && saved.Status() == domain.OrderStatusCancelled
		}), mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			return len(events) == 1
		})).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/orders/order-1/cancel", strings.NewReader(`{"reason":"payment failed"}`))
	req.SetPathValue("id", "order-1")
	rec := httptest.NewRecorder()

	CancelOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCancelOrderHandler_ValidatesOrderID(t *testing.T) {
	t.Parallel()

	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	req := httptest.NewRequest(http.MethodPost, "/orders//cancel", strings.NewReader(`{"reason":"payment failed"}`))
	rec := httptest.NewRecorder()

	CancelOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancelOrderHandler_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	req := httptest.NewRequest(http.MethodPost, "/orders/order-1/cancel", strings.NewReader(`{"reason":`))
	req.SetPathValue("id", "order-1")
	rec := httptest.NewRecorder()

	CancelOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
