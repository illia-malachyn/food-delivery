package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/illia-malachyn/food-delivery/order/application"
	mockapp "github.com/illia-malachyn/food-delivery/order/application/mocks"
)

func TestCreateOrderHandler_ReturnsCreatedOrderID(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	upcaster.EXPECT().
		Upcast(mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			return len(events) == 1
		})).
		RunAndReturn(func(events []application.IntegrationEvent) []application.IntegrationEvent {
			return events
		}).
		Once()

	repository.EXPECT().
		SaveOrder(mock.Anything, mock.AnythingOfType("*domain.Order"), mock.Anything).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"user_id":"user-1","item_id":"item-1","quantity":2}`))
	rec := httptest.NewRecorder()

	CreateOrderHandler(service).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response CreateOrderResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	_, parseErr := uuid.Parse(response.ID)
	require.NoError(t, parseErr)
}
