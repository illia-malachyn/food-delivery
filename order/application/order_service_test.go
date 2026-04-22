package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/order/application"
	"github.com/illia-malachyn/food-delivery/order/domain"
	mockapp "github.com/illia-malachyn/food-delivery/order/application/mocks"
	"github.com/stretchr/testify/mock"
)

func TestOrderServiceCreate_UsesExpecterMocks(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	upcastedEvents := []application.IntegrationEvent{
		application.OrderPlacedEventV2{
			Version:    2,
			OrderID:    "id-upcasted",
			CustomerID: "user-1",
			ItemID:     "item-1",
			Quantity:   2,
			OccurredAt: time.Now(),
			Source:     "order-service",
		},
	}

	upcaster.EXPECT().
		Upcast(mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			if len(events) != 1 {
				return false
			}

			placed, ok := events[0].(application.OrderPlacedEvent)
			return ok &&
				placed.Version == 1 &&
				placed.UserID == "user-1" &&
				placed.ItemID == "item-1" &&
				placed.Quantity == 2
		})).
		Return(upcastedEvents).
		Once()

	repository.EXPECT().
		SaveOrder(
			mock.Anything,
			mock.MatchedBy(func(order *domain.Order) bool {
				return order != nil &&
					order.UserID() == "user-1" &&
					order.ItemID() == "item-1" &&
					order.Quantity() == 2
			}),
			upcastedEvents,
		).
		Return(nil).
		Once()

	err := service.Create(context.Background(), &application.OrderDTO{
		UserId:   "user-1",
		ItemId:   "item-1",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
}
