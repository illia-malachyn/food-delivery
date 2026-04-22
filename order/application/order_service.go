package application

import (
	"context"

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

func (s *OrderService) Create(ctx context.Context, orderDTO *OrderDTO) error {
	order, err := domain.NewOrder(orderDTO.UserId, orderDTO.ItemId, orderDTO.Quantity)
	if err != nil {
		return err
	}

	if err = order.Place(); err != nil {
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
		}
	}

	return integrationEvents
}
