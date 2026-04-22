package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/order/application"
	mockapp "github.com/illia-malachyn/food-delivery/order/application/mocks"
	"github.com/illia-malachyn/food-delivery/order/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
}

func TestOrderServiceConfirm_DoesNotPublishIntegrationEvents(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	order, err := domain.ReconstructOrder("id-42", "user-1", "item-1", 2, domain.OrderStatusPlaced)
	require.NoError(t, err)

	repository.EXPECT().
		GetOrderById(mock.Anything, "id-42").
		Return(order, nil).
		Once()

	upcaster.EXPECT().
		Upcast(mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			return len(events) == 0
		})).
		Return([]application.IntegrationEvent{}).
		Once()

	repository.EXPECT().
		SaveOrder(
			mock.Anything,
			mock.MatchedBy(func(saved *domain.Order) bool {
				return saved != nil && saved.Status() == domain.OrderStatusConfirmed
			}),
			[]application.IntegrationEvent{},
		).
		Return(nil).
		Once()

	err = service.Confirm(context.Background(), "id-42")
	require.NoError(t, err)
}

func TestOrderServiceCancel_PublishesOrderCancelledEvent(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	order, err := domain.ReconstructOrder("id-7", "user-2", "item-9", 1, domain.OrderStatusPlaced)
	require.NoError(t, err)

	repository.EXPECT().
		GetOrderById(mock.Anything, "id-7").
		Return(order, nil).
		Once()

	upcaster.EXPECT().
		Upcast(mock.MatchedBy(func(events []application.IntegrationEvent) bool {
			if len(events) != 1 {
				return false
			}
			cancelled, ok := events[0].(application.OrderCancelledEvent)
			return ok && cancelled.Version == 1 && cancelled.OrderID == "id-7" && cancelled.Reason == "payment failed"
		})).
		RunAndReturn(func(events []application.IntegrationEvent) []application.IntegrationEvent {
			return events
		}).
		Once()

	repository.EXPECT().
		SaveOrder(
			mock.Anything,
			mock.MatchedBy(func(saved *domain.Order) bool {
				return saved != nil && saved.Status() == domain.OrderStatusCancelled
			}),
			mock.MatchedBy(func(events []application.IntegrationEvent) bool {
				if len(events) != 1 {
					return false
				}
				_, ok := events[0].(application.OrderCancelledEvent)
				return ok
			}),
		).
		Return(nil).
		Once()

	err = service.Cancel(context.Background(), "id-7", "payment failed")
	require.NoError(t, err)
}

func TestOrderServiceConfirm_PropagatesRepositoryError(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	repository.EXPECT().
		GetOrderById(mock.Anything, "missing").
		Return(nil, errors.New("not found")).
		Once()

	err := service.Confirm(context.Background(), "missing")
	require.Error(t, err)
}

func TestOrderServiceCancel_RejectsConfirmedOrder(t *testing.T) {
	repository := mockapp.NewOrderRepository(t)
	upcaster := mockapp.NewEventUpcaster(t)
	service := application.NewOrderService(repository, upcaster)

	order, err := domain.ReconstructOrder("id-9", "user-3", "item-3", 1, domain.OrderStatusConfirmed)
	require.NoError(t, err)

	repository.EXPECT().
		GetOrderById(mock.Anything, "id-9").
		Return(order, nil).
		Once()

	err = service.Cancel(context.Background(), "id-9", "too late")
	assert.ErrorIs(t, err, domain.ErrInvalidStateTransition)
}
