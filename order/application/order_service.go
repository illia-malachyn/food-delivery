package application

import (
	"context"
	"fmt"

	"github.com/illia-malachyn/food-delivery/order/domain"
)

type OrderService struct {
	orderRepository OrderRepository
	eventUpcaster   EventUpcaster
}

func NewOrderService(repository OrderRepository, eventUpcaster EventUpcaster) *OrderService {
	return &OrderService{
		orderRepository: repository,
		eventUpcaster:   eventUpcaster,
	}
}

func (s *OrderService) Create(ctx context.Context, orderDTO *OrderDTO) (string, error) {
	if orderDTO == nil {
		return "", fmt.Errorf("orderDTO is required")
	}

	order, err := domain.NewOrder(orderDTO.UserId, orderDTO.ItemId, orderDTO.Quantity)
	if err != nil {
		return "", err
	}

	if err = order.Place(); err != nil {
		return "", err
	}

	integrationEvents := mapToIntegrationEvents(order.FlushEvents())
	upcastedEvents := s.eventUpcaster.Upcast(integrationEvents)

	if err = s.orderRepository.SaveOrder(ctx, order, upcastedEvents); err != nil {
		return "", err
	}

	return order.ID(), nil
}

func (s *OrderService) Confirm(ctx context.Context, orderID string) error {
	order, err := s.orderRepository.GetOrderById(ctx, orderID)
	if err != nil {
		return err
	}

	if err = order.Confirm(); err != nil {
		return err
	}

	integrationEvents := mapToIntegrationEvents(order.FlushEvents())
	upcastedEvents := s.eventUpcaster.Upcast(integrationEvents)

	return s.orderRepository.SaveOrder(ctx, order, upcastedEvents)
}

func (s *OrderService) Cancel(ctx context.Context, orderID, reason string) error {
	order, err := s.orderRepository.GetOrderById(ctx, orderID)
	if err != nil {
		return err
	}

	if err = order.Cancel(reason); err != nil {
		return err
	}

	integrationEvents := mapToIntegrationEvents(order.FlushEvents())
	upcastedEvents := s.eventUpcaster.Upcast(integrationEvents)

	return s.orderRepository.SaveOrder(ctx, order, upcastedEvents)
}

func mapToIntegrationEvents(domainEvents []domain.DomainEvent) []IntegrationEvent {
	integrationEvents := make([]IntegrationEvent, 0, len(domainEvents))

	for _, event := range domainEvents {
		switch e := event.(type) {
		case domain.OrderPlacedEvent:
			integrationEvents = append(integrationEvents, OrderPlacedEvent{
				Version:    1,
				OrderID:    e.OrderID,
				UserID:     e.UserID,
				ItemID:     e.ItemID,
				Quantity:   e.Quantity,
				OccurredAt: e.OccurredAt,
			})
		case domain.OrderCancelledEvent:
			integrationEvents = append(integrationEvents, OrderCancelledEvent{
				Version:    1,
				OrderID:    e.OrderID,
				Reason:     e.Reason,
				OccurredAt: e.OccurredAt,
			})
		}
	}

	return integrationEvents
}
